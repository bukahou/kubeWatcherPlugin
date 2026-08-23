// atlhyper_master_v2/deployer/service.go
// Deployer 服务实现 — CD 轮询/渲染/部署
package deployer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/github"
	"AtlHyper/atlhyper_master_v2/mq"
	"AtlHyper/common/logger"
	"AtlHyper/model_v3/command"
)

type service struct {
	ghClient github.Client
	db       *database.DB
	bus      mq.Producer

	mu            sync.Mutex
	cancel        context.CancelFunc
	lastCommitSHA string
	deploying     map[string]bool // 正在部署的路径，防重复
}

// NewService 创建 Deployer 服务
func NewService(ghClient github.Client, db *database.DB, bus mq.Producer) Deployer {
	return &service{
		ghClient:  ghClient,
		db:        db,
		bus:       bus,
		deploying: make(map[string]bool),
	}
}

func (s *service) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	// 从 DB 恢复 lastCommitSHA，避免重启后首次轮询部署所有路径
	s.restoreLastCommitSHA(ctx)

	go s.pollLoop(ctx)
	logger.Info("[Deployer] started")
	return nil
}

// restoreLastCommitSHA 从最近的部署历史中恢复上次的 Config 仓库 commit SHA
func (s *service) restoreLastCommitSHA(ctx context.Context) {
	config := s.loadConfig(ctx)
	if config == nil {
		return
	}
	paths := parsePaths(config.Paths)
	if len(paths) == 0 {
		return
	}
	// 取所有路径中最新的一条部署记录的 commitSHA
	var latestTime time.Time
	var latestSHA string
	for _, p := range paths {
		record, err := s.db.DeployHistory.GetLatestByPath(ctx, config.ClusterID, p)
		if err != nil || record == nil {
			continue
		}
		if record.DeployedAt.After(latestTime) {
			latestTime = record.DeployedAt
			latestSHA = record.CommitSHA
		}
	}
	if latestSHA != "" {
		s.mu.Lock()
		s.lastCommitSHA = latestSHA
		s.mu.Unlock()
		logger.Info("[Deployer] restored lastCommitSHA from DB", "sha", latestSHA[:min(8, len(latestSHA))])
	}
}

func (s *service) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	logger.Info("[Deployer] stopped")
	return nil
}

func (s *service) pollLoop(ctx context.Context) {
	// 使用固定间隔轮询，每次动态读取配置（支持热更新）
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			config := s.loadConfig(ctx)
			if config == nil {
				continue // 配置尚未就绪，等待下一轮
			}
			if !config.AutoDeploy {
				continue
			}
			s.checkAndDeploy(ctx, config)
		}
	}
}

// loadConfig 从数据库加载部署配置（不依赖 clusters 表）
func (s *service) loadConfig(ctx context.Context) *database.DeployConfig {
	configs, err := s.db.DeployConfig.List(ctx)
	if err != nil || len(configs) == 0 {
		return nil
	}
	return configs[0]
}

func (s *service) checkAndDeploy(ctx context.Context, config *database.DeployConfig) {
	repo := config.RepoURL
	if repo == "" {
		return
	}

	// 获取最新 commit SHA
	sha, err := s.ghClient.GetLatestCommitSHA(ctx, repo, "main")
	if err != nil {
		logger.Error("[Deployer] failed to get latest commit", "error", err)
		return
	}

	s.mu.Lock()
	lastSHA := s.lastCommitSHA
	s.mu.Unlock()

	if sha == lastSHA {
		return
	}

	shortLast := lastSHA
	if len(shortLast) > 8 {
		shortLast = shortLast[:8]
	}
	logger.Info("[Deployer] new commit detected", "sha", sha[:min(8, len(sha))], "previous", shortLast)

	// 确定受影响的路径
	var affectedPaths []string
	var compareResult *github.CompareResult
	if lastSHA == "" {
		// 首次运行，部署所有配置的路径
		affectedPaths = parsePaths(config.Paths)
	} else {
		// 比较 commit 找出变更文件（同时获取 compare URL 和行数统计）
		cr, err := s.ghClient.CompareCommitsDetail(ctx, repo, lastSHA, sha)
		if err != nil {
			logger.Error("[Deployer] compare failed", "error", err)
			affectedPaths = parsePaths(config.Paths)
		} else {
			compareResult = cr
			affectedPaths = matchPaths(parsePaths(config.Paths), cr.Files)
		}
	}

	if len(affectedPaths) == 0 {
		s.mu.Lock()
		s.lastCommitSHA = sha
		s.mu.Unlock()
		return
	}

	// 部署每个受影响的路径（跳过正在部署的）
	for _, path := range affectedPaths {
		s.mu.Lock()
		if s.deploying[path] {
			s.mu.Unlock()
			logger.Info("[Deployer] skipping path (already deploying)", "path", path)
			continue
		}
		s.deploying[path] = true
		s.mu.Unlock()

		s.deployPath(ctx, config, path, sha, "auto", compareResult)
		s.clearDeploying(path)
	}

	s.mu.Lock()
	s.lastCommitSHA = sha
	s.mu.Unlock()
}

func (s *service) deployPath(ctx context.Context, config *database.DeployConfig, path, commitSHA, trigger string, compareResult *github.CompareResult) {
	clusterID := config.ClusterID

	record := &database.DeployHistory{
		ClusterID:    clusterID,
		Path:         path,
		CommitSHA:    commitSHA,
		ChangedFiles: "[]",
		DeployedAt:   time.Now(),
		Trigger:      trigger,
		Status:       "pending",
	}

	start := time.Now()

	// 使用 kustomize 构建 manifests
	manifests, err := s.kustomizeBuild(ctx, config.RepoURL, path, commitSHA)
	if err != nil {
		record.Status = "failed"
		record.ErrorMessage = fmt.Sprintf("kustomize build failed: %v", err)
		record.DurationMs = int(time.Since(start).Milliseconds())
		_ = s.db.DeployHistory.Create(ctx, record)
		logger.Error("[Deployer] kustomize build failed", "path", path, "error", err)
		return
	}

	// 从 manifests 提取 namespace
	record.Namespace = extractNamespace(manifests)

	// 统计资源数量
	record.ResourceTotal = strings.Count(manifests, "\nkind:")
	if record.ResourceTotal == 0 && strings.HasPrefix(manifests, "kind:") {
		record.ResourceTotal = 1
	}

	// 从 manifests 解析源码仓库信息（image tag → source SHA → GitHub API）
	s.enrichSourceInfo(ctx, record, manifests, config.RepoURL, compareResult)

	// 通过 MQ 发送 apply_manifests 指令
	cmd := &command.Command{
		ID:        fmt.Sprintf("deploy-%s-%d", path, time.Now().UnixMilli()),
		ClusterID: clusterID,
		Action:    command.ActionApplyManifests,
		Params: map[string]any{
			"manifests": manifests,
			"path":      path,
		},
		Source:    "deployer",
		CreatedAt: time.Now(),
	}

	if err := s.bus.EnqueueCommand(clusterID, mq.TopicOps, cmd); err != nil {
		record.Status = "failed"
		record.ErrorMessage = fmt.Sprintf("enqueue command failed: %v", err)
		record.DurationMs = int(time.Since(start).Milliseconds())
		_ = s.db.DeployHistory.Create(ctx, record)
		logger.Error("[Deployer] enqueue failed", "path", path, "error", err)
		return
	}

	// 等待结果（超时 120 秒）
	result, err := s.bus.WaitCommandResult(ctx, cmd.ID, 120*time.Second)
	if err != nil {
		record.Status = "failed"
		record.ErrorMessage = fmt.Sprintf("command timeout: %v", err)
	} else if result != nil && result.Success {
		record.Status = "success"
		// 从 Agent 返回的 Output 中解析实际资源变更数
		var applyResult struct {
			ResourceTotal   int    `json:"resourceTotal"`
			ResourceChanged int    `json:"resourceChanged"`
			ErrorMessage    string `json:"errorMessage"`
		}
		if json.Unmarshal([]byte(result.Output), &applyResult) == nil {
			record.ResourceTotal = applyResult.ResourceTotal
			record.ResourceChanged = applyResult.ResourceChanged
			if applyResult.ErrorMessage != "" {
				record.ErrorMessage = applyResult.ErrorMessage
			}
		}
	} else if result != nil {
		record.Status = "failed"
		record.ErrorMessage = result.Error
	}

	record.DurationMs = int(time.Since(start).Milliseconds())
	_ = s.db.DeployHistory.Create(ctx, record)
	logger.Info("[Deployer] deployed", "path", path, "status", record.Status, "durationMs", record.DurationMs)
}

func (s *service) SyncNow(ctx context.Context, path string) error {
	// 防重复：检查该路径是否正在部署
	s.mu.Lock()
	if s.deploying[path] {
		s.mu.Unlock()
		return fmt.Errorf("path %s is already deploying", path)
	}
	s.deploying[path] = true
	s.mu.Unlock()

	config := s.loadConfig(ctx)
	if config == nil {
		s.clearDeploying(path)
		return fmt.Errorf("no deploy config found")
	}

	sha, err := s.ghClient.GetLatestCommitSHA(ctx, config.RepoURL, "main")
	if err != nil {
		s.clearDeploying(path)
		return fmt.Errorf("get latest commit: %w", err)
	}

	// 手动同步时尝试获取 compare 信息
	var compareResult *github.CompareResult
	s.mu.Lock()
	lastSHA := s.lastCommitSHA
	s.mu.Unlock()
	if lastSHA != "" && lastSHA != sha {
		cr, err := s.ghClient.CompareCommitsDetail(ctx, config.RepoURL, lastSHA, sha)
		if err == nil {
			compareResult = cr
		}
	}

	// 使用独立 context，避免 HTTP 请求完成后 context 被 cancel
	go func() {
		defer s.clearDeploying(path)
		s.deployPath(context.Background(), config, path, sha, "manual", compareResult)
	}()
	return nil
}

func (s *service) clearDeploying(path string) {
	s.mu.Lock()
	delete(s.deploying, path)
	s.mu.Unlock()
}

func (s *service) GetPathStatus(ctx context.Context) ([]PathStatus, error) {
	config := s.loadConfig(ctx)
	if config == nil {
		return nil, nil
	}

	paths := parsePaths(config.Paths)
	var statuses []PathStatus
	for _, p := range paths {
		latest, _ := s.db.DeployHistory.GetLatestByPath(ctx, config.ClusterID, p)
		status := PathStatus{
			Path: p,
		}
		if latest != nil {
			status.Namespace = latest.Namespace
			status.LastSyncAt = latest.DeployedAt
			status.InSync = latest.Status == "success"
			status.ResourceCount = latest.ResourceTotal
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

// kustomizeBuild 从 GitHub 下载文件并执行 kustomize build
func (s *service) kustomizeBuild(ctx context.Context, repo, path, ref string) (string, error) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "kustomize-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	// 从 GitHub 下载路径下的所有文件
	entries, err := s.ghClient.ReadDirectory(ctx, repo, path, ref)
	if err != nil {
		return "", fmt.Errorf("read directory %s: %w", path, err)
	}

	for _, entry := range entries {
		if entry.Type != "file" {
			continue
		}
		content, err := s.ghClient.ReadFile(ctx, repo, filepath.Join(path, entry.Name), ref)
		if err != nil {
			return "", fmt.Errorf("read file %s: %w", entry.Name, err)
		}
		filePath := filepath.Join(tmpDir, entry.Name)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return "", err
		}
	}

	// 执行 kustomize build
	cmd := exec.CommandContext(ctx, "kubectl", "kustomize", tmpDir)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("kustomize build: %s", string(exitErr.Stderr))
		}
		return "", err
	}

	return string(output), nil
}

// parsePaths 解析 JSON 编码的路径字符串
func parsePaths(pathsStr string) []string {
	pathsStr = strings.TrimSpace(pathsStr)
	if pathsStr == "" || pathsStr == "[]" {
		return nil
	}
	// 简单解析: 去除方括号并按逗号分割
	pathsStr = strings.Trim(pathsStr, "[]")
	var result []string
	for _, p := range strings.Split(pathsStr, ",") {
		p = strings.Trim(strings.TrimSpace(p), `"`)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// matchPaths 找出受变更文件影响的配置路径
func matchPaths(configuredPaths []string, changedFiles []github.ChangedFile) []string {
	affected := make(map[string]bool)
	for _, f := range changedFiles {
		for _, p := range configuredPaths {
			if strings.HasPrefix(f.Filename, p+"/") || strings.HasPrefix(f.Filename, p) {
				affected[p] = true
			}
		}
	}
	var result []string
	for p := range affected {
		result = append(result, p)
	}
	return result
}

// extractNamespace 从 YAML manifests 中提取第一个 namespace
func extractNamespace(manifests string) string {
	for _, line := range strings.Split(manifests, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "namespace:") {
			ns := strings.TrimSpace(strings.TrimPrefix(line, "namespace:"))
			ns = strings.Trim(ns, `"'`)
			if ns != "" {
				return ns
			}
		}
	}
	return "default"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// === 源码仓库信息解析（source.json 方案） ===

// sourceJSON 是 CI 写入 Config 仓库的源码元数据文件
// 支持两种命名：
//   - 单服务: {path}/source.json（传统模式，如 AtlHyper）
//   - 多服务: {path}/source-{service}.json（微服务模式，如 Geass V2）
type sourceJSON struct {
	Repo   string `json:"repo"`   // 源码仓库 (e.g. "bukahou/geass-v2-auth")
	SHA    string `json:"sha"`    // 源码 commit SHA (full 40-char)
	Branch string `json:"branch"` // 源码分支 (e.g. "main")
}

// enrichSourceInfo 读取 source.json 解析源码仓库信息，填充到 DeployHistory
// configCompare 是 Config 仓库的 commit compare 结果，用于定位 source-*.json
func (s *service) enrichSourceInfo(ctx context.Context, record *database.DeployHistory, manifests, configRepo string, configCompare *github.CompareResult) {
	src := s.readSourceJSON(ctx, configRepo, record.Path, record.CommitSHA, configCompare)
	if src == nil {
		// source.json 不存在（CI 尚未适配），回退显示 Config 仓库 commit 信息
		s.enrichConfigInfo(ctx, record, configRepo)
		return
	}

	record.SourceRepo = src.Repo
	record.SourceCommitSHA = src.SHA

	// 获取源码 commit 详情
	srcCommit, err := s.ghClient.GetCommitByRef(ctx, src.Repo, src.SHA)
	if err == nil && srcCommit != nil {
		record.CommitMessage = srcCommit.Message
		record.CommitAuthor = srcCommit.Author
		record.CommitAvatarURL = srcCommit.AvatarURL
	}

	// 获取源码关联 PR
	srcPR, err := s.ghClient.GetPRByCommit(ctx, src.Repo, src.SHA)
	if err == nil && srcPR != nil {
		record.PRNumber = srcPR.Number
		record.PRTitle = srcPR.Title
		record.PRURL = srcPR.URL
	}

	// 源码变更文件：与上一次部署的源码 SHA 对比
	prevSHA := s.getPreviousSourceSHA(ctx, record.ClusterID, record.Path)
	if prevSHA != "" && prevSHA != src.SHA {
		srcCompare, err := s.ghClient.CompareCommitsDetail(ctx, src.Repo, prevSHA, src.SHA)
		if err == nil && srcCompare != nil {
			record.CompareURL = srcCompare.HTMLURL
			if filesData, err := json.Marshal(srcCompare.Files); err == nil {
				record.ChangedFiles = string(filesData)
			}
		}
	}

	logger.Info("[Deployer] resolved source info", "sourceRepo", src.Repo, "sourceSHA", src.SHA[:min(8, len(src.SHA))])
}

// enrichConfigInfo 回退方案：使用 Config 仓库的 commit 信息（source.json 不存在时）
func (s *service) enrichConfigInfo(ctx context.Context, record *database.DeployHistory, configRepo string) {
	commit, err := s.ghClient.GetCommitByRef(ctx, configRepo, record.CommitSHA)
	if err == nil && commit != nil {
		record.CommitMessage = commit.Message
		record.CommitAuthor = commit.Author
		record.CommitAvatarURL = commit.AvatarURL
	}
	logger.Info("[Deployer] using config repo commit info (no source.json)", "path", record.Path)
}

// readSourceJSON 从 Config 仓库读取 source 元数据
// 查找优先级：
//  1. 从 Config commit changed files 中找本次变更的 source-*.json（微服务模式）
//  2. 回退读取 {path}/source.json（传统模式）
func (s *service) readSourceJSON(ctx context.Context, configRepo, path, ref string, configCompare *github.CompareResult) *sourceJSON {
	// 优先：从 Config commit 的 changed files 中找 source-*.json
	if configCompare != nil {
		prefix := path + "/source-"
		for _, f := range configCompare.Files {
			if strings.HasPrefix(f.Filename, prefix) && strings.HasSuffix(f.Filename, ".json") {
				src := s.readSourceFile(ctx, configRepo, f.Filename, ref)
				if src != nil {
					return src
				}
			}
		}
	}

	// 回退：传统单文件 source.json
	return s.readSourceFile(ctx, configRepo, path+"/source.json", ref)
}

// readSourceFile 读取并解析单个 source JSON 文件
func (s *service) readSourceFile(ctx context.Context, configRepo, filePath, ref string) *sourceJSON {
	content, err := s.ghClient.ReadFile(ctx, configRepo, filePath, ref)
	if err != nil {
		return nil
	}
	var src sourceJSON
	if err := json.Unmarshal([]byte(content), &src); err != nil {
		logger.Error("[Deployer] failed to parse source json", "path", filePath, "error", err)
		return nil
	}
	if src.Repo == "" || src.SHA == "" {
		return nil
	}
	return &src
}

// getPreviousSourceSHA 从上一次部署记录获取源码 commit SHA
func (s *service) getPreviousSourceSHA(ctx context.Context, clusterID, path string) string {
	latest, err := s.db.DeployHistory.GetLatestByPath(ctx, clusterID, path)
	if err != nil || latest == nil {
		return ""
	}
	return latest.SourceCommitSHA
}

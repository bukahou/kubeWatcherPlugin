// atlhyper_master_v2/gateway/handler/admin/deploy.go
// 部署管理 Handler — 配置/kustomize 路径/历史
package admin

import (
	"context"
	"encoding/json"
	"net/http"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/deployer"
	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/github"
)

// DeployHandler 部署管理 Handler
// DeployQuery / DeployOps 本 handler 所需的最小 service 能力。
// 此前直接持有 database repository —— 绕过 service 层，
// 违反 CLAUDE.md「Gateway 直接访问 Database」禁令。
type DeployQuery interface {
	GetDeployConfig(ctx context.Context, clusterID string) (*database.DeployConfig, error)
	ListDeployHistory(ctx context.Context, opts database.DeployHistoryQueryOpts) ([]*database.DeployHistory, error)
	CountDeployHistory(ctx context.Context, opts database.DeployHistoryQueryOpts) (int, error)
	GetGitHubInstallation(ctx context.Context) (*database.GitHubInstallation, error)
}

type DeployOps interface {
	UpsertDeployConfig(ctx context.Context, cfg *database.DeployConfig) error
}

type DeployHandler struct {
	ghClient github.Client
	querySvc DeployQuery
	opsSvc   DeployOps
	deployer deployer.Deployer
}

// NewDeployHandler 创建 DeployHandler
func NewDeployHandler(
	ghClient github.Client,
	querySvc DeployQuery,
	opsSvc DeployOps,
) *DeployHandler {
	return &DeployHandler{
		ghClient: ghClient,
		querySvc: querySvc,
		opsSvc:   opsSvc,
	}
}

// SetDeployer 设置 Deployer（可选注入）
func (h *DeployHandler) SetDeployer(d deployer.Deployer) {
	h.deployer = d
}

// Status GET /api/deploy/status — 同步状态
func (h *DeployHandler) Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if h.deployer == nil {
		handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"data": []deployer.PathStatus{},
		})
		return
	}

	statuses, err := h.deployer.GetPathStatus(r.Context())
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if statuses == nil {
		statuses = []deployer.PathStatus{}
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data": statuses,
	})
}

// SyncNow POST /api/deploy/sync — 触发立即同步
func (h *DeployHandler) SyncNow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if h.deployer == nil {
		handler.WriteError(w, http.StatusServiceUnavailable, "Deployer 未启用")
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Path == "" {
		handler.WriteError(w, http.StatusBadRequest, "path required")
		return
	}

	if err := h.deployer.SyncNow(r.Context(), req.Path); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "sync triggered",
	})
}

// Rollback POST /api/deploy/rollback — 回滚到指定 commit
func (h *DeployHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Path            string `json:"path"`
		TargetCommitSha string `json:"targetCommitSha"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Path == "" || req.TargetCommitSha == "" {
		handler.WriteError(w, http.StatusBadRequest, "path and targetCommitSha required")
		return
	}

	// 当前为占位实现，完整回滚需要读取目标 commit 的文件
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message":         "rollback triggered",
		"path":            req.Path,
		"targetCommitSha": req.TargetCommitSha,
	})
}

// Config GET/PUT /api/deploy/config — 部署配置
func (h *DeployHandler) Config(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getConfig(w, r)
	case http.MethodPut:
		h.saveConfig(w, r)
	default:
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (h *DeployHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	clusterID := r.URL.Query().Get("clusterId")
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "缺少 clusterId 参数")
		return
	}

	config, err := h.querySvc.GetDeployConfig(r.Context(), clusterID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "获取配置失败")
		return
	}

	if config == nil {
		handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"data": nil,
		})
		return
	}

	// 解析 paths JSON
	var paths []string
	json.Unmarshal([]byte(config.Paths), &paths)

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]interface{}{
			"repoUrl":     config.RepoURL,
			"paths":       paths,
			"intervalSec": config.IntervalSec,
			"autoDeploy":  config.AutoDeploy,
			"clusterId":   config.ClusterID,
		},
	})
}

func (h *DeployHandler) saveConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClusterID   string   `json:"clusterId"`
		RepoURL     string   `json:"repoUrl"`
		Paths       []string `json:"paths"`
		IntervalSec int      `json:"intervalSec"`
		AutoDeploy  bool     `json:"autoDeploy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	if req.ClusterID == "" || req.RepoURL == "" {
		handler.WriteError(w, http.StatusBadRequest, "缺少必填参数")
		return
	}

	pathsJSON, _ := json.Marshal(req.Paths)
	if err := h.opsSvc.UpsertDeployConfig(r.Context(), &database.DeployConfig{
		ClusterID:   req.ClusterID,
		RepoURL:     req.RepoURL,
		Paths:       string(pathsJSON),
		IntervalSec: req.IntervalSec,
		AutoDeploy:  req.AutoDeploy,
	}); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "保存配置失败")
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "配置已保存",
	})
}

// KustomizePaths GET /api/deploy/kustomize-paths — 扫描仓库中的 kustomize 路径
func (h *DeployHandler) KustomizePaths(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	repo := r.URL.Query().Get("repo")
	if repo == "" {
		handler.WriteError(w, http.StatusBadRequest, "缺少 repo 参数")
		return
	}

	paths, err := h.ghClient.ScanKustomizePaths(r.Context(), repo, "")
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "扫描 kustomize 路径失败: "+err.Error())
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data": paths,
	})
}

// TestConnection POST /api/deploy/test-connection — 测试 GitHub 连接
func (h *DeployHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	inst, err := h.querySvc.GetGitHubInstallation(r.Context())
	if err != nil || inst == nil {
		handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"data": map[string]bool{"success": false},
		})
		return
	}

	// 测试能否获取 Installation Token
	_, err = h.ghClient.GetInstallationToken(r.Context())
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]bool{"success": err == nil},
	})
}

// History GET /api/deploy/history — 部署历史
func (h *DeployHandler) History(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("clusterId")
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "缺少 clusterId 参数")
		return
	}

	opts := database.DeployHistoryQueryOpts{
		ClusterID: clusterID,
		Path:      r.URL.Query().Get("path"),
		Limit:     50,
	}

	records, err := h.querySvc.ListDeployHistory(r.Context(), opts)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "获取部署历史失败")
		return
	}

	total, _ := h.querySvc.CountDeployHistory(r.Context(), opts)

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data":  records,
		"total": total,
	})
}

// atlhyper_master_v2/aiops/enricher/enricher.go
// AIOps 事件 AI 增强编排服务
// 通过 ai.AIService 接口调用 LLM，不直接操作 ai/llm 包
package enricher

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/ai"
	"AtlHyper/atlhyper_master_v2/ai/prompts"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/common/logger"
)

var log = logger.Module("AIOps-Enricher")

// SummarizeResponse 事件摘要响应
type SummarizeResponse struct {
	IncidentID        string           `json:"incidentId"`
	Summary           string           `json:"summary"`
	RootCauseAnalysis string           `json:"rootCauseAnalysis"`
	Recommendations   []Recommendation `json:"recommendations"`
	SimilarIncidents  []SimilarMatch   `json:"similarIncidents"`
	GeneratedAt       int64            `json:"generatedAt"`
	ReportID          int64            `json:"reportId,omitempty"`
	FromCache         bool             `json:"fromCache"`
}

// Recommendation 处置建议
type Recommendation struct {
	Priority    int    `json:"priority"`
	Action      string `json:"action"`
	Reason      string `json:"reason"`
	Impact      string `json:"impact"`
	IsAutomatic bool   `json:"isAutomatic"`
}

// SimilarMatch 相似事件匹配
type SimilarMatch struct {
	IncidentID string  `json:"incidentId"`
	Similarity float64 `json:"similarity"`
	RootCause  string  `json:"rootCause"`
	Resolution string  `json:"resolution"`
	OccurredAt string  `json:"occurredAt"`
}

// Token 预估常量
const (
	MaxPromptChars  = 16000 // Prompt 最大字符数（~4000 tokens）
	WarnPromptChars = 12000 // 超过此值记录 warning 日志
)

// 默认配置
const (
	defaultCooldown       = 60 * time.Second // Rate Limit 冷却时间
	defaultMaxCache       = 200              // 最大缓存条目数
	defaultCacheTTL       = 24 * time.Hour   // 缓存 TTL
	rateLimitExpiry       = 1 * time.Hour    // Rate Limit 条目过期清理阈值
	maxConcurrentAnalysis = 3                // 最多同时进行的后台分析数
	analysisTimeout       = 5 * time.Minute  // 深度分析超时
)

// 可缓存的事件状态（已结束，数据不再变化）
var cacheableStates = map[string]bool{
	"recovery": true,
	"stable":   true,
}

// cachedResult 缓存的 AI 分析结果
type cachedResult struct {
	response *SummarizeResponse
	cachedAt time.Time
}

// Enricher AIOps 事件 AI 增强编排服务
type Enricher struct {
	incidentRepo database.AIOpsIncidentRepository
	reportRepo   database.AIReportRepository // 报告持久化（可选）
	aiService    ai.AIService                // 通过接口调用 AI 能力
	store        datahub.Store               // 数据存储（读取 OTelSnapshot，可选）

	// 后台自动触发器（可选）
	bgTrigger *backgroundTrigger

	// 并发信号量：限制同时进行的后台分析数量
	concurrencySem chan struct{}

	// Rate Limit: 同一事件在 cooldown 内不允许重复调用 LLM
	rateMu    sync.Mutex
	lastCalls map[string]time.Time // incidentID → 上次调用时间
	cooldown  time.Duration

	// 缓存: 已完结事件的 AI 结果
	cacheMu  sync.RWMutex
	cache    map[string]*cachedResult
	maxCache int
	cacheTTL time.Duration
}

// NewEnricher 创建 AIOps 事件 AI 增强编排服务
func NewEnricher(
	incidentRepo database.AIOpsIncidentRepository,
	reportRepo database.AIReportRepository,
	aiService ai.AIService,
) *Enricher {
	return &Enricher{
		incidentRepo:   incidentRepo,
		reportRepo:     reportRepo,
		aiService:      aiService,
		concurrencySem: make(chan struct{}, maxConcurrentAnalysis),
		lastCalls:      make(map[string]time.Time),
		cooldown:       defaultCooldown,
		cache:          make(map[string]*cachedResult),
		maxCache:       defaultMaxCache,
		cacheTTL:       defaultCacheTTL,
	}
}

// SetStore 设置数据存储（用于读取 OTelSnapshot 丰富事件上下文）
func (e *Enricher) SetStore(store datahub.Store) {
	e.store = store
}

// EnableBackgroundTrigger 启用后台自动触发器
func (e *Enricher) EnableBackgroundTrigger(budgetRepo database.AIRoleBudgetRepository) {
	e.bgTrigger = newBackgroundTrigger(e, budgetRepo)
	log.Info("后台自动分析触发器已启用")
}

// NotifyIncidentEvent 通知事件创建/升级（供 AIOps 引擎回调）
func (e *Enricher) NotifyIncidentEvent(incidentID, severity, trigger string) {
	if e.bgTrigger == nil {
		return
	}
	e.bgTrigger.Submit(incidentID, severity, trigger)
}

// TriggerAnalysis 手动触发深度分析（异步）
func (e *Enricher) TriggerAnalysis(incidentID string) {
	log.Info("触发深度分析", "incident", incidentID)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), analysisTimeout)
		defer cancel()

		if err := e.runAnalysis(ctx, incidentID, "manual"); err != nil {
			log.Error("深度分析失败", "incident", incidentID, "err", err)
		} else {
			log.Info("深度分析完成", "incident", incidentID)
		}
	}()
}

// Stop 停止后台任务
func (e *Enricher) Stop() {
	if e.bgTrigger != nil {
		e.bgTrigger.Stop()
	}
}

// ==================== Summarize（background 角色）====================

// Summarize 生成事件 AI 摘要（用户手动触发）
// 优先查询已有报告，无报告时才调 LLM
func (e *Enricher) Summarize(ctx context.Context, incidentID string) (*SummarizeResponse, error) {
	// 优先查询已有 background 报告
	if e.reportRepo != nil {
		reports, err := e.reportRepo.ListByIncident(ctx, incidentID)
		if err == nil && len(reports) > 0 {
			for _, r := range reports {
				if r.Role == "background" {
					return reportToSummarizeResponse(r), nil
				}
			}
		}
	}

	// 无已有报告，调 LLM 生成
	result, incident, completeResult, err := e.summarizeCore(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	e.saveBackgroundReport(ctx, incident, result, "manual", completeResult)
	return result, nil
}

// SummarizeBackground 后台自动分析（指定 trigger 类型）
func (e *Enricher) SummarizeBackground(ctx context.Context, incidentID, trigger string) (*SummarizeResponse, error) {
	// 并发上限检查
	select {
	case e.concurrencySem <- struct{}{}:
		defer func() { <-e.concurrencySem }()
	default:
		return nil, fmt.Errorf("后台分析并发上限（%d），跳过", maxConcurrentAnalysis)
	}

	result, incident, completeResult, err := e.summarizeCore(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	e.saveBackgroundReport(ctx, incident, result, trigger, completeResult)
	return result, nil
}

// summarizeCore 核心分析流程（通过 ai.AIService.Complete 调用 LLM）
// 流程: 缓存 → Rate Limit → 查数据 → 构建 Prompt → LLM → 解析 → 缓存
func (e *Enricher) summarizeCore(ctx context.Context, incidentID string) (*SummarizeResponse, *database.AIOpsIncident, *ai.CompleteResult, error) {
	// 1. 查缓存
	if cached := e.getCache(incidentID); cached != nil {
		log.Debug("缓存命中", "incident", incidentID)
		return cached, nil, nil, nil
	}

	// 2. Rate Limit 检查
	if err := e.checkRateLimit(incidentID); err != nil {
		return nil, nil, nil, err
	}

	// 3. 查询事件数据
	incident, err := e.incidentRepo.GetByID(ctx, incidentID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("查询事件失败: %w", err)
	}
	if incident == nil {
		return nil, nil, nil, fmt.Errorf("事件不存在: %s", incidentID)
	}

	entities, err := e.incidentRepo.GetEntities(ctx, incidentID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("查询受影响实体失败: %w", err)
	}

	timeline, err := e.incidentRepo.GetTimeline(ctx, incidentID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("查询时间线失败: %w", err)
	}

	// 4. 查询历史相似事件
	var historical []*database.AIOpsIncident
	if incident.RootCause != "" {
		since := time.Now().Add(-90 * 24 * time.Hour)
		historical, _ = e.incidentRepo.ListByEntity(ctx, incident.RootCause, since)
	}

	// 5. 构建 Prompt + 截断
	prompt := e.buildPromptWithTruncation(incident, entities, timeline, historical)

	// 6. 通过 ai.AIService 调用 LLM（预算扣减在 Complete 内部处理）
	completeResult, err := e.aiService.Complete(ctx, &ai.CompleteRequest{
		Role:         ai.RoleBackground,
		SystemPrompt: prompt.System,
		UserPrompt:   prompt.User,
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("LLM 调用失败: %w", err)
	}

	log.Debug("LLM 响应", "incident", incidentID, "len", len(completeResult.Response))

	// 7. 解析结构化输出
	result, err := parseResponse(completeResult.Response, incidentID, historical)
	if err != nil {
		return nil, nil, nil, err
	}

	// 8. 更新 rate limit
	e.recordCall(incidentID)

	// 9. 缓存（已完结事件）
	if cacheableStates[incident.State] {
		e.setCache(incidentID, result)
	}

	return result, incident, completeResult, nil
}

// ==================== Analysis（analysis 角色）====================

// runAnalysis 执行深度分析（通过 ai.AIService.Analyze）
func (e *Enricher) runAnalysis(ctx context.Context, incidentID, trigger string) error {
	// 1. 查询事件数据
	incident, err := e.incidentRepo.GetByID(ctx, incidentID)
	if err != nil || incident == nil {
		return fmt.Errorf("查询事件失败: %w", err)
	}

	entities, _ := e.incidentRepo.GetEntities(ctx, incidentID)
	timeline, _ := e.incidentRepo.GetTimeline(ctx, incidentID)

	// 2. 构建上下文
	incidentCtx := BuildIncidentContext(incident, entities, timeline, nil)

	// 3. 调用 ai.AIService.Analyze（多轮 Tool Calling）
	result, err := e.aiService.Analyze(ctx, &ai.AnalyzeRequest{
		ClusterID:    incident.ClusterID,
		Role:         ai.RoleAnalysis,
		SystemPrompt: prompts.BuildAnalysisPrompt(),
		UserPrompt:   prompts.BuildAnalysisUserPrompt(incidentCtx),
	})
	if err != nil {
		return fmt.Errorf("深度分析失败: %w", err)
	}

	// 4. 保存分析报告（仅当有实际内容时）
	if result.Response == "" {
		log.Warn("深度分析 LLM 返回空响应，跳过保存", "incident", incidentID, "toolCalls", result.ToolCalls)
		return fmt.Errorf("LLM 返回空响应（已使用 %d tokens，%d 轮 tool calls）", result.InputTokens+result.OutputTokens, result.ToolCalls)
	}
	e.saveAnalysisReport(ctx, incidentID, incident, result, trigger)

	log.Info("深度分析完成",
		"incident", incidentID,
		"tools", result.ToolCalls,
		"tokens", result.InputTokens+result.OutputTokens,
	)
	return nil
}

// saveAnalysisReport 保存深度分析报告
func (e *Enricher) saveAnalysisReport(ctx context.Context, incidentID string, incident *database.AIOpsIncident, result *ai.AnalyzeResult, trigger string) {
	if e.reportRepo == nil {
		return
	}

	// 解析 LLM 输出为结构化报告
	parsed := parseAnalysisResult(result.Response, incidentID)

	recsJSON, _ := json.Marshal(parsed.Recommendations)
	stepsJSON, _ := json.Marshal(result.Steps)

	report := &database.AIReport{
		IncidentID:         incidentID,
		ClusterID:          incident.ClusterID,
		Role:               "analysis",
		Trigger:            trigger,
		Summary:            parsed.Summary,
		RootCauseAnalysis:  parsed.RootCauseAnalysis,
		Recommendations:    string(recsJSON),
		InvestigationSteps: string(stepsJSON),
		ProviderName:       result.ProviderName,
		Model:              result.Model,
		InputTokens:        result.InputTokens,
		OutputTokens:       result.OutputTokens,
		CreatedAt:          time.Now(),
	}

	if err := e.reportRepo.Create(ctx, report); err != nil {
		log.Warn("保存分析报告失败", "incident", incidentID, "err", err)
	}
}

// parseAnalysisResult 从 LLM 输出中解析分析报告
func parseAnalysisResult(raw, incidentID string) *SummarizeResponse {
	jsonStr := extractJSON(raw)

	var parsed struct {
		Summary           string `json:"summary"`
		RootCauseAnalysis string `json:"rootCauseAnalysis"`
		Recommendations   []struct {
			Priority int    `json:"priority"`
			Action   string `json:"action"`
			Reason   string `json:"reason"`
			Impact   string `json:"impact"`
		} `json:"recommendations"`
		Confidence float64 `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return &SummarizeResponse{
			IncidentID:  incidentID,
			Summary:     raw,
			GeneratedAt: time.Now().UnixMilli(),
		}
	}

	recs := make([]Recommendation, len(parsed.Recommendations))
	for i, r := range parsed.Recommendations {
		recs[i] = Recommendation{
			Priority: r.Priority,
			Action:   r.Action,
			Reason:   r.Reason,
			Impact:   r.Impact,
		}
	}

	return &SummarizeResponse{
		IncidentID:        incidentID,
		Summary:           parsed.Summary,
		RootCauseAnalysis: parsed.RootCauseAnalysis,
		Recommendations:   recs,
		GeneratedAt:       time.Now().UnixMilli(),
	}
}

// ==================== Rate Limit ====================

func (e *Enricher) checkRateLimit(incidentID string) error {
	e.rateMu.Lock()
	defer e.rateMu.Unlock()

	now := time.Now()
	for id, t := range e.lastCalls {
		if now.Sub(t) > rateLimitExpiry {
			delete(e.lastCalls, id)
		}
	}

	if last, ok := e.lastCalls[incidentID]; ok {
		remaining := e.cooldown - now.Sub(last)
		if remaining > 0 {
			return fmt.Errorf("请等待 %d 秒后再试", int(remaining.Seconds())+1)
		}
	}
	return nil
}

func (e *Enricher) recordCall(incidentID string) {
	e.rateMu.Lock()
	defer e.rateMu.Unlock()
	e.lastCalls[incidentID] = time.Now()
}

// ==================== 结果缓存 ====================

func (e *Enricher) getCache(incidentID string) *SummarizeResponse {
	e.cacheMu.RLock()
	entry, ok := e.cache[incidentID]
	e.cacheMu.RUnlock()

	if !ok {
		return nil
	}
	if time.Since(entry.cachedAt) > e.cacheTTL {
		e.cacheMu.Lock()
		delete(e.cache, incidentID)
		e.cacheMu.Unlock()
		return nil
	}
	return entry.response
}

func (e *Enricher) setCache(incidentID string, resp *SummarizeResponse) {
	e.cacheMu.Lock()
	defer e.cacheMu.Unlock()

	if len(e.cache) >= e.maxCache {
		var oldestID string
		var oldestTime time.Time
		for id, entry := range e.cache {
			if oldestID == "" || entry.cachedAt.Before(oldestTime) {
				oldestID = id
				oldestTime = entry.cachedAt
			}
		}
		if oldestID != "" {
			delete(e.cache, oldestID)
		}
	}

	e.cache[incidentID] = &cachedResult{
		response: resp,
		cachedAt: time.Now(),
	}
}

// ==================== Prompt 构建 ====================

// buildPromptWithTruncation 构建 Prompt，超限时逐步截断
func (e *Enricher) buildPromptWithTruncation(
	incident *database.AIOpsIncident,
	entities []*database.AIOpsIncidentEntity,
	timeline []*database.AIOpsIncidentTimeline,
	historical []*database.AIOpsIncident,
) *prompts.PromptPair {
	maxChars := MaxPromptChars
	warnChars := maxChars * 3 / 4

	incidentCtx := BuildIncidentContext(incident, entities, timeline, historical)

	// 丰富 OTel 上下文（如果 Store 可用）
	if e.store != nil && incident.ClusterID != "" {
		if snapshot, err := e.store.GetSnapshot(incident.ClusterID); err == nil && snapshot != nil {
			traces, logs, sloCtx := buildOTelContext(snapshot.OTel, entities)
			incidentCtx.RecentErrorTraces = traces
			incidentCtx.RecentErrorLogs = logs
			incidentCtx.SLOContext = sloCtx
		}
	}

	prompt := prompts.BuildBackgroundPrompt(incidentCtx)
	totalChars := len(prompt.System) + len(prompt.User)

	if totalChars > warnChars {
		log.Warn("Prompt 较长", "chars", totalChars, "warn_threshold", warnChars)
	}

	if totalChars <= maxChars {
		return prompt
	}

	// 超限 → 逐步截断: historical → timeline → entities
	log.Warn("Prompt 超限，开始截断", "chars", totalChars, "max", maxChars)

	truncSteps := []struct{ hist, tl, ent int }{
		{len(historical) / 2, len(timeline), len(entities)},
		{len(historical) / 2, len(timeline) / 2, len(entities)},
		{len(historical) / 2, len(timeline) / 2, len(entities) / 2},
		{0, len(timeline) / 2, len(entities) / 2},
		{0, 0, len(entities) / 2},
		{0, 0, 0},
	}

	for _, step := range truncSteps {
		h := truncateSlice(historical, step.hist)
		t := truncateSlice(timeline, step.tl)
		en := truncateSlice(entities, step.ent)

		rebuilt := BuildIncidentContext(incident, en, t, h)
		prompt = prompts.BuildBackgroundPrompt(rebuilt)
		totalChars = len(prompt.System) + len(prompt.User)

		if totalChars <= maxChars {
			log.Info("Prompt 截断成功", "chars", totalChars,
				"historical", len(h), "timeline", len(t), "entities", len(en))
			return prompt
		}
	}

	log.Warn("Prompt 截断后仍超限（兜底返回）", "chars", totalChars)
	return prompt
}

func truncateSlice[T any](s []T, maxLen int) []T {
	if maxLen <= 0 || len(s) == 0 {
		return nil
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// ==================== 报告持久化 ====================

// reportToSummarizeResponse 将 DB 报告转换为 SummarizeResponse
func reportToSummarizeResponse(r *database.AIReport) *SummarizeResponse {
	resp := &SummarizeResponse{
		IncidentID:        r.IncidentID,
		Summary:           r.Summary,
		RootCauseAnalysis: r.RootCauseAnalysis,
		GeneratedAt:       r.CreatedAt.UnixMilli(),
		ReportID:          r.ID,
		FromCache:         true,
	}

	if r.Recommendations != "" {
		var recs []Recommendation
		if err := json.Unmarshal([]byte(r.Recommendations), &recs); err == nil {
			resp.Recommendations = recs
		}
	}
	if resp.Recommendations == nil {
		resp.Recommendations = []Recommendation{}
	}

	if r.SimilarIncidents != "" {
		var sims []SimilarMatch
		if err := json.Unmarshal([]byte(r.SimilarIncidents), &sims); err == nil {
			resp.SimilarIncidents = sims
		}
	}
	if resp.SimilarIncidents == nil {
		resp.SimilarIncidents = []SimilarMatch{}
	}

	return resp
}

// saveBackgroundReport 持久化 background 分析报告
func (e *Enricher) saveBackgroundReport(ctx context.Context, incident *database.AIOpsIncident, result *SummarizeResponse, trigger string, cr *ai.CompleteResult) {
	if e.reportRepo == nil || incident == nil {
		return
	}

	recsJSON, _ := json.Marshal(result.Recommendations)
	similarsJSON, _ := json.Marshal(result.SimilarIncidents)

	report := &database.AIReport{
		IncidentID:        incident.ID,
		ClusterID:         incident.ClusterID,
		Role:              "background",
		Trigger:           trigger,
		Summary:           result.Summary,
		RootCauseAnalysis: result.RootCauseAnalysis,
		Recommendations:   string(recsJSON),
		SimilarIncidents:  string(similarsJSON),
		CreatedAt:         time.Now(),
	}

	if cr != nil {
		report.ProviderName = cr.ProviderName
		report.Model = cr.Model
		report.InputTokens = cr.InputTokens
		report.OutputTokens = cr.OutputTokens
	}

	if err := e.reportRepo.Create(ctx, report); err != nil {
		log.Warn("保存 AI 报告失败", "incident", incident.ID, "err", err)
	}
}

// ==================== JSON 解析 ====================

// parseResponse 解析 background LLM 响应为结构化结果
func parseResponse(raw string, incidentID string, historical []*database.AIOpsIncident) (*SummarizeResponse, error) {
	jsonStr := extractJSON(raw)

	var parsed struct {
		Summary           string `json:"summary"`
		RootCauseAnalysis string `json:"rootCauseAnalysis"`
		Recommendations   []struct {
			Priority int    `json:"priority"`
			Action   string `json:"action"`
			Reason   string `json:"reason"`
			Impact   string `json:"impact"`
		} `json:"recommendations"`
		SimilarPattern string `json:"similarPattern"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return &SummarizeResponse{
			IncidentID:       incidentID,
			Summary:          raw,
			SimilarIncidents: buildSimilarMatches(historical),
			GeneratedAt:      time.Now().UnixMilli(),
		}, nil
	}

	recommendations := make([]Recommendation, len(parsed.Recommendations))
	for i, r := range parsed.Recommendations {
		recommendations[i] = Recommendation{
			Priority:    r.Priority,
			Action:      r.Action,
			Reason:      r.Reason,
			Impact:      r.Impact,
			IsAutomatic: false,
		}
	}

	return &SummarizeResponse{
		IncidentID:        incidentID,
		Summary:           parsed.Summary,
		RootCauseAnalysis: parsed.RootCauseAnalysis,
		Recommendations:   recommendations,
		SimilarIncidents:  buildSimilarMatches(historical),
		GeneratedAt:       time.Now().UnixMilli(),
	}, nil
}

// extractJSON 从 LLM 输出中提取 JSON 内容
func extractJSON(text string) string {
	if start := strings.Index(text, "```json"); start != -1 {
		jsonStart := start + 7
		if end := strings.Index(text[jsonStart:], "```"); end != -1 {
			return strings.TrimSpace(text[jsonStart : jsonStart+end])
		}
	}
	if start := strings.Index(text, "```"); start != -1 {
		codeStart := start + 3
		if end := strings.Index(text[codeStart:], "```"); end != -1 {
			candidate := strings.TrimSpace(text[codeStart : codeStart+end])
			if len(candidate) > 0 && candidate[0] == '{' {
				return candidate
			}
		}
	}
	if start := strings.Index(text, "{"); start != -1 {
		if end := strings.LastIndex(text, "}"); end > start {
			return text[start : end+1]
		}
	}
	return text
}

// buildSimilarMatches 从历史事件构建相似事件列表
func buildSimilarMatches(historical []*database.AIOpsIncident) []SimilarMatch {
	if len(historical) == 0 {
		return []SimilarMatch{}
	}

	matches := make([]SimilarMatch, 0, len(historical))
	for i, inc := range historical {
		similarity := 0.9 - float64(i)*0.1
		if similarity < 0.3 {
			similarity = 0.3
		}
		matches = append(matches, SimilarMatch{
			IncidentID: inc.ID,
			Similarity: similarity,
			RootCause:  inc.RootCause,
			OccurredAt: inc.StartedAt.Format(time.RFC3339),
		})
	}
	return matches
}

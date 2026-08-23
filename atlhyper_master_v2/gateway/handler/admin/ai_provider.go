// atlhyper_master_v2/gateway/handler/ai_provider.go
// AI Provider 管理 API Handler
package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/ai"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/service"
)

// supportedProviders 支持的提供商 ID 列表
var supportedProviders = []string{"gemini", "openai", "anthropic", "ollama"}

// providerNames 提供商名称映射
var providerNames = map[string]string{
	"gemini":    "Google Gemini",
	"openai":    "OpenAI",
	"anthropic": "Anthropic Claude",
	"ollama":    "Ollama (本地部署)",
}

// AIProviderHandler AI Provider Handler
type AIProviderHandler struct {
	svc service.Service
}

// NewAIProviderHandler 创建 AIProviderHandler
func NewAIProviderHandler(svc service.Service) *AIProviderHandler {
	return &AIProviderHandler{svc: svc}
}

// ==================== Response Types ====================

// ProviderResponse プロバイダー情報レスポンス
type ProviderResponse struct {
	ID                    int64    `json:"id"`
	Name                  string   `json:"name"`
	Provider              string   `json:"provider"`
	Model                 string   `json:"model"`
	BaseURL               string   `json:"baseUrl,omitempty"`
	Description           string   `json:"description"`
	APIKeyMasked          string   `json:"apiKeyMasked"`
	APIKeySet             bool     `json:"apiKeySet"`
	Roles                 []string `json:"roles"`
	ContextWindowOverride int      `json:"contextWindowOverride"`
	Status                string   `json:"status"`
	TotalRequests         int64    `json:"totalRequests"`
	TotalTokens           int64    `json:"totalTokens"`
	TotalCost             float64  `json:"totalCost"`
	LastUsedAt            *string  `json:"lastUsedAt,omitempty"`
	LastError             string   `json:"lastError,omitempty"`
	CreatedAt             string   `json:"createdAt"`
	UpdatedAt             string   `json:"updatedAt"`
}

// AISettingsResponse AI 全局设置レスポンス
type AISettingsResponse struct {
	ToolTimeout int  `json:"toolTimeout"`
	ChatReady   bool `json:"chatReady"`
}

// ProviderListResponse プロバイダー一覧レスポンス
type ProviderListResponse struct {
	Providers []ProviderResponse  `json:"providers"`
	Settings  AISettingsResponse  `json:"settings"`
	Models    []ProviderModelInfo `json:"models"`
}

// ProviderModelInfo モデル情報
type ProviderModelInfo struct {
	Provider string   `json:"provider"`
	Name     string   `json:"name"`
	Models   []string `json:"models"`
}

// ProviderCreateRequest プロバイダー作成リクエスト
type ProviderCreateRequest struct {
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	APIKey      string `json:"apiKey"`
	Model       string `json:"model"`
	BaseURL     string `json:"baseUrl"`
	Description string `json:"description"`
}

// ProviderUpdateRequest プロバイダー更新リクエスト
type ProviderUpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	Provider    *string `json:"provider,omitempty"`
	APIKey      *string `json:"apiKey,omitempty"`
	Model       *string `json:"model,omitempty"`
	BaseURL     *string `json:"baseUrl,omitempty"`
	Description *string `json:"description,omitempty"`
}

// ==================== Handlers ====================

// ProvidersHandler プロバイダー一覧・作成
// GET  /api/v2/ai/providers -> 一覧取得
// POST /api/v2/ai/providers -> 新規作成
func (h *AIProviderHandler) ProvidersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listProviders(w, r)
	case http.MethodPost:
		h.createProvider(w, r)
	default:
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ProviderHandler 個別プロバイダー操作
// GET    /api/v2/ai/providers/{id} -> 取得
// PUT    /api/v2/ai/providers/{id} -> 更新
// DELETE /api/v2/ai/providers/{id} -> 削除
// PUT    /api/v2/ai/providers/{id}/roles -> 角色分配
func (h *AIProviderHandler) ProviderHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/ai/providers/")
	parts := strings.Split(path, "/")
	idStr := parts[0]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid provider id")
		return
	}

	// 子路径路由: /api/v2/ai/providers/{id}/roles
	if len(parts) >= 2 && parts[1] == "roles" {
		h.ProviderRolesHandler(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getProvider(w, r, id)
	case http.MethodPut, http.MethodPatch:
		h.updateProvider(w, r, id)
	case http.MethodDelete:
		h.deleteProvider(w, r, id)
	default:
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// SettingsHandler AI 全局設定操作
// GET /api/v2/ai/settings -> 取得
// PUT /api/v2/ai/settings -> 更新
func (h *AIProviderHandler) SettingsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getSettings(w, r)
	case http.MethodPut, http.MethodPatch:
		h.updateSettings(w, r)
	default:
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ==================== Implementation ====================

// listProviders プロバイダー一覧取得
func (h *AIProviderHandler) listProviders(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	providers, err := h.svc.ListAIProviders(ctx)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list providers")
		return
	}

	settings, _ := h.svc.GetAISettings(ctx)
	toolTimeout := 30
	if settings != nil {
		toolTimeout = settings.ToolTimeout
	}

	models := h.loadModelsGrouped(ctx)

	// 检查 chat 角色是否已分配
	chatReady := false
	for _, p := range providers {
		if slices.Contains(p.Roles, "chat") {
			chatReady = true
			break
		}
	}

	resp := ProviderListResponse{
		Providers: make([]ProviderResponse, 0, len(providers)),
		Settings: AISettingsResponse{
			ToolTimeout: toolTimeout,
			ChatReady:   chatReady,
		},
		Models: models,
	}

	for _, p := range providers {
		resp.Providers = append(resp.Providers, toProviderResponse(p))
	}

	handler.WriteJSON(w, http.StatusOK, resp)
}

// createProvider プロバイダー作成
func (h *AIProviderHandler) createProvider(w http.ResponseWriter, r *http.Request) {
	var req ProviderCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		handler.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Provider == "" {
		handler.WriteError(w, http.StatusBadRequest, "provider is required")
		return
	}
	if !slices.Contains(supportedProviders, req.Provider) {
		handler.WriteError(w, http.StatusBadRequest, "unsupported provider type")
		return
	}
	// Ollama 不需要 API Key，其他提供商必须
	if req.APIKey == "" && req.Provider != "ollama" {
		handler.WriteError(w, http.StatusBadRequest, "api_key is required")
		return
	}
	if req.Model == "" {
		handler.WriteError(w, http.StatusBadRequest, "model is required")
		return
	}
	if req.BaseURL != "" {
		if !isValidBaseURL(req.BaseURL) {
			handler.WriteError(w, http.StatusBadRequest, "invalid base_url format")
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	now := time.Now()
	provider := &database.AIProvider{
		Name:        req.Name,
		Provider:    req.Provider,
		APIKey:      req.APIKey,
		Model:       req.Model,
		BaseURL:     req.BaseURL,
		Description: req.Description,
		Status:      "unknown",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.svc.CreateAIProvider(ctx, provider); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to create provider")
		return
	}

	handler.WriteJSON(w, http.StatusCreated, toProviderResponse(provider))
}

// getProvider プロバイダー取得
func (h *AIProviderHandler) getProvider(w http.ResponseWriter, r *http.Request, id int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	provider, err := h.svc.GetAIProviderByID(ctx, id)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to get provider")
		return
	}
	if provider == nil {
		handler.WriteError(w, http.StatusNotFound, "provider not found")
		return
	}

	handler.WriteJSON(w, http.StatusOK, toProviderResponse(provider))
}

// updateProvider プロバイダー更新
func (h *AIProviderHandler) updateProvider(w http.ResponseWriter, r *http.Request, id int64) {
	var req ProviderUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	provider, err := h.svc.GetAIProviderByID(ctx, id)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to get provider")
		return
	}
	if provider == nil {
		handler.WriteError(w, http.StatusNotFound, "provider not found")
		return
	}

	if req.Name != nil {
		provider.Name = *req.Name
	}
	if req.Provider != nil {
		provider.Provider = *req.Provider
	}
	if req.APIKey != nil && *req.APIKey != "" {
		provider.APIKey = *req.APIKey
	}
	if req.Model != nil {
		provider.Model = *req.Model
	}
	if req.BaseURL != nil {
		provider.BaseURL = *req.BaseURL
	}
	if req.Description != nil {
		provider.Description = *req.Description
	}
	provider.UpdatedAt = time.Now()

	if err := h.svc.UpdateAIProvider(ctx, provider); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to update provider")
		return
	}

	handler.WriteJSON(w, http.StatusOK, toProviderResponse(provider))
}

// deleteProvider プロバイダー削除
func (h *AIProviderHandler) deleteProvider(w http.ResponseWriter, r *http.Request, id int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := h.svc.DeleteAIProvider(ctx, id); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to delete provider")
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// getSettings AI 全局設定取得
func (h *AIProviderHandler) getSettings(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	settings, _ := h.svc.GetAISettings(ctx)
	toolTimeout := 30
	if settings != nil {
		toolTimeout = settings.ToolTimeout
	}

	// 检查是否有 Provider 分配了 chat 角色
	chatReady := false
	providers, _ := h.svc.ListAIProviders(ctx)
	for _, p := range providers {
		if slices.Contains(p.Roles, "chat") {
			chatReady = true
			break
		}
	}

	handler.WriteJSON(w, http.StatusOK, AISettingsResponse{
		ToolTimeout: toolTimeout,
		ChatReady:   chatReady,
	})
}

// updateSettings AI 全局設定更新
func (h *AIProviderHandler) updateSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ToolTimeout *int `json:"toolTimeout,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	settings, _ := h.svc.GetAISettings(ctx)
	if settings == nil {
		settings = &database.AISettings{ToolTimeout: 30}
	}

	if req.ToolTimeout != nil {
		settings.ToolTimeout = *req.ToolTimeout
	}
	settings.UpdatedAt = time.Now()

	if err := h.svc.UpdateAISettings(ctx, settings); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to update settings")
		return
	}

	// 检查 chat 角色
	chatReady := false
	providers, _ := h.svc.ListAIProviders(ctx)
	for _, p := range providers {
		if slices.Contains(p.Roles, "chat") {
			chatReady = true
			break
		}
	}

	handler.WriteJSON(w, http.StatusOK, AISettingsResponse{
		ToolTimeout: settings.ToolTimeout,
		ChatReady:   chatReady,
	})
}

// ==================== Helpers ====================

// toProviderResponse DB モデルをレスポンスに変換
func toProviderResponse(p *database.AIProvider) ProviderResponse {
	roles := p.Roles
	if roles == nil {
		roles = []string{}
	}
	resp := ProviderResponse{
		ID:                    p.ID,
		Name:                  p.Name,
		Provider:              p.Provider,
		Model:                 p.Model,
		BaseURL:               p.BaseURL,
		Description:           p.Description,
		APIKeyMasked:          maskAPIKey(p.APIKey),
		APIKeySet:             p.APIKey != "",
		Roles:                 roles,
		ContextWindowOverride: p.ContextWindowOverride,
		Status:                p.Status,
		TotalRequests:         p.TotalRequests,
		TotalTokens:           p.TotalTokens,
		TotalCost:             p.TotalCost,
		LastError:             p.LastError,
		CreatedAt:             p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:             p.UpdatedAt.Format(time.RFC3339),
	}
	if p.LastUsedAt != nil {
		s := p.LastUsedAt.Format(time.RFC3339)
		resp.LastUsedAt = &s
	}
	return resp
}

// ProviderRolesHandler Provider 角色分配
// PUT /api/v2/ai/providers/{id}/roles → 设置角色
func (h *AIProviderHandler) ProviderRolesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// 从路径提取 provider ID: /api/v2/ai/providers/{id}/roles
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/ai/providers/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "roles" {
		handler.WriteError(w, http.StatusBadRequest, "invalid path")
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid provider id")
		return
	}

	var req struct {
		Roles []string `json:"roles"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 校验角色有效性
	for _, role := range req.Roles {
		if !ai.IsValidRole(role) {
			handler.WriteError(w, http.StatusBadRequest, "invalid role: "+role)
			return
		}
	}

	// 校验互斥约束
	providers, err := h.svc.ListAIProviders(ctx)
	if err != nil {
		log.Warn("获取 Provider 列表失败", "err", err)
	}
	for _, role := range req.Roles {
		for _, p := range providers {
			if p.ID == id {
				continue
			}
			for _, existingRole := range p.Roles {
				if existingRole == role {
					handler.WriteError(w, http.StatusConflict,
						"角色 "+role+" 已被 ["+p.Name+"] 持有，请先移除")
					return
				}
			}
		}
	}

	// 更新角色
	if err := h.svc.UpdateAIProviderRoles(ctx, id, req.Roles); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to update roles")
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "角色更新成功",
		"roles":   req.Roles,
	})
}

// RolesOverviewHandler 角色总览
// GET /api/v2/ai/roles → 获取所有角色状态
func (h *AIProviderHandler) RolesOverviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	providers, err := h.svc.ListAIProviders(ctx)
	if err != nil {
		log.Warn("获取 Provider 列表失败", "err", err)
	}

	// 构建角色名映射
	roleNames := map[string]string{
		"background": "后台分析",
		"chat":       "交互对话",
		"analysis":   "深度分析",
	}

	type roleProviderInfo struct {
		ID            int64  `json:"id"`
		Name          string `json:"name"`
		Model         string `json:"model"`
		ContextWindow int    `json:"contextWindow"`
	}
	type roleOverview struct {
		Role     string            `json:"role"`
		RoleName string            `json:"roleName"`
		Provider *roleProviderInfo `json:"provider"`
	}

	roles := []roleOverview{}
	for _, role := range ai.ValidRoles {
		item := roleOverview{
			Role:     role,
			RoleName: roleNames[role],
		}
		// 查找持有该角色的 Provider
		for _, p := range providers {
			for _, r := range p.Roles {
				if r == role {
					cw := p.ContextWindowOverride
					item.Provider = &roleProviderInfo{
						ID:            p.ID,
						Name:          p.Name,
						Model:         p.Model,
						ContextWindow: cw,
					}
					break
				}
			}
			if item.Provider != nil {
				break
			}
		}
		roles = append(roles, item)
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    roles,
	})
}

// ==================== Budget Handlers ====================

// BudgetsHandler 角色预算列表
// GET /api/v2/ai/budgets → 获取所有角色的预算配置
func (h *AIProviderHandler) BudgetsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	budgets, err := h.svc.ListAIRoleBudgets(ctx)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list budgets")
		return
	}

	type budgetResponse struct {
		Role string `json:"role"`
		// 日限额
		DailyInputTokenLimit  int `json:"dailyInputTokenLimit"`
		DailyOutputTokenLimit int `json:"dailyOutputTokenLimit"`
		DailyCallLimit        int `json:"dailyCallLimit"`
		// 日消耗
		DailyInputTokensUsed  int    `json:"dailyInputTokensUsed"`
		DailyOutputTokensUsed int    `json:"dailyOutputTokensUsed"`
		DailyCallsUsed        int    `json:"dailyCallsUsed"`
		DailyResetAt          string `json:"dailyResetAt,omitempty"`
		// 月限额
		MonthlyInputTokenLimit  int `json:"monthlyInputTokenLimit"`
		MonthlyOutputTokenLimit int `json:"monthlyOutputTokenLimit"`
		MonthlyCallLimit        int `json:"monthlyCallLimit"`
		// 月消耗
		MonthlyInputTokensUsed  int    `json:"monthlyInputTokensUsed"`
		MonthlyOutputTokensUsed int    `json:"monthlyOutputTokensUsed"`
		MonthlyCallsUsed        int    `json:"monthlyCallsUsed"`
		MonthlyResetAt          string `json:"monthlyResetAt,omitempty"`
		// 配置
		AutoTriggerMinSeverity string `json:"autoTriggerMinSeverity"`
		AutoTriggerMode        string `json:"autoTriggerMode"`
		ScheduleStartTime      string `json:"scheduleStartTime,omitempty"`
		ScheduleEndTime        string `json:"scheduleEndTime,omitempty"`
		FallbackProviderID     *int64 `json:"fallbackProviderId"`
	}

	items := make([]budgetResponse, 0, len(budgets))
	for _, b := range budgets {
		item := budgetResponse{
			Role:                    b.Role,
			DailyInputTokenLimit:    b.DailyInputTokenLimit,
			DailyOutputTokenLimit:   b.DailyOutputTokenLimit,
			DailyCallLimit:          b.DailyCallLimit,
			DailyInputTokensUsed:    b.DailyInputTokensUsed,
			DailyOutputTokensUsed:   b.DailyOutputTokensUsed,
			DailyCallsUsed:          b.DailyCallsUsed,
			MonthlyInputTokenLimit:  b.MonthlyInputTokenLimit,
			MonthlyOutputTokenLimit: b.MonthlyOutputTokenLimit,
			MonthlyCallLimit:        b.MonthlyCallLimit,
			MonthlyInputTokensUsed:  b.MonthlyInputTokensUsed,
			MonthlyOutputTokensUsed: b.MonthlyOutputTokensUsed,
			MonthlyCallsUsed:        b.MonthlyCallsUsed,
			AutoTriggerMinSeverity:  b.AutoTriggerMinSeverity,
			AutoTriggerMode:         b.AutoTriggerMode,
			ScheduleStartTime:       b.ScheduleStartTime,
			ScheduleEndTime:         b.ScheduleEndTime,
			FallbackProviderID:      b.FallbackProviderID,
		}
		if b.DailyResetAt != nil {
			item.DailyResetAt = b.DailyResetAt.Format(time.RFC3339)
		}
		if b.MonthlyResetAt != nil {
			item.MonthlyResetAt = b.MonthlyResetAt.Format(time.RFC3339)
		}
		items = append(items, item)
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    items,
	})
}

// BudgetHandler 单个角色预算更新
// PUT /api/v2/ai/budgets/{role}
func (h *AIProviderHandler) BudgetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	role := strings.TrimPrefix(r.URL.Path, "/api/v2/ai/budgets/")
	if !ai.IsValidRole(role) {
		handler.WriteError(w, http.StatusBadRequest, "invalid role: "+role)
		return
	}

	var req struct {
		DailyInputTokenLimit    *int    `json:"dailyInputTokenLimit,omitempty"`
		DailyOutputTokenLimit   *int    `json:"dailyOutputTokenLimit,omitempty"`
		DailyCallLimit          *int    `json:"dailyCallLimit,omitempty"`
		MonthlyInputTokenLimit  *int    `json:"monthlyInputTokenLimit,omitempty"`
		MonthlyOutputTokenLimit *int    `json:"monthlyOutputTokenLimit,omitempty"`
		MonthlyCallLimit        *int    `json:"monthlyCallLimit,omitempty"`
		AutoTriggerMinSeverity  *string `json:"autoTriggerMinSeverity,omitempty"`
		AutoTriggerMode         *string `json:"autoTriggerMode,omitempty"`
		ScheduleStartTime       *string `json:"scheduleStartTime,omitempty"`
		ScheduleEndTime         *string `json:"scheduleEndTime,omitempty"`
		FallbackProviderID      *int64  `json:"fallbackProviderId,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// 校验 severity 值
	validSeverities := map[string]bool{"critical": true, "high": true, "medium": true, "low": true, "off": true}
	if req.AutoTriggerMinSeverity != nil && !validSeverities[*req.AutoTriggerMinSeverity] {
		handler.WriteError(w, http.StatusBadRequest, "invalid severity: "+*req.AutoTriggerMinSeverity)
		return
	}
	// 校验触发模式
	validModes := map[string]bool{"auto": true, "manual": true, "schedule": true}
	if req.AutoTriggerMode != nil && !validModes[*req.AutoTriggerMode] {
		handler.WriteError(w, http.StatusBadRequest, "invalid trigger mode: "+*req.AutoTriggerMode)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 获取现有预算（可能不存在）
	existing, err := h.svc.ListAIRoleBudgets(ctx)
	if err != nil {
		log.Warn("获取角色预算列表失败", "err", err)
	}
	var budget *database.AIRoleBudget
	for _, b := range existing {
		if b.Role == role {
			budget = b
			break
		}
	}
	if budget == nil {
		budget = &database.AIRoleBudget{
			Role:                   role,
			AutoTriggerMinSeverity: "critical",
		}
	}

	// 应用更新
	if req.DailyInputTokenLimit != nil {
		budget.DailyInputTokenLimit = *req.DailyInputTokenLimit
	}
	if req.DailyOutputTokenLimit != nil {
		budget.DailyOutputTokenLimit = *req.DailyOutputTokenLimit
	}
	if req.DailyCallLimit != nil {
		budget.DailyCallLimit = *req.DailyCallLimit
	}
	if req.MonthlyInputTokenLimit != nil {
		budget.MonthlyInputTokenLimit = *req.MonthlyInputTokenLimit
	}
	if req.MonthlyOutputTokenLimit != nil {
		budget.MonthlyOutputTokenLimit = *req.MonthlyOutputTokenLimit
	}
	if req.MonthlyCallLimit != nil {
		budget.MonthlyCallLimit = *req.MonthlyCallLimit
	}
	if req.AutoTriggerMinSeverity != nil {
		budget.AutoTriggerMinSeverity = *req.AutoTriggerMinSeverity
	}
	if req.AutoTriggerMode != nil {
		budget.AutoTriggerMode = *req.AutoTriggerMode
	}
	if req.ScheduleStartTime != nil {
		budget.ScheduleStartTime = *req.ScheduleStartTime
	}
	if req.ScheduleEndTime != nil {
		budget.ScheduleEndTime = *req.ScheduleEndTime
	}
	if req.FallbackProviderID != nil {
		budget.FallbackProviderID = req.FallbackProviderID
	}
	budget.UpdatedAt = time.Now()

	if err := h.svc.UpdateAIRoleBudget(ctx, budget); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to update budget")
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "预算配置已更新",
		"role":    role,
	})
}

// ==================== AI Reports (調用歴史) ====================

// AIReportsHandler 调用历史列表
// GET /api/v2/ai/reports?role=background&limit=20&offset=0
func (h *AIProviderHandler) AIReportsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 解析参数
	role := r.URL.Query().Get("role") // 可选，空=全部
	limit := 20
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	reports, total, err := h.svc.ListRecentAIReports(ctx, role, limit, offset)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list reports")
		return
	}

	type reportItem struct {
		ID           int64  `json:"id"`
		IncidentID   string `json:"incidentId"`
		ClusterID    string `json:"clusterId"`
		Role         string `json:"role"`
		Trigger      string `json:"trigger"`
		Summary      string `json:"summary"`
		ProviderName string `json:"providerName"`
		Model        string `json:"model"`
		InputTokens  int    `json:"inputTokens"`
		OutputTokens int    `json:"outputTokens"`
		DurationMs   int64  `json:"durationMs"`
		CreatedAt    string `json:"createdAt"`
	}

	items := make([]reportItem, 0, len(reports))
	for _, rpt := range reports {
		items = append(items, reportItem{
			ID:           rpt.ID,
			IncidentID:   rpt.IncidentID,
			ClusterID:    rpt.ClusterID,
			Role:         rpt.Role,
			Trigger:      rpt.Trigger,
			Summary:      rpt.Summary,
			ProviderName: rpt.ProviderName,
			Model:        rpt.Model,
			InputTokens:  rpt.InputTokens,
			OutputTokens: rpt.OutputTokens,
			DurationMs:   rpt.DurationMs,
			CreatedAt:    rpt.CreatedAt.Format(time.RFC3339),
		})
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    items,
		"total":   total,
	})
}

// ==================== Helpers ====================

// loadModelsGrouped プロバイダー別モデル一覧取得
func (h *AIProviderHandler) loadModelsGrouped(ctx context.Context) []ProviderModelInfo {
	models, err := h.svc.ListAIModels(ctx)
	if err != nil {
		return []ProviderModelInfo{}
	}

	grouped := make(map[string][]string)
	for _, m := range models {
		grouped[m.Provider] = append(grouped[m.Provider], m.Model)
	}

	result := []ProviderModelInfo{}
	for _, pid := range supportedProviders {
		if ms, ok := grouped[pid]; ok {
			result = append(result, ProviderModelInfo{
				Provider: pid,
				Name:     providerNames[pid],
				Models:   ms,
			})
		}
	}

	return result
}

// isValidBaseURL 校验 BaseURL 格式
func isValidBaseURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

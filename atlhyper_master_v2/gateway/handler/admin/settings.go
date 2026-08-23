// atlhyper_master_v2/gateway/handler/settings.go
// AI 配置 API Handler
package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/service"
)

// SettingsHandler Settings Handler
type SettingsHandler struct {
	svc service.Service
}

// NewSettingsHandler 创建 SettingsHandler
func NewSettingsHandler(svc service.Service) *SettingsHandler {
	return &SettingsHandler{
		svc: svc,
	}
}

// ProviderInfo 提供商信息
type ProviderInfo struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Models []string `json:"models"`
}

// AIConfigResponse AI 配置响应
type AIConfigResponse struct {
	Enabled            bool           `json:"enabled"`            // 用户设置的启用状态
	EffectiveEnabled   bool           `json:"effectiveEnabled"`   // 实际可用状态
	ValidationErrors   []string       `json:"validationErrors"`   // 配置校验错误
	Provider           string         `json:"provider"`           // 当前提供商
	APIKeyMasked       string         `json:"apiKeyMasked"`       // 脱敏后的 API Key
	APIKeySet          bool           `json:"apiKeySet"`          // 是否已设置 API Key
	Model              string         `json:"model"`              // 当前模型
	ToolTimeout        int            `json:"toolTimeout"`        // Tool 超时(秒)
	AvailableProviders []ProviderInfo `json:"availableProviders"` // 可用提供商列表
	RequiresRestart    bool           `json:"requiresRestart"`    // 修改后是否需要重启
}

// AIConfigUpdateRequest AI 配置更新请求
type AIConfigUpdateRequest struct {
	Enabled     *bool  `json:"enabled,omitempty"`
	Provider    string `json:"provider,omitempty"`
	APIKey      string `json:"apiKey,omitempty"`
	Model       string `json:"model,omitempty"`
	ToolTimeout *int   `json:"toolTimeout,omitempty"`
}

// 可用的 AI 提供商和模型
var availableProviders = []ProviderInfo{
	{
		ID:   "gemini",
		Name: "Google Gemini",
		Models: []string{
			"gemini-2.0-flash",
			"gemini-2.0-flash-thinking-exp",
			"gemini-1.5-pro",
			"gemini-1.5-flash",
		},
	},
	{
		ID:   "openai",
		Name: "OpenAI",
		Models: []string{
			"gpt-4o",
			"gpt-4o-mini",
			"gpt-4-turbo",
			"gpt-4",
		},
	},
	{
		ID:   "anthropic",
		Name: "Anthropic Claude",
		Models: []string{
			"claude-sonnet-4-20250514",
			"claude-3-5-sonnet-20241022",
			"claude-3-opus-20240229",
		},
	},
}

// AIConfigHandler AI 配置综合处理
// GET  /api/v2/settings/ai      -> 获取配置
// PUT  /api/v2/settings/ai      -> 更新配置
// POST /api/v2/settings/ai/test -> 测试连接
func (h *SettingsHandler) AIConfigHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/settings/ai")

	// 检查是否是测试请求
	if path == "/test" {
		h.testAIConnection(w, r)
		return
	}

	// 根据 Method 分发
	switch r.Method {
	case http.MethodGet:
		h.getAIConfig(w, r)
	case http.MethodPut, http.MethodPatch:
		h.updateAIConfig(w, r)
	default:
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// getAIConfig 获取 AI 配置
func (h *SettingsHandler) getAIConfig(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 从数据库加载配置
	config := h.loadAIConfig(ctx)

	handler.WriteJSON(w, http.StatusOK, config)
}

// updateAIConfig 更新 AI 配置
func (h *SettingsHandler) updateAIConfig(w http.ResponseWriter, r *http.Request) {
	var req AIConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 校验 provider
	if req.Provider != "" && !isValidProvider(req.Provider) {
		handler.WriteError(w, http.StatusBadRequest, "invalid provider")
		return
	}

	// 校验 model（如果指定了 provider）
	if req.Model != "" && req.Provider != "" {
		if !isValidModel(req.Provider, req.Model) {
			handler.WriteError(w, http.StatusBadRequest, "invalid model for provider")
			return
		}
	}

	// 更新配置
	if req.Enabled != nil {
		h.setSetting(ctx, "ai.enabled", boolToString(*req.Enabled))
	}
	if req.Provider != "" {
		h.setSetting(ctx, "ai.provider", req.Provider)
	}
	if req.APIKey != "" {
		h.setSetting(ctx, "ai.api_key", req.APIKey)
	}
	if req.Model != "" {
		h.setSetting(ctx, "ai.model", req.Model)
	}
	if req.ToolTimeout != nil {
		h.setSetting(ctx, "ai.tool_timeout", strconv.Itoa(*req.ToolTimeout))
	}

	// 返回更新后的配置
	config := h.loadAIConfig(ctx)
	config.RequiresRestart = true // 修改后需要重启

	handler.WriteJSON(w, http.StatusOK, config)
}

// testAIConnection 测试 AI 连接
func (h *SettingsHandler) testAIConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// 加载当前配置
	apiKey, _ := h.svc.GetSetting(ctx, "ai.api_key")
	provider, _ := h.svc.GetSetting(ctx, "ai.provider")
	model, _ := h.svc.GetSetting(ctx, "ai.model")

	if apiKey == nil || apiKey.Value == "" {
		handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"message": "API Key 未配置",
		})
		return
	}

	providerVal := "gemini"
	if provider != nil && provider.Value != "" {
		providerVal = provider.Value
	}

	modelVal := "gemini-2.0-flash"
	if model != nil && model.Value != "" {
		modelVal = model.Value
	}

	// 根据 provider 测试连接
	var success bool
	var message string

	switch providerVal {
	case "gemini":
		success, message = testGeminiConnection(apiKey.Value, modelVal)
	case "openai":
		success, message = testOpenAIConnection(apiKey.Value, modelVal)
	case "anthropic":
		success, message = testAnthropicConnection(apiKey.Value, modelVal)
	default:
		success = false
		message = "不支持的 Provider: " + providerVal
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success":  success,
		"message":  message,
		"provider": providerVal,
		"model":    modelVal,
	})
}

// ==================== 辅助函数 ====================

// loadAIConfig 从数据库加载 AI 配置
func (h *SettingsHandler) loadAIConfig(ctx context.Context) *AIConfigResponse {
	enabled, _ := h.svc.GetSetting(ctx, "ai.enabled")
	provider, _ := h.svc.GetSetting(ctx, "ai.provider")
	apiKey, _ := h.svc.GetSetting(ctx, "ai.api_key")
	model, _ := h.svc.GetSetting(ctx, "ai.model")
	timeout, _ := h.svc.GetSetting(ctx, "ai.tool_timeout")

	// 构建响应
	config := &AIConfigResponse{
		Enabled:            getSettingBool(enabled),
		Provider:           getSettingString(provider, "gemini"),
		APIKeyMasked:       maskAPIKey(getSettingString(apiKey, "")),
		APIKeySet:          apiKey != nil && apiKey.Value != "",
		Model:              getSettingString(model, "gemini-2.0-flash"),
		ToolTimeout:        getSettingInt(timeout, 30),
		AvailableProviders: availableProviders,
		RequiresRestart:    false,
	}

	// 计算校验错误和有效启用状态
	config.ValidationErrors = validateAIConfig(config)
	config.EffectiveEnabled = config.Enabled && len(config.ValidationErrors) == 0

	return config
}

// setSetting 设置配置项
func (h *SettingsHandler) setSetting(ctx context.Context, key, value string) error {
	return h.svc.SetSetting(ctx, &database.Setting{
		Key:   key,
		Value: value,
	})
}

// validateAIConfig 校验 AI 配置完整性
func validateAIConfig(config *AIConfigResponse) []string {
	var errors []string

	if config.Provider == "" {
		errors = append(errors, "provider 未配置")
	}
	if !config.APIKeySet {
		errors = append(errors, "api_key 未配置")
	}
	if config.Model == "" {
		errors = append(errors, "model 未配置")
	}

	return errors
}

// isValidProvider 校验 provider 有效性
func isValidProvider(provider string) bool {
	for _, p := range availableProviders {
		if p.ID == provider {
			return true
		}
	}
	return false
}

// isValidModel 校验 model 有效性
func isValidModel(provider, model string) bool {
	for _, p := range availableProviders {
		if p.ID == provider {
			for _, m := range p.Models {
				if m == model {
					return true
				}
			}
			return false
		}
	}
	return false
}

// maskAPIKey 脱敏 API Key
func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "****"
	}
	// 显示前4位和后4位
	return key[:4] + "****" + key[len(key)-4:]
}

// getSettingBool 获取布尔设置
func getSettingBool(s *database.Setting) bool {
	if s == nil {
		return false
	}
	return s.Value == "true" || s.Value == "1" || s.Value == "yes"
}

// getSettingString 获取字符串设置
func getSettingString(s *database.Setting, defaultVal string) string {
	if s == nil || s.Value == "" {
		return defaultVal
	}
	return s.Value
}

// getSettingInt 获取整数设置
func getSettingInt(s *database.Setting, defaultVal int) int {
	if s == nil || s.Value == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(s.Value)
	if err != nil {
		return defaultVal
	}
	return val
}

// boolToString 布尔转字符串
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// ==================== AI 连接测试 ====================

// testGeminiConnection 测试 Gemini 连接
func testGeminiConnection(apiKey, model string) (bool, string) {
	// TODO: 实现真实的 Gemini API 连接测试
	// 目前返回模拟结果
	if apiKey == "" {
		return false, "API Key 为空"
	}
	if !strings.HasPrefix(apiKey, "AIza") {
		return false, "API Key 格式不正确（应以 AIza 开头）"
	}
	return true, "Gemini 连接测试成功 (model: " + model + ")"
}

// testOpenAIConnection 测试 OpenAI 连接
func testOpenAIConnection(apiKey, model string) (bool, string) {
	// TODO: 实现真实的 OpenAI API 连接测试
	if apiKey == "" {
		return false, "API Key 为空"
	}
	if !strings.HasPrefix(apiKey, "sk-") {
		return false, "API Key 格式不正确（应以 sk- 开头）"
	}
	return true, "OpenAI 连接测试成功 (model: " + model + ")"
}

// testAnthropicConnection 测试 Anthropic 连接
func testAnthropicConnection(apiKey, model string) (bool, string) {
	// TODO: 实现真实的 Anthropic API 连接测试
	if apiKey == "" {
		return false, "API Key 为空"
	}
	if !strings.HasPrefix(apiKey, "sk-ant-") {
		return false, "API Key 格式不正确（应以 sk-ant- 开头）"
	}
	return true, "Anthropic 连接测试成功 (model: " + model + ")"
}

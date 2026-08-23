// atlhyper_master_v2/gateway/handler/notify.go
// 通知配置 API Handler
package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/service"
)

// NotifyHandler 通知 Handler
type NotifyHandler struct {
	svc service.Service
}

// NewNotifyHandler 创建 NotifyHandler
func NewNotifyHandler(svc service.Service) *NotifyHandler {
	return &NotifyHandler{
		svc: svc,
	}
}

// ChannelResponse 渠道响应
type ChannelResponse struct {
	ID               int64           `json:"id"`
	Type             string          `json:"type"`
	Name             string          `json:"name"`
	Enabled          bool            `json:"enabled"`          // 用户设置的启用状态
	EffectiveEnabled bool            `json:"effectiveEnabled"` // 实际可用状态（启用+配置完整）
	ValidationErrors []string        `json:"validationErrors"` // 配置校验错误
	Config           json.RawMessage `json:"config"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
}

// ListChannels 列出通知渠道
func (h *NotifyHandler) ListChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	channels, err := h.svc.ListNotifyChannels(ctx)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list channels")
		return
	}

	// 转换响应（含校验信息）
	var responses []ChannelResponse
	for _, ch := range channels {
		responses = append(responses, toChannelResponse(ch))
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"channels": responses,
		"total":    len(responses),
	})
}

// UpdateChannelRequest 更新通知渠道请求
type UpdateChannelRequest struct {
	Enabled *bool           `json:"enabled,omitempty"`
	Name    string          `json:"name,omitempty"`
	Config  json.RawMessage `json:"config,omitempty"`
}

// ChannelHandler 通知渠道综合处理（Admin 权限）
// 根据 Method 和 Path 分发到对应的处理函数
// GET  /api/v2/notify/channels/{type}       -> 获取详情
// PUT  /api/v2/notify/channels/{type}       -> 更新配置
// POST /api/v2/notify/channels/{type}/test  -> 测试发送
func (h *NotifyHandler) ChannelHandler(w http.ResponseWriter, r *http.Request) {
	// 解析路径
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/notify/channels/")

	// 检查是否是测试请求
	if strings.HasSuffix(path, "/test") {
		h.testChannel(w, r, strings.TrimSuffix(path, "/test"))
		return
	}

	// 根据 Method 分发
	switch r.Method {
	case http.MethodGet:
		h.getChannel(w, r, strings.TrimSuffix(path, "/"))
	case http.MethodPut, http.MethodPatch:
		h.updateChannel(w, r, strings.TrimSuffix(path, "/"))
	default:
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// getChannel 获取通知渠道详情
func (h *NotifyHandler) getChannel(w http.ResponseWriter, r *http.Request, channelType string) {
	if channelType == "" {
		handler.WriteError(w, http.StatusBadRequest, "channel type required")
		return
	}

	if !isValidChannelType(channelType) {
		handler.WriteError(w, http.StatusBadRequest, "invalid channel type")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	channel, err := h.svc.GetNotifyChannelByType(ctx, channelType)
	if err != nil || channel == nil {
		handler.WriteError(w, http.StatusNotFound, "channel not found")
		return
	}

	handler.WriteJSON(w, http.StatusOK, toChannelResponse(channel))
}

// UpdateChannel 更新通知渠道（保留用于独立路由）
func (h *NotifyHandler) UpdateChannel(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/notify/channels/")
	channelType := strings.TrimSuffix(path, "/")
	h.updateChannel(w, r, channelType)
}

// updateChannel 更新通知渠道（内部方法）
func (h *NotifyHandler) updateChannel(w http.ResponseWriter, r *http.Request, channelType string) {
	if channelType == "" {
		handler.WriteError(w, http.StatusBadRequest, "channel type required")
		return
	}

	if !isValidChannelType(channelType) {
		handler.WriteError(w, http.StatusBadRequest, "invalid channel type")
		return
	}

	var req UpdateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 获取现有配置
	existing, err := h.svc.GetNotifyChannelByType(ctx, channelType)
	if err != nil || existing == nil {
		// 不存在则创建
		existing = &database.NotifyChannel{
			Type:    channelType,
			Name:    channelType,
			Enabled: false,
			Config:  "{}",
		}
	}

	// 更新字段
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Config != nil {
		existing.Config = string(req.Config)
	}

	// 保存
	if existing.ID == 0 {
		err = h.svc.CreateNotifyChannel(ctx, existing)
	} else {
		err = h.svc.UpdateNotifyChannel(ctx, existing)
	}
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to update channel")
		return
	}

	handler.WriteJSON(w, http.StatusOK, toChannelResponse(existing))
}

// testChannel 测试通知渠道（内部方法）
// 已迁移到独立的 tester 模块（端口 9080）
// 此处保留空实现，返回提示信息
func (h *NotifyHandler) testChannel(w http.ResponseWriter, r *http.Request, channelType string) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// 测试功能已迁移到独立端口
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "测试功能已迁移到 :9080/test/notifier/" + channelType,
		"channel": channelType,
		"success": false,
	})
}

// ==================== 辅助函数 ====================

// validChannelTypes 有效的渠道类型
var validChannelTypes = map[string]bool{
	"email":    true,
	"mail":     true, // alias
	"slack":    true,
	"webhook":  true,
	"dingtalk": true,
}

// isValidChannelType 校验渠道类型
func isValidChannelType(channelType string) bool {
	return validChannelTypes[channelType]
}

// validateChannelConfig 校验渠道配置完整性
// 返回校验错误列表，空列表表示配置完整
func validateChannelConfig(channelType, config string) []string {
	var errors []string

	switch channelType {
	case "slack":
		var cfg database.SlackConfig
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			errors = append(errors, "配置格式错误")
			return errors
		}
		if cfg.WebhookURL == "" {
			errors = append(errors, "webhook_url 未配置")
		}

	case "email", "mail":
		var cfg database.EmailConfig
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			errors = append(errors, "配置格式错误")
			return errors
		}
		if cfg.SMTPHost == "" {
			errors = append(errors, "smtp_host 未配置")
		}
		if cfg.SMTPPort == 0 {
			errors = append(errors, "smtp_port 未配置")
		}
		if cfg.SMTPUser == "" {
			errors = append(errors, "smtp_user 未配置")
		}
		if cfg.SMTPPassword == "" {
			errors = append(errors, "smtp_password 未配置")
		}
		// from_address 统一使用 smtp_user，无需单独验证
		if len(cfg.ToAddresses) == 0 {
			errors = append(errors, "to_addresses 未配置（至少需要一个收件人）")
		}

	case "webhook":
		// webhook 类型需要 url
		var cfg struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			errors = append(errors, "配置格式错误")
			return errors
		}
		if cfg.URL == "" {
			errors = append(errors, "url 未配置")
		}

	case "dingtalk":
		// 钉钉需要 webhook
		var cfg struct {
			WebhookURL string `json:"webhook_url"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			errors = append(errors, "配置格式错误")
			return errors
		}
		if cfg.WebhookURL == "" {
			errors = append(errors, "webhook_url 未配置")
		}
	}

	return errors
}

// toChannelResponse 转换为响应结构（含校验）
func toChannelResponse(ch *database.NotifyChannel) ChannelResponse {
	validationErrors := validateChannelConfig(ch.Type, ch.Config)
	effectiveEnabled := ch.Enabled && len(validationErrors) == 0

	return ChannelResponse{
		ID:               ch.ID,
		Type:             ch.Type,
		Name:             ch.Name,
		Enabled:          ch.Enabled,
		EffectiveEnabled: effectiveEnabled,
		ValidationErrors: validationErrors,
		Config:           json.RawMessage(ch.Config),
		CreatedAt:        ch.CreatedAt,
		UpdatedAt:        ch.UpdatedAt,
	}
}

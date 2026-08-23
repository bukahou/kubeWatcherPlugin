// atlhyper_master_v2/gateway/handler/audit.go
// 审计日志 API Handler
package admin

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/service"
)

// AuditHandler 审计日志 Handler
type AuditHandler struct {
	svc service.Query
}

// NewAuditHandler 创建 AuditHandler
func NewAuditHandler(svc service.Query) *AuditHandler {
	return &AuditHandler{
		svc: svc,
	}
}

// AuditLogResponse 审计日志响应
type AuditLogResponse struct {
	ID             int64  `json:"id"`
	Timestamp      string `json:"timestamp"`
	UserID         int64  `json:"userId"`
	Username       string `json:"username"`
	Role           int    `json:"role"`
	Source         string `json:"source"`
	Action         string `json:"action"`
	Resource       string `json:"resource"`
	Method         string `json:"method"`
	RequestSummary string `json:"requestSummary,omitempty"`
	Status         int    `json:"status"`
	Success        bool   `json:"success"`
	ErrorMessage   string `json:"errorMessage,omitempty"`
	IP             string `json:"ip"`
	DurationMs     int64  `json:"durationMs"`
}

// List 获取审计日志列表
// GET /api/v2/audit/logs
// 查询参数：user_id, source, action, since, until, limit, offset
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 解析查询参数
	query := r.URL.Query()
	opts := database.AuditQueryOpts{
		Limit:  50, // 默认
		Offset: 0,
	}

	if userIDStr := query.Get("userId"); userIDStr != "" {
		if userID, err := strconv.ParseInt(userIDStr, 10, 64); err == nil {
			opts.UserID = userID
		}
	}

	if source := query.Get("source"); source != "" {
		opts.Source = source
	}

	if action := query.Get("action"); action != "" {
		opts.Action = action
	}

	if since := query.Get("since"); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			opts.Since = t
		}
	}

	if until := query.Get("until"); until != "" {
		if t, err := time.Parse(time.RFC3339, until); err == nil {
			opts.Until = t
		}
	}

	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			if limit > 200 {
				limit = 200 // 最大限制
			}
			opts.Limit = limit
		}
	}

	if offsetStr := query.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			opts.Offset = offset
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 查询日志
	logs, err := h.svc.ListAuditLogs(ctx, opts)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "获取审计日志失败")
		return
	}

	// 查询总数
	total, err := h.svc.CountAuditLogs(ctx, opts)
	if err != nil {
		total = int64(len(logs))
	}

	// 转换响应
	var responses []AuditLogResponse
	for _, log := range logs {
		responses = append(responses, AuditLogResponse{
			ID:             log.ID,
			Timestamp:      log.Timestamp.Format(time.RFC3339),
			UserID:         log.UserID,
			Username:       log.Username,
			Role:           log.Role,
			Source:         log.Source,
			Action:         log.Action,
			Resource:       log.Resource,
			Method:         log.Method,
			RequestSummary: summarizeRequest(log.RequestBody),
			Status:         log.StatusCode,
			Success:        log.Success,
			ErrorMessage:   log.ErrorMessage,
			IP:             log.IP,
			DurationMs:     log.DurationMs,
		})
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    responses,
		"total":   total,
	})
}

// summarizeRequest 摘要请求体（脱敏）
func summarizeRequest(body string) string {
	if len(body) > 100 {
		return body[:100] + "..."
	}
	return body
}

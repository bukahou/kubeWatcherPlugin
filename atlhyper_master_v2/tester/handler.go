// atlhyper_master_v2/tester/handler.go
// HTTP Handler
package tester

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// Handler HTTP Handler
type Handler struct {
	registry *Registry
}

// NewHandler 创建 Handler
func NewHandler(registry *Registry) *Handler {
	return &Handler{
		registry: registry,
	}
}

// Health 健康检查
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"testers": h.registry.List(),
		"time":    time.Now().Format(time.RFC3339),
	})
}

// Test 执行测试
// 路由: POST /test/{tester}/{target}
// 例如: POST /test/notifier/slack
func (h *Handler) Test(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// 解析路径: /test/{tester}/{target}
	path := strings.TrimPrefix(r.URL.Path, "/test/")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) < 1 || parts[0] == "" {
		h.writeError(w, http.StatusBadRequest, "tester name required")
		return
	}

	testerName := parts[0]
	target := ""
	if len(parts) > 1 {
		target = parts[1]
	}

	// 执行测试
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result := h.registry.Run(ctx, testerName, target)

	status := http.StatusOK
	if !result.Success {
		status = http.StatusBadRequest
	}

	h.writeJSON(w, status, result)
}

// List 列出所有测试器
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"testers": h.registry.List(),
	})
}

// writeJSON 写入 JSON 响应
func (h *Handler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError 写入错误响应
func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]any{
		"error": message,
	})
}

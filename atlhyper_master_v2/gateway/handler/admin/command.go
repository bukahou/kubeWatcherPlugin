// atlhyper_master_v2/gateway/handler/command.go
// 指令下发 API Handler
// 创建指令通过 CommandService，查询状态通过 Query
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
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/service"
	"AtlHyper/common/logger"
)

var cmdHandlerLog = logger.Module("CommandHandler")

// CommandHandler 指令 Handler
type CommandHandler struct {
	svc service.Service
}

// NewCommandHandler 创建 CommandHandler
func NewCommandHandler(svc service.Service) *CommandHandler {
	return &CommandHandler{svc: svc}
}

// CreateCommandRequest 创建指令请求
type CreateCommandRequest struct {
	ClusterID       string                 `json:"clusterId"`
	Action          string                 `json:"action"`
	TargetKind      string                 `json:"targetKind,omitempty"`
	TargetNamespace string                 `json:"targetNamespace,omitempty"`
	TargetName      string                 `json:"targetName,omitempty"`
	Params          map[string]interface{} `json:"params,omitempty"`
}

// Create 创建指令
func (h *CommandHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req CreateCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// 创建指令
	resp, err := h.svc.CreateCommand(&model.CreateCommandRequest{
		ClusterID:       req.ClusterID,
		Action:          req.Action,
		TargetKind:      req.TargetKind,
		TargetNamespace: req.TargetNamespace,
		TargetName:      req.TargetName,
		Params:          req.Params,
		Source:          "web",
	})
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	handler.WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"commandId": resp.CommandID,
		"status":    resp.Status,
	})
}

// GetStatus 获取指令状态
func (h *CommandHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// 解析 commandID: /api/v2/commands/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/commands/")
	commandID := strings.TrimSuffix(path, "/")
	if commandID == "" {
		handler.WriteError(w, http.StatusBadRequest, "command_id required")
		return
	}

	// 通过 Query 层查询状态
	status, err := h.svc.GetCommandStatus(r.Context(), commandID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to get command status")
		return
	}
	if status == nil {
		handler.WriteError(w, http.StatusNotFound, "command not found")
		return
	}

	handler.WriteJSON(w, http.StatusOK, status)
}

// CommandHistoryResponse 命令历史响应
type CommandHistoryResponse struct {
	ID              int64      `json:"id"`
	CommandID       string     `json:"commandId"`
	ClusterID       string     `json:"clusterId"`
	Source          string     `json:"source"`
	UserID          int64      `json:"userId"`
	Action          string     `json:"action"`
	TargetKind      string     `json:"targetKind"`
	TargetNamespace string     `json:"targetNamespace"`
	TargetName      string     `json:"targetName"`
	Params          string     `json:"params"`
	Status          string     `json:"status"`
	Result          string     `json:"result"`
	ErrorMessage    string     `json:"errorMessage"`
	CreatedAt       time.Time  `json:"createdAt"`
	StartedAt       *time.Time `json:"startedAt"`
	FinishedAt      *time.Time `json:"finishedAt"`
	DurationMs      int64      `json:"durationMs"`
}

// ListHistory 获取命令历史列表
// GET /api/v2/commands/history?cluster_id=&source=&status=&action=&search=&limit=&offset=
func (h *CommandHandler) ListHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	query := r.URL.Query()
	opts := database.CommandQueryOpts{
		ClusterID: query.Get("clusterId"),
		Source:    query.Get("source"),
		Status:    query.Get("status"),
		Action:    query.Get("action"),
		Search:    query.Get("search"),
		Limit:     parseIntOrDefault(query.Get("limit"), 20),
		Offset:    parseIntOrDefault(query.Get("offset"), 0),
	}

	// 限制最大查询数量
	if opts.Limit > 100 {
		opts.Limit = 100
	}
	if opts.Limit <= 0 {
		opts.Limit = 20
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 查询列表
	commands, err := h.svc.ListCommandHistory(ctx, opts)
	if err != nil {
		cmdHandlerLog.Error("查询命令列表失败", "err", err)
		handler.WriteError(w, http.StatusInternalServerError, "failed to list commands: "+err.Error())
		return
	}

	// 查询总数
	total, err := h.svc.CountCommandHistory(ctx, opts)
	if err != nil {
		cmdHandlerLog.Error("统计命令数量失败", "err", err)
		handler.WriteError(w, http.StatusInternalServerError, "failed to count commands: "+err.Error())
		return
	}

	// 转换响应（确保返回空数组而非 null）
	responses := make([]CommandHistoryResponse, 0, len(commands))
	for _, cmd := range commands {
		responses = append(responses, CommandHistoryResponse{
			ID:              cmd.ID,
			CommandID:       cmd.CommandID,
			ClusterID:       cmd.ClusterID,
			Source:          cmd.Source,
			UserID:          cmd.UserID,
			Action:          cmd.Action,
			TargetKind:      cmd.TargetKind,
			TargetNamespace: cmd.TargetNamespace,
			TargetName:      cmd.TargetName,
			Params:          cmd.Params,
			Status:          cmd.Status,
			Result:          cmd.Result,
			ErrorMessage:    cmd.ErrorMessage,
			CreatedAt:       cmd.CreatedAt,
			StartedAt:       cmd.StartedAt,
			FinishedAt:      cmd.FinishedAt,
			DurationMs:      cmd.DurationMs,
		})
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"commands": responses,
		"total":    total,
	})
}

// parseIntOrDefault 解析整数，失败返回默认值
func parseIntOrDefault(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return val
}

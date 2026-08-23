// atlhyper_master_v2/gateway/handler/slo_targets.go
// SLO 目标管理与状态历史 Handler 方法
package slo

import (
	"encoding/json"
	"net/http"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/model"
)

// ==================== 目标管理 ====================

// Targets 处理 /api/v2/slo/targets
func (h *SLOHandler) Targets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getTargets(w, r)
	case http.MethodPut:
		h.updateTarget(w, r)
	default:
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *SLOHandler) getTargets(w http.ResponseWriter, r *http.Request) {
	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		clusterID = h.defaultClusterID(r.Context())
	}

	targets, err := h.querySvc.GetSLOTargets(r.Context(), clusterID)
	if err != nil {
		sloLog.Error("获取 targets 失败", "err", err)
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handler.WriteJSON(w, http.StatusOK, targets)
}

func (h *SLOHandler) updateTarget(w http.ResponseWriter, r *http.Request) {
	var req model.UpdateSLOTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Host == "" {
		handler.WriteError(w, http.StatusBadRequest, "host required")
		return
	}

	if req.ClusterID == "" {
		req.ClusterID = h.defaultClusterID(r.Context())
	}

	if err := h.opsSvc.UpsertSLOTarget(r.Context(), &req); err != nil {
		sloLog.Error("更新 target 失败", "err", err)
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ==================== 状态历史 ====================

// StatusHistory GET /api/v2/slo/status-history
// 状态历史不再写入 SQLite，返回空数组
func (h *SLOHandler) StatusHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	handler.WriteJSON(w, http.StatusOK, []model.SLOStatusHistoryItem{})
}

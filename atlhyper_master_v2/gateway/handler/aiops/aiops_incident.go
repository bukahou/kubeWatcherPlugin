// atlhyper_master_v2/gateway/handler/aiops_incident.go
// AIOps 事件 API Handler
package aiops

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/service"
)

// AIOpsIncidentHandler AIOps 事件 Handler
type AIOpsIncidentHandler struct {
	svc service.Query
}

// NewAIOpsIncidentHandler 创建 Handler
func NewAIOpsIncidentHandler(svc service.Query) *AIOpsIncidentHandler {
	return &AIOpsIncidentHandler{svc: svc}
}

// List 事件列表
// GET /api/v2/aiops/incidents?cluster={id}&state={state}&severity={severity}&from={time}&to={time}&limit={n}&offset={n}
func (h *AIOpsIncidentHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query()
	opts := aiops.IncidentQueryOpts{
		ClusterID: q.Get("cluster"),
		State:     q.Get("state"),
		Severity:  q.Get("severity"),
	}

	if from := q.Get("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			opts.From = t
		}
	}
	if to := q.Get("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			opts.To = t
		}
	}
	if limit := q.Get("limit"); limit != "" {
		if n, err := strconv.Atoi(limit); err == nil && n > 0 {
			opts.Limit = n
		}
	}
	if offset := q.Get("offset"); offset != "" {
		if n, err := strconv.Atoi(offset); err == nil && n >= 0 {
			opts.Offset = n
		}
	}

	if opts.Limit == 0 {
		opts.Limit = 50
	}

	incidents, total, err := h.svc.GetAIOpsIncidents(r.Context(), opts)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"message": "获取成功",
		"data":    incidents,
		"total":   total,
	})
}

// Detail 事件详情
// GET /api/v2/aiops/incidents/{id}
func (h *AIOpsIncidentHandler) Detail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// 从路径提取 ID: /api/v2/aiops/incidents/{id}
	path := r.URL.Path
	id := strings.TrimPrefix(path, "/api/v2/aiops/incidents/")
	if id == "" || id == path {
		handler.WriteError(w, http.StatusBadRequest, "missing incident id")
		return
	}

	detail, err := h.svc.GetAIOpsIncidentDetail(r.Context(), id)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if detail == nil {
		handler.WriteError(w, http.StatusNotFound, "incident not found")
		return
	}

	handler.WriteJSON(w, http.StatusOK, ScaleIncidentDetail(detail))
}

// Stats 事件统计
// GET /api/v2/aiops/incidents/stats?cluster={id}&period=7d
func (h *AIOpsIncidentHandler) Stats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clusterID := handler.ClusterIDFromQuery(r)
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "missing cluster_id parameter")
		return
	}

	since := parsePeriod(r.URL.Query().Get("period"))

	stats, err := h.svc.GetAIOpsIncidentStats(r.Context(), clusterID, since)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handler.WriteJSON(w, http.StatusOK, stats)
}

// Patterns 历史事件模式
// GET /api/v2/aiops/incidents/patterns?entity={key}&period=30d
func (h *AIOpsIncidentHandler) Patterns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	entityKey := r.URL.Query().Get("entity")
	if entityKey == "" {
		handler.WriteError(w, http.StatusBadRequest, "missing entity parameter")
		return
	}

	since := parsePeriod(r.URL.Query().Get("period"))

	patterns, err := h.svc.GetAIOpsIncidentPatterns(r.Context(), entityKey, since)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handler.WriteJSON(w, http.StatusOK, patterns)
}

// parsePeriod 解析时间周期字符串（如 "7d", "30d", "24h"）
func parsePeriod(period string) time.Time {
	now := time.Now()
	if period == "" {
		return now.AddDate(0, 0, -7) // 默认 7 天
	}

	if strings.HasSuffix(period, "d") {
		if days, err := strconv.Atoi(strings.TrimSuffix(period, "d")); err == nil {
			return now.AddDate(0, 0, -days)
		}
	}
	if strings.HasSuffix(period, "h") {
		if hours, err := strconv.Atoi(strings.TrimSuffix(period, "h")); err == nil {
			return now.Add(-time.Duration(hours) * time.Hour)
		}
	}

	return now.AddDate(0, 0, -7)
}

// atlhyper_master_v2/gateway/handler/aiops_graph.go
// AIOps 依赖图 API Handler
package aiops

import (
	"net/http"
	"strconv"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/service"
)

// AIOpsGraphHandler AIOps 依赖图 Handler
type AIOpsGraphHandler struct {
	svc service.Query
}

// NewAIOpsGraphHandler 创建 Handler
func NewAIOpsGraphHandler(svc service.Query) *AIOpsGraphHandler {
	return &AIOpsGraphHandler{svc: svc}
}

// Graph 获取完整依赖图
// GET /api/v2/aiops/graph?cluster={id}
func (h *AIOpsGraphHandler) Graph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clusterID := handler.ClusterIDFromQuery(r)
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "missing cluster_id parameter")
		return
	}

	graph, err := h.svc.GetAIOpsGraph(r.Context(), clusterID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handler.WriteJSON(w, http.StatusOK, graph)
}

// Trace 链路追踪
// GET /api/v2/aiops/graph/trace?cluster={id}&from={key}&direction=upstream&depth=5
func (h *AIOpsGraphHandler) Trace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clusterID := handler.ClusterIDFromQuery(r)
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "missing cluster_id parameter")
		return
	}

	fromKey := r.URL.Query().Get("from")
	if fromKey == "" {
		handler.WriteError(w, http.StatusBadRequest, "missing from parameter")
		return
	}

	direction := r.URL.Query().Get("direction")
	if direction == "" {
		direction = "downstream"
	}

	maxDepth := 5
	if d := r.URL.Query().Get("depth"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			maxDepth = parsed
		}
	}

	result, err := h.svc.GetAIOpsGraphTrace(r.Context(), clusterID, fromKey, direction, maxDepth)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handler.WriteJSON(w, http.StatusOK, result)
}

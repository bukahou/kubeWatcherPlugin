// atlhyper_master_v2/gateway/handler/aiops_risk.go
// AIOps 风险评分 API Handler
package aiops

import (
	"net/http"
	"strconv"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/service"
)

// AIOpsRiskHandler AIOps 风险评分 Handler
type AIOpsRiskHandler struct {
	svc service.Query
}

// NewAIOpsRiskHandler 创建 Handler
func NewAIOpsRiskHandler(svc service.Query) *AIOpsRiskHandler {
	return &AIOpsRiskHandler{svc: svc}
}

// ClusterRisk 获取集群风险评分
// GET /api/v2/aiops/risk/cluster?cluster={id}
func (h *AIOpsRiskHandler) ClusterRisk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clusterID := handler.ClusterIDFromQuery(r)
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "missing cluster_id parameter")
		return
	}

	risk, err := h.svc.GetAIOpsClusterRisk(r.Context(), clusterID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handler.WriteJSON(w, http.StatusOK, ScaleClusterRisk(risk))
}

// EntityRisks 获取实体风险列表
// GET /api/v2/aiops/risk/entities?cluster={id}&sort=r_final&limit=20
func (h *AIOpsRiskHandler) EntityRisks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clusterID := handler.ClusterIDFromQuery(r)
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "missing cluster_id parameter")
		return
	}

	sortBy := r.URL.Query().Get("sort")
	if sortBy == "" {
		sortBy = "r_final"
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	risks, err := h.svc.GetAIOpsEntityRisks(r.Context(), clusterID, sortBy, limit)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handler.WriteJSON(w, http.StatusOK, ScaleEntityRisks(risks))
}

// EntityRisk 获取单个实体的风险详情
// GET /api/v2/aiops/risk/entity?cluster={id}&entity={key}
func (h *AIOpsRiskHandler) EntityRisk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clusterID := handler.ClusterIDFromQuery(r)
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "missing cluster_id parameter")
		return
	}

	entityKey := r.URL.Query().Get("entity")
	if entityKey == "" {
		handler.WriteError(w, http.StatusBadRequest, "missing entity parameter")
		return
	}

	detail, err := h.svc.GetAIOpsEntityRisk(r.Context(), clusterID, entityKey)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handler.WriteJSON(w, http.StatusOK, ScaleEntityRiskDetail(detail))
}

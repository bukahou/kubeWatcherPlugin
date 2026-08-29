// atlhyper_master_v2/gateway/handler/aiops_baseline.go
// AIOps 基线 API Handler
package aiops

import (
	"AtlHyper/atlhyper_master_v2/aiops"
	"math"
	"net/http"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/service"
)

// AIOpsBaselineHandler AIOps 基线 Handler
type AIOpsBaselineHandler struct {
	svc service.Query
}

// NewAIOpsBaselineHandler 创建 Handler
func NewAIOpsBaselineHandler(svc service.Query) *AIOpsBaselineHandler {
	return &AIOpsBaselineHandler{svc: svc}
}

// Baseline 获取实体基线状态
// GET /api/v2/aiops/baseline?cluster={id}&entity={key}
func (h *AIOpsBaselineHandler) Baseline(w http.ResponseWriter, r *http.Request) {
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

	baseline, err := h.svc.GetAIOpsBaseline(r.Context(), clusterID, entityKey)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handler.WriteJSON(w, http.StatusOK, buildBaselineResponse(baseline))
}

// ──────────────────────────────────────────────────────────────
// 响应模型
// ──────────────────────────────────────────────────────────────
//
// 引擎的 BaselineState 只有原始 EMA/方差/采样数，前端无法据此判断
// 「这个基线能不能信」。冷启动阈值（ColdStartMinCount）是引擎内部常量，
// 由后端随响应下发 —— 前端硬编码副本会在阈值调整时静默失效。

type baselineStateResponse struct {
	MetricName      string  `json:"metricName"`
	EMA             float64 `json:"ema"`
	Variance        float64 `json:"variance"`
	StdDev          float64 `json:"stdDev"` // √variance，前端直接展示，不在前端做开方
	Count           int64   `json:"count"`
	ConsecutiveZero int64   `json:"consecutiveZero"`
	UpdatedAt       int64   `json:"updatedAt"`
	Ready           bool    `json:"ready"` // 采样数已达冷启动阈值，基线可用于异常判定
}

type baselineResponse struct {
	EntityKey         string                  `json:"entityKey"`
	States            []baselineStateResponse `json:"states"`
	ColdStartMinCount int64                   `json:"coldStartMinCount"`
}

func buildBaselineResponse(src *aiops.EntityBaseline) *baselineResponse {
	if src == nil {
		return nil
	}
	threshold := int64(aiops.ColdStartMinCount)
	states := make([]baselineStateResponse, 0, len(src.States))
	for _, st := range src.States {
		if st == nil {
			continue
		}
		states = append(states, baselineStateResponse{
			MetricName:      st.MetricName,
			EMA:             st.EMA,
			Variance:        st.Variance,
			StdDev:          math.Sqrt(st.Variance),
			Count:           st.Count,
			ConsecutiveZero: st.ConsecutiveZero,
			UpdatedAt:       st.UpdatedAt,
			Ready:           st.Count >= threshold,
		})
	}
	return &baselineResponse{
		EntityKey:         src.EntityKey,
		States:            states,
		ColdStartMinCount: threshold,
	}
}

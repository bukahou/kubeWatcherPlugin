// observe_freshness.go 信号新鲜度 Handler
package observe

import (
	"net/http"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
)

// Freshness GET /api/v2/observe/freshness
//
// 四个信号页共用：空数据时用它区分「没有流量」和「采集异常」。
func (h *ObserveHandler) Freshness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	freshness, err := h.querySvc.GetSignalFreshness(r.Context(), clusterID)
	if err != nil || freshness == nil {
		handler.WriteError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    freshness,
	})
}

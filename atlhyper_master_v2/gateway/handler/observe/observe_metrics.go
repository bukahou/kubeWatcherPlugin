// atlhyper_master_v2/gateway/handler/observe_metrics.go
// Metrics 信号域 Handler 方法
package observe

import (
	"net/http"
	"strconv"
	"strings"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
)

// MetricsSummary GET /api/v2/observe/metrics/summary (Dashboard: 快照直读)
func (h *ObserveHandler) MetricsSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	otel, err := h.querySvc.GetOTelSnapshot(r.Context(), clusterID)
	if err != nil || otel == nil || otel.MetricsSummary == nil {
		handler.WriteError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    otel.MetricsSummary,
	})
}

// MetricsNodes GET /api/v2/observe/metrics/nodes (Dashboard: 快照直读)
func (h *ObserveHandler) MetricsNodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	otel, err := h.querySvc.GetOTelSnapshot(r.Context(), clusterID)
	if err != nil || otel == nil || otel.MetricsNodes == nil {
		handler.WriteError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    otel.MetricsNodes,
	})
}

// MetricsHardware GET /api/v2/observe/metrics/hardware
//
// 硬件健康矩阵：每格自带 status，缺传感器的格子为 null（前端显示「无数据」）
func (h *ObserveHandler) MetricsHardware(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	health, err := h.querySvc.GetHardwareHealth(r.Context(), clusterID)
	if err != nil || health == nil {
		handler.WriteError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    health,
	})
}

// MetricsNodeRoute GET /api/v2/observe/metrics/nodes/{name}[/series]
//
// 单节点详情: 从快照 MetricsNodes 中过滤
// 节点时序: 优先从预聚合时序读取，≤15min 降级到 OTel Ring Buffer
func (h *ObserveHandler) MetricsNodeRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	// 解析路径: /api/v2/observe/metrics/nodes/{name}[/series]
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/observe/metrics/nodes/")
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		handler.WriteError(w, http.StatusBadRequest, "node name is required")
		return
	}

	parts := strings.SplitN(path, "/", 2)
	nodeName := parts[0]

	otel, err := h.querySvc.GetOTelSnapshot(r.Context(), clusterID)
	if err != nil || otel == nil {
		handler.WriteError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}

	if len(parts) == 2 && parts[1] == "series" {
		// GET /api/v2/observe/metrics/nodes/{name}/series
		metric := r.URL.Query().Get("metric")
		minutes := 30
		if v := r.URL.Query().Get("minutes"); v != "" {
			if m, err := strconv.Atoi(v); err == nil && m > 0 {
				minutes = m
			}
		}

		points, _, err := resolveNodeSeries(h.querySvc, r.Context(), clusterID, nodeName, metric, minutes)
		if err == nil && len(points) > 0 {
			handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"message": "获取成功",
				"data": map[string]interface{}{
					"metric": metric,
					"points": points,
				},
			})
			return
		}

		// 层 3: Command/MQ → ClickHouse（>60min，暂返回未就绪）
		handler.WriteError(w, http.StatusNotFound, "时序数据未就绪")
	} else {
		// GET /api/v2/observe/metrics/nodes/{name} — 从快照过滤
		if otel.MetricsNodes != nil {
			for _, node := range otel.MetricsNodes {
				if node.NodeName == nodeName {
					handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
						"message": "获取成功",
						"data":    node,
					})
					return
				}
			}
		}
		handler.WriteError(w, http.StatusNotFound, "节点未找到")
	}
}

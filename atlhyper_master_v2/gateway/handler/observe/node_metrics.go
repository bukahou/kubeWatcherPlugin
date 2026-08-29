// Package handler Gateway HTTP 处理器
//
// node_metrics.go - 节点指标 API 处理器
//
// 数据源: OTelSnapshot（内存直读）+ Ring Buffer / Concentrator / Command(CH)
// 替代原 SQLite node_metrics_latest/node_metrics_history 表
package observe

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/model/convert"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/command"
	"AtlHyper/model_v3/metrics"
)

// NodeMetricsHandler 节点指标处理器
//
// 依赖:
//   - querySvc: OTelSnapshot 读取（实时指标 + Ring Buffer + Concentrator）
//   - ops: Command 发送（>60min 历史查询 → Agent → ClickHouse）
//   - bus: 等待 Command 结果
type NodeMetricsHandler struct {
	querySvc ObserveQuery
	ops      ObserveOps
}

// NewNodeMetricsHandler 创建节点指标处理器
func NewNodeMetricsHandler(querySvc ObserveQuery, ops ObserveOps) *NodeMetricsHandler {
	return &NodeMetricsHandler{
		querySvc: querySvc,
		ops:      ops,
	}
}

// Route 路由分发
// 处理 /api/v2/node-metrics/* 路径
func (h *NodeMetricsHandler) Route(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// 解析路径: /api/v2/node-metrics/{nodeName}[/history]
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/node-metrics")
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		// /api/v2/node-metrics 或 /api/v2/node-metrics/ -> 列表
		h.List(w, r)
		return
	}

	// 检查是否有 /history 后缀
	if strings.HasSuffix(path, "/history") {
		nodeName := strings.TrimSuffix(path, "/history")
		h.getHistory(w, r, nodeName)
		return
	}

	// /api/v2/node-metrics/{nodeName} -> 详情
	h.getDetail(w, r, path)
}

// List 获取集群所有节点指标
// GET /api/v2/node-metrics?cluster_id=xxx
//
// 数据源: OTelSnapshot.NodeMetrics（内存快照）→ convert → API 模型
func (h *NodeMetricsHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	// 从内存快照读取节点指标
	snapshot, err := h.querySvc.GetSnapshot(r.Context(), clusterID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if snapshot == nil || snapshot.OTel == nil || len(snapshot.OTel.MetricsNodes) == 0 {
		handler.WriteJSON(w, http.StatusOK, model.ClusterNodeMetricsResponse{
			Summary: model.ClusterMetricsSummary{},
			Nodes:   []model.NodeMetricsSnapshot{},
		})
		return
	}

	// 构建节点列表
	nodes := make([]*metrics.NodeMetrics, 0, len(snapshot.OTel.MetricsNodes))
	for i := range snapshot.OTel.MetricsNodes {
		nodes = append(nodes, &snapshot.OTel.MetricsNodes[i])
	}

	// 计算汇总统计并转换为 API 响应
	summary := calculateSummary(nodes)

	handler.WriteJSON(w, http.StatusOK, model.ClusterNodeMetricsResponse{
		Summary: convert.ClusterMetricsSummary(summary),
		Nodes:   convert.NodeMetricsSnapshots(nodes),
	})
}

// getDetail 获取单节点详情
//
// 数据源: OTelSnapshot.NodeMetrics[nodeName]
func (h *NodeMetricsHandler) getDetail(w http.ResponseWriter, r *http.Request, nodeName string) {
	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	snapshot, err := h.querySvc.GetSnapshot(r.Context(), clusterID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if snapshot == nil || snapshot.OTel == nil || len(snapshot.OTel.MetricsNodes) == 0 {
		handler.WriteError(w, http.StatusNotFound, "node metrics not found")
		return
	}

	var nodeMetrics *metrics.NodeMetrics
	for i := range snapshot.OTel.MetricsNodes {
		if snapshot.OTel.MetricsNodes[i].NodeName == nodeName {
			nodeMetrics = &snapshot.OTel.MetricsNodes[i]
			break
		}
	}
	if nodeMetrics == nil {
		handler.WriteError(w, http.StatusNotFound, "node metrics not found")
		return
	}

	handler.WriteJSON(w, http.StatusOK, convert.NodeMetricsSnapshot(nodeMetrics))
}

// getHistory 获取节点历史数据
//
// 3 层路由:
//   - ≤15min: Ring Buffer（10s 精度）
//   - ≤60min: Concentrator 预聚合（1min 精度）
//   - >60min: Command → Agent → ClickHouse
func (h *NodeMetricsHandler) getHistory(w http.ResponseWriter, r *http.Request, nodeName string) {
	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	// 解析时间范围
	hours := 24
	if hoursStr := r.URL.Query().Get("hours"); hoursStr != "" {
		if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 {
			hours = h
		}
	}

	minutes := hours * 60
	end := time.Now()
	start := end.Add(-time.Duration(hours) * time.Hour)

	// 层 1: Ring Buffer（≤15min）— 10s 精度
	if minutes <= 15 {
		since := time.Now().Add(-time.Duration(minutes) * time.Minute)
		entries, err := h.querySvc.GetOTelTimeline(r.Context(), clusterID, since)
		if err == nil && len(entries) > 0 {
			data := buildNodeHistoryFromTimeline(entries, nodeName)
			if hasData(data) {
				handler.WriteJSON(w, http.StatusOK, model.NodeMetricsHistoryResponse{
					NodeName: nodeName,
					Start:    start,
					End:      end,
					Data:     data,
				})
				return
			}
		}
	}

	// 层 2: Concentrator 预聚合（≤60min）— 1min 精度
	if minutes <= 60 {
		otel, err := h.querySvc.GetOTelSnapshot(r.Context(), clusterID)
		if err == nil && otel != nil && otel.NodeMetricsSeries != nil {
			for _, ns := range otel.NodeMetricsSeries {
				if ns.NodeName == nodeName {
					points := filterNodePointsByMinutes(ns.Points, minutes)
					data := buildNodeHistoryFromConcentrator(points)
					handler.WriteJSON(w, http.StatusOK, model.NodeMetricsHistoryResponse{
						NodeName: nodeName,
						Start:    start,
						End:      end,
						Data:     data,
					})
					return
				}
			}
		}
	}

	// 层 3: Command → Agent → ClickHouse（>60min）
	h.getHistoryFromCH(w, r, clusterID, nodeName, start, end, hours)
}

// getHistoryFromCH 通过 Command 管道从 ClickHouse 获取历史数据
func (h *NodeMetricsHandler) getHistoryFromCH(w http.ResponseWriter, r *http.Request, clusterID, nodeName string, start, end time.Time, hours int) {
	params := map[string]interface{}{
		"sub_action": "get_history",
		"node_name":  nodeName,
		"since":      (time.Duration(hours) * time.Hour).String(),
	}

	// 同步执行指令（创建 + 等待结果，30s 超时）
	result, err := h.ops.ExecuteCommandSync(r.Context(), &model.CreateCommandRequest{
		ClusterID: clusterID,
		Action:    command.ActionQueryMetrics,
		Params:    params,
		Source:    "web",
	}, 30*time.Second)
	if err != nil {
		if strings.Contains(err.Error(), "create command:") {
			handler.WriteError(w, http.StatusInternalServerError, "创建查询指令失败: "+err.Error())
		} else {
			handler.WriteError(w, http.StatusGatewayTimeout, "查询超时，请稍后重试")
		}
		return
	}
	if result == nil {
		handler.WriteError(w, http.StatusGatewayTimeout, "查询超时，请稍后重试")
		return
	}

	if !result.Success {
		errMsg := result.Error
		if errMsg == "" {
			errMsg = "查询失败"
		}
		handler.WriteError(w, http.StatusInternalServerError, errMsg)
		return
	}

	// 透传 Agent 返回的 JSON
	var data map[string][]model.TimeSeriesPoint
	if err := json.Unmarshal([]byte(result.Output), &data); err != nil {
		// 直接透传原始 JSON
		handler.WriteJSON(w, http.StatusOK, model.NodeMetricsHistoryResponse{
			NodeName: nodeName,
			Start:    start,
			End:      end,
			Data:     map[string][]model.TimeSeriesPoint{},
		})
		return
	}

	handler.WriteJSON(w, http.StatusOK, model.NodeMetricsHistoryResponse{
		NodeName: nodeName,
		Start:    start,
		End:      end,
		Data:     data,
	})
}

// calculateSummary 计算集群汇总统计
func calculateSummary(nodes []*metrics.NodeMetrics) metrics.Summary {
	summary := metrics.Summary{
		TotalNodes:  len(nodes),
		OnlineNodes: len(nodes),
	}

	if len(nodes) == 0 {
		return summary
	}

	var (
		totalCPU, totalMem float64
		maxCPU, maxMem     float64
		totalTemp          float64
		tempCount          int
	)

	for _, node := range nodes {
		// CPU
		totalCPU += node.CPU.UsagePct
		if node.CPU.UsagePct > maxCPU {
			maxCPU = node.CPU.UsagePct
		}

		// 内存
		totalMem += node.Memory.UsagePct
		if node.Memory.UsagePct > maxMem {
			maxMem = node.Memory.UsagePct
		}

		// 温度
		if node.Temperature.CPUTempC > 0 {
			totalTemp += node.Temperature.CPUTempC
			tempCount++
			if node.Temperature.CPUTempC > summary.MaxCPUTemp {
				summary.MaxCPUTemp = node.Temperature.CPUTempC
			}
		}
	}

	n := float64(len(nodes))
	summary.AvgCPUPct = totalCPU / n
	summary.AvgMemPct = totalMem / n
	summary.MaxCPUPct = maxCPU
	summary.MaxMemPct = maxMem

	if tempCount > 0 {
		summary.MaxCPUTemp = totalTemp / float64(tempCount)
	}

	return summary
}

// ==================== 历史数据构建辅助函数 ====================

// buildNodeHistoryFromTimeline 从 Ring Buffer 时间线构建历史数据
func buildNodeHistoryFromTimeline(entries []cluster.OTelEntry, nodeName string) map[string][]model.TimeSeriesPoint {
	data := map[string][]model.TimeSeriesPoint{
		"cpu":    {},
		"memory": {},
		"disk":   {},
		"temp":   {},
	}

	for _, e := range entries {
		if e.Snapshot == nil || e.Snapshot.MetricsNodes == nil {
			continue
		}
		for _, node := range e.Snapshot.MetricsNodes {
			if node.NodeName == nodeName {
				ts := e.Timestamp.UTC().Format(time.RFC3339)
				data["cpu"] = append(data["cpu"], model.TimeSeriesPoint{Timestamp: ts, Value: node.CPU.UsagePct})
				data["memory"] = append(data["memory"], model.TimeSeriesPoint{Timestamp: ts, Value: node.Memory.UsagePct})
				if d := node.GetPrimaryDisk(); d != nil {
					data["disk"] = append(data["disk"], model.TimeSeriesPoint{Timestamp: ts, Value: d.UsagePct})
				}
				data["temp"] = append(data["temp"], model.TimeSeriesPoint{Timestamp: ts, Value: node.Temperature.CPUTempC})
				break
			}
		}
	}

	return data
}

// buildNodeHistoryFromConcentrator 从 Concentrator 预聚合时序构建历史数据
func buildNodeHistoryFromConcentrator(points []cluster.NodeMetricsPoint) map[string][]model.TimeSeriesPoint {
	data := map[string][]model.TimeSeriesPoint{
		"cpu":    make([]model.TimeSeriesPoint, 0, len(points)),
		"memory": make([]model.TimeSeriesPoint, 0, len(points)),
		"disk":   make([]model.TimeSeriesPoint, 0, len(points)),
		"temp":   make([]model.TimeSeriesPoint, 0, len(points)),
	}

	for _, p := range points {
		ts := p.Timestamp.UTC().Format(time.RFC3339)
		data["cpu"] = append(data["cpu"], model.TimeSeriesPoint{Timestamp: ts, Value: p.CPUPct})
		data["memory"] = append(data["memory"], model.TimeSeriesPoint{Timestamp: ts, Value: p.MemPct})
		data["disk"] = append(data["disk"], model.TimeSeriesPoint{Timestamp: ts, Value: p.DiskPct})
		data["temp"] = append(data["temp"], model.TimeSeriesPoint{Timestamp: ts, Value: p.CPUTempC})
	}

	return data
}

// hasData 检查历史数据是否有内容
func hasData(data map[string][]model.TimeSeriesPoint) bool {
	for _, points := range data {
		if len(points) > 0 {
			return true
		}
	}
	return false
}

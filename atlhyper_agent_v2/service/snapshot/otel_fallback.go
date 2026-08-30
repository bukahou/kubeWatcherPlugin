// atlhyper_agent_v2/service/snapshot/otel_fallback.go
// OTel 快照的逐信号回退
package snapshot

import "AtlHyper/model_v3/cluster"

// restoreMissingFromPrev 把本轮为 nil 的信号从上一份快照补回。
//
// 为什么需要（2026-08-30 实证）：9 路 dashboard 查询各自独立，
// 任一路失败只会 log.Warn 后 return，该字段留 nil；而带 nil 的快照
// 会被写进 30s TTL 缓存 → 之后 30 秒每轮都复制这个 nil →
// Master 见 MetricsNodes == nil 返 404「数据尚未就绪」→
// 面板空白 30 秒后突然恢复。用户观察到的「有时候没数据、有时候又出现」。
//
// 原则与 2026-08-29 修的 buildNodeMetrics 缺陷一致：
// **部分失败不该丢弃全部**。这里更进一步 —— 失败的那部分沿用上一份已知值，
// 优于让它空着（陈旧数据带新鲜度标记可辨识，空白则无从判断）。
//
// 仅补 nil，不补空切片：空切片是「查询成功但结果为空」的真实结论，
// 覆盖它会把「集群里没有节点」谎报成上一轮的节点列表。
// RestoredSignals 标记哪些信号是从上一份快照回退来的。
// 调用方据此跳过 Concentrator 摄入 —— 回退数据不是新采样点，
// 重复摄入会让预聚合时序把「查询失败」伪装成「值保持不变」。
type RestoredSignals struct {
	MetricsNodes bool
	APMServices  bool
	SLOIngress   bool
}

func restoreMissingFromPrev(fresh, prev *cluster.OTelSnapshot) RestoredSignals {
	var r RestoredSignals
	if fresh == nil || prev == nil {
		return r
	}

	// 切片型：nil = 查询失败，空切片 = 真实为空
	if fresh.MetricsNodes == nil {
		fresh.MetricsNodes = prev.MetricsNodes
		r.MetricsNodes = true
	}
	if fresh.APMServices == nil {
		fresh.APMServices = prev.APMServices
		r.APMServices = true
	}
	if fresh.SLOIngress == nil {
		fresh.SLOIngress = prev.SLOIngress
		r.SLOIngress = true
	}
	if fresh.APMOperations == nil {
		fresh.APMOperations = prev.APMOperations
	}
	if fresh.RecentTraces == nil {
		fresh.RecentTraces = prev.RecentTraces
	}
	if fresh.RecentLogs == nil {
		fresh.RecentLogs = prev.RecentLogs
	}

	// 指针型：nil = 未取到
	if fresh.MetricsSummary == nil {
		fresh.MetricsSummary = prev.MetricsSummary
	}
	if fresh.APMTopology == nil {
		fresh.APMTopology = prev.APMTopology
	}
	if fresh.SLOSummary == nil {
		fresh.SLOSummary = prev.SLOSummary
	}
	if fresh.LogsSummary == nil {
		fresh.LogsSummary = prev.LogsSummary
	}

	// 注意：Freshness 不回退 —— 它是「数据有多新」的判断依据，
	// 必须反映本轮实测，否则页面拿着旧数据却显示「一切正常」。
	// 预聚合时序（NodeMetricsSeries 等）由 Concentrator 每轮重算，同样不回退。
	return r
}

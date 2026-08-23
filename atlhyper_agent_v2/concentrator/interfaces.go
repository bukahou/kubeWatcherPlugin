// Package concentrator 本地时序聚合器接口
package concentrator

import (
	"time"

	"AtlHyper/model_v3/apm"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/metrics"
	"AtlHyper/model_v3/slo"
)

// TimeSeriesAggregator 本地时序聚合器接口
// 维护最近 1 小时的降采样数据（环形缓冲，1 分钟粒度）。
type TimeSeriesAggregator interface {
	// Ingest 摄入当前 OTel 快照数据，更新时序环
	Ingest(
		nodes []metrics.NodeMetrics,
		sloIngress []slo.IngressSLO,
		apmServices []apm.APMService,
		ts time.Time,
	)
	// FlushNodeSeries 输出所有节点的预聚合时序
	FlushNodeSeries() []cluster.NodeMetricsTimeSeries
	// FlushSLOSeries 输出所有服务的预聚合 SLO 时序
	FlushSLOSeries() []cluster.SLOServiceTimeSeries
	// FlushAPMSeries 输出所有服务的预聚合 APM 时序
	FlushAPMSeries() []cluster.APMServiceTimeSeries
}

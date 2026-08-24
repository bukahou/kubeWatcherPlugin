// observe.go 观测模块的跨信号响应模型
package model

// FreshnessStatus 信号的数据新鲜度状态。
//
// 存在的理由：页面上「没有流量」和「采集挂了」都表现为空白，但一个不用管、
// 一个要救火。让用户自己去查 ClickHouse 才能分辨，是观测平台的失职。
type FreshnessStatus string

const (
	// FreshnessLive 数据新鲜
	FreshnessLive FreshnessStatus = "live"
	// FreshnessIdle 超过阈值，但采集链路是好的 —— 就是没有请求
	FreshnessIdle FreshnessStatus = "idle"
	// FreshnessStale 采集异常：连拉取式的 metrics 都停了
	FreshnessStale FreshnessStatus = "stale"
	// FreshnessAbsent 从未有过数据（新集群，或该信号未接入）
	FreshnessAbsent FreshnessStatus = "absent"
)

// SignalFreshnessItem 单个信号的新鲜度
type SignalFreshnessItem struct {
	Signal     string          `json:"signal"`               // metrics / traces / logs
	LastDataAt string          `json:"lastDataAt,omitempty"` // RFC3339；从未有数据时为空
	LagSeconds int64           `json:"lagSeconds"`
	Status     FreshnessStatus `json:"status"`
}

// FreshnessResponse GET /api/v2/observe/freshness 的响应
type FreshnessResponse struct {
	Signals []SignalFreshnessItem `json:"signals"`
	// CollectorHealthy 采集链路是否健康。以 metrics 为准 —— 它是主动拉取的，
	// 只要节点活着就该有数据；它停了说明链路有问题，而不是没人访问。
	CollectorHealthy bool `json:"collectorHealthy"`
}

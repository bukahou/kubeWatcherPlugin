// Package slo 定义 SLO 数据模型
//
// 数据源: Traefik (otel_metrics_sum + otel_metrics_histogram)
//
//	Linkerd (otel_metrics_gauge)
package slo

import "time"

// ============================================================
// IngressSLO — Traefik 入口服务级指标
// ============================================================

type IngressSLO struct {
	ServiceKey    string            `json:"serviceKey"`
	DisplayName   string            `json:"displayName"`
	RPS           float64           `json:"rps"`
	SuccessRate   float64           `json:"successRate"`
	ErrorRate     float64           `json:"errorRate"`
	P50Ms         float64           `json:"p50Ms"`
	P90Ms         float64           `json:"p90Ms"`
	P95Ms         float64           `json:"p95Ms"`
	P99Ms         float64           `json:"p99Ms"`
	AvgMs         float64           `json:"avgMs"`
	StatusCodes   []StatusCodeCount `json:"statusCodes"`
	TotalRequests int64             `json:"totalRequests"`
	TotalErrors   int64             `json:"totalErrors"`

	// 延迟分布桶（Traefik histogram ExplicitBounds）
	LatencyBuckets []LatencyBucket `json:"latencyBuckets,omitempty"`
	// HTTP 方法分布
	Methods []MethodCount `json:"methods,omitempty"`
}

// LatencyBucket 延迟分布桶
type LatencyBucket struct {
	LE    float64 `json:"le"`    // 上界 (ms)
	Count int64   `json:"count"` // 请求数
}

// MethodCount HTTP 方法计数
type MethodCount struct {
	Method string `json:"method"`
	Count  int64  `json:"count"`
}

// StatusCodeCount HTTP 状态码分布
type StatusCodeCount struct {
	Code  string `json:"code"`
	Count int64  `json:"count"`
}

// ============================================================
// ServiceSLO — Linkerd 服务网格指标
// ============================================================

// ============================================================
// ServiceEdge — 服务间调用关系（Linkerd outbound）
// ============================================================

// ============================================================
// SLO 时序数据
// ============================================================

type DataPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	RPS         float64   `json:"rps"`
	SuccessRate float64   `json:"successRate"`
	P50Ms       float64   `json:"p50Ms"`
	P99Ms       float64   `json:"p99Ms"`
}

type TimeSeries struct {
	Namespace string      `json:"namespace,omitempty"`
	Name      string      `json:"name"`
	Points    []DataPoint `json:"points"`
}

// ============================================================
// SLO 多窗口数据（1d/7d/30d 预聚合）
// ============================================================

// SLOWindowData 单个时间窗口的完整 SLO 数据
type SLOWindowData struct {
	Current  []IngressSLO      `json:"current"`            // 当前窗口聚合
	Previous []IngressSLO      `json:"previous,omitempty"` // 上一周期聚合（用于对比）
	History  []SLOHistoryPoint `json:"history,omitempty"`  // 时序数据

	// Mesh 数据（Linkerd 服务网格，按窗口时间范围聚合）
}

// SLOHistoryPoint 时序数据点（按桶聚合）
type SLOHistoryPoint struct {
	Timestamp     time.Time `json:"timestamp"`
	ServiceKey    string    `json:"serviceKey"`
	Availability  float64   `json:"availability"` // 0-100
	RPS           float64   `json:"rps"`
	ErrorRate     float64   `json:"errorRate"` // 0-100
	P95Ms         float64   `json:"p95Ms"`
	P99Ms         float64   `json:"p99Ms"`
	TotalRequests int64     `json:"totalRequests"`
}

// ============================================================
// SLOSummary — 仪表盘摘要
// ============================================================

type SLOSummary struct {
	TotalServices    int     `json:"totalServices"`
	HealthyServices  int     `json:"healthyServices"`
	WarningServices  int     `json:"warningServices"`
	CriticalServices int     `json:"criticalServices"`
	AvgSuccessRate   float64 `json:"avgSuccessRate"`
	TotalRPS         float64 `json:"totalRps"`
	AvgP99Ms         float64 `json:"avgP99Ms"`
}

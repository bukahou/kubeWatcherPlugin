// atlhyper_master_v2/model/slo.go
// SLO API 类型定义
package model

import "AtlHyper/atlhyper_master_v2/slo"

// ==================== API 响应类型 ====================

// DomainSLO 域名 SLO 信息
type DomainSLO struct {
	Host         string         `json:"host"`
	IngressName  string         `json:"ingressName"`
	IngressClass string         `json:"ingressClass"`
	Namespace    string         `json:"namespace"`
	TLS          bool           `json:"tls"`
	Target       *SLOTargetSpec `json:"target"`
	Budget       *SLOBudget     `json:"budget,omitempty"`
	Current      *SLOMetrics    `json:"current"`
	Previous     *SLOMetrics    `json:"previous,omitempty"`
	ErrorBudget  float64        `json:"errorBudgetRemaining"`
	Status       string         `json:"status"` // healthy / warning / critical
	Trend        string         `json:"trend"`  // up / down / stable
}

// SLOTargetSpec SLO 目标规格。
// 一个域名一组目标：固定滚动窗口 + 可用率 + P95 延迟。
// 页面上的时间范围切换只影响图表画多长，不改变目标。
type SLOTargetSpec struct {
	Availability float64 `json:"availability"`
	P95Latency   int     `json:"p95Latency"`
	WindowDays   int     `json:"windowDays"`
}

// SLOBudget 错误预算与多窗口燃烧率（判定已在后端完成，前端只渲染）
type SLOBudget struct {
	RemainingPct   float64              `json:"remainingPct"`
	AllowedEvents  int64                `json:"allowedEvents"`
	ConsumedEvents int64                `json:"consumedEvents"`
	BurnRates      []slo.BurnRateWindow `json:"burnRates"`
	// ExhaustHours 按当前燃烧率还能撑多少小时；0 表示窗口内不会耗尽
	ExhaustHours float64 `json:"exhaustHours"`
}

// SLOMetrics SLO 指标
type SLOMetrics struct {
	Availability   float64 `json:"availability"`
	P95Latency     int     `json:"p95Latency"`
	P99Latency     int     `json:"p99Latency"`
	ErrorRate      float64 `json:"errorRate"`
	RequestsPerSec float64 `json:"requestsPerSec"`
	TotalRequests  int64   `json:"totalRequests"`
	// GoodRequests / BadRequests 把分子分母摆出来。低流量时百分比极具误导性：
	// 10 个请求错 1 个，可用率 90% 看着像事故，其实什么都没发生。
	GoodRequests int64 `json:"goodRequests"`
	BadRequests  int64 `json:"badRequests"`
}

// SLOSummary SLO 汇总信息
type SLOSummary struct {
	TotalServices   int     `json:"totalServices"`
	TotalDomains    int     `json:"totalDomains"`
	HealthyCount    int     `json:"healthyCount"`
	WarningCount    int     `json:"warningCount"`
	CriticalCount   int     `json:"criticalCount"`
	AvgAvailability float64 `json:"avgAvailability"`
	AvgErrorBudget  float64 `json:"avgErrorBudget"`
	TotalRPS        float64 `json:"totalRps"`
}

// SLODomainsResponse 域名列表响应 (V1 兼容，使用 host/service key)
type SLODomainsResponse struct {
	Domains []DomainSLO `json:"domains"`
	Summary SLOSummary  `json:"summary"`
}

// ==================== V2 API 响应类型（按真实域名分组）====================

// DomainSLOResponseV2 域名级别的 SLO 响应 (V2)
// 以真实域名为单位，包含该域名下的所有后端服务
type DomainSLOResponseV2 struct {
	Domain               string         `json:"domain"`               // 真实域名（如 example.com）
	TLS                  bool           `json:"tls"`                  // 是否启用 TLS
	Services             []ServiceSLO   `json:"services"`             // 该域名下的所有后端服务
	Summary              *SLOMetrics    `json:"summary"`              // 域名级别汇总指标
	Previous             *SLOMetrics    `json:"previous,omitempty"`   // 上一周期汇总指标
	Target               *SLOTargetSpec `json:"target"`               // 该域名的 SLO 目标
	Budget               *SLOBudget     `json:"budget,omitempty"`     // 错误预算与燃烧率
	Status               string         `json:"status"`               // healthy / warning / critical
	ErrorBudgetRemaining float64        `json:"errorBudgetRemaining"` // 剩余错误预算
}

// ServiceSLO 后端服务级别的 SLO 数据（Metrics 的实际数据来源）
type ServiceSLO struct {
	ServiceKey  string         `json:"serviceKey"`           // Traefik service key (namespace-name-port@kubernetes)
	ServiceName string         `json:"serviceName"`          // 服务名称
	ServicePort int            `json:"servicePort"`          // 服务端口
	Namespace   string         `json:"namespace"`            // 命名空间
	Paths       []string       `json:"paths"`                // 使用该服务的路径列表
	IngressName string         `json:"ingressName"`          // IngressRoute/Ingress 名称
	Current     *SLOMetrics    `json:"current"`              // 当前周期指标
	Previous    *SLOMetrics    `json:"previous,omitempty"`   // 上一周期指标（用于对比）
	Target      *SLOTargetSpec `json:"target"`               // 该服务的 SLO 目标
	Budget      *SLOBudget     `json:"budget,omitempty"`     // 错误预算与燃烧率
	Status      string         `json:"status"`               // healthy / warning / critical
	ErrorBudget float64        `json:"errorBudgetRemaining"` // 剩余错误预算
}

// SLODomainsResponseV2 域名列表响应 (V2)
type SLODomainsResponseV2 struct {
	Domains []DomainSLOResponseV2 `json:"domains"`
	Summary SLOSummary            `json:"summary"`
}

// SLODomainHistoryItem 域名历史数据项
type SLODomainHistoryItem struct {
	Timestamp    string  `json:"timestamp"`
	Availability float64 `json:"availability"`
	P95Latency   int     `json:"p95Latency"`
	P99Latency   int     `json:"p99Latency"`
	RPS          float64 `json:"rps"`
	ErrorRate    float64 `json:"errorRate"`
	ErrorBudget  float64 `json:"errorBudget"`
}

// SLODomainHistoryResponse 域名历史响应
type SLODomainHistoryResponse struct {
	Host    string                 `json:"host"`
	History []SLODomainHistoryItem `json:"history"`
}

// SLOStatusHistoryItem 状态变更历史项
type SLOStatusHistoryItem struct {
	Host                 string  `json:"host"`
	TimeRange            string  `json:"timeRange"`
	OldStatus            string  `json:"oldStatus"`
	NewStatus            string  `json:"newStatus"`
	Availability         float64 `json:"availability"`
	P95Latency           int     `json:"p95Latency"`
	ErrorBudgetRemaining float64 `json:"errorBudgetRemaining"`
	ChangedAt            string  `json:"changedAt"`
}

// ==================== 延迟分布 API 类型 ====================

// LatencyBucket 延迟分布桶
type LatencyBucket struct {
	LE    float64 `json:"le"`    // 上界 (ms)
	Count int64   `json:"count"` // 该桶内的请求数
}

// MethodBreakdown HTTP 方法分布
type MethodBreakdown struct {
	Method string `json:"method"` // GET, POST, PUT, DELETE, OTHER
	Count  int64  `json:"count"`
}

// StatusCodeBreakdown 状态码分布
type StatusCodeBreakdown struct {
	Code  string `json:"code"` // "2xx", "3xx", "4xx", "5xx"
	Count int64  `json:"count"`
}

// LatencyDistributionResponse 延迟分布响应
type LatencyDistributionResponse struct {
	Domain        string                `json:"domain"`
	TotalRequests int64                 `json:"totalRequests"`
	P50LatencyMs  int                   `json:"p50LatencyMs"`
	P95LatencyMs  int                   `json:"p95LatencyMs"`
	P99LatencyMs  int                   `json:"p99LatencyMs"`
	AvgLatencyMs  int                   `json:"avgLatencyMs"`
	Buckets       []LatencyBucket       `json:"buckets"`
	Methods       []MethodBreakdown     `json:"methods"`
	StatusCodes   []StatusCodeBreakdown `json:"statusCodes"`
}

// ==================== SLOTarget API 响应类型 ====================

// SLOTargetResponse SLO 目标（API 响应，camelCase）
type SLOTargetResponse struct {
	ID                 int64   `json:"id"`
	ClusterID          string  `json:"clusterId"`
	Host               string  `json:"host"`
	WindowDays         int     `json:"windowDays"`
	AvailabilityTarget float64 `json:"availabilityTarget"`
	P95LatencyTarget   int     `json:"p95LatencyTarget"`
	CreatedAt          string  `json:"createdAt"`
	UpdatedAt          string  `json:"updatedAt"`
}

// ==================== API 请求类型 ====================

// UpdateSLOTargetRequest 更新 SLO 目标请求
type UpdateSLOTargetRequest struct {
	ClusterID          string  `json:"clusterId"`
	Host               string  `json:"host"`
	WindowDays         int     `json:"windowDays"` // 滚动窗口天数，0 = 用默认值
	AvailabilityTarget float64 `json:"availabilityTarget"`
	P95LatencyTarget   int     `json:"p95LatencyTarget"`
}

// SLOQueryParams SLO 查询参数
type SLOQueryParams struct {
	ClusterID string `form:"cluster_id"`
	TimeRange string `form:"time_range"` // "1d", "7d", "30d"
	Host      string `form:"host"`
	Limit     int    `form:"limit"`
	Offset    int    `form:"offset"`
}

// ==================== SLO 路由映射 API 响应类型 ====================

// SLORouteMapping SLO 路由映射（Service 层返回类型，不暴露 database 层）
type SLORouteMapping struct {
	Domain      string `json:"domain"`
	PathPrefix  string `json:"pathPrefix"`
	IngressName string `json:"ingressName"`
	Namespace   string `json:"namespace"`
	TLS         bool   `json:"tls"`
	ServiceKey  string `json:"serviceKey"`
	ServiceName string `json:"serviceName"`
	ServicePort int    `json:"servicePort"`
}

// ==================== 服务网格 API 响应类型 ====================

// ServiceMeshTopologyResponse 服务拓扑响应
type ServiceMeshTopologyResponse struct {
	Nodes []ServiceNodeResponse `json:"nodes"`
	Edges []ServiceEdgeResponse `json:"edges"`
}

// ServiceNodeResponse 服务节点响应
type ServiceNodeResponse struct {
	ID            string  `json:"id"` // "namespace/name"
	Name          string  `json:"name"`
	Namespace     string  `json:"namespace"`
	RPS           float64 `json:"rps"`
	AvgLatencyMs  float64 `json:"avgLatency"`
	P50LatencyMs  float64 `json:"p50Latency"`
	P95LatencyMs  float64 `json:"p95Latency"`
	P99LatencyMs  float64 `json:"p99Latency"`
	ErrorRate     float64 `json:"errorRate"`
	Availability  float64 `json:"availability"`
	Status        string  `json:"status"` // healthy/warning/critical
	MtlsEnabled   bool    `json:"mtlsEnabled"`
	TotalRequests int64   `json:"totalRequests"`
}

// ServiceEdgeResponse 服务拓扑边响应
type ServiceEdgeResponse struct {
	Source       string  `json:"source"` // "namespace/name"
	Target       string  `json:"target"`
	RPS          float64 `json:"rps"`
	AvgLatencyMs float64 `json:"avgLatency"`
	ErrorRate    float64 `json:"errorRate"`
}

// ServiceDetailResponse 服务详情响应
type ServiceDetailResponse struct {
	ServiceNodeResponse
	History        []ServiceHistoryPoint `json:"history"`
	Upstreams      []ServiceEdgeResponse `json:"upstreams"`
	Downstreams    []ServiceEdgeResponse `json:"downstreams"`
	StatusCodes    []StatusCodeBreakdown `json:"statusCodes"`
	LatencyBuckets []LatencyBucket       `json:"latencyBuckets"`
}

// ServiceHistoryPoint 服务历史数据点
type ServiceHistoryPoint struct {
	Timestamp    string  `json:"timestamp"`
	RPS          float64 `json:"rps"`
	P95LatencyMs float64 `json:"p95Latency"`
	ErrorRate    float64 `json:"errorRate"`
	Availability float64 `json:"availability"`
	MtlsEnabled  bool    `json:"mtlsEnabled"`
}

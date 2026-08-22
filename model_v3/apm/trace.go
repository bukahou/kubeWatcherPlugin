// Package apm 定义 APM（分布式追踪）数据模型
//
// 数据源: ClickHouse otel_traces 表
package apm

import (
	"time"

	model_v3 "AtlHyper/model_v3"
)

// ============================================================
// Span — otel_traces 行的领域模型
// ============================================================

// SpanError 从 OTel exception event 提取的结构化错误信息
type SpanError struct {
	Type       string `json:"type"`
	Message    string `json:"message"`
	Stacktrace string `json:"stacktrace,omitempty"`
}

// Span 分布式追踪中的单个操作
type Span struct {
	Timestamp     time.Time `json:"timestamp"`
	TraceId       string    `json:"traceId"`
	SpanId        string    `json:"spanId"`
	ParentSpanId  string    `json:"parentSpanId"`
	SpanName      string    `json:"spanName"`
	SpanKind      string    `json:"spanKind"`
	ServiceName   string    `json:"serviceName"`
	Duration      int64     `json:"duration"`   // 纳秒
	DurationMs    float64   `json:"durationMs"` // 毫秒
	StatusCode    string    `json:"statusCode"`
	StatusMessage string    `json:"statusMessage"`

	HTTP     *SpanHTTP    `json:"http,omitempty"`
	DB       *SpanDB      `json:"db,omitempty"`
	Resource SpanResource `json:"resource"`
	Events   []SpanEvent  `json:"events"`
	Error    *SpanError   `json:"error,omitempty"`
}

type SpanHTTP struct {
	Method     string `json:"method"`
	Route      string `json:"route,omitempty"`
	URL        string `json:"url,omitempty"`
	StatusCode int    `json:"statusCode,omitempty"`
	Server     string `json:"server,omitempty"`
	ServerPort int    `json:"serverPort,omitempty"`
}

type SpanDB struct {
	System    string `json:"system"`
	Name      string `json:"name,omitempty"`
	Operation string `json:"operation,omitempty"`
	Table     string `json:"table,omitempty"`
	Statement string `json:"statement,omitempty"`
}

type SpanResource struct {
	ServiceVersion string `json:"serviceVersion,omitempty"`
	InstanceId     string `json:"instanceId,omitempty"`
	PodName        string `json:"podName,omitempty"`
	NodeName       string `json:"nodeName,omitempty"`
	DeploymentName string `json:"deploymentName,omitempty"`
	NamespaceName  string `json:"namespaceName,omitempty"`
	ClusterName    string `json:"clusterName,omitempty"`
}

type SpanEvent struct {
	Timestamp  time.Time         `json:"timestamp"`
	Name       string            `json:"name"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

func (s *Span) IsError() bool  { return s.StatusCode == StatusCodeError }
func (s *Span) IsServer() bool { return s.SpanKind == SpanKindServer }
func (s *Span) IsClient() bool { return s.SpanKind == SpanKindClient }
func (s *Span) IsRoot() bool   { return s.ParentSpanId == "" }

// ============================================================
// TraceSummary — Trace 列表项
// ============================================================

type TraceSummary struct {
	TraceId       string    `json:"traceId"`
	RootService   string    `json:"rootService"`
	RootOperation string    `json:"rootOperation"`
	Services      []string  `json:"services"`
	DurationMs    float64   `json:"durationMs"`
	SpanCount     int       `json:"spanCount"`
	ServiceCount  int       `json:"serviceCount"`
	HasError      bool      `json:"hasError"`
	ErrorType     string    `json:"errorType,omitempty"`
	ErrorMessage  string    `json:"errorMessage,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

// ============================================================
// TraceDetail — 完整 Trace（瀑布图）
// ============================================================

type TraceDetail struct {
	TraceId      string  `json:"traceId"`
	DurationMs   float64 `json:"durationMs"`
	ServiceCount int     `json:"serviceCount"`
	SpanCount    int     `json:"spanCount"`
	Spans        []Span  `json:"spans"`
}

// ============================================================
// APMService — 服务级聚合统计
// ============================================================

type APMService struct {
	Name          string  `json:"name"`
	Namespace     string  `json:"namespace"`
	Environment   string  `json:"environment,omitempty"`
	SpanCount     int64   `json:"spanCount"`
	ErrorCount    int64   `json:"errorCount"`
	SuccessRate   float64 `json:"successRate"`
	AvgDurationMs float64 `json:"avgDurationMs"`
	P50Ms         float64 `json:"p50Ms"`
	P99Ms         float64 `json:"p99Ms"`
	RPS           float64 `json:"rps"`
}

// ============================================================
// OperationStats — 操作级聚合统计（GROUP BY ServiceName, SpanName）
// ============================================================

// OperationStats 操作级聚合统计（Kibana 模式：后端 GROUP BY，前端直接展示）
type OperationStats struct {
	ServiceName   string  `json:"serviceName"`
	OperationName string  `json:"operationName"` // SpanName
	SpanCount     int64   `json:"spanCount"`
	ErrorCount    int64   `json:"errorCount"`
	SuccessRate   float64 `json:"successRate"`   // 0-1
	AvgDurationMs float64 `json:"avgDurationMs"`
	P50Ms         float64 `json:"p50Ms"`
	P99Ms         float64 `json:"p99Ms"`
	RPS           float64 `json:"rps"`
}

// ============================================================
// APM 服务拓扑
// ============================================================

type TopologyNode struct {
	Id          string               `json:"id"`
	Name        string               `json:"name"`
	Namespace   string               `json:"namespace"`
	Type        string               `json:"type"` // "service", "database", "external"
	RPS         float64              `json:"rps"`
	SuccessRate float64              `json:"successRate"`
	P99Ms       float64              `json:"p99Ms"`
	Status      model_v3.HealthStatus `json:"status"`
}

type TopologyEdge struct {
	Source    string  `json:"source"`
	Target    string  `json:"target"`
	CallCount int64   `json:"callCount"`
	AvgMs     float64 `json:"avgMs"`
	ErrorRate float64 `json:"errorRate"`
}

type Topology struct {
	Nodes []TopologyNode `json:"nodes"`
	Edges []TopologyEdge `json:"edges"`
}

// ============================================================
// HTTPStats — HTTP 状态码分布统计
// ============================================================

type HTTPStats struct {
	StatusCode int    `json:"statusCode"`
	Method     string `json:"method"`
	Operation  string `json:"operation"`
	Count      int64  `json:"count"`
}

// ============================================================
// DBOperationStats — 数据库操作统计
// ============================================================

type DBOperationStats struct {
	DBSystem  string  `json:"dbSystem"`
	DBName    string  `json:"dbName"`
	Operation string  `json:"operation"`
	Table     string  `json:"table"`
	CallCount int64   `json:"callCount"`
	AvgMs     float64 `json:"avgMs"`
	P99Ms     float64 `json:"p99Ms"`
	ErrorRate float64 `json:"errorRate"`
}

// ============================================================
// TimePoint — 服务时序数据点（ClickHouse 按需聚合）
// ============================================================

type TimePoint struct {
	Timestamp   time.Time `json:"timestamp"`
	RPS         float64   `json:"rps"`
	SuccessRate float64   `json:"successRate"` // 0-1
	AvgMs       float64   `json:"avgMs"`
	P99Ms       float64   `json:"p99Ms"`
	ErrorCount  int64     `json:"errorCount"`
}

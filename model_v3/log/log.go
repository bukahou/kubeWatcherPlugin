// Package log 定义日志搜索数据模型
//
// 数据源: ClickHouse otel_logs 表
package log

import "time"

// Entry 单条日志记录
type Entry struct {
	Timestamp   time.Time         `json:"timestamp"`
	TraceId     string            `json:"traceId"`
	SpanId      string            `json:"spanId"`
	Severity    string            `json:"severity"`
	SeverityNum int32             `json:"severityNum"`
	ServiceName string            `json:"serviceName"`
	Body        string            `json:"body"`
	ScopeName   string            `json:"scopeName"`
	Attributes  map[string]string `json:"attributes"`
	Resource    map[string]string `json:"resource"`
}

func (l *Entry) HasTrace() bool { return l.TraceId != "" }

// Facet 分面统计项
type Facet struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}

// Facets 所有分面
type Facets struct {
	Services   []Facet `json:"services"`
	Severities []Facet `json:"severities"`
	Scopes     []Facet `json:"scopes"`
}

// QueryResult 日志搜索结果
type QueryResult struct {
	Logs   []Entry `json:"logs"`
	Total  int64   `json:"total"`
	Facets Facets  `json:"facets"`
}

// HistogramBucket 单个直方图桶（服务端预聚合）
type HistogramBucket struct {
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"`
	Count     int64     `json:"count"`
}

// HistogramResult 直方图查询结果
type HistogramResult struct {
	Buckets    []HistogramBucket `json:"buckets"`
	IntervalMs int64             `json:"intervalMs"`
}

// Package repository 定义数据访问接口
//
// 所有 Repository 接口统一在此文件定义，实现按职责分布在子包中:
//   - k8s/   K8s 资源仓库 (通过 K8s API Server 采集)
//   - ch/    ClickHouse 仓库 (查询 OTel 时序数据)
//
// 上层 (Service) 只依赖本包接口，不感知具体实现。
package repository

import (
	"context"
	"time"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/model_v3/apm"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/log"
	"AtlHyper/model_v3/metrics"
	"AtlHyper/model_v3/slo"
)

// =============================================================================
// K8s 资源仓库 — 实现: repository/k8s/
// =============================================================================

// PodRepository Pod 数据访问接口
type PodRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Pod, error)
	Get(ctx context.Context, namespace, name string) (*cluster.Pod, error)
	GetLogs(ctx context.Context, namespace, name string, opts model.LogOptions) (string, error)
}

// NodeRepository Node 数据访问接口
type NodeRepository interface {
	List(ctx context.Context, opts model.ListOptions) ([]cluster.Node, error)
	Get(ctx context.Context, name string) (*cluster.Node, error)
}

// DeploymentRepository Deployment 数据访问接口
type DeploymentRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Deployment, error)
	Get(ctx context.Context, namespace, name string) (*cluster.Deployment, error)
}

// StatefulSetRepository StatefulSet 数据访问接口
type StatefulSetRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.StatefulSet, error)
	Get(ctx context.Context, namespace, name string) (*cluster.StatefulSet, error)
}

// DaemonSetRepository DaemonSet 数据访问接口
type DaemonSetRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.DaemonSet, error)
	Get(ctx context.Context, namespace, name string) (*cluster.DaemonSet, error)
}

// ReplicaSetRepository ReplicaSet 数据访问接口
type ReplicaSetRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.ReplicaSet, error)
}

// ServiceRepository Service 数据访问接口
type ServiceRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Service, error)
	Get(ctx context.Context, namespace, name string) (*cluster.Service, error)
}

// IngressRepository Ingress 数据访问接口
type IngressRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Ingress, error)
	Get(ctx context.Context, namespace, name string) (*cluster.Ingress, error)
}

// ConfigMapRepository ConfigMap 数据访问接口
type ConfigMapRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.ConfigMap, error)
	Get(ctx context.Context, namespace, name string) (*cluster.ConfigMap, error)
}

// SecretRepository Secret 数据访问接口
type SecretRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Secret, error)
	Get(ctx context.Context, namespace, name string) (*cluster.Secret, error)
}

// NamespaceRepository Namespace 数据访问接口
type NamespaceRepository interface {
	List(ctx context.Context, opts model.ListOptions) ([]cluster.Namespace, error)
	Get(ctx context.Context, name string) (*cluster.Namespace, error)
}

// EventRepository Event 数据访问接口
type EventRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Event, error)
}

// JobRepository Job 数据访问接口
type JobRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Job, error)
	Get(ctx context.Context, namespace, name string) (*cluster.Job, error)
}

// CronJobRepository CronJob 数据访问接口
type CronJobRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.CronJob, error)
	Get(ctx context.Context, namespace, name string) (*cluster.CronJob, error)
}

// PersistentVolumeRepository PV 数据访问接口
type PersistentVolumeRepository interface {
	List(ctx context.Context, opts model.ListOptions) ([]cluster.PersistentVolume, error)
}

// PersistentVolumeClaimRepository PVC 数据访问接口
type PersistentVolumeClaimRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.PersistentVolumeClaim, error)
}

// ResourceQuotaRepository ResourceQuota 数据访问接口
type ResourceQuotaRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.ResourceQuota, error)
}

// LimitRangeRepository LimitRange 数据访问接口
type LimitRangeRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.LimitRange, error)
}

// NetworkPolicyRepository NetworkPolicy 数据访问接口
type NetworkPolicyRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.NetworkPolicy, error)
}

// ServiceAccountRepository ServiceAccount 数据访问接口
type ServiceAccountRepository interface {
	List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.ServiceAccount, error)
}

// GenericRepository 通用操作接口 (写操作 + 动态查询)
type GenericRepository interface {
	DeletePod(ctx context.Context, namespace, name string, opts model.DeleteOptions) error
	Delete(ctx context.Context, kind, namespace, name string, opts model.DeleteOptions) error
	ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) error
	RestartDeployment(ctx context.Context, namespace, name string) error
	UpdateDeploymentImage(ctx context.Context, namespace, name, container, image string) error
	CordonNode(ctx context.Context, name string) error
	UncordonNode(ctx context.Context, name string) error
	GetConfigMapData(ctx context.Context, namespace, name string) (map[string]string, error)
	GetSecretData(ctx context.Context, namespace, name string) (map[string]string, error)
	Execute(ctx context.Context, req *model.DynamicRequest) (*model.DynamicResponse, error)
}

// =============================================================================
// ClickHouse 仓库 — 实现: repository/ch/
// =============================================================================

// OTelSummaryRepository OTel 概览仓库（定期聚合，随快照上报）
type OTelSummaryRepository interface {
	// APM 概览
	GetAPMSummary(ctx context.Context) (totalServices, healthyServices int, totalRPS, avgSuccessRate, avgP99Ms float64, err error)
	// SLO 概览
	GetSLOSummary(ctx context.Context) (ingressServices int, ingressAvgRPS float64, meshServices int, meshAvgMTLS float64, err error)
	// 基础设施指标概览
	GetMetricsSummary(ctx context.Context) (monitoredNodes int, avgCPUPct, avgMemPct, maxCPUPct, maxMemPct float64, err error)
}

// TraceQueryRepository Trace 查询仓库（按需查询）
//
// startTime/endTime: 绝对时间（RFC3339），优先于 since 相对时间。
// 空字符串表示不使用绝对时间，回退到 since 相对时间。
type TraceQueryRepository interface {
	ListTraces(ctx context.Context, service, operation string, minDurationMs float64, limit int, since time.Duration, sort string, startTime, endTime string, statusCode, method string) ([]apm.TraceSummary, error)
	GetTraceDetail(ctx context.Context, traceID string) (*apm.TraceDetail, error)
	ListServices(ctx context.Context, since time.Duration, startTime, endTime string) ([]apm.APMService, error)
	GetTopology(ctx context.Context, since time.Duration, startTime, endTime string) (*apm.Topology, error)
	ListOperations(ctx context.Context, since time.Duration, startTime, endTime string) ([]apm.OperationStats, error)
	GetHTTPStats(ctx context.Context, service string, since time.Duration, startTime, endTime string) ([]apm.HTTPStats, error)
	GetDBStats(ctx context.Context, service string, since time.Duration, startTime, endTime string) ([]apm.DBOperationStats, error)
	GetServiceTimeSeries(ctx context.Context, service string, since time.Duration) ([]apm.TimePoint, error)
}

// LogQueryOptions 日志查询选项
type LogQueryOptions struct {
	Query     string        // Body 全文搜索
	Service   string        // ServiceName 过滤
	Level     string        // SeverityText 过滤
	Scope     string        // ScopeName 过滤
	TraceId   string        // TraceId 精确匹配（跨信号关联）
	SpanId    string        // SpanId 精确匹配（跨信号关联）
	Limit     int           // 每页条数
	Offset    int           // 分页偏移
	Since     time.Duration // 时间范围（相对）
	StartTime string        // 绝对开始时间（RFC3339，brush 选区）
	EndTime   string        // 绝对结束时间（RFC3339，brush 选区）
}

// LogQueryRepository Log 查询仓库（按需查询）
type LogQueryRepository interface {
	QueryLogs(ctx context.Context, opts LogQueryOptions) (*log.QueryResult, error)
	QueryHistogram(ctx context.Context, opts LogQueryOptions) (*log.HistogramResult, error)
	GetSummary(ctx context.Context) (*log.Summary, error)
	ListRecentEntries(ctx context.Context, limit int) ([]log.Entry, error)
}

// MetricsQueryRepository Metrics 查询仓库（按需查询）
type MetricsQueryRepository interface {
	ListAllNodeMetrics(ctx context.Context) ([]metrics.NodeMetrics, error)
	GetNodeMetrics(ctx context.Context, nodeName string) (*metrics.NodeMetrics, error)
	GetNodeMetricsSeries(ctx context.Context, nodeName string, metric string, since time.Duration) ([]metrics.Point, error)
	GetMetricsSummary(ctx context.Context) (*metrics.Summary, error)
	// GetNodeMetricsHistory 获取节点历史时序（按指标分组: cpu/memory/disk/temp）
	// 返回格式与 NodeMetricsHistoryResponse.Data 一致
	GetNodeMetricsHistory(ctx context.Context, nodeName string, since time.Duration) (map[string][]metrics.Point, error)
}

// =============================================================================
// Dashboard 信号域子接口 — 按 Metrics/APM/SLO/Logs 拆分
// =============================================================================

// MetricsDashboardRepository Metrics Dashboard 数据采集
type MetricsDashboardRepository interface {
	GetMetricsSummary(ctx context.Context) (*metrics.Summary, error)
	ListAllNodeMetrics(ctx context.Context) ([]metrics.NodeMetrics, error)
}

// APMDashboardRepository APM Dashboard 数据采集
type APMDashboardRepository interface {
	ListAPMServices(ctx context.Context) ([]apm.APMService, error)
	GetAPMTopology(ctx context.Context) (*apm.Topology, error)
	ListAPMOperations(ctx context.Context) ([]apm.OperationStats, error)
	ListRecentTraces(ctx context.Context, limit int) ([]apm.TraceSummary, error)
}

// SLODashboardRepository SLO Dashboard 数据采集
type SLODashboardRepository interface {
	GetSLOSummary(ctx context.Context) (*slo.SLOSummary, error)
	ListIngressSLO(ctx context.Context, since time.Duration) ([]slo.IngressSLO, error)
	ListIngressSLOPrevious(ctx context.Context, since time.Duration) ([]slo.IngressSLO, error)
	GetIngressSLOHistory(ctx context.Context, since, bucket time.Duration) ([]slo.SLOHistoryPoint, error)
	// 注: 服务网格 SLO 已移除 —— SLO 只做 ingress 外部视角,
	// 服务间调用质量由 APM 承担。见 docs/design/active/slo-ingress-contract-design.md
}

// LogsDashboardRepository Logs Dashboard 数据采集
type LogsDashboardRepository interface {
	GetLogsSummary(ctx context.Context) (*log.Summary, error)
	ListRecentLogs(ctx context.Context, limit int) ([]log.Entry, error)
}

// OTelDashboardRepository Dashboard 数据采集（组合接口，兼容现有代码）
//
// 组合委托 MetricsQueryRepository、TraceQueryRepository、SLOQueryRepository，
// 不写新 SQL，仅复用已有查询方法。
type OTelDashboardRepository interface {
	MetricsDashboardRepository
	APMDashboardRepository
	SLODashboardRepository
	LogsDashboardRepository
}

// SLOQueryRepository SLO 查询仓库（按需查询）
type SLOQueryRepository interface {
	ListIngressSLO(ctx context.Context, since time.Duration) ([]slo.IngressSLO, error)
	ListIngressSLOPrevious(ctx context.Context, since time.Duration) ([]slo.IngressSLO, error)
	GetIngressSLOHistory(ctx context.Context, since, bucket time.Duration) ([]slo.SLOHistoryPoint, error)
	// 注: 服务网格 SLO 已移除 —— SLO 只做 ingress 外部视角,
	// 服务间调用质量由 APM 承担。见 docs/design/active/slo-ingress-contract-design.md
	GetSLOTimeSeries(ctx context.Context, name string, since time.Duration) (*slo.TimeSeries, error)
	GetSLOSummary(ctx context.Context) (*slo.SLOSummary, error)
}

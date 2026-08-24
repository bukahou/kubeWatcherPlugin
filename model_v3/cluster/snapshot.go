package cluster

import (
	"time"

	"AtlHyper/model_v3/apm"
	"AtlHyper/model_v3/log"
	"AtlHyper/model_v3/metrics"
	"AtlHyper/model_v3/slo"
)

// ClusterSnapshot Agent 采集的完整集群状态
//
// NodeMetrics 和 SLO 不再随快照上报，改由 Master 从 ClickHouse 查询。
type ClusterSnapshot struct {
	ClusterID string    `json:"clusterId"`
	FetchedAt time.Time `json:"fetchedAt"`

	// 工作负载
	Pods         []Pod         `json:"pods"`
	Deployments  []Deployment  `json:"deployments"`
	StatefulSets []StatefulSet `json:"statefulSets"`
	DaemonSets   []DaemonSet   `json:"daemonSets"`
	ReplicaSets  []ReplicaSet  `json:"replicaSets,omitempty"`
	Jobs         []Job         `json:"jobs,omitempty"`
	CronJobs     []CronJob     `json:"cronJobs,omitempty"`

	// 网络
	Services  []Service `json:"services"`
	Ingresses []Ingress `json:"ingresses"`

	// 配置
	Namespaces []Namespace `json:"namespaces"`
	ConfigMaps []ConfigMap `json:"configMaps"`
	Secrets    []Secret    `json:"secrets,omitempty"`

	// 策略与配额
	ResourceQuotas  []ResourceQuota  `json:"resourceQuotas,omitempty"`
	LimitRanges     []LimitRange     `json:"limitRanges,omitempty"`
	NetworkPolicies []NetworkPolicy  `json:"networkPolicies,omitempty"`
	ServiceAccounts []ServiceAccount `json:"serviceAccounts,omitempty"`

	// 存储
	PersistentVolumes      []PersistentVolume      `json:"pvs,omitempty"`
	PersistentVolumeClaims []PersistentVolumeClaim `json:"pvcs,omitempty"`

	// 集群
	Nodes  []Node  `json:"nodes"`
	Events []Event `json:"events"`

	// OTel 可观测性快照（Agent 从 ClickHouse 定期聚合，随快照上报）
	OTel *OTelSnapshot `json:"otel,omitempty"`

	// 摘要
	Summary ClusterSummary `json:"summary"`
}

// ClusterSummary 集群摘要统计
type ClusterSummary struct {
	TotalNodes         int `json:"totalNodes"`
	ReadyNodes         int `json:"readyNodes"`
	TotalPods          int `json:"totalPods"`
	RunningPods        int `json:"runningPods"`
	PendingPods        int `json:"pendingPods"`
	FailedPods         int `json:"failedPods"`
	TotalDeployments   int `json:"totalDeployments"`
	HealthyDeployments int `json:"healthyDeployments"`
	TotalStatefulSets  int `json:"totalStatefulSets"`
	TotalDaemonSets    int `json:"totalDaemonSets"`
	TotalServices      int `json:"totalServices"`
	TotalIngresses     int `json:"totalIngresses"`
	TotalNamespaces    int `json:"totalNamespaces"`
	TotalEvents        int `json:"totalEvents"`
	WarningEvents      int `json:"warningEvents"`
}

func (s *ClusterSnapshot) GenerateSummary() ClusterSummary {
	sum := ClusterSummary{
		TotalNodes: len(s.Nodes), TotalPods: len(s.Pods),
		TotalDeployments: len(s.Deployments), TotalStatefulSets: len(s.StatefulSets),
		TotalDaemonSets: len(s.DaemonSets), TotalServices: len(s.Services),
		TotalIngresses: len(s.Ingresses), TotalNamespaces: len(s.Namespaces),
		TotalEvents: len(s.Events),
	}
	for _, n := range s.Nodes {
		if n.IsReady() {
			sum.ReadyNodes++
		}
	}
	for _, p := range s.Pods {
		switch p.Status.Phase {
		case "Running":
			sum.RunningPods++
		case "Pending":
			sum.PendingPods++
		case "Failed":
			sum.FailedPods++
		}
	}
	for _, d := range s.Deployments {
		if d.IsHealthy() {
			sum.HealthyDeployments++
		}
	}
	for _, e := range s.Events {
		if e.IsWarning() {
			sum.WarningEvents++
		}
	}
	return sum
}

func (s *ClusterSnapshot) GetNodeReadyPercent() float64 {
	if s.Summary.TotalNodes == 0 {
		return 0
	}
	return float64(s.Summary.ReadyNodes) / float64(s.Summary.TotalNodes) * 100
}

func (s *ClusterSnapshot) GetPodRunningPercent() float64 {
	if s.Summary.TotalPods == 0 {
		return 0
	}
	return float64(s.Summary.RunningPods) / float64(s.Summary.TotalPods) * 100
}

// OTelEntry 带时间戳的 OTel 快照（时间线中的一条记录）
type OTelEntry struct {
	Snapshot  *OTelSnapshot `json:"snapshot"`
	Timestamp time.Time     `json:"timestamp"`
}

// OTelSnapshot OTel 可观测性快照（Agent 从 ClickHouse 定期聚合）
//
// 包含两类数据：
//   - 标量摘要（15 个字段）：用于概览卡片展示
//   - Dashboard 列表（8 个字段）：用于 Dashboard 页面直读，避免 Command 机制的延迟
//
// 随快照一起上报给 Master，Master 内存直读即可返回前端。
// 详细数据（Trace Detail、Log Query 等）仍通过 Command 机制按需查询。
type OTelSnapshot struct {
	// ===== 标量摘要 =====

	// APM 服务概览
	TotalServices   int     `json:"totalServices"`
	HealthyServices int     `json:"healthyServices"`
	TotalRPS        float64 `json:"totalRps"`
	AvgSuccessRate  float64 `json:"avgSuccessRate"`
	AvgP99Ms        float64 `json:"avgP99Ms"`

	// SLO 概览
	IngressServices int     `json:"ingressServices"`
	IngressAvgRPS   float64 `json:"ingressAvgRps"`

	// 基础设施指标概览
	MonitoredNodes int     `json:"monitoredNodes"`
	AvgCPUPct      float64 `json:"avgCpuPct"`
	AvgMemPct      float64 `json:"avgMemPct"`
	MaxCPUPct      float64 `json:"maxCpuPct"`
	MaxMemPct      float64 `json:"maxMemPct"`

	// ===== Dashboard 列表数据 =====

	// Metrics Dashboard
	MetricsSummary *metrics.Summary      `json:"metricsSummary,omitempty"`
	MetricsNodes   []metrics.NodeMetrics `json:"metricsNodes,omitempty"`

	// APM Dashboard
	APMServices []apm.APMService `json:"apmServices,omitempty"`
	APMTopology *apm.Topology    `json:"apmTopology,omitempty"`

	// SLO Dashboard
	SLOSummary *slo.SLOSummary  `json:"sloSummary,omitempty"`
	SLOIngress []slo.IngressSLO `json:"sloIngress,omitempty"`

	// ===== 扩展数据（内存时间线 + Dashboard 首屏） =====

	// APM 操作级聚合统计（GROUP BY ServiceName, SpanName，15 分钟窗口）
	APMOperations []apm.OperationStats `json:"apmOperations,omitempty"`
	// 最近 Traces（无过滤，Dashboard 首屏用）
	RecentTraces []apm.TraceSummary `json:"recentTraces,omitempty"`
	// 最近日志条目（5 分钟窗口，最多 500 条）
	RecentLogs []log.Entry `json:"recentLogs,omitempty"`
	// 日志统计摘要（5 分钟窗口）
	LogsSummary *log.Summary `json:"logsSummary,omitempty"`

	// ===== 多窗口 SLO 数据（1d/7d/30d 预聚合） =====

	// SLOWindows 按时间窗口预聚合的 Ingress SLO 数据
	// key: "1d", "7d", "30d"
	SLOWindows map[string]*slo.SLOWindowData `json:"sloWindows,omitempty"`

	// ===== 预聚合时序（Agent Concentrator 生成，1min 粒度 × 60 点 = 1h） =====

	// 节点指标时序（每个节点最近 1h）
	NodeMetricsSeries []NodeMetricsTimeSeries `json:"nodeMetricsSeries,omitempty"`
	// SLO 服务时序（每个服务最近 1h）
	SLOTimeSeries []SLOServiceTimeSeries `json:"sloTimeSeries,omitempty"`
	// APM 服务时序（每个服务最近 1h）
	APMTimeSeries []APMServiceTimeSeries `json:"apmTimeSeries,omitempty"`

	// Freshness 各信号最近一条数据的时间。
	// 用来在页面上区分「没有流量」和「采集挂了」—— 两者在页面上都是空白，
	// 但一个不用管，一个要救火。nil 表示上报方（旧版 Agent）不采集。
	Freshness *SignalFreshness `json:"freshness,omitempty"`
}

// SignalFreshness 各信号最近一条数据的时间戳。
//
// 之所以三个信号都要：metrics 是 Collector 主动拉取的，只要节点活着就一定有数据；
// traces / logs 由请求触发，安静一段时间很正常。两者的非对称正好能区分
// 「没人访问」和「采集链路断了」—— 单看某一个信号是分不出来的。
type SignalFreshness struct {
	MetricsAt time.Time `json:"metricsAt"`
	TracesAt  time.Time `json:"tracesAt"`
	LogsAt    time.Time `json:"logsAt"`
}

// NodeMetricsTimeSeries 单节点预聚合时序
type NodeMetricsTimeSeries struct {
	NodeName string             `json:"nodeName"`
	Points   []NodeMetricsPoint `json:"points"`
}

// NodeMetricsPoint 节点指标时序数据点（1 分钟粒度，25 字段）
type NodeMetricsPoint struct {
	Timestamp time.Time `json:"timestamp"`

	// CPU（7 字段）
	CPUPct    float64 `json:"cpuPct"`
	UserPct   float64 `json:"userPct"`
	SystemPct float64 `json:"systemPct"`
	IOWaitPct float64 `json:"iowaitPct"`
	Load1     float64 `json:"load1"`
	Load5     float64 `json:"load5"`
	Load15    float64 `json:"load15"`

	// Memory（2 字段）
	MemPct       float64 `json:"memPct"`
	SwapUsagePct float64 `json:"swapUsagePct"`

	// Disk — 主磁盘（4 字段）
	DiskPct       float64 `json:"diskPct"`
	DiskReadBps   float64 `json:"diskReadBps"`
	DiskWriteBps  float64 `json:"diskWriteBps"`
	DiskIOUtilPct float64 `json:"diskIOUtilPct"`

	// Network — 主网卡（4 字段）
	NetRxBps    float64 `json:"netRxBps"`
	NetTxBps    float64 `json:"netTxBps"`
	NetRxPktSec float64 `json:"netRxPktSec"`
	NetTxPktSec float64 `json:"netTxPktSec"`

	// Temperature（1 字段）
	CPUTempC float64 `json:"cpuTempC"`

	// PSI（3 字段）
	CPUSomePct float64 `json:"cpuSomePct"`
	MemSomePct float64 `json:"memSomePct"`
	IOSomePct  float64 `json:"ioSomePct"`

	// TCP（2 字段）
	TCPEstab    int64 `json:"tcpEstab"`
	SocketsUsed int64 `json:"socketsUsed"`
}

// SLOServiceTimeSeries 单服务预聚合时序
type SLOServiceTimeSeries struct {
	ServiceName string         `json:"serviceName"`
	Points      []SLOTimePoint `json:"points"`
}

// SLOTimePoint SLO 时序数据点（1 分钟粒度，6 字段）
type SLOTimePoint struct {
	Timestamp   time.Time `json:"timestamp"`
	RPS         float64   `json:"rps"`
	SuccessRate float64   `json:"successRate"`
	P50Ms       float64   `json:"p50Ms"`
	P99Ms       float64   `json:"p99Ms"`
	ErrorRate   float64   `json:"errorRate"`
}

// APMServiceTimeSeries 单服务 APM 预聚合时序
type APMServiceTimeSeries struct {
	ServiceName string         `json:"serviceName"`
	Namespace   string         `json:"namespace"`
	Points      []APMTimePoint `json:"points"`
}

// APMTimePoint APM 时序数据点（1 分钟粒度）
type APMTimePoint struct {
	Timestamp   time.Time `json:"timestamp"`
	RPS         float64   `json:"rps"`
	SuccessRate float64   `json:"successRate"`
	AvgMs       float64   `json:"avgMs"`
	P99Ms       float64   `json:"p99Ms"`
	ErrorCount  int64     `json:"errorCount"`
}

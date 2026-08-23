// atlhyper_master_v2/model/overview.go
// 集群概览 Web API 响应类型（camelCase JSON tag）
package model

import "time"

// ==================== 顶层响应 ====================

// ClusterOverview 集群概览响应
type ClusterOverview struct {
	ClusterID string            `json:"clusterId"`
	Cards     OverviewCards     `json:"cards"`
	Workloads OverviewWorkloads `json:"workloads"`
	Alerts    OverviewAlerts    `json:"alerts"`
	Nodes     OverviewNodes     `json:"nodes"`
}

// ==================== 卡片数据 ====================

// OverviewCards 概览卡片
type OverviewCards struct {
	ClusterHealth OverviewClusterHealth `json:"clusterHealth"`
	NodeReady     OverviewResourceReady `json:"nodeReady"`
	CPUUsage      OverviewPercent       `json:"cpuUsage"`
	MemUsage      OverviewPercent       `json:"memUsage"`
	Events24h     int                   `json:"events24h"`
}

// OverviewClusterHealth 集群健康状态
type OverviewClusterHealth struct {
	Status           string  `json:"status"`
	Reason           string  `json:"reason,omitempty"`
	NodeReadyPercent float64 `json:"nodeReadyPercent"`
	PodReadyPercent  float64 `json:"podReadyPercent"`
}

// OverviewResourceReady 资源就绪
type OverviewResourceReady struct {
	Total   int     `json:"total"`
	Ready   int     `json:"ready"`
	Percent float64 `json:"percent"`
}

// OverviewPercent 百分比
type OverviewPercent struct {
	Percent float64 `json:"percent"`
}

// ==================== 工作负载 ====================

// OverviewWorkloads 工作负载概览
type OverviewWorkloads struct {
	Summary   OverviewWorkloadSummary `json:"summary"`
	PodStatus OverviewPodStatus       `json:"podStatus"`
	PeakStats *OverviewPeakStats      `json:"peakStats,omitempty"`
}

// OverviewWorkloadSummary 工作负载汇总
type OverviewWorkloadSummary struct {
	Deployments  OverviewWorkloadStatus `json:"deployments"`
	DaemonSets   OverviewWorkloadStatus `json:"daemonsets"`
	StatefulSets OverviewWorkloadStatus `json:"statefulsets"`
	Jobs         OverviewJobStatus      `json:"jobs"`
}

// OverviewWorkloadStatus 工作负载状态
type OverviewWorkloadStatus struct {
	Total int `json:"total"`
	Ready int `json:"ready"`
}

// OverviewJobStatus Job 状态
type OverviewJobStatus struct {
	Total     int `json:"total"`
	Running   int `json:"running"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
}

// OverviewPodStatus Pod 状态分布
type OverviewPodStatus struct {
	Total            int     `json:"total"`
	Running          int     `json:"running"`
	Pending          int     `json:"pending"`
	Failed           int     `json:"failed"`
	Succeeded        int     `json:"succeeded"`
	Unknown          int     `json:"unknown"`
	RunningPercent   float64 `json:"runningPercent"`
	PendingPercent   float64 `json:"pendingPercent"`
	FailedPercent    float64 `json:"failedPercent"`
	SucceededPercent float64 `json:"succeededPercent"`
}

// OverviewPeakStats 峰值统计
type OverviewPeakStats struct {
	PeakCPU     float64 `json:"peakCpu"`
	PeakCPUNode string  `json:"peakCpuNode"`
	PeakMem     float64 `json:"peakMem"`
	PeakMemNode string  `json:"peakMemNode"`
	HasData     bool    `json:"hasData"`
}

// ==================== 告警 ====================

// OverviewAlerts 告警数据
type OverviewAlerts struct {
	Trend  []OverviewAlertTrend  `json:"trend"`
	Totals OverviewAlertTotals   `json:"totals"`
	Recent []OverviewRecentAlert `json:"recent"`
}

// OverviewAlertTrend 告警趋势点
type OverviewAlertTrend struct {
	At    time.Time      `json:"at"`
	Kinds map[string]int `json:"kinds"`
}

// OverviewAlertTotals 告警统计
type OverviewAlertTotals struct {
	Critical int `json:"critical"`
	Warning  int `json:"warning"`
	Info     int `json:"info"`
}

// OverviewRecentAlert 最近告警
type OverviewRecentAlert struct {
	Timestamp string `json:"timestamp"`
	Severity  string `json:"severity"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Message   string `json:"message"`
	Reason    string `json:"reason"`
}

// ==================== 节点 ====================

// OverviewNodes 节点数据
type OverviewNodes struct {
	Usage []OverviewNodeUsage `json:"usage"`
}

// OverviewNodeUsage 节点使用率
type OverviewNodeUsage struct {
	Node     string  `json:"node"`
	CPUUsage float64 `json:"cpuUsage"`
	MemUsage float64 `json:"memUsage"`
}

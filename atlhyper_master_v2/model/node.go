// atlhyper_master_v2/model/node.go
// Node Web API 响应类型（camelCase JSON tag，扁平结构，单位已转换）
package model

// NodeItem Node 列表项（扁平，单位已转换）
type NodeItem struct {
	Name         string  `json:"name"`
	Ready        bool    `json:"ready"`
	InternalIP   string  `json:"internalIP"`
	OSImage      string  `json:"osImage"`
	Architecture string  `json:"architecture"`
	CPUCores     float64 `json:"cpuCores"`
	MemoryGiB    float64 `json:"memoryGiB"`
	Schedulable  bool    `json:"schedulable"`
}

// NodeOverviewCards Node 概览统计
type NodeOverviewCards struct {
	TotalNodes  int     `json:"totalNodes"`
	ReadyNodes  int     `json:"readyNodes"`
	TotalCPU    float64 `json:"totalCPU"`
	TotalMemGiB float64 `json:"totalMemoryGiB"`
}

// NodeOverview Node 概览
type NodeOverview struct {
	Cards NodeOverviewCards `json:"cards"`
	Rows  []NodeItem        `json:"rows"`
}

// NodeDetail Node 详情（扁平，单位已转换）
type NodeDetail struct {
	// 基本信息
	Name        string   `json:"name"`
	Roles       []string `json:"roles,omitempty"`
	Ready       bool     `json:"ready"`
	Schedulable bool     `json:"schedulable"`
	Age         string   `json:"age,omitempty"`
	CreatedAt   string   `json:"createdAt"`

	// 地址与系统
	Hostname     string `json:"hostname,omitempty"`
	InternalIP   string `json:"internalIP,omitempty"`
	ExternalIP   string `json:"externalIP,omitempty"`
	OSImage      string `json:"osImage,omitempty"`
	OS           string `json:"os,omitempty"`
	Architecture string `json:"architecture,omitempty"`
	Kernel       string `json:"kernel,omitempty"`
	CRI          string `json:"cri,omitempty"`
	Kubelet      string `json:"kubelet,omitempty"`
	KubeProxy    string `json:"kubeProxy,omitempty"`

	// 资源容量（单位已转换）
	CPUCapacityCores    float64 `json:"cpuCapacityCores,omitempty"`
	CPUAllocatableCores float64 `json:"cpuAllocatableCores,omitempty"`
	MemCapacityGiB      float64 `json:"memCapacityGiB,omitempty"`
	MemAllocatableGiB   float64 `json:"memAllocatableGiB,omitempty"`
	PodsCapacity        int     `json:"podsCapacity,omitempty"`
	PodsAllocatable     int     `json:"podsAllocatable,omitempty"`
	EphemeralStorageGiB float64 `json:"ephemeralStorageGiB,omitempty"`

	// 当前指标
	CPUUsageCores float64 `json:"cpuUsageCores,omitempty"`
	CPUUtilPct    float64 `json:"cpuUtilPct,omitempty"`
	MemUsageGiB   float64 `json:"memUsageGiB,omitempty"`
	MemUtilPct    float64 `json:"memUtilPct,omitempty"`
	PodsUsed      int     `json:"podsUsed,omitempty"`
	PodsUtilPct   float64 `json:"podsUtilPct,omitempty"`

	// 压力状态
	PressureMemory     bool `json:"pressureMemory,omitempty"`
	PressureDisk       bool `json:"pressureDisk,omitempty"`
	PressurePID        bool `json:"pressurePID,omitempty"`
	NetworkUnavailable bool `json:"networkUnavailable,omitempty"`

	// 调度
	PodCIDRs   []string `json:"podCIDRs,omitempty"`
	ProviderID string   `json:"providerID,omitempty"`

	// 条件/污点/标签
	Conditions []NodeConditionResponse `json:"conditions,omitempty"`
	Taints     []NodeTaintResponse     `json:"taints,omitempty"`
	Labels     map[string]string       `json:"labels,omitempty"`

	// 诊断
	Badges  []string `json:"badges,omitempty"`
	Reason  string   `json:"reason,omitempty"`
	Message string   `json:"message,omitempty"`
}

// NodeConditionResponse Node Condition 响应
type NodeConditionResponse struct {
	Type      string `json:"type"`
	Status    string `json:"status"`
	Reason    string `json:"reason,omitempty"`
	Message   string `json:"message,omitempty"`
	Heartbeat string `json:"heartbeat,omitempty"`
	ChangedAt string `json:"changedAt,omitempty"`
}

// NodeTaintResponse Node Taint 响应
type NodeTaintResponse struct {
	Key    string `json:"key"`
	Value  string `json:"value,omitempty"`
	Effect string `json:"effect"`
}

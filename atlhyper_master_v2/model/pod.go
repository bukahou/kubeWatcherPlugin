// atlhyper_master_v2/model/pod.go
// Pod Web API 响应类型（camelCase JSON tag，扁平结构）
package model

// PodItem Pod 列表项（扁平）
type PodItem struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Deployment string `json:"deployment"`
	Ready      string `json:"ready"`
	Phase      string `json:"phase"`
	Restarts   int32  `json:"restarts"`
	CPUText    string `json:"cpuText"`
	MemoryText string `json:"memoryText"`
	StartTime  string `json:"startTime"`
	Node       string `json:"node"`
	Age        string `json:"age,omitempty"`
}

// PodOverviewCards Pod 概览统计
type PodOverviewCards struct {
	Running int `json:"running"`
	Pending int `json:"pending"`
	Failed  int `json:"failed"`
	Unknown int `json:"unknown"`
}

// PodOverview Pod 概览
type PodOverview struct {
	Cards PodOverviewCards `json:"cards"`
	Pods  []PodItem        `json:"pods"`
}

// PodDetail Pod 详情（扁平 + 嵌套容器/卷）
type PodDetail struct {
	// 基本信息
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Controller string `json:"controller,omitempty"`
	Phase      string `json:"phase"`
	Ready      string `json:"ready"`
	Restarts   int32  `json:"restarts"`
	StartTime  string `json:"startTime"`
	Age        string `json:"age,omitempty"`
	Node       string `json:"node"`
	PodIP      string `json:"podIP,omitempty"`
	HostIP     string `json:"hostIP,omitempty"`
	QoSClass   string `json:"qosClass,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Message    string `json:"message,omitempty"`

	// 调度
	RestartPolicy                 string            `json:"restartPolicy,omitempty"`
	PriorityClassName             string            `json:"priorityClassName,omitempty"`
	RuntimeClassName              string            `json:"runtimeClassName,omitempty"`
	TerminationGracePeriodSeconds *int64            `json:"terminationGracePeriodSeconds,omitempty"`
	Tolerations                   interface{}       `json:"tolerations,omitempty"`
	Affinity                      interface{}       `json:"affinity,omitempty"`
	NodeSelector                  map[string]string `json:"nodeSelector,omitempty"`

	// 网络
	HostNetwork        bool     `json:"hostNetwork,omitempty"`
	PodIPs             []string `json:"podIPs,omitempty"`
	DNSPolicy          string   `json:"dnsPolicy,omitempty"`
	ServiceAccountName string   `json:"serviceAccountName,omitempty"`

	// 指标
	CPUUsage string `json:"cpuUsage,omitempty"`
	MemUsage string `json:"memUsage,omitempty"`

	// 容器
	Containers []PodContainerResponse `json:"containers"`

	// 存储卷
	Volumes []PodVolumeResponse `json:"volumes,omitempty"`
}

// PodContainerResponse 容器响应
type PodContainerResponse struct {
	Name                 string            `json:"name"`
	Image                string            `json:"image"`
	ImagePullPolicy      string            `json:"imagePullPolicy,omitempty"`
	Ports                interface{}       `json:"ports,omitempty"`
	Envs                 interface{}       `json:"envs,omitempty"`
	VolumeMounts         interface{}       `json:"volumeMounts,omitempty"`
	Requests             map[string]string `json:"requests,omitempty"`
	Limits               map[string]string `json:"limits,omitempty"`
	ReadinessProbe       interface{}       `json:"readinessProbe,omitempty"`
	LivenessProbe        interface{}       `json:"livenessProbe,omitempty"`
	StartupProbe         interface{}       `json:"startupProbe,omitempty"`
	State                string            `json:"state,omitempty"`
	RestartCount         int32             `json:"restartCount"`
	LastTerminatedReason string            `json:"lastTerminatedReason,omitempty"`
}

// PodVolumeResponse 存储卷响应
type PodVolumeResponse struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	SourceBrief string `json:"sourceBrief,omitempty"`
}

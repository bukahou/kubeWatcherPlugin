// atlhyper_master_v2/model/deployment.go
// Deployment Web API 响应类型（camelCase JSON tag，扁平结构）
package model

// DeploymentItem Deployment 列表项（扁平）
type DeploymentItem struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Image      string `json:"image"`
	Replicas   string `json:"replicas"`
	LabelCount int    `json:"labelCount"`
	AnnoCount  int    `json:"annoCount"`
	CreatedAt  string `json:"createdAt"`
}

// DeploymentOverviewCards Deployment 概览统计
type DeploymentOverviewCards struct {
	TotalDeployments int   `json:"totalDeployments"`
	Namespaces       int   `json:"namespaces"`
	TotalReplicas    int32 `json:"totalReplicas"`
	ReadyReplicas    int32 `json:"readyReplicas"`
}

// DeploymentOverview Deployment 概览
type DeploymentOverview struct {
	Cards DeploymentOverviewCards `json:"cards"`
	Rows  []DeploymentItem        `json:"rows"`
}

// DeploymentDetail Deployment 详情（扁平顶层 + 嵌套子结构）
type DeploymentDetail struct {
	// 基本信息（扁平）
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Strategy    string `json:"strategy"`
	Replicas    int32  `json:"replicas"`
	Updated     int32  `json:"updated"`
	Ready       int32  `json:"ready"`
	Available   int32  `json:"available"`
	Unavailable int32  `json:"unavailable,omitempty"`
	Paused      bool   `json:"paused,omitempty"`
	Selector    string `json:"selector,omitempty"`
	CreatedAt   string `json:"createdAt"`
	Age         string `json:"age,omitempty"`

	// 嵌套结构（使用 interface{} 保持前端灵活性）
	Spec        interface{} `json:"spec,omitempty"`
	Template    interface{} `json:"template,omitempty"`
	Status      interface{} `json:"status,omitempty"`
	Conditions  interface{} `json:"conditions,omitempty"`
	Rollout     interface{} `json:"rollout,omitempty"`
	ReplicaSets interface{} `json:"replicaSets,omitempty"`

	// 元数据
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// atlhyper_master_v2/model/service.go
// Service Web API 响应类型（camelCase JSON tag，扁平结构）
package model

// ServiceItem Service 列表项（扁平）
type ServiceItem struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Type      string `json:"type"`
	ClusterIP string `json:"clusterIP"`
	Ports     string `json:"ports"` // 格式化字符串，如 "80:30080/TCP→8080"
	Protocol  string `json:"protocol"`
	Selector  string `json:"selector"` // 格式化字符串，如 "app=nginx,tier=frontend"
	CreatedAt string `json:"createdAt"`
}

// ServiceOverviewCards Service 概览统计
type ServiceOverviewCards struct {
	TotalServices    int `json:"totalServices"`
	ExternalServices int `json:"externalServices"`
	InternalServices int `json:"internalServices"`
	HeadlessServices int `json:"headlessServices"`
}

// ServiceOverview Service 概览
type ServiceOverview struct {
	Cards ServiceOverviewCards `json:"cards"`
	Rows  []ServiceItem        `json:"rows"`
}

// ServiceDetail Service 详情（扁平 + 嵌套端口/后端）
type ServiceDetail struct {
	// 基本信息
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Type      string `json:"type"`
	CreatedAt string `json:"createdAt"`
	Age       string `json:"age,omitempty"`

	// 选择器
	Selector map[string]string `json:"selector,omitempty"`

	// 端口
	Ports []ServicePortResponse `json:"ports,omitempty"`

	// 网络
	ClusterIPs          []string `json:"clusterIPs,omitempty"`
	ExternalIPs         []string `json:"externalIPs,omitempty"`
	LoadBalancerIngress []string `json:"loadBalancerIngress,omitempty"`

	// 会话与流量策略
	SessionAffinity       string `json:"sessionAffinity,omitempty"`
	ExternalTrafficPolicy string `json:"externalTrafficPolicy,omitempty"`
	InternalTrafficPolicy string `json:"internalTrafficPolicy,omitempty"`

	// IP 族
	IPFamilies     []string `json:"ipFamilies,omitempty"`
	IPFamilyPolicy string   `json:"ipFamilyPolicy,omitempty"`

	// 后端端点
	Backends interface{} `json:"backends,omitempty"`

	// 诊断
	Badges []string `json:"badges,omitempty"`
}

// ServicePortResponse Service 端口响应
type ServicePortResponse struct {
	Name        string `json:"name,omitempty"`
	Protocol    string `json:"protocol"`
	Port        int32  `json:"port"`
	TargetPort  string `json:"targetPort"`
	NodePort    int32  `json:"nodePort,omitempty"`
	AppProtocol string `json:"appProtocol,omitempty"`
}

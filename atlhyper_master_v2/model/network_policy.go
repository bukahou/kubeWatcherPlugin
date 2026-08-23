// atlhyper_master_v2/model/network_policy.go
// NetworkPolicy Web API 响应类型（camelCase JSON tag，扁平结构）
package model

// NetworkPolicyItem NetworkPolicy 列表项
type NetworkPolicyItem struct {
	Name             string   `json:"name"`
	Namespace        string   `json:"namespace"`
	PodSelector      string   `json:"podSelector"`
	PolicyTypes      []string `json:"policyTypes"`
	IngressRuleCount int      `json:"ingressRuleCount"`
	EgressRuleCount  int      `json:"egressRuleCount"`
	CreatedAt        string   `json:"createdAt"`
	Age              string   `json:"age"`
}

// NetworkPolicyDetail NetworkPolicy 详情
type NetworkPolicyDetail struct {
	Name             string      `json:"name"`
	Namespace        string      `json:"namespace"`
	PodSelector      string      `json:"podSelector"`
	PolicyTypes      []string    `json:"policyTypes"`
	IngressRuleCount int         `json:"ingressRuleCount"`
	EgressRuleCount  int         `json:"egressRuleCount"`
	IngressRules     interface{} `json:"ingressRules,omitempty"`
	EgressRules      interface{} `json:"egressRules,omitempty"`
	CreatedAt        string      `json:"createdAt"`
	Age              string      `json:"age"`

	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

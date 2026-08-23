package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/model_v3/cluster"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// =============================================================================
// ResourceQuota 仓库
// =============================================================================

type resourceQuotaRepository struct {
	client sdk.K8sClient
}

// NewResourceQuotaRepository 创建 ResourceQuota 仓库
func NewResourceQuotaRepository(client sdk.K8sClient) repository.ResourceQuotaRepository {
	return &resourceQuotaRepository{client: client}
}

func (r *resourceQuotaRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.ResourceQuota, error) {
	k8sQuotas, err := r.client.ListResourceQuotas(ctx, namespace, sdk.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	})
	if err != nil {
		return nil, err
	}

	quotas := make([]cluster.ResourceQuota, 0, len(k8sQuotas))
	for i := range k8sQuotas {
		quotas = append(quotas, ConvertResourceQuota(&k8sQuotas[i]))
	}
	return quotas, nil
}

// =============================================================================
// LimitRange 仓库
// =============================================================================

type limitRangeRepository struct {
	client sdk.K8sClient
}

// NewLimitRangeRepository 创建 LimitRange 仓库
func NewLimitRangeRepository(client sdk.K8sClient) repository.LimitRangeRepository {
	return &limitRangeRepository{client: client}
}

func (r *limitRangeRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.LimitRange, error) {
	k8sLRs, err := r.client.ListLimitRanges(ctx, namespace, sdk.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	})
	if err != nil {
		return nil, err
	}

	lrs := make([]cluster.LimitRange, 0, len(k8sLRs))
	for i := range k8sLRs {
		lrs = append(lrs, ConvertLimitRange(&k8sLRs[i]))
	}
	return lrs, nil
}

// =============================================================================
// NetworkPolicy 仓库
// =============================================================================

type networkPolicyRepository struct {
	client sdk.K8sClient
}

// NewNetworkPolicyRepository 创建 NetworkPolicy 仓库
func NewNetworkPolicyRepository(client sdk.K8sClient) repository.NetworkPolicyRepository {
	return &networkPolicyRepository{client: client}
}

func (r *networkPolicyRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.NetworkPolicy, error) {
	k8sNPs, err := r.client.ListNetworkPolicies(ctx, namespace, sdk.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	})
	if err != nil {
		return nil, err
	}

	nps := make([]cluster.NetworkPolicy, 0, len(k8sNPs))
	for i := range k8sNPs {
		nps = append(nps, ConvertNetworkPolicy(&k8sNPs[i]))
	}
	return nps, nil
}

// =============================================================================
// ServiceAccount 仓库
// =============================================================================

type serviceAccountRepository struct {
	client sdk.K8sClient
}

// NewServiceAccountRepository 创建 ServiceAccount 仓库
func NewServiceAccountRepository(client sdk.K8sClient) repository.ServiceAccountRepository {
	return &serviceAccountRepository{client: client}
}

func (r *serviceAccountRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.ServiceAccount, error) {
	k8sSAs, err := r.client.ListServiceAccounts(ctx, namespace, sdk.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	})
	if err != nil {
		return nil, err
	}

	sas := make([]cluster.ServiceAccount, 0, len(k8sSAs))
	for i := range k8sSAs {
		sas = append(sas, ConvertServiceAccount(&k8sSAs[i]))
	}
	return sas, nil
}

// =============================================================================
// 辅助函数
// =============================================================================

// formatAge 格式化时间为易读的 Age 字符串
func formatAge(t time.Time) string {
	d := time.Since(t)
	days := int(d.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	hours := int(d.Hours())
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	minutes := int(d.Minutes())
	if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return "0m"
}

// =============================================================================
// 转换函数
// =============================================================================

// ConvertResourceQuota 转换 K8s ResourceQuota 到 model_v3
func ConvertResourceQuota(k8sRQ *corev1.ResourceQuota) cluster.ResourceQuota {
	rq := cluster.ResourceQuota{
		Name:        k8sRQ.Name,
		Namespace:   k8sRQ.Namespace,
		CreatedAt:   k8sRQ.CreationTimestamp.Format(time.RFC3339),
		Age:         formatAge(k8sRQ.CreationTimestamp.Time),
		Labels:      k8sRQ.Labels,
		Annotations: k8sRQ.Annotations,
	}

	// 转换 Scopes
	for _, scope := range k8sRQ.Spec.Scopes {
		rq.Scopes = append(rq.Scopes, string(scope))
	}

	// 转换 Hard
	if len(k8sRQ.Spec.Hard) > 0 {
		rq.Hard = make(map[string]string)
		for k, v := range k8sRQ.Spec.Hard {
			rq.Hard[string(k)] = v.String()
		}
	}

	// 转换 Used
	if len(k8sRQ.Status.Used) > 0 {
		rq.Used = make(map[string]string)
		for k, v := range k8sRQ.Status.Used {
			rq.Used[string(k)] = v.String()
		}
	}

	return rq
}

// ConvertLimitRange 转换 K8s LimitRange 到 model_v3
func ConvertLimitRange(k8sLR *corev1.LimitRange) cluster.LimitRange {
	lr := cluster.LimitRange{
		Name:        k8sLR.Name,
		Namespace:   k8sLR.Namespace,
		CreatedAt:   k8sLR.CreationTimestamp.Format(time.RFC3339),
		Age:         formatAge(k8sLR.CreationTimestamp.Time),
		Labels:      k8sLR.Labels,
		Annotations: k8sLR.Annotations,
	}

	// 转换 Items
	for _, item := range k8sLR.Spec.Limits {
		lrItem := cluster.LimitRangeItem{
			Type: string(item.Type),
		}

		// Default
		if len(item.Default) > 0 {
			lrItem.Default = make(map[string]string)
			for k, v := range item.Default {
				lrItem.Default[string(k)] = v.String()
			}
		}

		// DefaultRequest
		if len(item.DefaultRequest) > 0 {
			lrItem.DefaultRequest = make(map[string]string)
			for k, v := range item.DefaultRequest {
				lrItem.DefaultRequest[string(k)] = v.String()
			}
		}

		// Max
		if len(item.Max) > 0 {
			lrItem.Max = make(map[string]string)
			for k, v := range item.Max {
				lrItem.Max[string(k)] = v.String()
			}
		}

		// Min
		if len(item.Min) > 0 {
			lrItem.Min = make(map[string]string)
			for k, v := range item.Min {
				lrItem.Min[string(k)] = v.String()
			}
		}

		// MaxLimitRequestRatio
		if len(item.MaxLimitRequestRatio) > 0 {
			lrItem.MaxLimitRequestRatio = make(map[string]string)
			for k, v := range item.MaxLimitRequestRatio {
				lrItem.MaxLimitRequestRatio[string(k)] = v.String()
			}
		}

		lr.Items = append(lr.Items, lrItem)
	}

	return lr
}

// ConvertNetworkPolicy 转换 K8s NetworkPolicy 到 model_v3
func ConvertNetworkPolicy(k8sNP *networkingv1.NetworkPolicy) cluster.NetworkPolicy {
	np := cluster.NetworkPolicy{
		Name:             k8sNP.Name,
		Namespace:        k8sNP.Namespace,
		CreatedAt:        k8sNP.CreationTimestamp.Format(time.RFC3339),
		Age:              formatAge(k8sNP.CreationTimestamp.Time),
		IngressRuleCount: len(k8sNP.Spec.Ingress),
		EgressRuleCount:  len(k8sNP.Spec.Egress),
		Labels:           k8sNP.Labels,
		Annotations:      k8sNP.Annotations,
	}

	// Pod Selector - 转换为 JSON 字符串
	if k8sNP.Spec.PodSelector.Size() > 0 {
		if b, err := json.Marshal(k8sNP.Spec.PodSelector); err == nil {
			np.PodSelector = string(b)
		}
	}

	// Policy Types
	for _, pt := range k8sNP.Spec.PolicyTypes {
		np.PolicyTypes = append(np.PolicyTypes, string(pt))
	}

	// Ingress Rules
	for _, rule := range k8sNP.Spec.Ingress {
		np.IngressRules = append(np.IngressRules, convertNetworkPolicyIngressRule(rule))
	}

	// Egress Rules
	for _, rule := range k8sNP.Spec.Egress {
		np.EgressRules = append(np.EgressRules, convertNetworkPolicyEgressRule(rule))
	}

	return np
}

// convertNetworkPolicyIngressRule 转换入站规则
func convertNetworkPolicyIngressRule(rule networkingv1.NetworkPolicyIngressRule) cluster.NetworkPolicyRule {
	r := cluster.NetworkPolicyRule{}

	for _, from := range rule.From {
		r.Peers = append(r.Peers, convertNetworkPolicyPeer(from))
	}
	for _, port := range rule.Ports {
		r.Ports = append(r.Ports, convertNetworkPolicyPort(port))
	}

	return r
}

// convertNetworkPolicyEgressRule 转换出站规则
func convertNetworkPolicyEgressRule(rule networkingv1.NetworkPolicyEgressRule) cluster.NetworkPolicyRule {
	r := cluster.NetworkPolicyRule{}

	for _, to := range rule.To {
		r.Peers = append(r.Peers, convertNetworkPolicyPeer(to))
	}
	for _, port := range rule.Ports {
		r.Ports = append(r.Ports, convertNetworkPolicyPort(port))
	}

	return r
}

// convertNetworkPolicyPeer 转换网络策略对端
func convertNetworkPolicyPeer(peer networkingv1.NetworkPolicyPeer) cluster.NetworkPolicyPeer {
	p := cluster.NetworkPolicyPeer{}

	if peer.PodSelector != nil {
		p.Type = "podSelector"
		if b, err := json.Marshal(peer.PodSelector); err == nil {
			p.Selector = string(b)
		}
	} else if peer.NamespaceSelector != nil {
		p.Type = "namespaceSelector"
		if b, err := json.Marshal(peer.NamespaceSelector); err == nil {
			p.Selector = string(b)
		}
	} else if peer.IPBlock != nil {
		p.Type = "ipBlock"
		p.CIDR = peer.IPBlock.CIDR
		p.Except = peer.IPBlock.Except
	}

	return p
}

// convertNetworkPolicyPort 转换网络策略端口
func convertNetworkPolicyPort(port networkingv1.NetworkPolicyPort) cluster.NetworkPolicyPort {
	p := cluster.NetworkPolicyPort{
		Protocol: "TCP",
	}

	if port.Protocol != nil {
		p.Protocol = string(*port.Protocol)
	}
	if port.Port != nil {
		p.Port = port.Port.String()
	}
	if port.EndPort != nil {
		p.EndPort = port.EndPort
	}

	return p
}

// ConvertServiceAccount 转换 K8s ServiceAccount 到 model_v3
func ConvertServiceAccount(k8sSA *corev1.ServiceAccount) cluster.ServiceAccount {
	sa := cluster.ServiceAccount{
		Name:                         k8sSA.Name,
		Namespace:                    k8sSA.Namespace,
		CreatedAt:                    k8sSA.CreationTimestamp.Format(time.RFC3339),
		Age:                          formatAge(k8sSA.CreationTimestamp.Time),
		SecretsCount:                 len(k8sSA.Secrets),
		ImagePullSecretsCount:        len(k8sSA.ImagePullSecrets),
		AutomountServiceAccountToken: k8sSA.AutomountServiceAccountToken,
		Labels:                       k8sSA.Labels,
		Annotations:                  k8sSA.Annotations,
	}

	// Secret 名称列表
	for _, s := range k8sSA.Secrets {
		sa.SecretNames = append(sa.SecretNames, s.Name)
	}
	// ImagePullSecret 名称列表
	for _, s := range k8sSA.ImagePullSecrets {
		sa.ImagePullSecretNames = append(sa.ImagePullSecretNames, s.Name)
	}

	return sa
}

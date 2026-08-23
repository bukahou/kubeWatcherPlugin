package k8s

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/model_v3/cluster"
)

// ingressRepository Ingress 仓库实现
type ingressRepository struct {
	client sdk.K8sClient
}

// NewIngressRepository 创建 Ingress 仓库
func NewIngressRepository(client sdk.K8sClient) repository.IngressRepository {
	return &ingressRepository{client: client}
}

// List 列出 Ingress
func (r *ingressRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Ingress, error) {
	k8sIngresses, err := r.client.ListIngresses(ctx, namespace, sdk.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	})
	if err != nil {
		return nil, err
	}

	ingresses := make([]cluster.Ingress, 0, len(k8sIngresses))
	for i := range k8sIngresses {
		ingresses = append(ingresses, ConvertIngress(&k8sIngresses[i]))
	}
	return ingresses, nil
}

// Get 获取单个 Ingress
func (r *ingressRepository) Get(ctx context.Context, namespace, name string) (*cluster.Ingress, error) {
	k8sIngress, err := r.client.GetIngress(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	ingress := ConvertIngress(k8sIngress)
	return &ingress, nil
}

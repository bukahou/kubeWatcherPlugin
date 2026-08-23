package k8s

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/model_v3/cluster"
)

// namespaceRepository Namespace 仓库实现
type namespaceRepository struct {
	client sdk.K8sClient
}

// NewNamespaceRepository 创建 Namespace 仓库
func NewNamespaceRepository(client sdk.K8sClient) repository.NamespaceRepository {
	return &namespaceRepository{client: client}
}

// List 列出 Namespace
func (r *namespaceRepository) List(ctx context.Context, opts model.ListOptions) ([]cluster.Namespace, error) {
	k8sNamespaces, err := r.client.ListNamespaces(ctx, sdk.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	})
	if err != nil {
		return nil, err
	}

	namespaces := make([]cluster.Namespace, 0, len(k8sNamespaces))
	for i := range k8sNamespaces {
		namespaces = append(namespaces, ConvertNamespace(&k8sNamespaces[i]))
	}
	return namespaces, nil
}

// Get 获取单个 Namespace
func (r *namespaceRepository) Get(ctx context.Context, name string) (*cluster.Namespace, error) {
	k8sNamespace, err := r.client.GetNamespace(ctx, name)
	if err != nil {
		return nil, err
	}
	namespace := ConvertNamespace(k8sNamespace)
	return &namespace, nil
}

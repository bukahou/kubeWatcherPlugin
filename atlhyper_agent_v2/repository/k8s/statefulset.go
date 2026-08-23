package k8s

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/model_v3/cluster"
)

// statefulSetRepository StatefulSet 仓库实现
type statefulSetRepository struct {
	client sdk.K8sClient
}

// NewStatefulSetRepository 创建 StatefulSet 仓库
func NewStatefulSetRepository(client sdk.K8sClient) repository.StatefulSetRepository {
	return &statefulSetRepository{client: client}
}

// List 列出 StatefulSet
func (r *statefulSetRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.StatefulSet, error) {
	k8sStatefulSets, err := r.client.ListStatefulSets(ctx, namespace, sdk.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	})
	if err != nil {
		return nil, err
	}

	statefulSets := make([]cluster.StatefulSet, 0, len(k8sStatefulSets))
	for i := range k8sStatefulSets {
		statefulSets = append(statefulSets, ConvertStatefulSet(&k8sStatefulSets[i]))
	}
	return statefulSets, nil
}

// Get 获取单个 StatefulSet
func (r *statefulSetRepository) Get(ctx context.Context, namespace, name string) (*cluster.StatefulSet, error) {
	k8sStatefulSet, err := r.client.GetStatefulSet(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	statefulSet := ConvertStatefulSet(k8sStatefulSet)
	return &statefulSet, nil
}

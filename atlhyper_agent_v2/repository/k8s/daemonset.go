package k8s

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/model_v3/cluster"
)

// daemonSetRepository DaemonSet 仓库实现
type daemonSetRepository struct {
	client sdk.K8sClient
}

// NewDaemonSetRepository 创建 DaemonSet 仓库
func NewDaemonSetRepository(client sdk.K8sClient) repository.DaemonSetRepository {
	return &daemonSetRepository{client: client}
}

// List 列出 DaemonSet
func (r *daemonSetRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.DaemonSet, error) {
	k8sDaemonSets, err := r.client.ListDaemonSets(ctx, namespace, sdk.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	})
	if err != nil {
		return nil, err
	}

	daemonSets := make([]cluster.DaemonSet, 0, len(k8sDaemonSets))
	for i := range k8sDaemonSets {
		daemonSets = append(daemonSets, ConvertDaemonSet(&k8sDaemonSets[i]))
	}
	return daemonSets, nil
}

// Get 获取单个 DaemonSet
func (r *daemonSetRepository) Get(ctx context.Context, namespace, name string) (*cluster.DaemonSet, error) {
	k8sDaemonSet, err := r.client.GetDaemonSet(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	daemonSet := ConvertDaemonSet(k8sDaemonSet)
	return &daemonSet, nil
}

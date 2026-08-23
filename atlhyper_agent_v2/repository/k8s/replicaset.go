package k8s

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/model_v3/cluster"
)

// replicaSetRepository ReplicaSet 仓库实现
type replicaSetRepository struct {
	client sdk.K8sClient
}

// NewReplicaSetRepository 创建 ReplicaSet 仓库
func NewReplicaSetRepository(client sdk.K8sClient) repository.ReplicaSetRepository {
	return &replicaSetRepository{client: client}
}

// List 列出 ReplicaSet
func (r *replicaSetRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.ReplicaSet, error) {
	k8sReplicaSets, err := r.client.ListReplicaSets(ctx, namespace, sdk.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	})
	if err != nil {
		return nil, err
	}

	replicaSets := make([]cluster.ReplicaSet, 0, len(k8sReplicaSets))
	for i := range k8sReplicaSets {
		replicaSets = append(replicaSets, ConvertReplicaSet(&k8sReplicaSets[i]))
	}
	return replicaSets, nil
}

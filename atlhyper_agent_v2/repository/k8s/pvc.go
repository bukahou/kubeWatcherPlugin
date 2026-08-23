package k8s

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/model_v3/cluster"
)

// persistentVolumeClaimRepository PVC 仓库实现
type persistentVolumeClaimRepository struct {
	client sdk.K8sClient
}

// NewPersistentVolumeClaimRepository 创建 PVC 仓库
func NewPersistentVolumeClaimRepository(client sdk.K8sClient) repository.PersistentVolumeClaimRepository {
	return &persistentVolumeClaimRepository{client: client}
}

// List 列出 PersistentVolumeClaim
func (r *persistentVolumeClaimRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.PersistentVolumeClaim, error) {
	k8sPVCs, err := r.client.ListPersistentVolumeClaims(ctx, namespace, sdk.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	})
	if err != nil {
		return nil, err
	}

	pvcs := make([]cluster.PersistentVolumeClaim, 0, len(k8sPVCs))
	for i := range k8sPVCs {
		pvcs = append(pvcs, ConvertPersistentVolumeClaim(&k8sPVCs[i]))
	}
	return pvcs, nil
}

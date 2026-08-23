package k8s

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/model_v3/cluster"
)

// configMapRepository ConfigMap 仓库实现
type configMapRepository struct {
	client sdk.K8sClient
}

// NewConfigMapRepository 创建 ConfigMap 仓库
func NewConfigMapRepository(client sdk.K8sClient) repository.ConfigMapRepository {
	return &configMapRepository{client: client}
}

// List 列出 ConfigMap
func (r *configMapRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.ConfigMap, error) {
	k8sConfigMaps, err := r.client.ListConfigMaps(ctx, namespace, sdk.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	})
	if err != nil {
		return nil, err
	}

	configMaps := make([]cluster.ConfigMap, 0, len(k8sConfigMaps))
	for i := range k8sConfigMaps {
		configMaps = append(configMaps, ConvertConfigMap(&k8sConfigMaps[i]))
	}
	return configMaps, nil
}

// Get 获取单个 ConfigMap
func (r *configMapRepository) Get(ctx context.Context, namespace, name string) (*cluster.ConfigMap, error) {
	k8sConfigMap, err := r.client.GetConfigMap(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	configMap := ConvertConfigMap(k8sConfigMap)
	return &configMap, nil
}

package k8s

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/model_v3/cluster"
)

// secretRepository Secret 仓库实现
type secretRepository struct {
	client sdk.K8sClient
}

// NewSecretRepository 创建 Secret 仓库
func NewSecretRepository(client sdk.K8sClient) repository.SecretRepository {
	return &secretRepository{client: client}
}

// List 列出 Secret
func (r *secretRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Secret, error) {
	k8sSecrets, err := r.client.ListSecrets(ctx, namespace, sdk.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	})
	if err != nil {
		return nil, err
	}

	secrets := make([]cluster.Secret, 0, len(k8sSecrets))
	for i := range k8sSecrets {
		secrets = append(secrets, ConvertSecret(&k8sSecrets[i]))
	}
	return secrets, nil
}

// Get 获取单个 Secret
func (r *secretRepository) Get(ctx context.Context, namespace, name string) (*cluster.Secret, error) {
	k8sSecret, err := r.client.GetSecret(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	secret := ConvertSecret(k8sSecret)
	return &secret, nil
}

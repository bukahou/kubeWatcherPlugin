package k8s

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/model_v3/cluster"
)

// deploymentRepository Deployment 仓库实现
type deploymentRepository struct {
	client sdk.K8sClient
}

// NewDeploymentRepository 创建 Deployment 仓库
func NewDeploymentRepository(client sdk.K8sClient) repository.DeploymentRepository {
	return &deploymentRepository{client: client}
}

// List 列出 Deployment
func (r *deploymentRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Deployment, error) {
	k8sDeployments, err := r.client.ListDeployments(ctx, namespace, sdk.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	})
	if err != nil {
		return nil, err
	}

	deployments := make([]cluster.Deployment, 0, len(k8sDeployments))
	for i := range k8sDeployments {
		deployments = append(deployments, ConvertDeployment(&k8sDeployments[i]))
	}
	return deployments, nil
}

// Get 获取单个 Deployment
func (r *deploymentRepository) Get(ctx context.Context, namespace, name string) (*cluster.Deployment, error) {
	k8sDeployment, err := r.client.GetDeployment(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	deployment := ConvertDeployment(k8sDeployment)
	return &deployment, nil
}

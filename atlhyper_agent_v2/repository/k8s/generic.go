package k8s

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
)

// genericRepository 通用操作仓库实现
type genericRepository struct {
	client sdk.K8sClient
}

// NewGenericRepository 创建通用操作仓库
func NewGenericRepository(client sdk.K8sClient) repository.GenericRepository {
	return &genericRepository{client: client}
}

// =============================================================================
// 删除操作
// =============================================================================

// DeletePod 删除 Pod
func (r *genericRepository) DeletePod(ctx context.Context, namespace, name string, opts model.DeleteOptions) error {
	return r.client.DeletePod(ctx, namespace, name, sdk.DeleteOptions{
		GracePeriodSeconds: opts.GracePeriodSeconds,
		Force:              opts.Force,
	})
}

// Delete 删除任意资源
func (r *genericRepository) Delete(ctx context.Context, kind, namespace, name string, opts model.DeleteOptions) error {
	gvk := sdk.GroupVersionKind{Kind: kind}
	return r.client.Delete(ctx, gvk, namespace, name, sdk.DeleteOptions{
		GracePeriodSeconds: opts.GracePeriodSeconds,
		Force:              opts.Force,
	})
}

// =============================================================================
// Deployment 操作
// =============================================================================

// ScaleDeployment 扩缩容 Deployment
func (r *genericRepository) ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) error {
	return r.client.UpdateDeploymentScale(ctx, namespace, name, replicas)
}

// RestartDeployment 重启 Deployment
func (r *genericRepository) RestartDeployment(ctx context.Context, namespace, name string) error {
	return r.client.RestartDeployment(ctx, namespace, name)
}

// UpdateDeploymentImage 更新容器镜像
func (r *genericRepository) UpdateDeploymentImage(ctx context.Context, namespace, name, container, image string) error {
	return r.client.UpdateDeploymentImage(ctx, namespace, name, container, image)
}

// =============================================================================
// Node 操作
// =============================================================================

// CordonNode 封锁节点
func (r *genericRepository) CordonNode(ctx context.Context, name string) error {
	return r.client.CordonNode(ctx, name)
}

// UncordonNode 解封节点
func (r *genericRepository) UncordonNode(ctx context.Context, name string) error {
	return r.client.UncordonNode(ctx, name)
}

// =============================================================================
// 配置数据获取
// =============================================================================

// GetConfigMapData 获取 ConfigMap 数据内容
func (r *genericRepository) GetConfigMapData(ctx context.Context, namespace, name string) (map[string]string, error) {
	cm, err := r.client.GetConfigMap(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	return cm.Data, nil
}

// GetSecretData 获取 Secret 数据内容（base64 解码后）
func (r *genericRepository) GetSecretData(ctx context.Context, namespace, name string) (map[string]string, error) {
	secret, err := r.client.GetSecret(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	// 将 []byte 转换为 string
	result := make(map[string]string, len(secret.Data))
	for k, v := range secret.Data {
		result[k] = string(v)
	}
	return result, nil
}

// =============================================================================
// 动态查询
// =============================================================================

// Execute 执行动态查询 (仅 GET)
func (r *genericRepository) Execute(ctx context.Context, req *model.DynamicRequest) (*model.DynamicResponse, error) {
	sdkReq := sdk.DynamicRequest{
		Path:  req.Path,
		Query: req.Query,
	}

	sdkResp, err := r.client.Dynamic(ctx, sdkReq)
	if err != nil {
		return nil, err
	}

	return &model.DynamicResponse{
		StatusCode: sdkResp.StatusCode,
		Body:       sdkResp.Body,
	}, nil
}

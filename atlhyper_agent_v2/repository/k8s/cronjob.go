package k8s

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/model_v3/cluster"
)

// cronJobRepository CronJob 仓库实现
type cronJobRepository struct {
	client sdk.K8sClient
}

// NewCronJobRepository 创建 CronJob 仓库
func NewCronJobRepository(client sdk.K8sClient) repository.CronJobRepository {
	return &cronJobRepository{client: client}
}

// List 列出 CronJob
func (r *cronJobRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.CronJob, error) {
	k8sCronJobs, err := r.client.ListCronJobs(ctx, namespace, sdk.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	})
	if err != nil {
		return nil, err
	}

	cronJobs := make([]cluster.CronJob, 0, len(k8sCronJobs))
	for i := range k8sCronJobs {
		cronJobs = append(cronJobs, ConvertCronJob(&k8sCronJobs[i]))
	}
	return cronJobs, nil
}

// Get 获取单个 CronJob
func (r *cronJobRepository) Get(ctx context.Context, namespace, name string) (*cluster.CronJob, error) {
	k8sCronJob, err := r.client.GetCronJob(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	cronJob := ConvertCronJob(k8sCronJob)
	return &cronJob, nil
}

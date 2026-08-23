package k8s

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/model_v3/cluster"
)

// jobRepository Job 仓库实现
type jobRepository struct {
	client sdk.K8sClient
}

// NewJobRepository 创建 Job 仓库
func NewJobRepository(client sdk.K8sClient) repository.JobRepository {
	return &jobRepository{client: client}
}

// List 列出 Job
func (r *jobRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Job, error) {
	k8sJobs, err := r.client.ListJobs(ctx, namespace, sdk.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	})
	if err != nil {
		return nil, err
	}

	jobs := make([]cluster.Job, 0, len(k8sJobs))
	for i := range k8sJobs {
		jobs = append(jobs, ConvertJob(&k8sJobs[i]))
	}
	return jobs, nil
}

// Get 获取单个 Job
func (r *jobRepository) Get(ctx context.Context, namespace, name string) (*cluster.Job, error) {
	k8sJob, err := r.client.GetJob(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	job := ConvertJob(k8sJob)
	return &job, nil
}

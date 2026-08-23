package k8s

import (
	"context"
	"fmt"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	model_v3 "AtlHyper/model_v3"
	"AtlHyper/model_v3/cluster"
)

// podRepository Pod 仓库实现
//
// 所有方法都委托给 sdk.K8sClient，并转换返回类型。
type podRepository struct {
	client sdk.K8sClient
}

// NewPodRepository 创建 Pod 仓库
func NewPodRepository(client sdk.K8sClient) repository.PodRepository {
	return &podRepository{client: client}
}

// List 列出 Pod
func (r *podRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Pod, error) {
	k8sPods, err := r.client.ListPods(ctx, namespace, sdk.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	})
	if err != nil {
		return nil, err
	}

	// 获取 Pod Metrics
	podMetrics, _ := r.client.ListPodMetrics(ctx)

	pods := make([]cluster.Pod, 0, len(k8sPods))
	for i := range k8sPods {
		pod := ConvertPod(&k8sPods[i])

		// 关联 metrics
		key := pod.GetNamespace() + "/" + pod.GetName()
		if pm, ok := podMetrics[key]; ok {
			// 汇总所有容器的 CPU 和 Memory
			pod.Status.CPUUsage, pod.Status.MemoryUsage = aggregateContainerMetrics(pm.Containers)
		}

		pods = append(pods, pod)
	}
	return pods, nil
}

// aggregateContainerMetrics 汇总容器 metrics
func aggregateContainerMetrics(containers []sdk.ContainerMetrics) (string, string) {
	if len(containers) == 0 {
		return "", ""
	}
	if len(containers) == 1 {
		return containers[0].CPU, containers[0].Memory
	}
	var totalCPU int64 // 毫核
	var totalMem int64 // 字节
	for _, c := range containers {
		totalCPU += model_v3.ParseCPU(c.CPU)
		totalMem += model_v3.ParseMemory(c.Memory)
	}
	cpuStr := fmt.Sprintf("%dm", totalCPU)
	memStr := fmt.Sprintf("%dKi", totalMem/1024)
	return cpuStr, memStr
}

// Get 获取单个 Pod
func (r *podRepository) Get(ctx context.Context, namespace, name string) (*cluster.Pod, error) {
	k8sPod, err := r.client.GetPod(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	pod := ConvertPod(k8sPod)
	return &pod, nil
}

// GetLogs 获取 Pod 日志
func (r *podRepository) GetLogs(ctx context.Context, namespace, name string, opts model.LogOptions) (string, error) {
	return r.client.GetPodLogs(ctx, namespace, name, sdk.LogOptions{
		Container:    opts.Container,
		TailLines:    opts.TailLines,
		SinceSeconds: opts.SinceSeconds,
		Timestamps:   opts.Timestamps,
		Previous:     opts.Previous,
	})
}

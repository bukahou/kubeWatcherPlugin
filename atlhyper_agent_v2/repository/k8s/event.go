package k8s

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/model_v3/cluster"
)

// eventRepository Event 仓库实现
type eventRepository struct {
	client sdk.K8sClient
}

// NewEventRepository 创建 Event 仓库
func NewEventRepository(client sdk.K8sClient) repository.EventRepository {
	return &eventRepository{client: client}
}

// List 列出 Event
func (r *eventRepository) List(ctx context.Context, namespace string, opts model.ListOptions) ([]cluster.Event, error) {
	k8sEvents, err := r.client.ListEvents(ctx, namespace, sdk.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	})
	if err != nil {
		return nil, err
	}

	events := make([]cluster.Event, 0, len(k8sEvents))
	for i := range k8sEvents {
		events = append(events, ConvertEvent(&k8sEvents[i]))
	}
	return events, nil
}

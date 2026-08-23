package query

import (
	"context"
	"testing"
	"time"

	model_v3 "AtlHyper/model_v3"
	agentmodel "AtlHyper/model_v3/agent"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/command"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
)

// ==================== Mock: datahub.Store (overview 专用) ====================

type mockStoreForOverview struct {
	snapshots     map[string]*cluster.ClusterSnapshot
	agents        []agentmodel.AgentInfo
	events        map[string][]cluster.Event
	agentStatuses map[string]*agentmodel.AgentStatus
	otelTimeline  []cluster.OTelEntry
}

func (m *mockStoreForOverview) SetSnapshot(clusterID string, snapshot *cluster.ClusterSnapshot) error {
	return nil
}
func (m *mockStoreForOverview) GetSnapshot(clusterID string) (*cluster.ClusterSnapshot, error) {
	if m.snapshots != nil {
		return m.snapshots[clusterID], nil
	}
	return nil, nil
}
func (m *mockStoreForOverview) UpdateHeartbeat(clusterID string) error { return nil }
func (m *mockStoreForOverview) GetAgentStatus(clusterID string) (*agentmodel.AgentStatus, error) {
	if m.agentStatuses != nil {
		return m.agentStatuses[clusterID], nil
	}
	return nil, nil
}
func (m *mockStoreForOverview) ListAgents() ([]agentmodel.AgentInfo, error) {
	return m.agents, nil
}
func (m *mockStoreForOverview) GetEvents(clusterID string) ([]cluster.Event, error) {
	if m.events != nil {
		return m.events[clusterID], nil
	}
	return nil, nil
}
func (m *mockStoreForOverview) GetOTelTimeline(clusterID string, since time.Time) ([]cluster.OTelEntry, error) {
	return m.otelTimeline, nil
}
func (m *mockStoreForOverview) Start() error { return nil }
func (m *mockStoreForOverview) Stop() error  { return nil }

// ==================== Mock: mq.Producer (overview 专用) ====================

type mockBusForOverview struct {
	status *command.Status
	err    error
}

func (m *mockBusForOverview) EnqueueCommand(clusterID, topic string, cmd *command.Command) error {
	return m.err
}
func (m *mockBusForOverview) GetCommandStatus(cmdID string) (*command.Status, error) {
	return m.status, m.err
}
func (m *mockBusForOverview) WaitCommandResult(ctx context.Context, cmdID string, timeout time.Duration) (*command.Result, error) {
	return nil, m.err
}

// ==================== Phase 1: 单资源查询测试 ====================

// --- GetPod ---

func TestGetPod_Found(t *testing.T) {
	store := &mockStoreForOverview{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				Pods: []cluster.Pod{
					{
						Summary: cluster.PodSummary{Name: "nginx", Namespace: "default"},
						Status:  cluster.PodStatus{Phase: "Running"},
					},
					{
						Summary: cluster.PodSummary{Name: "redis", Namespace: "cache"},
						Status:  cluster.PodStatus{Phase: "Running"},
					},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	pod, err := svc.GetPod(context.Background(), "cluster-1", "cache", "redis")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pod == nil {
		t.Fatal("expected pod, got nil")
	}
	if pod.GetName() != "redis" {
		t.Errorf("expected name=redis, got %s", pod.GetName())
	}
	if pod.GetNamespace() != "cache" {
		t.Errorf("expected namespace=cache, got %s", pod.GetNamespace())
	}
}

func TestGetPod_NotFound(t *testing.T) {
	store := &mockStoreForOverview{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				Pods: []cluster.Pod{
					{Summary: cluster.PodSummary{Name: "nginx", Namespace: "default"}},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	pod, err := svc.GetPod(context.Background(), "cluster-1", "default", "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pod != nil {
		t.Errorf("expected nil for not-found, got %+v", pod)
	}
}

func TestGetPod_NoSnapshot(t *testing.T) {
	store := &mockStoreForOverview{snapshots: map[string]*cluster.ClusterSnapshot{}}
	svc := &QueryService{store: store}

	pod, err := svc.GetPod(context.Background(), "unknown-cluster", "default", "nginx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pod != nil {
		t.Errorf("expected nil for no-snapshot, got %+v", pod)
	}
}

// --- GetNode ---

func TestGetNode_Found(t *testing.T) {
	store := &mockStoreForOverview{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				Nodes: []cluster.Node{
					{Summary: cluster.NodeSummary{Name: "node-1"}},
					{Summary: cluster.NodeSummary{Name: "node-2"}},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	node, err := svc.GetNode(context.Background(), "cluster-1", "node-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node == nil {
		t.Fatal("expected node, got nil")
	}
	if node.GetName() != "node-2" {
		t.Errorf("expected name=node-2, got %s", node.GetName())
	}
}

func TestGetNode_NotFound(t *testing.T) {
	store := &mockStoreForOverview{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				Nodes: []cluster.Node{
					{Summary: cluster.NodeSummary{Name: "node-1"}},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	node, err := svc.GetNode(context.Background(), "cluster-1", "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node != nil {
		t.Errorf("expected nil for not-found, got %+v", node)
	}
}

// --- GetDeployment ---

func TestGetDeployment_Found(t *testing.T) {
	store := &mockStoreForOverview{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				Deployments: []cluster.Deployment{
					{Summary: cluster.DeploymentSummary{Name: "web", Namespace: "prod"}},
					{Summary: cluster.DeploymentSummary{Name: "api", Namespace: "prod"}},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	dep, err := svc.GetDeployment(context.Background(), "cluster-1", "prod", "api")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dep == nil {
		t.Fatal("expected deployment, got nil")
	}
	if dep.GetName() != "api" {
		t.Errorf("expected name=api, got %s", dep.GetName())
	}
	if dep.GetNamespace() != "prod" {
		t.Errorf("expected namespace=prod, got %s", dep.GetNamespace())
	}
}

func TestGetDeployment_NotFound(t *testing.T) {
	store := &mockStoreForOverview{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				Deployments: []cluster.Deployment{
					{Summary: cluster.DeploymentSummary{Name: "web", Namespace: "prod"}},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	dep, err := svc.GetDeployment(context.Background(), "cluster-1", "prod", "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dep != nil {
		t.Errorf("expected nil for not-found, got %+v", dep)
	}
}

// --- GetDeploymentByReplicaSet ---

func TestGetDeploymentByReplicaSet_Found(t *testing.T) {
	store := &mockStoreForOverview{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				Deployments: []cluster.Deployment{
					{Summary: cluster.DeploymentSummary{Name: "web-app", Namespace: "prod"}},
					{Summary: cluster.DeploymentSummary{Name: "api-server", Namespace: "prod"}},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	// RS 名 = deployment名-hash
	dep, err := svc.GetDeploymentByReplicaSet(context.Background(), "cluster-1", "prod", "api-server-7f8d9c6b5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dep == nil {
		t.Fatal("expected deployment, got nil")
	}
	if dep.GetName() != "api-server" {
		t.Errorf("expected name=api-server, got %s", dep.GetName())
	}
}

func TestGetDeploymentByReplicaSet_NotFound(t *testing.T) {
	store := &mockStoreForOverview{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				Deployments: []cluster.Deployment{
					{Summary: cluster.DeploymentSummary{Name: "web-app", Namespace: "prod"}},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	dep, err := svc.GetDeploymentByReplicaSet(context.Background(), "cluster-1", "prod", "nonexistent-7f8d9c6b5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dep != nil {
		t.Errorf("expected nil for not-found, got %+v", dep)
	}
}

func TestGetDeploymentByReplicaSet_WrongNamespace(t *testing.T) {
	store := &mockStoreForOverview{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				Deployments: []cluster.Deployment{
					{Summary: cluster.DeploymentSummary{Name: "web-app", Namespace: "prod"}},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	// RS 名前缀匹配但 namespace 不同
	dep, err := svc.GetDeploymentByReplicaSet(context.Background(), "cluster-1", "staging", "web-app-7f8d9c6b5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dep != nil {
		t.Errorf("expected nil for wrong-namespace, got %+v", dep)
	}
}

// ==================== Phase 1: Event 查询测试 ====================

func makeTestEvents() []cluster.Event {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	return []cluster.Event{
		{
			CommonMeta:     model_v3.CommonMeta{Name: "evt-1"},
			Type:           "Warning",
			Reason:         "OOMKilled",
			Message:        "Container killed due to OOM",
			InvolvedObject: model_v3.ResourceRef{Kind: "Pod", Namespace: "default", Name: "nginx"},
			LastTimestamp:  now,
		},
		{
			CommonMeta:     model_v3.CommonMeta{Name: "evt-2"},
			Type:           "Normal",
			Reason:         "Pulled",
			Message:        "Successfully pulled image",
			InvolvedObject: model_v3.ResourceRef{Kind: "Pod", Namespace: "default", Name: "redis"},
			LastTimestamp:  now.Add(-1 * time.Hour),
		},
		{
			CommonMeta:     model_v3.CommonMeta{Name: "evt-3"},
			Type:           "Warning",
			Reason:         "BackOff",
			Message:        "Back-off restarting failed container",
			InvolvedObject: model_v3.ResourceRef{Kind: "Pod", Namespace: "kube-system", Name: "coredns"},
			LastTimestamp:  now.Add(-2 * time.Hour),
		},
		{
			CommonMeta:     model_v3.CommonMeta{Name: "evt-4"},
			Type:           "Normal",
			Reason:         "Scheduled",
			Message:        "Successfully assigned pod",
			InvolvedObject: model_v3.ResourceRef{Kind: "Pod", Namespace: "default", Name: "nginx"},
			LastTimestamp:  now.Add(-3 * time.Hour),
		},
	}
}

// --- GetEvents ---

func TestGetEvents_NoFilter(t *testing.T) {
	events := makeTestEvents()
	store := &mockStoreForOverview{
		events: map[string][]cluster.Event{"cluster-1": events},
	}
	svc := &QueryService{store: store}

	result, err := svc.GetEvents(context.Background(), "cluster-1", model.EventQueryOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 4 {
		t.Fatalf("expected 4 events, got %d", len(result))
	}
}

func TestGetEvents_TypeFilter(t *testing.T) {
	events := makeTestEvents()
	store := &mockStoreForOverview{
		events: map[string][]cluster.Event{"cluster-1": events},
	}
	svc := &QueryService{store: store}

	result, err := svc.GetEvents(context.Background(), "cluster-1", model.EventQueryOpts{Type: "Warning"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 Warning events, got %d", len(result))
	}
	for _, e := range result {
		if e.Type != "Warning" {
			t.Errorf("expected type=Warning, got %s", e.Type)
		}
	}
}

func TestGetEvents_ReasonFilter(t *testing.T) {
	events := makeTestEvents()
	store := &mockStoreForOverview{
		events: map[string][]cluster.Event{"cluster-1": events},
	}
	svc := &QueryService{store: store}

	result, err := svc.GetEvents(context.Background(), "cluster-1", model.EventQueryOpts{Reason: "OOMKilled"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 event with Reason=OOMKilled, got %d", len(result))
	}
	if result[0].Reason != "OOMKilled" {
		t.Errorf("expected reason=OOMKilled, got %s", result[0].Reason)
	}
}

func TestGetEvents_SinceFilter(t *testing.T) {
	events := makeTestEvents()
	store := &mockStoreForOverview{
		events: map[string][]cluster.Event{"cluster-1": events},
	}
	svc := &QueryService{store: store}

	// Since = 1.5h ago → should get evt-1 (now) and evt-2 (1h ago), exclude evt-3 (2h ago) and evt-4 (3h ago)
	since := time.Date(2025, 6, 1, 10, 30, 0, 0, time.UTC) // 1.5h before now
	result, err := svc.GetEvents(context.Background(), "cluster-1", model.EventQueryOpts{Since: since})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 events after since, got %d", len(result))
	}
}

func TestGetEvents_Pagination(t *testing.T) {
	events := makeTestEvents()
	store := &mockStoreForOverview{
		events: map[string][]cluster.Event{"cluster-1": events},
	}
	svc := &QueryService{store: store}

	// Offset=1, Limit=2 → skip first, take next 2
	result, err := svc.GetEvents(context.Background(), "cluster-1", model.EventQueryOpts{Offset: 1, Limit: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 events with pagination, got %d", len(result))
	}
	// After offset=1, the first element should be evt-2
	if result[0].CommonMeta.Name != "evt-2" {
		t.Errorf("expected first paginated event=evt-2, got %s", result[0].CommonMeta.Name)
	}
}

func TestGetEvents_NoSnapshot(t *testing.T) {
	store := &mockStoreForOverview{events: map[string][]cluster.Event{}}
	svc := &QueryService{store: store}

	result, err := svc.GetEvents(context.Background(), "unknown-cluster", model.EventQueryOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 events for unknown cluster, got %d", len(result))
	}
}

// --- GetEventsByResource ---

func TestGetEventsByResource_Found(t *testing.T) {
	events := makeTestEvents()
	store := &mockStoreForOverview{
		events: map[string][]cluster.Event{"cluster-1": events},
	}
	svc := &QueryService{store: store}

	result, err := svc.GetEventsByResource(context.Background(), "cluster-1", "Pod", "default", "nginx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// evt-1 和 evt-4 都是 Pod/default/nginx
	if len(result) != 2 {
		t.Fatalf("expected 2 events for Pod/default/nginx, got %d", len(result))
	}
	for _, e := range result {
		if e.InvolvedObject.Name != "nginx" {
			t.Errorf("expected involved name=nginx, got %s", e.InvolvedObject.Name)
		}
	}
}

func TestGetEventsByResource_NotFound(t *testing.T) {
	events := makeTestEvents()
	store := &mockStoreForOverview{
		events: map[string][]cluster.Event{"cluster-1": events},
	}
	svc := &QueryService{store: store}

	result, err := svc.GetEventsByResource(context.Background(), "cluster-1", "Deployment", "prod", "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 events for non-matching resource, got %d", len(result))
	}
}

// ==================== Phase 2: 透传方法测试 ====================

func TestGetAgentStatus_Delegate(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	injectedStatus := &agentmodel.AgentStatus{
		ClusterID:     "cluster-1",
		Status:        "online",
		LastHeartbeat: now,
		LastSnapshot:  now.Add(-30 * time.Second),
	}
	store := &mockStoreForOverview{
		agentStatuses: map[string]*agentmodel.AgentStatus{
			"cluster-1": injectedStatus,
		},
	}
	svc := &QueryService{store: store}

	status, err := svc.GetAgentStatus(context.Background(), "cluster-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status == nil {
		t.Fatal("expected status, got nil")
	}
	if status.ClusterID != "cluster-1" {
		t.Errorf("expected clusterID=cluster-1, got %s", status.ClusterID)
	}
	if status.Status != "online" {
		t.Errorf("expected status=online, got %s", status.Status)
	}
	if status != injectedStatus {
		t.Error("expected GetAgentStatus to return the exact object from store (transparent delegation)")
	}
}

func TestGetCommandStatus_Delegate(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	injectedStatus := &command.Status{
		CommandID: "cmd-123",
		Status:    "completed",
		CreatedAt: now,
	}
	bus := &mockBusForOverview{status: injectedStatus}
	svc := &QueryService{bus: bus}

	status, err := svc.GetCommandStatus(context.Background(), "cmd-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status == nil {
		t.Fatal("expected status, got nil")
	}
	if status.CommandID != "cmd-123" {
		t.Errorf("expected commandID=cmd-123, got %s", status.CommandID)
	}
	if status.Status != "completed" {
		t.Errorf("expected status=completed, got %s", status.Status)
	}
	if status != injectedStatus {
		t.Error("expected GetCommandStatus to return the exact object from bus (transparent delegation)")
	}
}

// ==================== Phase 2: 集群查询测试 ====================

func TestListClusters_WithSnapshots(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	store := &mockStoreForOverview{
		agents: []agentmodel.AgentInfo{
			{ClusterID: "cluster-1", Status: "online", LastHeartbeat: now},
			{ClusterID: "cluster-2", Status: "offline", LastHeartbeat: now.Add(-5 * time.Minute)},
		},
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				Nodes: []cluster.Node{
					{Summary: cluster.NodeSummary{Name: "node-1"}},
					{Summary: cluster.NodeSummary{Name: "node-2"}},
				},
				Pods: []cluster.Pod{
					{Summary: cluster.PodSummary{Name: "pod-1", Namespace: "default"}},
					{Summary: cluster.PodSummary{Name: "pod-2", Namespace: "default"}},
					{Summary: cluster.PodSummary{Name: "pod-3", Namespace: "kube-system"}},
				},
				OTel: &cluster.OTelSnapshot{}, // OTel 可用
			},
			"cluster-2": {
				Nodes: []cluster.Node{
					{Summary: cluster.NodeSummary{Name: "node-a"}},
				},
				Pods: []cluster.Pod{
					{Summary: cluster.PodSummary{Name: "pod-a", Namespace: "default"}},
				},
				// OTel 为 nil → OTelAvailable=false
			},
		},
	}
	svc := &QueryService{store: store}

	result, err := svc.ListClusters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(result))
	}

	// cluster-1: 2 nodes, 3 pods, OTel available
	c1 := result[0]
	if c1.ClusterID != "cluster-1" {
		t.Errorf("expected clusterID=cluster-1, got %s", c1.ClusterID)
	}
	if c1.NodeCount != 2 {
		t.Errorf("expected nodeCount=2, got %d", c1.NodeCount)
	}
	if c1.PodCount != 3 {
		t.Errorf("expected podCount=3, got %d", c1.PodCount)
	}
	if !c1.OTelAvailable {
		t.Error("expected OTelAvailable=true for cluster-1")
	}

	// cluster-2: 1 node, 1 pod, no OTel
	c2 := result[1]
	if c2.ClusterID != "cluster-2" {
		t.Errorf("expected clusterID=cluster-2, got %s", c2.ClusterID)
	}
	if c2.NodeCount != 1 {
		t.Errorf("expected nodeCount=1, got %d", c2.NodeCount)
	}
	if c2.PodCount != 1 {
		t.Errorf("expected podCount=1, got %d", c2.PodCount)
	}
	if c2.OTelAvailable {
		t.Error("expected OTelAvailable=false for cluster-2")
	}
}

func TestListClusters_Empty(t *testing.T) {
	store := &mockStoreForOverview{agents: []agentmodel.AgentInfo{}}
	svc := &QueryService{store: store}

	result, err := svc.ListClusters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 clusters, got %d", len(result))
	}
}

func TestGetCluster_Found(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	snapshot := &cluster.ClusterSnapshot{
		Nodes: []cluster.Node{{Summary: cluster.NodeSummary{Name: "node-1"}}},
		Pods:  []cluster.Pod{{Summary: cluster.PodSummary{Name: "pod-1", Namespace: "default"}}},
	}
	agentStatus := &agentmodel.AgentStatus{
		ClusterID:     "cluster-1",
		Status:        "online",
		LastHeartbeat: now,
	}
	store := &mockStoreForOverview{
		snapshots:     map[string]*cluster.ClusterSnapshot{"cluster-1": snapshot},
		agentStatuses: map[string]*agentmodel.AgentStatus{"cluster-1": agentStatus},
	}
	svc := &QueryService{store: store}

	detail, err := svc.GetCluster(context.Background(), "cluster-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail == nil {
		t.Fatal("expected ClusterDetail, got nil")
	}
	if detail.ClusterID != "cluster-1" {
		t.Errorf("expected clusterID=cluster-1, got %s", detail.ClusterID)
	}
	if detail.Status == nil {
		t.Fatal("expected Status != nil")
	}
	if detail.Status.Status != "online" {
		t.Errorf("expected status=online, got %s", detail.Status.Status)
	}
	if detail.Snapshot == nil {
		t.Fatal("expected Snapshot != nil")
	}
	if len(detail.Snapshot.Nodes) != 1 {
		t.Errorf("expected 1 node in snapshot, got %d", len(detail.Snapshot.Nodes))
	}
}

func TestGetCluster_NoSnapshot(t *testing.T) {
	// clusterID 不在 snapshots 中 → GetSnapshot 返回 nil
	// GetCluster 仍然返回 ClusterDetail（Snapshot=nil, Status=nil）
	store := &mockStoreForOverview{
		snapshots: map[string]*cluster.ClusterSnapshot{},
	}
	svc := &QueryService{store: store}

	detail, err := svc.GetCluster(context.Background(), "unknown-cluster")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail == nil {
		t.Fatal("expected ClusterDetail even without snapshot, got nil")
	}
	if detail.ClusterID != "unknown-cluster" {
		t.Errorf("expected clusterID=unknown-cluster, got %s", detail.ClusterID)
	}
	if detail.Snapshot != nil {
		t.Errorf("expected Snapshot=nil, got %+v", detail.Snapshot)
	}
	// Status 也应为 nil（store 中无此 clusterID 的 AgentStatus）
	if detail.Status != nil {
		t.Errorf("expected Status=nil, got %+v", detail.Status)
	}
}

// ==================== Phase 3: GetOverview 测试 ====================

// --- Mock: database.ClusterEventRepository (overview 专用) ---

type mockEventRepoForOverview struct {
	hourlyKindCounts []database.HourlyKindCount
	clusterEvents    []*database.ClusterEvent
}

func (m *mockEventRepoForOverview) Upsert(ctx context.Context, event *database.ClusterEvent) error {
	return nil
}
func (m *mockEventRepoForOverview) UpsertBatch(ctx context.Context, events []*database.ClusterEvent) error {
	return nil
}
func (m *mockEventRepoForOverview) ListByCluster(ctx context.Context, clusterID string, opts database.EventQueryOpts) ([]*database.ClusterEvent, error) {
	return m.clusterEvents, nil
}
func (m *mockEventRepoForOverview) ListByInvolvedResource(ctx context.Context, clusterID, kind, namespace, name string) ([]*database.ClusterEvent, error) {
	return nil, nil
}
func (m *mockEventRepoForOverview) ListByType(ctx context.Context, clusterID, eventType string, since time.Time) ([]*database.ClusterEvent, error) {
	return nil, nil
}
func (m *mockEventRepoForOverview) GetLatestEventID(ctx context.Context) (int64, error) {
	return 0, nil
}
func (m *mockEventRepoForOverview) GetEventsSince(ctx context.Context, sinceID int64) ([]*database.ClusterEvent, error) {
	return nil, nil
}
func (m *mockEventRepoForOverview) DeleteBefore(ctx context.Context, clusterID string, before time.Time) (int64, error) {
	return 0, nil
}
func (m *mockEventRepoForOverview) DeleteOldest(ctx context.Context, clusterID string, keepCount int) (int64, error) {
	return 0, nil
}
func (m *mockEventRepoForOverview) CountByCluster(ctx context.Context, clusterID string) (int64, error) {
	return 0, nil
}
func (m *mockEventRepoForOverview) CountByHour(ctx context.Context, clusterID string, hours int) ([]database.HourlyEventCount, error) {
	return nil, nil
}
func (m *mockEventRepoForOverview) CountByHourAndKind(ctx context.Context, clusterID string, hours int) ([]database.HourlyKindCount, error) {
	return m.hourlyKindCounts, nil
}

// --- GetOverview 测试用例 ---

func TestGetOverview_NoSnapshot(t *testing.T) {
	store := &mockStoreForOverview{snapshots: map[string]*cluster.ClusterSnapshot{}}
	svc := &QueryService{store: store}

	overview, err := svc.GetOverview(context.Background(), "unknown-cluster")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overview != nil {
		t.Errorf("expected nil for no-snapshot, got %+v", overview)
	}
}

func TestGetOverview_BasicCards(t *testing.T) {
	// 最小快照：2 nodes (1 ready), 3 pods (2 running)
	store := &mockStoreForOverview{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				Summary: cluster.ClusterSummary{
					TotalNodes:  2,
					ReadyNodes:  1,
					TotalPods:   3,
					RunningPods: 2,
				},
				Nodes: []cluster.Node{
					{
						Summary:     cluster.NodeSummary{Name: "node-1"},
						Allocatable: cluster.NodeResources{CPU: "2", Memory: "4Gi"},
					},
					{
						Summary:     cluster.NodeSummary{Name: "node-2"},
						Allocatable: cluster.NodeResources{CPU: "2", Memory: "4Gi"},
					},
				},
				Pods: []cluster.Pod{
					{Summary: cluster.PodSummary{Name: "p1", Namespace: "default"}, Status: cluster.PodStatus{Phase: "Running"}},
					{Summary: cluster.PodSummary{Name: "p2", Namespace: "default"}, Status: cluster.PodStatus{Phase: "Running"}},
					{Summary: cluster.PodSummary{Name: "p3", Namespace: "default"}, Status: cluster.PodStatus{Phase: "Pending"}},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	overview, err := svc.GetOverview(context.Background(), "cluster-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overview == nil {
		t.Fatal("expected overview, got nil")
	}

	// 验证 Cards
	cards := overview.Cards

	// NodeReady: 1/2 = 50%
	if cards.NodeReady.Total != 2 {
		t.Errorf("expected NodeReady.Total=2, got %d", cards.NodeReady.Total)
	}
	if cards.NodeReady.Ready != 1 {
		t.Errorf("expected NodeReady.Ready=1, got %d", cards.NodeReady.Ready)
	}
	if cards.NodeReady.Percent != 50.0 {
		t.Errorf("expected NodeReady.Percent=50, got %f", cards.NodeReady.Percent)
	}

	// ClusterHealth: nodeReady=50% < 70 → Unhealthy
	if cards.ClusterHealth.Status != "Unhealthy" {
		t.Errorf("expected health=Unhealthy (nodeReady=50%%), got %s", cards.ClusterHealth.Status)
	}
	if cards.ClusterHealth.NodeReadyPercent != 50.0 {
		t.Errorf("expected nodeReadyPercent=50, got %f", cards.ClusterHealth.NodeReadyPercent)
	}

	// PodReadyPercent: runningPods/totalPods = 2/3 ≈ 66.67%
	expectedPodPct := float64(2) / float64(3) * 100
	if cards.ClusterHealth.PodReadyPercent != expectedPodPct {
		t.Errorf("expected podReadyPercent=%f, got %f", expectedPodPct, cards.ClusterHealth.PodReadyPercent)
	}
}

func TestGetOverview_WorkloadStats(t *testing.T) {
	store := &mockStoreForOverview{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				Summary: cluster.ClusterSummary{TotalNodes: 1, ReadyNodes: 1, TotalPods: 5, RunningPods: 3},
				Nodes:   []cluster.Node{{Summary: cluster.NodeSummary{Name: "node-1"}, Allocatable: cluster.NodeResources{CPU: "4", Memory: "8Gi"}}},
				Deployments: []cluster.Deployment{
					{Summary: cluster.DeploymentSummary{Name: "dep-1", Namespace: "default", Replicas: 3, Ready: 3}}, // healthy
					{Summary: cluster.DeploymentSummary{Name: "dep-2", Namespace: "default", Replicas: 2, Ready: 1}}, // not healthy
				},
				StatefulSets: []cluster.StatefulSet{
					{Summary: cluster.StatefulSetSummary{Name: "sts-1", Namespace: "default", Replicas: 2, Ready: 2}}, // healthy
				},
				DaemonSets: []cluster.DaemonSet{
					{Summary: cluster.DaemonSetSummary{Name: "ds-1", Namespace: "kube-system", DesiredNumberScheduled: 3, NumberReady: 3}}, // healthy
					{Summary: cluster.DaemonSetSummary{Name: "ds-2", Namespace: "kube-system", DesiredNumberScheduled: 2, NumberReady: 1}}, // not healthy
				},
				Jobs: []cluster.Job{
					{CommonMeta: model_v3.CommonMeta{Name: "job-1"}, Active: 1, Succeeded: 0, Failed: 0, Complete: false}, // running
					{CommonMeta: model_v3.CommonMeta{Name: "job-2"}, Active: 0, Succeeded: 1, Failed: 0, Complete: true},  // succeeded
					{CommonMeta: model_v3.CommonMeta{Name: "job-3"}, Active: 0, Succeeded: 0, Failed: 2, Complete: false}, // failed
				},
				Pods: []cluster.Pod{
					{Summary: cluster.PodSummary{Name: "p1"}, Status: cluster.PodStatus{Phase: "Running"}},
					{Summary: cluster.PodSummary{Name: "p2"}, Status: cluster.PodStatus{Phase: "Running"}},
					{Summary: cluster.PodSummary{Name: "p3"}, Status: cluster.PodStatus{Phase: "Pending"}},
					{Summary: cluster.PodSummary{Name: "p4"}, Status: cluster.PodStatus{Phase: "Failed"}},
					{Summary: cluster.PodSummary{Name: "p5"}, Status: cluster.PodStatus{Phase: "Succeeded"}},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	overview, err := svc.GetOverview(context.Background(), "cluster-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	w := overview.Workloads

	// Deployments: 2 total, 1 healthy
	if w.Summary.Deployments.Total != 2 {
		t.Errorf("expected Deployments.Total=2, got %d", w.Summary.Deployments.Total)
	}
	if w.Summary.Deployments.Ready != 1 {
		t.Errorf("expected Deployments.Ready=1, got %d", w.Summary.Deployments.Ready)
	}

	// StatefulSets: 1 total, 1 healthy
	if w.Summary.StatefulSets.Total != 1 {
		t.Errorf("expected StatefulSets.Total=1, got %d", w.Summary.StatefulSets.Total)
	}
	if w.Summary.StatefulSets.Ready != 1 {
		t.Errorf("expected StatefulSets.Ready=1, got %d", w.Summary.StatefulSets.Ready)
	}

	// DaemonSets: 2 total, 1 healthy
	if w.Summary.DaemonSets.Total != 2 {
		t.Errorf("expected DaemonSets.Total=2, got %d", w.Summary.DaemonSets.Total)
	}
	if w.Summary.DaemonSets.Ready != 1 {
		t.Errorf("expected DaemonSets.Ready=1, got %d", w.Summary.DaemonSets.Ready)
	}

	// Jobs: 3 total, 1 running, 1 succeeded, 1 failed
	if w.Summary.Jobs.Total != 3 {
		t.Errorf("expected Jobs.Total=3, got %d", w.Summary.Jobs.Total)
	}
	if w.Summary.Jobs.Running != 1 {
		t.Errorf("expected Jobs.Running=1, got %d", w.Summary.Jobs.Running)
	}
	if w.Summary.Jobs.Succeeded != 1 {
		t.Errorf("expected Jobs.Succeeded=1, got %d", w.Summary.Jobs.Succeeded)
	}
	if w.Summary.Jobs.Failed != 1 {
		t.Errorf("expected Jobs.Failed=1, got %d", w.Summary.Jobs.Failed)
	}

	// PodStatus: 5 total, 2 running, 1 pending, 1 failed, 1 succeeded
	ps := w.PodStatus
	if ps.Total != 5 {
		t.Errorf("expected PodStatus.Total=5, got %d", ps.Total)
	}
	if ps.Running != 2 {
		t.Errorf("expected Running=2, got %d", ps.Running)
	}
	if ps.Pending != 1 {
		t.Errorf("expected Pending=1, got %d", ps.Pending)
	}
	if ps.Failed != 1 {
		t.Errorf("expected Failed=1, got %d", ps.Failed)
	}
	if ps.Succeeded != 1 {
		t.Errorf("expected Succeeded=1, got %d", ps.Succeeded)
	}
	// RunningPercent = 2/5 * 100 = 40%
	if ps.RunningPercent != 40.0 {
		t.Errorf("expected RunningPercent=40, got %f", ps.RunningPercent)
	}
}

func TestGetOverview_NodeUsageAndPeak(t *testing.T) {
	// 2 个节点有 Metrics 数据
	store := &mockStoreForOverview{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				Summary: cluster.ClusterSummary{TotalNodes: 2, ReadyNodes: 2, TotalPods: 1, RunningPods: 1},
				Nodes: []cluster.Node{
					{
						Summary:     cluster.NodeSummary{Name: "node-1"},
						Allocatable: cluster.NodeResources{CPU: "4", Memory: "8Gi"},
						Metrics: &cluster.NodeResourceUsage{
							CPU:    cluster.NodeResourceMetric{Usage: "2"},   // 2000m / 4000m = 50%
							Memory: cluster.NodeResourceMetric{Usage: "2Gi"}, // 2Gi / 8Gi = 25%
						},
					},
					{
						Summary:     cluster.NodeSummary{Name: "node-2"},
						Allocatable: cluster.NodeResources{CPU: "2", Memory: "4Gi"},
						Metrics: &cluster.NodeResourceUsage{
							CPU:    cluster.NodeResourceMetric{Usage: "1800m"}, // 1800m / 2000m = 90%
							Memory: cluster.NodeResourceMetric{Usage: "3Gi"},   // 3Gi / 4Gi = 75%
						},
					},
				},
				Pods: []cluster.Pod{
					{Summary: cluster.PodSummary{Name: "p1"}, Status: cluster.PodStatus{Phase: "Running"}},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	overview, err := svc.GetOverview(context.Background(), "cluster-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// PeakStats
	peak := overview.Workloads.PeakStats
	if peak == nil {
		t.Fatal("expected PeakStats, got nil")
	}
	if !peak.HasData {
		t.Error("expected HasData=true")
	}
	// Peak CPU: node-2 = 90%
	if peak.PeakCPUNode != "node-2" {
		t.Errorf("expected PeakCPUNode=node-2, got %s", peak.PeakCPUNode)
	}
	if peak.PeakCPU != 90.0 {
		t.Errorf("expected PeakCPU=90, got %f", peak.PeakCPU)
	}
	// Peak Mem: node-2 = 75%
	if peak.PeakMemNode != "node-2" {
		t.Errorf("expected PeakMemNode=node-2, got %s", peak.PeakMemNode)
	}
	if peak.PeakMem != 75.0 {
		t.Errorf("expected PeakMem=75, got %f", peak.PeakMem)
	}

	// NodeUsage: 2 个节点都有 Metrics → 2 条记录
	nodeUsages := overview.Nodes.Usage
	if len(nodeUsages) != 2 {
		t.Fatalf("expected 2 node usages, got %d", len(nodeUsages))
	}

	// Cluster 总使用率
	// CPU: (2000+1800) / (4000+2000) = 3800/6000 ≈ 63.33%
	// Mem: (2Gi+3Gi) / (8Gi+4Gi) = 5/12 ≈ 41.67%
	cards := overview.Cards
	expectedCPU := float64(2000+1800) / float64(4000+2000) * 100
	if cards.CPUUsage.Percent != expectedCPU {
		t.Errorf("expected CPUUsage=%f, got %f", expectedCPU, cards.CPUUsage.Percent)
	}
}

func TestGetOverview_NoMetrics(t *testing.T) {
	// 节点没有 Metrics 数据
	store := &mockStoreForOverview{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				Summary: cluster.ClusterSummary{TotalNodes: 1, ReadyNodes: 1, TotalPods: 1, RunningPods: 1},
				Nodes: []cluster.Node{
					{
						Summary:     cluster.NodeSummary{Name: "node-1"},
						Allocatable: cluster.NodeResources{CPU: "4", Memory: "8Gi"},
						// Metrics = nil
					},
				},
				Pods: []cluster.Pod{
					{Summary: cluster.PodSummary{Name: "p1"}, Status: cluster.PodStatus{Phase: "Running"}},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	overview, err := svc.GetOverview(context.Background(), "cluster-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// PeakStats.HasData = false
	peak := overview.Workloads.PeakStats
	if peak == nil {
		t.Fatal("expected PeakStats, got nil")
	}
	if peak.HasData {
		t.Error("expected HasData=false when no metrics")
	}

	// NodeUsage 为空
	if len(overview.Nodes.Usage) != 0 {
		t.Errorf("expected 0 node usages, got %d", len(overview.Nodes.Usage))
	}

	// CPU/Mem 使用率 = 0
	if overview.Cards.CPUUsage.Percent != 0 {
		t.Errorf("expected CPUUsage=0, got %f", overview.Cards.CPUUsage.Percent)
	}
	if overview.Cards.MemUsage.Percent != 0 {
		t.Errorf("expected MemUsage=0, got %f", overview.Cards.MemUsage.Percent)
	}
}

func TestGetOverview_AlertsFromDB(t *testing.T) {
	now := time.Now()
	currentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

	// 构造 hourly kind counts，使用当前小时的 key 以确保命中 hourToIndex
	hourKey := currentHour.Format("2006-01-02T15")

	eventRepo := &mockEventRepoForOverview{
		hourlyKindCounts: []database.HourlyKindCount{
			{Hour: hourKey, Kind: "Pod", Count: 5},
			{Hour: hourKey, Kind: "Node", Count: 2},
		},
		clusterEvents: []*database.ClusterEvent{
			{
				ClusterID:         "cluster-1",
				InvolvedKind:      "Pod",
				InvolvedNamespace: "default",
				InvolvedName:      "nginx-abc",
				Message:           "Back-off restarting",
				Reason:            "BackOff",
				LastTimestamp:     now.Add(-10 * time.Minute),
			},
		},
	}

	store := &mockStoreForOverview{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				Summary: cluster.ClusterSummary{TotalNodes: 1, ReadyNodes: 1, TotalPods: 1, RunningPods: 1},
				Nodes:   []cluster.Node{{Summary: cluster.NodeSummary{Name: "n1"}, Allocatable: cluster.NodeResources{CPU: "2", Memory: "4Gi"}}},
				Pods:    []cluster.Pod{{Summary: cluster.PodSummary{Name: "p1"}, Status: cluster.PodStatus{Phase: "Running"}}},
			},
		},
	}
	svc := &QueryService{store: store, eventRepo: eventRepo}

	overview, err := svc.GetOverview(context.Background(), "cluster-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// AlertTrend 应有 24 个点
	if len(overview.Alerts.Trend) != 24 {
		t.Fatalf("expected 24 trend points, got %d", len(overview.Alerts.Trend))
	}

	// Events24h = 5 + 2 = 7
	if overview.Cards.Events24h != 7 {
		t.Errorf("expected Events24h=7, got %d", overview.Cards.Events24h)
	}

	// Totals.Warning = 7（全部告警数）
	if overview.Alerts.Totals.Warning != 7 {
		t.Errorf("expected Totals.Warning=7, got %d", overview.Alerts.Totals.Warning)
	}

	// RecentAlerts 应有 1 条
	if len(overview.Alerts.Recent) != 1 {
		t.Fatalf("expected 1 recent alert, got %d", len(overview.Alerts.Recent))
	}
	alert := overview.Alerts.Recent[0]
	if alert.Kind != "Pod" {
		t.Errorf("expected Kind=Pod, got %s", alert.Kind)
	}
	if alert.Severity != "warning" {
		t.Errorf("expected Severity=warning, got %s", alert.Severity)
	}
	if alert.Name != "nginx-abc" {
		t.Errorf("expected Name=nginx-abc, got %s", alert.Name)
	}
	if alert.Reason != "BackOff" {
		t.Errorf("expected Reason=BackOff, got %s", alert.Reason)
	}
}

func TestGetOverview_NilEventRepo(t *testing.T) {
	store := &mockStoreForOverview{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				Summary: cluster.ClusterSummary{TotalNodes: 1, ReadyNodes: 1, TotalPods: 1, RunningPods: 1},
				Nodes:   []cluster.Node{{Summary: cluster.NodeSummary{Name: "n1"}, Allocatable: cluster.NodeResources{CPU: "2", Memory: "4Gi"}}},
				Pods:    []cluster.Pod{{Summary: cluster.PodSummary{Name: "p1"}, Status: cluster.PodStatus{Phase: "Running"}}},
			},
		},
	}
	// eventRepo = nil
	svc := &QueryService{store: store, eventRepo: nil}

	overview, err := svc.GetOverview(context.Background(), "cluster-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overview == nil {
		t.Fatal("expected overview, got nil")
	}

	// 不 panic，告警相关字段为零值
	if overview.Cards.Events24h != 0 {
		t.Errorf("expected Events24h=0 with nil eventRepo, got %d", overview.Cards.Events24h)
	}
	if len(overview.Alerts.Recent) != 0 {
		t.Errorf("expected 0 recent alerts with nil eventRepo, got %d", len(overview.Alerts.Recent))
	}
	if overview.Alerts.Totals.Warning != 0 {
		t.Errorf("expected Totals.Warning=0, got %d", overview.Alerts.Totals.Warning)
	}
	// AlertTrend 仍有 24 个点（空时间线）
	if len(overview.Alerts.Trend) != 24 {
		t.Errorf("expected 24 trend points even with nil eventRepo, got %d", len(overview.Alerts.Trend))
	}
}

// ==================== OTel 查询测试（otel.go） ====================

func TestGetOTelSnapshot_Found(t *testing.T) {
	otel := &cluster.OTelSnapshot{
		TotalServices:   5,
		HealthyServices: 4,
		TotalRPS:        120.5,
	}
	store := &mockStoreForOverview{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {OTel: otel},
		},
	}
	svc := &QueryService{store: store}

	result, err := svc.GetOTelSnapshot(context.Background(), "cluster-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected OTelSnapshot, got nil")
	}
	if result != otel {
		t.Error("expected exact same OTelSnapshot pointer from snapshot")
	}
	if result.TotalServices != 5 {
		t.Errorf("expected TotalServices=5, got %d", result.TotalServices)
	}
}

func TestGetOTelSnapshot_NoOTel(t *testing.T) {
	store := &mockStoreForOverview{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				// OTel = nil
				Nodes: []cluster.Node{{Summary: cluster.NodeSummary{Name: "n1"}}},
			},
		},
	}
	svc := &QueryService{store: store}

	result, err := svc.GetOTelSnapshot(context.Background(), "cluster-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil when OTel is nil, got %+v", result)
	}
}

func TestGetOTelSnapshot_NoSnapshot(t *testing.T) {
	store := &mockStoreForOverview{snapshots: map[string]*cluster.ClusterSnapshot{}}
	svc := &QueryService{store: store}

	result, err := svc.GetOTelSnapshot(context.Background(), "unknown-cluster")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for no-snapshot, got %+v", result)
	}
}

func TestGetOTelTimeline_Delegate(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	entries := []cluster.OTelEntry{
		{
			Timestamp: now,
			Snapshot:  &cluster.OTelSnapshot{TotalServices: 3},
		},
		{
			Timestamp: now.Add(-5 * time.Minute),
			Snapshot:  &cluster.OTelSnapshot{TotalServices: 2},
		},
	}
	store := &mockStoreForOverview{otelTimeline: entries}
	svc := &QueryService{store: store}

	result, err := svc.GetOTelTimeline(context.Background(), "cluster-1", now.Add(-10*time.Minute))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	if result[0].Snapshot.TotalServices != 3 {
		t.Errorf("expected first entry TotalServices=3, got %d", result[0].Snapshot.TotalServices)
	}
}

func TestGetOTelTimeline_Empty(t *testing.T) {
	store := &mockStoreForOverview{otelTimeline: nil}
	svc := &QueryService{store: store}

	result, err := svc.GetOTelTimeline(context.Background(), "cluster-1", time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 透传 store 返回的 nil
	if result != nil {
		t.Errorf("expected nil for empty timeline, got %d entries", len(result))
	}
}

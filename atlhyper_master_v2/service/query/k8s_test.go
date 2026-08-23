package query

import (
	"context"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/model"
	model_v3 "AtlHyper/model_v3"
	agentmodel "AtlHyper/model_v3/agent"
	"AtlHyper/model_v3/cluster"
)

// ==================== Mock: datahub.Store (k8s 专用，最小实现) ====================

type mockStoreForK8s struct {
	snapshots map[string]*cluster.ClusterSnapshot
}

func (m *mockStoreForK8s) SetSnapshot(clusterID string, snapshot *cluster.ClusterSnapshot) error {
	return nil
}
func (m *mockStoreForK8s) GetSnapshot(clusterID string) (*cluster.ClusterSnapshot, error) {
	if m.snapshots != nil {
		return m.snapshots[clusterID], nil
	}
	return nil, nil
}
func (m *mockStoreForK8s) UpdateHeartbeat(clusterID string) error { return nil }
func (m *mockStoreForK8s) GetAgentStatus(clusterID string) (*agentmodel.AgentStatus, error) {
	return nil, nil
}
func (m *mockStoreForK8s) ListAgents() ([]agentmodel.AgentInfo, error) { return nil, nil }
func (m *mockStoreForK8s) GetEvents(clusterID string) ([]cluster.Event, error) {
	return nil, nil
}
func (m *mockStoreForK8s) GetOTelTimeline(clusterID string, since time.Time) ([]cluster.OTelEntry, error) {
	return nil, nil
}
func (m *mockStoreForK8s) Start() error { return nil }
func (m *mockStoreForK8s) Stop() error  { return nil }

// ==================== Phase 1: 透传方法测试 ====================

func TestGetSnapshot_Delegate(t *testing.T) {
	snapshot := &cluster.ClusterSnapshot{
		ClusterID: "cluster-1",
		Nodes:     []cluster.Node{{Summary: cluster.NodeSummary{Name: "node-1"}}},
	}
	store := &mockStoreForK8s{
		snapshots: map[string]*cluster.ClusterSnapshot{"cluster-1": snapshot},
	}
	svc := &QueryService{store: store}

	result, err := svc.GetSnapshot(context.Background(), "cluster-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != snapshot {
		t.Error("expected exact same snapshot pointer (transparent delegation)")
	}
}

func TestGetSnapshot_NoData(t *testing.T) {
	store := &mockStoreForK8s{snapshots: map[string]*cluster.ClusterSnapshot{}}
	svc := &QueryService{store: store}

	result, err := svc.GetSnapshot(context.Background(), "unknown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %+v", result)
	}
}

func TestGetNodes_Found(t *testing.T) {
	store := &mockStoreForK8s{
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

	result, err := svc.GetNodes(context.Background(), "cluster-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(result))
	}
	if result[0].GetName() != "node-1" {
		t.Errorf("expected node-1, got %s", result[0].GetName())
	}
}

func TestGetNodes_NoSnapshot(t *testing.T) {
	store := &mockStoreForK8s{snapshots: map[string]*cluster.ClusterSnapshot{}}
	svc := &QueryService{store: store}

	result, err := svc.GetNodes(context.Background(), "unknown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %+v", result)
	}
}

func TestGetNamespaces_Found(t *testing.T) {
	store := &mockStoreForK8s{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				Namespaces: []cluster.Namespace{
					{Summary: cluster.NamespaceSummary{Name: "default"}},
					{Summary: cluster.NamespaceSummary{Name: "kube-system"}},
					{Summary: cluster.NamespaceSummary{Name: "monitoring"}},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	result, err := svc.GetNamespaces(context.Background(), "cluster-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 namespaces, got %d", len(result))
	}
}

func TestGetNamespaces_NoSnapshot(t *testing.T) {
	store := &mockStoreForK8s{snapshots: map[string]*cluster.ClusterSnapshot{}}
	svc := &QueryService{store: store}

	result, err := svc.GetNamespaces(context.Background(), "unknown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %+v", result)
	}
}

func TestGetPersistentVolumes_Found(t *testing.T) {
	store := &mockStoreForK8s{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				PersistentVolumes: []cluster.PersistentVolume{
					{CommonMeta: model_v3.CommonMeta{Name: "pv-1"}, Capacity: "10Gi", Phase: "Bound"},
					{CommonMeta: model_v3.CommonMeta{Name: "pv-2"}, Capacity: "20Gi", Phase: "Available"},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	result, err := svc.GetPersistentVolumes(context.Background(), "cluster-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 PVs, got %d", len(result))
	}
	if result[0].Name != "pv-1" {
		t.Errorf("expected pv-1, got %s", result[0].Name)
	}
}

func TestGetPersistentVolumes_NoSnapshot(t *testing.T) {
	store := &mockStoreForK8s{snapshots: map[string]*cluster.ClusterSnapshot{}}
	svc := &QueryService{store: store}

	result, err := svc.GetPersistentVolumes(context.Background(), "unknown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %+v", result)
	}
}

// ==================== Phase 2: GetPods 复杂查询测试 ====================

// testPods 构造 5 个测试用 Pod，覆盖不同 namespace/node/phase/metrics 组合
func testPods() []cluster.Pod {
	return []cluster.Pod{
		{
			Summary: cluster.PodSummary{Name: "pod-1", Namespace: "default", NodeName: "node-a"},
			Status:  cluster.PodStatus{Phase: "Running", CPUUsage: "100000000n", MemoryUsage: "2097152Ki"},
		},
		{
			Summary: cluster.PodSummary{Name: "pod-2", Namespace: "default", NodeName: "node-b"},
			Status:  cluster.PodStatus{Phase: "Running", CPUUsage: "2500m", MemoryUsage: "128Mi"},
		},
		{
			Summary: cluster.PodSummary{Name: "pod-3", Namespace: "kube-system", NodeName: "node-a"},
			Status:  cluster.PodStatus{Phase: "Running", CPUUsage: "50m", MemoryUsage: "64Mi"},
		},
		{
			Summary: cluster.PodSummary{Name: "pod-4", Namespace: "kube-system", NodeName: "node-b"},
			Status:  cluster.PodStatus{Phase: "Pending", CPUUsage: "", MemoryUsage: ""},
		},
		{
			Summary: cluster.PodSummary{Name: "pod-5", Namespace: "monitoring", NodeName: "node-a"},
			Status:  cluster.PodStatus{Phase: "Failed", CPUUsage: "1500000000n", MemoryUsage: "2048Mi"},
		},
	}
}

func newK8sSvcWithPods() *QueryService {
	store := &mockStoreForK8s{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {Pods: testPods()},
		},
	}
	return &QueryService{store: store}
}

func TestGetPods_NoFilter(t *testing.T) {
	svc := newK8sSvcWithPods()

	result, err := svc.GetPods(context.Background(), "cluster-1", model.PodQueryOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 5 {
		t.Fatalf("expected 5 pods, got %d", len(result))
	}
}

func TestGetPods_NamespaceFilter(t *testing.T) {
	svc := newK8sSvcWithPods()

	result, err := svc.GetPods(context.Background(), "cluster-1", model.PodQueryOpts{Namespace: "kube-system"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 pods in kube-system, got %d", len(result))
	}
	for _, p := range result {
		if p.GetNamespace() != "kube-system" {
			t.Errorf("expected namespace kube-system, got %s for pod %s", p.GetNamespace(), p.Summary.Name)
		}
	}
}

func TestGetPods_NodeNameFilter(t *testing.T) {
	svc := newK8sSvcWithPods()

	result, err := svc.GetPods(context.Background(), "cluster-1", model.PodQueryOpts{NodeName: "node-a"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// pod-1(default/node-a), pod-3(kube-system/node-a), pod-5(monitoring/node-a)
	if len(result) != 3 {
		t.Fatalf("expected 3 pods on node-a, got %d", len(result))
	}
	for _, p := range result {
		if p.GetNodeName() != "node-a" {
			t.Errorf("expected nodeName node-a, got %s for pod %s", p.GetNodeName(), p.Summary.Name)
		}
	}
}

func TestGetPods_PhaseFilter(t *testing.T) {
	svc := newK8sSvcWithPods()

	result, err := svc.GetPods(context.Background(), "cluster-1", model.PodQueryOpts{Phase: "Running"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// pod-1, pod-2, pod-3 are Running
	if len(result) != 3 {
		t.Fatalf("expected 3 Running pods, got %d", len(result))
	}
	for _, p := range result {
		if p.Status.Phase != "Running" {
			t.Errorf("expected phase Running, got %s for pod %s", p.Status.Phase, p.Summary.Name)
		}
	}
}

func TestGetPods_CombinedFilter(t *testing.T) {
	svc := newK8sSvcWithPods()

	// namespace=default AND nodeName=node-a → only pod-1
	result, err := svc.GetPods(context.Background(), "cluster-1", model.PodQueryOpts{
		Namespace: "default",
		NodeName:  "node-a",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 pod (default+node-a), got %d", len(result))
	}
	if result[0].Summary.Name != "pod-1" {
		t.Errorf("expected pod-1, got %s", result[0].Summary.Name)
	}

	// namespace=kube-system AND phase=Pending → only pod-4
	result2, err := svc.GetPods(context.Background(), "cluster-1", model.PodQueryOpts{
		Namespace: "kube-system",
		Phase:     "Pending",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result2) != 1 {
		t.Fatalf("expected 1 pod (kube-system+Pending), got %d", len(result2))
	}
	if result2[0].Summary.Name != "pod-4" {
		t.Errorf("expected pod-4, got %s", result2[0].Summary.Name)
	}
}

func TestGetPods_Pagination(t *testing.T) {
	svc := newK8sSvcWithPods()

	// Limit=2 → first 2 pods
	result, err := svc.GetPods(context.Background(), "cluster-1", model.PodQueryOpts{Limit: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 pods with limit=2, got %d", len(result))
	}

	// Offset=2, Limit=2 → pods at index 2,3
	result2, err := svc.GetPods(context.Background(), "cluster-1", model.PodQueryOpts{Offset: 2, Limit: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result2) != 2 {
		t.Fatalf("expected 2 pods with offset=2,limit=2, got %d", len(result2))
	}

	// Offset=4 → last 1 pod
	result3, err := svc.GetPods(context.Background(), "cluster-1", model.PodQueryOpts{Offset: 4})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result3) != 1 {
		t.Fatalf("expected 1 pod with offset=4, got %d", len(result3))
	}
}

func TestGetPods_MetricsFormat(t *testing.T) {
	svc := newK8sSvcWithPods()

	result, err := svc.GetPods(context.Background(), "cluster-1", model.PodQueryOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// pod-1: 100000000n → 100m, 2097152Ki → 2.00Gi
	if result[0].Status.CPUUsage != "100m" {
		t.Errorf("pod-1 CPU: expected 100m, got %s", result[0].Status.CPUUsage)
	}
	if result[0].Status.MemoryUsage != "2.00Gi" {
		t.Errorf("pod-1 Memory: expected 2.00Gi, got %s", result[0].Status.MemoryUsage)
	}

	// pod-2: 2500m → 2.50, 128Mi → 128Mi
	if result[1].Status.CPUUsage != "2.50" {
		t.Errorf("pod-2 CPU: expected 2.50, got %s", result[1].Status.CPUUsage)
	}
	if result[1].Status.MemoryUsage != "128Mi" {
		t.Errorf("pod-2 Memory: expected 128Mi, got %s", result[1].Status.MemoryUsage)
	}

	// pod-3: 50m → 50m, 64Mi → 64Mi
	if result[2].Status.CPUUsage != "50m" {
		t.Errorf("pod-3 CPU: expected 50m, got %s", result[2].Status.CPUUsage)
	}
	if result[2].Status.MemoryUsage != "64Mi" {
		t.Errorf("pod-3 Memory: expected 64Mi, got %s", result[2].Status.MemoryUsage)
	}

	// pod-4: "" → "", "" → ""
	if result[3].Status.CPUUsage != "" {
		t.Errorf("pod-4 CPU: expected empty, got %s", result[3].Status.CPUUsage)
	}
	if result[3].Status.MemoryUsage != "" {
		t.Errorf("pod-4 Memory: expected empty, got %s", result[3].Status.MemoryUsage)
	}

	// pod-5: 1500000000n → 1.50, 2048Mi → 2.00Gi
	if result[4].Status.CPUUsage != "1.50" {
		t.Errorf("pod-5 CPU: expected 1.50, got %s", result[4].Status.CPUUsage)
	}
	if result[4].Status.MemoryUsage != "2.00Gi" {
		t.Errorf("pod-5 Memory: expected 2.00Gi, got %s", result[4].Status.MemoryUsage)
	}
}

func TestGetPods_NoSnapshot(t *testing.T) {
	store := &mockStoreForK8s{snapshots: map[string]*cluster.ClusterSnapshot{}}
	svc := &QueryService{store: store}

	result, err := svc.GetPods(context.Background(), "unknown", model.PodQueryOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %+v", result)
	}
}

// ==================== Phase 3: Namespace 过滤方法测试 ====================

// --- 代表性深度测试: GetDeployments (Summary.GetNamespace() 模式) ---

func TestGetDeployments_All(t *testing.T) {
	store := &mockStoreForK8s{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				Deployments: []cluster.Deployment{
					{Summary: cluster.DeploymentSummary{Name: "deploy-1", Namespace: "default"}},
					{Summary: cluster.DeploymentSummary{Name: "deploy-2", Namespace: "kube-system"}},
					{Summary: cluster.DeploymentSummary{Name: "deploy-3", Namespace: "default"}},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	result, err := svc.GetDeployments(context.Background(), "cluster-1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 deployments, got %d", len(result))
	}
}

func TestGetDeployments_Filtered(t *testing.T) {
	store := &mockStoreForK8s{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				Deployments: []cluster.Deployment{
					{Summary: cluster.DeploymentSummary{Name: "deploy-1", Namespace: "default"}},
					{Summary: cluster.DeploymentSummary{Name: "deploy-2", Namespace: "kube-system"}},
					{Summary: cluster.DeploymentSummary{Name: "deploy-3", Namespace: "default"}},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	result, err := svc.GetDeployments(context.Background(), "cluster-1", "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 deployments in default, got %d", len(result))
	}
	for _, d := range result {
		if d.GetNamespace() != "default" {
			t.Errorf("expected namespace default, got %s", d.GetNamespace())
		}
	}
}

func TestGetDeployments_NoSnapshot(t *testing.T) {
	store := &mockStoreForK8s{snapshots: map[string]*cluster.ClusterSnapshot{}}
	svc := &QueryService{store: store}

	result, err := svc.GetDeployments(context.Background(), "unknown", "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %+v", result)
	}
}

// --- 代表性深度测试: GetConfigMaps (CommonMeta.Namespace 模式) ---

func TestGetConfigMaps_All(t *testing.T) {
	store := &mockStoreForK8s{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				ConfigMaps: []cluster.ConfigMap{
					{CommonMeta: model_v3.CommonMeta{Name: "cm-1", Namespace: "default"}},
					{CommonMeta: model_v3.CommonMeta{Name: "cm-2", Namespace: "kube-system"}},
					{CommonMeta: model_v3.CommonMeta{Name: "cm-3", Namespace: "default"}},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	result, err := svc.GetConfigMaps(context.Background(), "cluster-1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 configmaps, got %d", len(result))
	}
}

func TestGetConfigMaps_Filtered(t *testing.T) {
	store := &mockStoreForK8s{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"cluster-1": {
				ConfigMaps: []cluster.ConfigMap{
					{CommonMeta: model_v3.CommonMeta{Name: "cm-1", Namespace: "default"}},
					{CommonMeta: model_v3.CommonMeta{Name: "cm-2", Namespace: "kube-system"}},
					{CommonMeta: model_v3.CommonMeta{Name: "cm-3", Namespace: "default"}},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	result, err := svc.GetConfigMaps(context.Background(), "cluster-1", "kube-system")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 configmap in kube-system, got %d", len(result))
	}
	if result[0].Name != "cm-2" {
		t.Errorf("expected cm-2, got %s", result[0].Name)
	}
}

func TestGetConfigMaps_NoSnapshot(t *testing.T) {
	store := &mockStoreForK8s{snapshots: map[string]*cluster.ClusterSnapshot{}}
	svc := &QueryService{store: store}

	result, err := svc.GetConfigMaps(context.Background(), "unknown", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %+v", result)
	}
}

// --- 烟雾测试: 剩余 12 个 namespace 过滤方法 ---

// testSnapshotForSmoke 构造包含所有资源类型的快照，每类 3 个（2 个 default + 1 个 kube-system）
func testSnapshotForSmoke() *cluster.ClusterSnapshot {
	return &cluster.ClusterSnapshot{
		// Pattern A: Summary.Namespace (GetNamespace())
		Services: []cluster.Service{
			{Summary: cluster.ServiceSummary{Name: "svc-1", Namespace: "default"}},
			{Summary: cluster.ServiceSummary{Name: "svc-2", Namespace: "kube-system"}},
			{Summary: cluster.ServiceSummary{Name: "svc-3", Namespace: "default"}},
		},
		Ingresses: []cluster.Ingress{
			{Summary: cluster.IngressSummary{Name: "ing-1", Namespace: "default"}},
			{Summary: cluster.IngressSummary{Name: "ing-2", Namespace: "kube-system"}},
			{Summary: cluster.IngressSummary{Name: "ing-3", Namespace: "default"}},
		},
		DaemonSets: []cluster.DaemonSet{
			{Summary: cluster.DaemonSetSummary{Name: "ds-1", Namespace: "default"}},
			{Summary: cluster.DaemonSetSummary{Name: "ds-2", Namespace: "kube-system"}},
			{Summary: cluster.DaemonSetSummary{Name: "ds-3", Namespace: "default"}},
		},
		StatefulSets: []cluster.StatefulSet{
			{Summary: cluster.StatefulSetSummary{Name: "sts-1", Namespace: "default"}},
			{Summary: cluster.StatefulSetSummary{Name: "sts-2", Namespace: "kube-system"}},
			{Summary: cluster.StatefulSetSummary{Name: "sts-3", Namespace: "default"}},
		},
		// Pattern B: CommonMeta.Namespace
		Secrets: []cluster.Secret{
			{CommonMeta: model_v3.CommonMeta{Name: "sec-1", Namespace: "default"}},
			{CommonMeta: model_v3.CommonMeta{Name: "sec-2", Namespace: "kube-system"}},
			{CommonMeta: model_v3.CommonMeta{Name: "sec-3", Namespace: "default"}},
		},
		Jobs: []cluster.Job{
			{CommonMeta: model_v3.CommonMeta{Name: "job-1", Namespace: "default"}},
			{CommonMeta: model_v3.CommonMeta{Name: "job-2", Namespace: "kube-system"}},
			{CommonMeta: model_v3.CommonMeta{Name: "job-3", Namespace: "default"}},
		},
		CronJobs: []cluster.CronJob{
			{CommonMeta: model_v3.CommonMeta{Name: "cj-1", Namespace: "default"}},
			{CommonMeta: model_v3.CommonMeta{Name: "cj-2", Namespace: "kube-system"}},
			{CommonMeta: model_v3.CommonMeta{Name: "cj-3", Namespace: "default"}},
		},
		PersistentVolumeClaims: []cluster.PersistentVolumeClaim{
			{CommonMeta: model_v3.CommonMeta{Name: "pvc-1", Namespace: "default"}},
			{CommonMeta: model_v3.CommonMeta{Name: "pvc-2", Namespace: "kube-system"}},
			{CommonMeta: model_v3.CommonMeta{Name: "pvc-3", Namespace: "default"}},
		},
		// Pattern C: 直接 .Namespace 字段
		NetworkPolicies: []cluster.NetworkPolicy{
			{Name: "np-1", Namespace: "default"},
			{Name: "np-2", Namespace: "kube-system"},
			{Name: "np-3", Namespace: "default"},
		},
		ResourceQuotas: []cluster.ResourceQuota{
			{Name: "rq-1", Namespace: "default"},
			{Name: "rq-2", Namespace: "kube-system"},
			{Name: "rq-3", Namespace: "default"},
		},
		LimitRanges: []cluster.LimitRange{
			{Name: "lr-1", Namespace: "default"},
			{Name: "lr-2", Namespace: "kube-system"},
			{Name: "lr-3", Namespace: "default"},
		},
		ServiceAccounts: []cluster.ServiceAccount{
			{Name: "sa-1", Namespace: "default"},
			{Name: "sa-2", Namespace: "kube-system"},
			{Name: "sa-3", Namespace: "default"},
		},
	}
}

func TestNamespaceFilter_SmokeAll(t *testing.T) {
	store := &mockStoreForK8s{
		snapshots: map[string]*cluster.ClusterSnapshot{"cluster-1": testSnapshotForSmoke()},
	}
	svc := &QueryService{store: store}
	ctx := context.Background()

	tests := []struct {
		name string
		call func() (int, error)
	}{
		{"Services", func() (int, error) {
			r, e := svc.GetServices(ctx, "cluster-1", "")
			return len(r), e
		}},
		{"Ingresses", func() (int, error) {
			r, e := svc.GetIngresses(ctx, "cluster-1", "")
			return len(r), e
		}},
		{"DaemonSets", func() (int, error) {
			r, e := svc.GetDaemonSets(ctx, "cluster-1", "")
			return len(r), e
		}},
		{"StatefulSets", func() (int, error) {
			r, e := svc.GetStatefulSets(ctx, "cluster-1", "")
			return len(r), e
		}},
		{"Secrets", func() (int, error) {
			r, e := svc.GetSecrets(ctx, "cluster-1", "")
			return len(r), e
		}},
		{"Jobs", func() (int, error) {
			r, e := svc.GetJobs(ctx, "cluster-1", "")
			return len(r), e
		}},
		{"CronJobs", func() (int, error) {
			r, e := svc.GetCronJobs(ctx, "cluster-1", "")
			return len(r), e
		}},
		{"PersistentVolumeClaims", func() (int, error) {
			r, e := svc.GetPersistentVolumeClaims(ctx, "cluster-1", "")
			return len(r), e
		}},
		{"NetworkPolicies", func() (int, error) {
			r, e := svc.GetNetworkPolicies(ctx, "cluster-1", "")
			return len(r), e
		}},
		{"ResourceQuotas", func() (int, error) {
			r, e := svc.GetResourceQuotas(ctx, "cluster-1", "")
			return len(r), e
		}},
		{"LimitRanges", func() (int, error) {
			r, e := svc.GetLimitRanges(ctx, "cluster-1", "")
			return len(r), e
		}},
		{"ServiceAccounts", func() (int, error) {
			r, e := svc.GetServiceAccounts(ctx, "cluster-1", "")
			return len(r), e
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := tt.call()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if count != 3 {
				t.Errorf("expected 3, got %d", count)
			}
		})
	}
}

func TestNamespaceFilter_SmokeFiltered(t *testing.T) {
	store := &mockStoreForK8s{
		snapshots: map[string]*cluster.ClusterSnapshot{"cluster-1": testSnapshotForSmoke()},
	}
	svc := &QueryService{store: store}
	ctx := context.Background()

	tests := []struct {
		name    string
		call    func() (int, error)
		wantAll int // namespace="default" 的数量
	}{
		{"Services", func() (int, error) {
			r, e := svc.GetServices(ctx, "cluster-1", "default")
			return len(r), e
		}, 2},
		{"Ingresses", func() (int, error) {
			r, e := svc.GetIngresses(ctx, "cluster-1", "default")
			return len(r), e
		}, 2},
		{"DaemonSets", func() (int, error) {
			r, e := svc.GetDaemonSets(ctx, "cluster-1", "default")
			return len(r), e
		}, 2},
		{"StatefulSets", func() (int, error) {
			r, e := svc.GetStatefulSets(ctx, "cluster-1", "default")
			return len(r), e
		}, 2},
		{"Secrets", func() (int, error) {
			r, e := svc.GetSecrets(ctx, "cluster-1", "default")
			return len(r), e
		}, 2},
		{"Jobs", func() (int, error) {
			r, e := svc.GetJobs(ctx, "cluster-1", "default")
			return len(r), e
		}, 2},
		{"CronJobs", func() (int, error) {
			r, e := svc.GetCronJobs(ctx, "cluster-1", "default")
			return len(r), e
		}, 2},
		{"PersistentVolumeClaims", func() (int, error) {
			r, e := svc.GetPersistentVolumeClaims(ctx, "cluster-1", "default")
			return len(r), e
		}, 2},
		{"NetworkPolicies", func() (int, error) {
			r, e := svc.GetNetworkPolicies(ctx, "cluster-1", "default")
			return len(r), e
		}, 2},
		{"ResourceQuotas", func() (int, error) {
			r, e := svc.GetResourceQuotas(ctx, "cluster-1", "default")
			return len(r), e
		}, 2},
		{"LimitRanges", func() (int, error) {
			r, e := svc.GetLimitRanges(ctx, "cluster-1", "default")
			return len(r), e
		}, 2},
		{"ServiceAccounts", func() (int, error) {
			r, e := svc.GetServiceAccounts(ctx, "cluster-1", "default")
			return len(r), e
		}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := tt.call()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if count != tt.wantAll {
				t.Errorf("expected %d in default, got %d", tt.wantAll, count)
			}
		})
	}
}

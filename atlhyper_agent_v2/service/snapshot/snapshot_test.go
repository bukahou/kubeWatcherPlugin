package snapshot

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"AtlHyper/atlhyper_agent_v2/config"
	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/testutil/mock"
	model_v3 "AtlHyper/model_v3"
	"AtlHyper/model_v3/cluster"
)

// newTestService creates a snapshotService with all mocks wired up.
// Pass non-nil repos to override defaults; nil repos get empty mocks.
func newTestService(opts ...func(*snapshotService)) *snapshotService {
	svc := &snapshotService{
		clusterID:          "test-cluster",
		podRepo:            &mock.PodRepository{},
		nodeRepo:           &mock.NodeRepository{},
		deploymentRepo:     &mock.DeploymentRepository{},
		statefulSetRepo:    &mock.StatefulSetRepository{},
		daemonSetRepo:      &mock.DaemonSetRepository{},
		replicaSetRepo:     &mock.ReplicaSetRepository{},
		serviceRepo:        &mock.ServiceRepository{},
		ingressRepo:        &mock.IngressRepository{},
		configMapRepo:      &mock.ConfigMapRepository{},
		secretRepo:         &mock.SecretRepository{},
		namespaceRepo:      &mock.NamespaceRepository{},
		eventRepo:          &mock.EventRepository{},
		jobRepo:            &mock.JobRepository{},
		cronJobRepo:        &mock.CronJobRepository{},
		pvRepo:             &mock.PersistentVolumeRepository{},
		pvcRepo:            &mock.PersistentVolumeClaimRepository{},
		resourceQuotaRepo:  &mock.ResourceQuotaRepository{},
		limitRangeRepo:     &mock.LimitRangeRepository{},
		networkPolicyRepo:  &mock.NetworkPolicyRepository{},
		serviceAccountRepo: &mock.ServiceAccountRepository{},
	}
	for _, fn := range opts {
		fn(svc)
	}
	return svc
}

// ---------------------------------------------------------------------------
// TestCollect_AllResourcesPopulated
// ---------------------------------------------------------------------------

func TestCollect_AllResourcesPopulated(t *testing.T) {
	podRepo := &mock.PodRepository{
		ListFn: func(_ context.Context, _ string, _ model.ListOptions) ([]cluster.Pod, error) {
			return []cluster.Pod{{Summary: cluster.PodSummary{Name: "pod-1", Namespace: "default"}}}, nil
		},
	}
	nodeRepo := &mock.NodeRepository{
		ListFn: func(_ context.Context, _ model.ListOptions) ([]cluster.Node, error) {
			return []cluster.Node{{Summary: cluster.NodeSummary{Name: "node-1"}}}, nil
		},
	}
	deploymentRepo := &mock.DeploymentRepository{
		ListFn: func(_ context.Context, _ string, _ model.ListOptions) ([]cluster.Deployment, error) {
			return []cluster.Deployment{{Summary: cluster.DeploymentSummary{Name: "deploy-1", Namespace: "default"}}}, nil
		},
	}
	statefulSetRepo := &mock.StatefulSetRepository{
		ListFn: func(_ context.Context, _ string, _ model.ListOptions) ([]cluster.StatefulSet, error) {
			return []cluster.StatefulSet{{Summary: cluster.StatefulSetSummary{Name: "sts-1", Namespace: "default"}}}, nil
		},
	}
	daemonSetRepo := &mock.DaemonSetRepository{
		ListFn: func(_ context.Context, _ string, _ model.ListOptions) ([]cluster.DaemonSet, error) {
			return []cluster.DaemonSet{{Summary: cluster.DaemonSetSummary{Name: "ds-1", Namespace: "default"}}}, nil
		},
	}
	replicaSetRepo := &mock.ReplicaSetRepository{
		ListFn: func(_ context.Context, _ string, _ model.ListOptions) ([]cluster.ReplicaSet, error) {
			return []cluster.ReplicaSet{{CommonMeta: commonMeta("rs-1", "default")}}, nil
		},
	}
	serviceRepo := &mock.ServiceRepository{
		ListFn: func(_ context.Context, _ string, _ model.ListOptions) ([]cluster.Service, error) {
			return []cluster.Service{{Summary: cluster.ServiceSummary{Name: "svc-1", Namespace: "default"}}}, nil
		},
	}
	ingressRepo := &mock.IngressRepository{
		ListFn: func(_ context.Context, _ string, _ model.ListOptions) ([]cluster.Ingress, error) {
			return []cluster.Ingress{{Summary: cluster.IngressSummary{Name: "ing-1", Namespace: "default"}}}, nil
		},
	}
	configMapRepo := &mock.ConfigMapRepository{
		ListFn: func(_ context.Context, _ string, _ model.ListOptions) ([]cluster.ConfigMap, error) {
			return []cluster.ConfigMap{{CommonMeta: commonMeta("cm-1", "default")}}, nil
		},
	}
	secretRepo := &mock.SecretRepository{
		ListFn: func(_ context.Context, _ string, _ model.ListOptions) ([]cluster.Secret, error) {
			return []cluster.Secret{{CommonMeta: commonMeta("sec-1", "default")}}, nil
		},
	}
	namespaceRepo := &mock.NamespaceRepository{
		ListFn: func(_ context.Context, _ model.ListOptions) ([]cluster.Namespace, error) {
			return []cluster.Namespace{{Summary: cluster.NamespaceSummary{Name: "default"}}}, nil
		},
	}
	eventRepo := &mock.EventRepository{
		ListFn: func(_ context.Context, _ string, _ model.ListOptions) ([]cluster.Event, error) {
			return []cluster.Event{{CommonMeta: commonMeta("evt-1", "default"), Type: "Normal"}}, nil
		},
	}
	jobRepo := &mock.JobRepository{
		ListFn: func(_ context.Context, _ string, _ model.ListOptions) ([]cluster.Job, error) {
			return []cluster.Job{{CommonMeta: commonMeta("job-1", "default")}}, nil
		},
	}
	cronJobRepo := &mock.CronJobRepository{
		ListFn: func(_ context.Context, _ string, _ model.ListOptions) ([]cluster.CronJob, error) {
			return []cluster.CronJob{{CommonMeta: commonMeta("cj-1", "default")}}, nil
		},
	}
	pvRepo := &mock.PersistentVolumeRepository{
		ListFn: func(_ context.Context, _ model.ListOptions) ([]cluster.PersistentVolume, error) {
			return []cluster.PersistentVolume{{CommonMeta: commonMeta("pv-1", "")}}, nil
		},
	}
	pvcRepo := &mock.PersistentVolumeClaimRepository{
		ListFn: func(_ context.Context, _ string, _ model.ListOptions) ([]cluster.PersistentVolumeClaim, error) {
			return []cluster.PersistentVolumeClaim{{CommonMeta: commonMeta("pvc-1", "default")}}, nil
		},
	}
	rqRepo := &mock.ResourceQuotaRepository{
		ListFn: func(_ context.Context, _ string, _ model.ListOptions) ([]cluster.ResourceQuota, error) {
			return []cluster.ResourceQuota{{Name: "rq-1", Namespace: "default"}}, nil
		},
	}
	lrRepo := &mock.LimitRangeRepository{
		ListFn: func(_ context.Context, _ string, _ model.ListOptions) ([]cluster.LimitRange, error) {
			return []cluster.LimitRange{{Name: "lr-1", Namespace: "default"}}, nil
		},
	}
	npRepo := &mock.NetworkPolicyRepository{
		ListFn: func(_ context.Context, _ string, _ model.ListOptions) ([]cluster.NetworkPolicy, error) {
			return []cluster.NetworkPolicy{{Name: "np-1", Namespace: "default"}}, nil
		},
	}
	saRepo := &mock.ServiceAccountRepository{
		ListFn: func(_ context.Context, _ string, _ model.ListOptions) ([]cluster.ServiceAccount, error) {
			return []cluster.ServiceAccount{{Name: "sa-1", Namespace: "default"}}, nil
		},
	}

	svc := newTestService(func(s *snapshotService) {
		s.podRepo = podRepo
		s.nodeRepo = nodeRepo
		s.deploymentRepo = deploymentRepo
		s.statefulSetRepo = statefulSetRepo
		s.daemonSetRepo = daemonSetRepo
		s.replicaSetRepo = replicaSetRepo
		s.serviceRepo = serviceRepo
		s.ingressRepo = ingressRepo
		s.configMapRepo = configMapRepo
		s.secretRepo = secretRepo
		s.namespaceRepo = namespaceRepo
		s.eventRepo = eventRepo
		s.jobRepo = jobRepo
		s.cronJobRepo = cronJobRepo
		s.pvRepo = pvRepo
		s.pvcRepo = pvcRepo
		s.resourceQuotaRepo = rqRepo
		s.limitRangeRepo = lrRepo
		s.networkPolicyRepo = npRepo
		s.serviceAccountRepo = saRepo
	})

	snapshot, err := svc.Collect(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if snapshot.ClusterID != "test-cluster" {
		t.Errorf("ClusterID = %q, want %q", snapshot.ClusterID, "test-cluster")
	}
	if time.Since(snapshot.FetchedAt) > 5*time.Second {
		t.Errorf("FetchedAt is too old: %v", snapshot.FetchedAt)
	}

	// Assert all 20 resource slices are populated
	assertions := []struct {
		name  string
		count int
	}{
		{"Pods", len(snapshot.Pods)},
		{"Nodes", len(snapshot.Nodes)},
		{"Deployments", len(snapshot.Deployments)},
		{"StatefulSets", len(snapshot.StatefulSets)},
		{"DaemonSets", len(snapshot.DaemonSets)},
		{"ReplicaSets", len(snapshot.ReplicaSets)},
		{"Services", len(snapshot.Services)},
		{"Ingresses", len(snapshot.Ingresses)},
		{"ConfigMaps", len(snapshot.ConfigMaps)},
		{"Secrets", len(snapshot.Secrets)},
		{"Namespaces", len(snapshot.Namespaces)},
		{"Events", len(snapshot.Events)},
		{"Jobs", len(snapshot.Jobs)},
		{"CronJobs", len(snapshot.CronJobs)},
		{"PersistentVolumes", len(snapshot.PersistentVolumes)},
		{"PersistentVolumeClaims", len(snapshot.PersistentVolumeClaims)},
		{"ResourceQuotas", len(snapshot.ResourceQuotas)},
		{"LimitRanges", len(snapshot.LimitRanges)},
		{"NetworkPolicies", len(snapshot.NetworkPolicies)},
		{"ServiceAccounts", len(snapshot.ServiceAccounts)},
	}
	for _, a := range assertions {
		if a.count == 0 {
			t.Errorf("%s should not be empty", a.name)
		}
	}
}

// ---------------------------------------------------------------------------
// TestCollect_PartialFailure
// ---------------------------------------------------------------------------

func TestCollect_PartialFailure(t *testing.T) {
	podRepo := &mock.PodRepository{
		ListFn: func(_ context.Context, _ string, _ model.ListOptions) ([]cluster.Pod, error) {
			return nil, errors.New("pod list failed")
		},
	}
	nodeRepo := &mock.NodeRepository{
		ListFn: func(_ context.Context, _ model.ListOptions) ([]cluster.Node, error) {
			return []cluster.Node{{Summary: cluster.NodeSummary{Name: "node-1"}}}, nil
		},
	}
	deploymentRepo := &mock.DeploymentRepository{
		ListFn: func(_ context.Context, _ string, _ model.ListOptions) ([]cluster.Deployment, error) {
			return []cluster.Deployment{{Summary: cluster.DeploymentSummary{Name: "deploy-1"}}}, nil
		},
	}

	svc := newTestService(func(s *snapshotService) {
		s.podRepo = podRepo
		s.nodeRepo = nodeRepo
		s.deploymentRepo = deploymentRepo
	})

	snapshot, err := svc.Collect(context.Background())
	if err == nil {
		t.Fatal("expected an error from partial failure, got nil")
	}
	if snapshot.Pods != nil {
		t.Errorf("Pods should be nil when pod repo fails, got %d items", len(snapshot.Pods))
	}
	if snapshot.Nodes == nil || len(snapshot.Nodes) == 0 {
		t.Error("Nodes should still be populated despite pod failure")
	}
	if snapshot.Deployments == nil || len(snapshot.Deployments) == 0 {
		t.Error("Deployments should still be populated despite pod failure")
	}
}

// ---------------------------------------------------------------------------
// TestCollect_AllEmpty
// ---------------------------------------------------------------------------

func TestCollect_AllEmpty(t *testing.T) {
	// All repos return empty slices (default mock behavior returns nil)
	// Override with explicit empty slices to test that path
	podRepo := &mock.PodRepository{
		ListFn: func(_ context.Context, _ string, _ model.ListOptions) ([]cluster.Pod, error) {
			return []cluster.Pod{}, nil
		},
	}
	nodeRepo := &mock.NodeRepository{
		ListFn: func(_ context.Context, _ model.ListOptions) ([]cluster.Node, error) {
			return []cluster.Node{}, nil
		},
	}
	namespaceRepo := &mock.NamespaceRepository{
		ListFn: func(_ context.Context, _ model.ListOptions) ([]cluster.Namespace, error) {
			return []cluster.Namespace{}, nil
		},
	}

	svc := newTestService(func(s *snapshotService) {
		s.podRepo = podRepo
		s.nodeRepo = nodeRepo
		s.namespaceRepo = namespaceRepo
	})

	snapshot, err := svc.Collect(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if snapshot.ClusterID != "test-cluster" {
		t.Errorf("ClusterID = %q, want %q", snapshot.ClusterID, "test-cluster")
	}
	if len(snapshot.Pods) != 0 {
		t.Errorf("expected 0 pods, got %d", len(snapshot.Pods))
	}
	if len(snapshot.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(snapshot.Nodes))
	}
}

// ---------------------------------------------------------------------------
// TestCalculateNamespaceResources_BasicCounting
// ---------------------------------------------------------------------------

func TestCalculateNamespaceResources_BasicCounting(t *testing.T) {
	svc := newTestService()

	snapshot := &cluster.ClusterSnapshot{
		Namespaces: []cluster.Namespace{
			{Summary: cluster.NamespaceSummary{Name: "default"}},
			{Summary: cluster.NamespaceSummary{Name: "kube-system"}},
		},
		Pods: []cluster.Pod{
			{Summary: cluster.PodSummary{Name: "pod-1", Namespace: "default"}, Status: cluster.PodStatus{Phase: "Running"}},
			{Summary: cluster.PodSummary{Name: "pod-2", Namespace: "default"}, Status: cluster.PodStatus{Phase: "Running"}},
			{Summary: cluster.PodSummary{Name: "pod-3", Namespace: "default"}, Status: cluster.PodStatus{Phase: "Running"}},
			{Summary: cluster.PodSummary{Name: "pod-4", Namespace: "kube-system"}, Status: cluster.PodStatus{Phase: "Running"}},
		},
	}

	svc.calculateNamespaceResources(snapshot)

	// Find namespaces by name
	var defaultNS, kubeSystemNS *cluster.Namespace
	for i := range snapshot.Namespaces {
		switch snapshot.Namespaces[i].GetName() {
		case "default":
			defaultNS = &snapshot.Namespaces[i]
		case "kube-system":
			kubeSystemNS = &snapshot.Namespaces[i]
		}
	}

	if defaultNS == nil || kubeSystemNS == nil {
		t.Fatal("failed to find namespaces in snapshot")
	}

	if defaultNS.Resources.Pods != 3 {
		t.Errorf("default namespace pods = %d, want 3", defaultNS.Resources.Pods)
	}
	if kubeSystemNS.Resources.Pods != 1 {
		t.Errorf("kube-system namespace pods = %d, want 1", kubeSystemNS.Resources.Pods)
	}
}

// ---------------------------------------------------------------------------
// TestCalculateNamespaceResources_PodPhases
// ---------------------------------------------------------------------------

func TestCalculateNamespaceResources_PodPhases(t *testing.T) {
	svc := newTestService()

	snapshot := &cluster.ClusterSnapshot{
		Namespaces: []cluster.Namespace{
			{Summary: cluster.NamespaceSummary{Name: "default"}},
		},
		Pods: []cluster.Pod{
			{Summary: cluster.PodSummary{Name: "running-1", Namespace: "default"}, Status: cluster.PodStatus{Phase: "Running"}},
			{Summary: cluster.PodSummary{Name: "running-2", Namespace: "default"}, Status: cluster.PodStatus{Phase: "Running"}},
			{Summary: cluster.PodSummary{Name: "pending-1", Namespace: "default"}, Status: cluster.PodStatus{Phase: "Pending"}},
			{Summary: cluster.PodSummary{Name: "failed-1", Namespace: "default"}, Status: cluster.PodStatus{Phase: "Failed"}},
			{Summary: cluster.PodSummary{Name: "succeeded-1", Namespace: "default"}, Status: cluster.PodStatus{Phase: "Succeeded"}},
			{Summary: cluster.PodSummary{Name: "succeeded-2", Namespace: "default"}, Status: cluster.PodStatus{Phase: "Succeeded"}},
		},
	}

	svc.calculateNamespaceResources(snapshot)

	ns := &snapshot.Namespaces[0]
	if ns.Resources.Pods != 6 {
		t.Errorf("total pods = %d, want 6", ns.Resources.Pods)
	}
	if ns.Resources.PodsRunning != 2 {
		t.Errorf("running pods = %d, want 2", ns.Resources.PodsRunning)
	}
	if ns.Resources.PodsPending != 1 {
		t.Errorf("pending pods = %d, want 1", ns.Resources.PodsPending)
	}
	if ns.Resources.PodsFailed != 1 {
		t.Errorf("failed pods = %d, want 1", ns.Resources.PodsFailed)
	}
	if ns.Resources.PodsSucceeded != 2 {
		t.Errorf("succeeded pods = %d, want 2", ns.Resources.PodsSucceeded)
	}
}

// ---------------------------------------------------------------------------
// TestCalculateNamespaceResources_QuotaAssociation
// ---------------------------------------------------------------------------

func TestCalculateNamespaceResources_QuotaAssociation(t *testing.T) {
	svc := newTestService()

	snapshot := &cluster.ClusterSnapshot{
		Namespaces: []cluster.Namespace{
			{Summary: cluster.NamespaceSummary{Name: "ns-a"}},
			{Summary: cluster.NamespaceSummary{Name: "ns-b"}},
		},
		ResourceQuotas: []cluster.ResourceQuota{
			{Name: "quota-1", Namespace: "ns-a"},
			{Name: "quota-2", Namespace: "ns-a"},
			{Name: "quota-3", Namespace: "ns-b"},
		},
		LimitRanges: []cluster.LimitRange{
			{Name: "lr-1", Namespace: "ns-b"},
		},
	}

	svc.calculateNamespaceResources(snapshot)

	var nsA, nsB *cluster.Namespace
	for i := range snapshot.Namespaces {
		switch snapshot.Namespaces[i].GetName() {
		case "ns-a":
			nsA = &snapshot.Namespaces[i]
		case "ns-b":
			nsB = &snapshot.Namespaces[i]
		}
	}

	if nsA == nil || nsB == nil {
		t.Fatal("failed to find namespaces")
	}

	if len(nsA.Quotas) != 2 {
		t.Errorf("ns-a quotas = %d, want 2", len(nsA.Quotas))
	}
	if len(nsB.Quotas) != 1 {
		t.Errorf("ns-b quotas = %d, want 1", len(nsB.Quotas))
	}
	if len(nsA.LimitRanges) != 0 {
		t.Errorf("ns-a limitRanges = %d, want 0", len(nsA.LimitRanges))
	}
	if len(nsB.LimitRanges) != 1 {
		t.Errorf("ns-b limitRanges = %d, want 1", len(nsB.LimitRanges))
	}
}

// ---------------------------------------------------------------------------
// TestGetOTelSnapshot_NilRepo
// ---------------------------------------------------------------------------

func TestGetOTelSnapshot_NilRepo(t *testing.T) {
	svc := newTestService()
	// otelSummaryRepo is nil by default

	snapshot, err := svc.Collect(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if snapshot.OTel != nil {
		t.Errorf("OTel should be nil when otelSummaryRepo is nil, got %+v", snapshot.OTel)
	}
}

// ---------------------------------------------------------------------------
// TestGetOTelSnapshot_CacheBehavior
// ---------------------------------------------------------------------------

func TestGetOTelSnapshot_CacheBehavior(t *testing.T) {
	// Set cache TTL for test
	config.GlobalConfig.Scheduler.OTelCacheTTL = 5 * time.Minute

	var callCount int64
	otelRepo := &mock.OTelSummaryRepository{
		GetAPMSummaryFn: func(_ context.Context) (int, int, float64, float64, float64, error) {
			atomic.AddInt64(&callCount, 1)
			return 10, 8, 100.0, 99.5, 15.0, nil
		},
		GetSLOSummaryFn: func(_ context.Context) (int, float64, error) {
			return 5, 50.0, nil
		},
		GetMetricsSummaryFn: func(_ context.Context) (int, float64, float64, float64, float64, error) {
			return 4, 45.0, 60.0, 80.0, 85.0, nil
		},
	}

	svc := newTestService(func(s *snapshotService) {
		s.otelSummaryRepo = otelRepo
	})

	ctx := context.Background()

	// First call: should invoke GetAPMSummaryFn
	result1 := svc.getOTelSnapshot(ctx)
	if result1 == nil {
		t.Fatal("first call returned nil")
	}
	if result1.TotalServices != 10 {
		t.Errorf("TotalServices = %d, want 10", result1.TotalServices)
	}
	if atomic.LoadInt64(&callCount) != 1 {
		t.Errorf("GetAPMSummaryFn call count = %d, want 1", atomic.LoadInt64(&callCount))
	}

	// Second call (within TTL): should return cached result
	result2 := svc.getOTelSnapshot(ctx)
	if result2 == nil {
		t.Fatal("second call returned nil")
	}
	if atomic.LoadInt64(&callCount) != 1 {
		t.Errorf("GetAPMSummaryFn call count = %d after cached call, want 1", atomic.LoadInt64(&callCount))
	}

	// Verify cached data is the same
	if result2.TotalServices != 10 {
		t.Errorf("cached TotalServices = %d, want 10", result2.TotalServices)
	}
	if result2.IngressServices != 5 {
		t.Errorf("cached IngressServices = %d, want 5", result2.IngressServices)
	}
	if result2.MonitoredNodes != 4 {
		t.Errorf("cached MonitoredNodes = %d, want 4", result2.MonitoredNodes)
	}
}

// ---------------------------------------------------------------------------
// TestGetOTelSnapshot_CacheExpiry
// ---------------------------------------------------------------------------

func TestGetOTelSnapshot_CacheExpiry(t *testing.T) {
	// Set very short TTL to test expiry
	config.GlobalConfig.Scheduler.OTelCacheTTL = 1 * time.Millisecond

	var callCount int64
	otelRepo := &mock.OTelSummaryRepository{
		GetAPMSummaryFn: func(_ context.Context) (int, int, float64, float64, float64, error) {
			atomic.AddInt64(&callCount, 1)
			return 10, 8, 100.0, 99.5, 15.0, nil
		},
		GetSLOSummaryFn: func(_ context.Context) (int, float64, error) {
			return 5, 50.0, nil
		},
		GetMetricsSummaryFn: func(_ context.Context) (int, float64, float64, float64, float64, error) {
			return 4, 45.0, 60.0, 80.0, 85.0, nil
		},
	}

	svc := newTestService(func(s *snapshotService) {
		s.otelSummaryRepo = otelRepo
	})

	ctx := context.Background()

	// First call
	svc.getOTelSnapshot(ctx)
	if atomic.LoadInt64(&callCount) != 1 {
		t.Errorf("first call count = %d, want 1", atomic.LoadInt64(&callCount))
	}

	// Wait for cache to expire
	time.Sleep(5 * time.Millisecond)

	// Second call should hit repo again
	svc.getOTelSnapshot(ctx)
	if atomic.LoadInt64(&callCount) != 2 {
		t.Errorf("call count after expiry = %d, want 2", atomic.LoadInt64(&callCount))
	}
}

// ---------------------------------------------------------------------------
// TestCollect_SnapshotSummaryGenerated
// ---------------------------------------------------------------------------

func TestCollect_SnapshotSummaryGenerated(t *testing.T) {
	svc := newTestService(func(s *snapshotService) {
		s.podRepo = &mock.PodRepository{
			ListFn: func(_ context.Context, _ string, _ model.ListOptions) ([]cluster.Pod, error) {
				return []cluster.Pod{
					{Summary: cluster.PodSummary{Name: "p1", Namespace: "default"}, Status: cluster.PodStatus{Phase: "Running"}},
					{Summary: cluster.PodSummary{Name: "p2", Namespace: "default"}, Status: cluster.PodStatus{Phase: "Pending"}},
				}, nil
			},
		}
		s.nodeRepo = &mock.NodeRepository{
			ListFn: func(_ context.Context, _ model.ListOptions) ([]cluster.Node, error) {
				return []cluster.Node{
					{Summary: cluster.NodeSummary{Name: "n1", Ready: "True"}},
					{Summary: cluster.NodeSummary{Name: "n2", Ready: "False"}},
				}, nil
			},
		}
		s.namespaceRepo = &mock.NamespaceRepository{
			ListFn: func(_ context.Context, _ model.ListOptions) ([]cluster.Namespace, error) {
				return []cluster.Namespace{
					{Summary: cluster.NamespaceSummary{Name: "default"}},
				}, nil
			},
		}
	})

	snapshot, err := svc.Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if snapshot.Summary.TotalPods != 2 {
		t.Errorf("Summary.TotalPods = %d, want 2", snapshot.Summary.TotalPods)
	}
	if snapshot.Summary.RunningPods != 1 {
		t.Errorf("Summary.RunningPods = %d, want 1", snapshot.Summary.RunningPods)
	}
	if snapshot.Summary.PendingPods != 1 {
		t.Errorf("Summary.PendingPods = %d, want 1", snapshot.Summary.PendingPods)
	}
	if snapshot.Summary.TotalNodes != 2 {
		t.Errorf("Summary.TotalNodes = %d, want 2", snapshot.Summary.TotalNodes)
	}
	if snapshot.Summary.ReadyNodes != 1 {
		t.Errorf("Summary.ReadyNodes = %d, want 1", snapshot.Summary.ReadyNodes)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// commonMeta creates a minimal CommonMeta for testing resources that embed it.
func commonMeta(name, namespace string) model_v3.CommonMeta {
	return model_v3.CommonMeta{
		Name:      name,
		Namespace: namespace,
	}
}

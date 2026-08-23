// atlhyper_master_v2/aiops/baseline/extractor_test.go
// 容器级指标提取 + 确定性异常直注 + Event 关联信号 测试
package baseline

import (
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	model_v3 "AtlHyper/model_v3"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/slo"
)

// ==================== Phase 1: extractPodMetrics 容器级指标 ====================

func TestExtractPodMetrics_AllHealthy(t *testing.T) {
	snap := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{
			makePod("default", "web-abc", "Running", 0,
				makeContainer("nginx", "running", "", true, 0, ""),
				makeContainer("sidecar", "running", "", true, 0, ""),
				makeContainer("init-done", "running", "", true, 0, ""),
			),
		},
	}

	points := extractPodMetrics(snap)
	metrics := indexPoints(points, "default/pod/web-abc")

	assertMetric(t, metrics, "not_ready_containers", 0)
	assertMetric(t, metrics, "max_container_restarts", 0)
	assertMetric(t, metrics, "is_running", 1)
	assertMetric(t, metrics, "restart_count", 0)
}

func TestExtractPodMetrics_OneCrashLoopBackOff(t *testing.T) {
	snap := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{
			makePod("default", "mysql-0", "Running", 5,
				makeContainer("mysql", "waiting", "CrashLoopBackOff", false, 5, ""),
				makeContainer("sidecar", "running", "", true, 0, ""),
			),
		},
	}

	points := extractPodMetrics(snap)
	metrics := indexPoints(points, "default/pod/mysql-0")

	assertMetric(t, metrics, "not_ready_containers", 1)
	assertMetric(t, metrics, "max_container_restarts", 5)
	assertMetric(t, metrics, "is_running", 1) // Pod Phase 仍为 Running
}

func TestExtractPodMetrics_NoContainers(t *testing.T) {
	snap := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{
			makePod("kube-system", "pending-pod", "Pending", 0),
		},
	}

	points := extractPodMetrics(snap)
	metrics := indexPoints(points, "kube-system/pod/pending-pod")

	assertMetric(t, metrics, "not_ready_containers", 0)
	assertMetric(t, metrics, "max_container_restarts", 0)
	assertMetric(t, metrics, "is_running", 0)
}

// ==================== Phase 2: ExtractDeterministicAnomalies 容器异常 ====================

func TestExtractDeterministic_CrashLoopBackOff(t *testing.T) {
	snap := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{
			makePod("default", "crash-pod", "Running", 10,
				makeContainer("app", "waiting", "CrashLoopBackOff", false, 10, ""),
				makeContainer("sidecar", "running", "", true, 0, ""),
			),
		},
	}

	results := ExtractDeterministicAnomalies(snap)
	found := findResult(results, "default/pod/crash-pod", "container_anomaly")
	if found == nil {
		t.Fatal("CrashLoopBackOff 应生成 container_anomaly 结果")
	}
	if found.Score != 0.90 {
		t.Errorf("CrashLoopBackOff score 应为 0.90, got %.2f", found.Score)
	}
	if !found.IsAnomaly {
		t.Error("确定性异常 IsAnomaly 应为 true")
	}
}

func TestExtractDeterministic_OOMKilled(t *testing.T) {
	snap := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{
			makePod("default", "oom-pod", "Running", 3,
				makeContainer("app", "waiting", "OOMKilled", false, 3, "OOMKilled"),
			),
		},
	}

	results := ExtractDeterministicAnomalies(snap)
	found := findResult(results, "default/pod/oom-pod", "container_anomaly")
	if found == nil {
		t.Fatal("OOMKilled 应生成 container_anomaly 结果")
	}
	if found.Score != 0.95 {
		t.Errorf("OOMKilled score 应为 0.95, got %.2f", found.Score)
	}
}

func TestExtractDeterministic_ImagePullBackOff(t *testing.T) {
	snap := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{
			makePod("default", "img-pod", "Pending", 0,
				makeContainer("app", "waiting", "ImagePullBackOff", false, 0, ""),
			),
		},
	}

	results := ExtractDeterministicAnomalies(snap)
	found := findResult(results, "default/pod/img-pod", "container_anomaly")
	if found == nil {
		t.Fatal("ImagePullBackOff 应生成 container_anomaly 结果")
	}
	if found.Score != 0.70 {
		t.Errorf("ImagePullBackOff score 应为 0.70, got %.2f", found.Score)
	}
}

func TestExtractDeterministic_AllNormal(t *testing.T) {
	snap := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{
			makePod("default", "healthy-pod", "Running", 0,
				makeContainer("app", "running", "", true, 0, ""),
				makeContainer("sidecar", "running", "", true, 0, ""),
			),
		},
	}

	results := ExtractDeterministicAnomalies(snap)
	found := findResult(results, "default/pod/healthy-pod", "container_anomaly")
	if found != nil {
		t.Errorf("全正常 Pod 不应生成 container_anomaly, got score=%.2f", found.Score)
	}
}

func TestExtractDeterministic_MultiContainer_TakesWorst(t *testing.T) {
	// 一个容器 ImagePullBackOff(0.70), 另一个 OOMKilled(0.95) → 取最严重
	snap := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{
			makePod("default", "multi-bad", "Running", 5,
				makeContainer("app", "waiting", "ImagePullBackOff", false, 0, ""),
				makeContainer("worker", "waiting", "OOMKilled", false, 5, "OOMKilled"),
			),
		},
	}

	results := ExtractDeterministicAnomalies(snap)
	found := findResult(results, "default/pod/multi-bad", "container_anomaly")
	if found == nil {
		t.Fatal("多容器异常应生成结果")
	}
	if found.Score != 0.95 {
		t.Errorf("应取最严重容器 (OOMKilled=0.95), got %.2f", found.Score)
	}
}

func TestExtractDeterministic_TerminatedOOMKilled(t *testing.T) {
	snap := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{
			makePod("default", "term-oom", "Running", 1,
				makeContainerWithTime("app", "terminated", "", false, 1, "OOMKilled", ""),
			),
		},
	}

	results := ExtractDeterministicAnomalies(snap)
	found := findResult(results, "default/pod/term-oom", "container_anomaly")
	if found == nil {
		t.Fatal("terminated + LastTerminationReason=OOMKilled 应生成结果")
	}
	if found.Score != 0.95 {
		t.Errorf("OOMKilled score 应为 0.95, got %.2f", found.Score)
	}
}

func TestExtractDeterministic_RunningRecentCrash(t *testing.T) {
	// 容器 running 但 2 分钟前崩溃过（快照恰好抓到重启后的 running 瞬间）
	recentTime := time.Now().Add(-2 * time.Minute).Format(time.RFC3339)
	snap := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{
			makePod("geass", "geass-user-abc", "Running", 3,
				makeContainerWithTime("geass-user", "running", "", true, 3, "Error", recentTime),
				makeContainer("linkerd-proxy", "running", "", true, 0, ""),
			),
		},
	}

	results := ExtractDeterministicAnomalies(snap)
	found := findResult(results, "geass/pod/geass-user-abc", "container_anomaly")
	if found == nil {
		t.Fatal("running + 近期崩溃 (Error, 2min ago) 应生成 container_anomaly")
	}
	if found.Score != 0.75 {
		t.Errorf("RecentCrash score 应为 0.75, got %.2f", found.Score)
	}
}

func TestExtractDeterministic_RunningOldCrash(t *testing.T) {
	// 容器 running，崩溃是 1 小时前的 → 不应告警
	oldTime := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	snap := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{
			makePod("default", "old-crash", "Running", 2,
				makeContainerWithTime("app", "running", "", true, 2, "Error", oldTime),
			),
		},
	}

	results := ExtractDeterministicAnomalies(snap)
	found := findResult(results, "default/pod/old-crash", "container_anomaly")
	if found != nil {
		t.Errorf("超过 10 分钟的崩溃不应告警, got score=%.2f", found.Score)
	}
}

func TestExtractDeterministic_NotReady(t *testing.T) {
	// 容器 running 但 ready=false（readiness probe 失败）
	snap := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{
			makePod("geass", "geass-media-xyz", "Running", 0,
				makeContainer("geass-media", "running", "", false, 0, ""),
				makeContainer("linkerd-proxy", "running", "", true, 0, ""),
			),
		},
	}

	results := ExtractDeterministicAnomalies(snap)
	found := findResult(results, "geass/pod/geass-media-xyz", "container_anomaly")
	if found == nil {
		t.Fatal("running + ready=false 应生成 container_anomaly")
	}
	if found.Score != 0.60 {
		t.Errorf("NotReady score 应为 0.60, got %.2f", found.Score)
	}
}

// ==================== Phase 2: classifyContainerAnomaly ====================

func TestClassifyContainerAnomaly(t *testing.T) {
	tests := []struct {
		name       string
		container  cluster.PodContainerDetail
		wantReason string
	}{
		{
			"waiting_CrashLoopBackOff",
			cluster.PodContainerDetail{State: "waiting", StateReason: "CrashLoopBackOff"},
			"CrashLoopBackOff",
		},
		{
			"waiting_OOMKilled",
			cluster.PodContainerDetail{State: "waiting", StateReason: "OOMKilled"},
			"OOMKilled",
		},
		{
			"waiting_ImagePullBackOff",
			cluster.PodContainerDetail{State: "waiting", StateReason: "ImagePullBackOff"},
			"ImagePullBackOff",
		},
		{
			"waiting_ErrImagePull",
			cluster.PodContainerDetail{State: "waiting", StateReason: "ErrImagePull"},
			"ErrImagePull",
		},
		{
			"waiting_CreateContainerConfigError",
			cluster.PodContainerDetail{State: "waiting", StateReason: "CreateContainerConfigError"},
			"CreateContainerConfigError",
		},
		{
			"terminated_OOMKilled",
			cluster.PodContainerDetail{State: "terminated", LastTerminationReason: "OOMKilled"},
			"OOMKilled",
		},
		{
			"running_recent_crash",
			cluster.PodContainerDetail{
				State: "running", Ready: true, RestartCount: 3,
				LastTerminationReason: "Error",
				LastTerminationTime:   time.Now().Add(-2 * time.Minute).Format(time.RFC3339),
			},
			"RecentCrash",
		},
		{
			"running_recent_oom",
			cluster.PodContainerDetail{
				State: "running", Ready: true, RestartCount: 1,
				LastTerminationReason: "OOMKilled",
				LastTerminationTime:   time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
			},
			"OOMKilled",
		},
		{
			"running_old_crash_no_alert",
			cluster.PodContainerDetail{
				State: "running", Ready: true, RestartCount: 2,
				LastTerminationReason: "Error",
				LastTerminationTime:   time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
			},
			"",
		},
		{
			"running_not_ready",
			cluster.PodContainerDetail{State: "running", Ready: false},
			"NotReady",
		},
		{
			"running_normal",
			cluster.PodContainerDetail{State: "running", Ready: true},
			"",
		},
		{
			"waiting_ContainerCreating",
			cluster.PodContainerDetail{State: "waiting", StateReason: "ContainerCreating"},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyContainerAnomaly(&tt.container)
			if got != tt.wantReason {
				t.Errorf("classifyContainerAnomaly() = %q, want %q", got, tt.wantReason)
			}
		})
	}
}

// ==================== Phase 3: Event 关联异常 ====================

func TestExtractEventAnomalies_CriticalPodEvent(t *testing.T) {
	now := time.Now()
	snap := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{
			makePod("default", "crash-pod", "Running", 5,
				makeContainer("app", "waiting", "CrashLoopBackOff", false, 5, ""),
			),
		},
		Events: []cluster.Event{
			makeEvent("Warning", "BackOff", "Pod", "default", "crash-pod", now.Add(-2*time.Minute)),
		},
	}

	results := extractEventAnomalies(snap, now.Unix())
	found := findResult(results, "default/pod/crash-pod", "critical_event")
	if found == nil {
		t.Fatal("关联 Critical Event 应生成 critical_event 结果")
	}
	if found.Score != 0.85 {
		t.Errorf("critical_event score 应为 0.85, got %.2f", found.Score)
	}
}

func TestExtractEventAnomalies_OldEvent(t *testing.T) {
	now := time.Now()
	snap := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{
			makePod("default", "old-pod", "Running", 0,
				makeContainer("app", "running", "", true, 0, ""),
			),
		},
		Events: []cluster.Event{
			makeEvent("Warning", "BackOff", "Pod", "default", "old-pod", now.Add(-10*time.Minute)),
		},
	}

	results := extractEventAnomalies(snap, now.Unix())
	found := findResult(results, "default/pod/old-pod", "critical_event")
	if found != nil {
		t.Error("超过 5 分钟的 Event 不应生成结果")
	}
}

func TestExtractEventAnomalies_PodNotInSnapshot(t *testing.T) {
	now := time.Now()
	snap := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{}, // 无 Pod
		Events: []cluster.Event{
			makeEvent("Warning", "BackOff", "Pod", "default", "ghost-pod", now.Add(-1*time.Minute)),
		},
	}

	results := extractEventAnomalies(snap, now.Unix())
	if len(results) != 0 {
		t.Errorf("关联 Pod 不在快照中时不应生成结果, got %d", len(results))
	}
}

func TestExtractEventAnomalies_NormalEvent(t *testing.T) {
	now := time.Now()
	snap := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{
			makePod("default", "normal-pod", "Running", 0,
				makeContainer("app", "running", "", true, 0, ""),
			),
		},
		Events: []cluster.Event{
			makeEvent("Normal", "Scheduled", "Pod", "default", "normal-pod", now.Add(-1*time.Minute)),
		},
	}

	results := extractEventAnomalies(snap, now.Unix())
	if len(results) != 0 {
		t.Errorf("Normal Event 不应生成结果, got %d", len(results))
	}
}

func TestExtractEventAnomalies_NonPodEvent(t *testing.T) {
	now := time.Now()
	snap := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{
			makePod("default", "some-pod", "Running", 0,
				makeContainer("app", "running", "", true, 0, ""),
			),
		},
		Events: []cluster.Event{
			makeEvent("Warning", "NodeNotReady", "Node", "", "worker-1", now.Add(-1*time.Minute)),
		},
	}

	results := extractEventAnomalies(snap, now.Unix())
	if len(results) != 0 {
		t.Errorf("非 Pod Event 不应生成 Pod 结果, got %d", len(results))
	}
}

func TestExtractEventAnomalies_DeduplicatePerPod(t *testing.T) {
	now := time.Now()
	snap := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{
			makePod("default", "dup-pod", "Running", 3,
				makeContainer("app", "waiting", "CrashLoopBackOff", false, 3, ""),
			),
		},
		Events: []cluster.Event{
			makeEvent("Warning", "BackOff", "Pod", "default", "dup-pod", now.Add(-1*time.Minute)),
			makeEvent("Warning", "Failed", "Pod", "default", "dup-pod", now.Add(-2*time.Minute)),
		},
	}

	results := extractEventAnomalies(snap, now.Unix())
	count := 0
	for _, r := range results {
		if r.EntityKey == "default/pod/dup-pod" && r.MetricName == "critical_event" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("每个 Pod 只应报告一次 critical_event, got %d", count)
	}
}

// ==================== 辅助函数 ====================

func makePod(namespace, name, phase string, restarts int32, containers ...cluster.PodContainerDetail) cluster.Pod {
	return cluster.Pod{
		Summary: cluster.PodSummary{
			Name:      name,
			Namespace: namespace,
		},
		Status: cluster.PodStatus{
			Phase:    phase,
			Restarts: restarts,
		},
		Containers: containers,
	}
}

func makeContainer(name, state, stateReason string, ready bool, restarts int32, lastTermReason string) cluster.PodContainerDetail {
	return cluster.PodContainerDetail{
		Name:                  name,
		State:                 state,
		StateReason:           stateReason,
		Ready:                 ready,
		RestartCount:          restarts,
		LastTerminationReason: lastTermReason,
	}
}

func makeContainerWithTime(name, state, stateReason string, ready bool, restarts int32, lastTermReason, lastTermTime string) cluster.PodContainerDetail {
	return cluster.PodContainerDetail{
		Name:                  name,
		State:                 state,
		StateReason:           stateReason,
		Ready:                 ready,
		RestartCount:          restarts,
		LastTerminationReason: lastTermReason,
		LastTerminationTime:   lastTermTime,
	}
}

func makeEvent(typ, reason, involvedKind, involvedNS, involvedName string, lastTimestamp time.Time) cluster.Event {
	return cluster.Event{
		CommonMeta: model_v3.CommonMeta{
			Name:      reason + "-event",
			Namespace: involvedNS,
		},
		Type:           typ,
		Reason:         reason,
		LastTimestamp:  lastTimestamp,
		InvolvedObject: model_v3.ResourceRef{Kind: involvedKind, Namespace: involvedNS, Name: involvedName},
	}
}

// indexPoints 将指标点按 metricName 索引
func indexPoints(points []aiops.MetricDataPoint, entityKey string) map[string]float64 {
	m := make(map[string]float64)
	for _, p := range points {
		if p.EntityKey == entityKey {
			m[p.MetricName] = p.Value
		}
	}
	return m
}

func assertMetric(t *testing.T, metrics map[string]float64, name string, expected float64) {
	t.Helper()
	got, ok := metrics[name]
	if !ok {
		t.Errorf("缺少指标 %s", name)
		return
	}
	if got != expected {
		t.Errorf("指标 %s = %.2f, want %.2f", name, got, expected)
	}
}

// findResult 在结果中查找匹配的异常
func findResult(results []*aiops.AnomalyResult, entityKey, metricName string) *aiops.AnomalyResult {
	for _, r := range results {
		if r.EntityKey == entityKey && r.MetricName == metricName {
			return r
		}
	}
	return nil
}

// findAllResults 查找所有匹配 metricName 的结果
func findAllResults(results []*aiops.AnomalyResult, metricName string) []*aiops.AnomalyResult {
	var matched []*aiops.AnomalyResult
	for _, r := range results {
		if r.MetricName == metricName {
			matched = append(matched, r)
		}
	}
	return matched
}

// makePodWithOwner 创建带 Owner 信息的 Pod
func makePodWithOwner(namespace, name, phase string, restarts int32, ownerKind, ownerName string, containers ...cluster.PodContainerDetail) cluster.Pod {
	return cluster.Pod{
		Summary: cluster.PodSummary{
			Name:      name,
			Namespace: namespace,
			OwnerKind: ownerKind,
			OwnerName: ownerName,
		},
		Status: cluster.PodStatus{
			Phase:    phase,
			Restarts: restarts,
		},
		Containers: containers,
	}
}

// makeDeployment 创建 Deployment
func makeDeployment(namespace, name string, replicas, ready int32) cluster.Deployment {
	return cluster.Deployment{
		Summary: cluster.DeploymentSummary{
			Name:      name,
			Namespace: namespace,
			Replicas:  replicas,
			Ready:     ready,
		},
	}
}

// makeReplicaSet 创建 ReplicaSet（通过 OwnerKind/OwnerName 关联 Deployment）
func makeReplicaSet(namespace, name, ownerName string) cluster.ReplicaSet {
	return cluster.ReplicaSet{
		CommonMeta: model_v3.CommonMeta{
			Name:      name,
			Namespace: namespace,
			OwnerKind: "Deployment",
			OwnerName: ownerName,
		},
	}
}

// ==================== Phase 4: Deployment 影响比例异常 ====================

func TestExtractDeploymentImpact_75Percent(t *testing.T) {
	// 4 个 Pod，3 个不健康 → 75% → score=0.95
	snap := &cluster.ClusterSnapshot{
		Deployments: []cluster.Deployment{
			makeDeployment("default", "web", 4, 1),
		},
		ReplicaSets: []cluster.ReplicaSet{
			makeReplicaSet("default", "web-rs-abc", "web"),
		},
		Pods: []cluster.Pod{
			makePodWithOwner("default", "web-1", "Running", 5, "ReplicaSet", "web-rs-abc",
				makeContainer("app", "waiting", "CrashLoopBackOff", false, 5, ""),
			),
			makePodWithOwner("default", "web-2", "Running", 3, "ReplicaSet", "web-rs-abc",
				makeContainer("app", "waiting", "CrashLoopBackOff", false, 3, ""),
			),
			makePodWithOwner("default", "web-3", "Running", 2, "ReplicaSet", "web-rs-abc",
				makeContainer("app", "waiting", "CrashLoopBackOff", false, 2, ""),
			),
			makePodWithOwner("default", "web-4", "Running", 0, "ReplicaSet", "web-rs-abc",
				makeContainer("app", "running", "", true, 0, ""),
			),
		},
	}

	results := ExtractDeterministicAnomalies(snap)
	impacts := findAllResults(results, "deployment_impact")
	if len(impacts) != 3 {
		t.Fatalf("75%% 不健康应有 3 个 Pod 收到信号, got %d", len(impacts))
	}
	for _, r := range impacts {
		if r.Score != 0.95 {
			t.Errorf("75%% 不可用 score 应为 0.95, got %.2f (entity=%s)", r.Score, r.EntityKey)
		}
	}
	// 健康 Pod 不应收到
	healthy := findResult(results, "default/pod/web-4", "deployment_impact")
	if healthy != nil {
		t.Error("健康 Pod 不应收到 deployment_impact 信号")
	}
}

func TestExtractDeploymentImpact_50Percent(t *testing.T) {
	// 4 个 Pod，2 个不健康 → 50% → score=0.80
	snap := &cluster.ClusterSnapshot{
		Deployments: []cluster.Deployment{
			makeDeployment("default", "api", 4, 2),
		},
		ReplicaSets: []cluster.ReplicaSet{
			makeReplicaSet("default", "api-rs-xyz", "api"),
		},
		Pods: []cluster.Pod{
			makePodWithOwner("default", "api-1", "Running", 3, "ReplicaSet", "api-rs-xyz",
				makeContainer("app", "waiting", "CrashLoopBackOff", false, 3, ""),
			),
			makePodWithOwner("default", "api-2", "Running", 2, "ReplicaSet", "api-rs-xyz",
				makeContainer("app", "running", "", false, 0, ""), // Ready=false
			),
			makePodWithOwner("default", "api-3", "Running", 0, "ReplicaSet", "api-rs-xyz",
				makeContainer("app", "running", "", true, 0, ""),
			),
			makePodWithOwner("default", "api-4", "Running", 0, "ReplicaSet", "api-rs-xyz",
				makeContainer("app", "running", "", true, 0, ""),
			),
		},
	}

	results := ExtractDeterministicAnomalies(snap)
	impacts := findAllResults(results, "deployment_impact")
	if len(impacts) != 2 {
		t.Fatalf("50%% 不健康应有 2 个 Pod 收到信号, got %d", len(impacts))
	}
	for _, r := range impacts {
		if r.Score != 0.80 {
			t.Errorf("50%% 不可用 score 应为 0.80, got %.2f (entity=%s)", r.Score, r.EntityKey)
		}
	}
}

func TestExtractDeploymentImpact_25Percent_NoInjection(t *testing.T) {
	// 4 个 Pod，1 个不健康 → 25% → 不注入
	snap := &cluster.ClusterSnapshot{
		Deployments: []cluster.Deployment{
			makeDeployment("default", "worker", 4, 3),
		},
		ReplicaSets: []cluster.ReplicaSet{
			makeReplicaSet("default", "worker-rs-def", "worker"),
		},
		Pods: []cluster.Pod{
			makePodWithOwner("default", "worker-1", "Running", 3, "ReplicaSet", "worker-rs-def",
				makeContainer("app", "waiting", "CrashLoopBackOff", false, 3, ""),
			),
			makePodWithOwner("default", "worker-2", "Running", 0, "ReplicaSet", "worker-rs-def",
				makeContainer("app", "running", "", true, 0, ""),
			),
			makePodWithOwner("default", "worker-3", "Running", 0, "ReplicaSet", "worker-rs-def",
				makeContainer("app", "running", "", true, 0, ""),
			),
			makePodWithOwner("default", "worker-4", "Running", 0, "ReplicaSet", "worker-rs-def",
				makeContainer("app", "running", "", true, 0, ""),
			),
		},
	}

	results := ExtractDeterministicAnomalies(snap)
	impacts := findAllResults(results, "deployment_impact")
	if len(impacts) != 0 {
		t.Errorf("25%% 不可用不应注入, got %d 个信号", len(impacts))
	}
}

func TestExtractDeploymentImpact_HealthyPodNoSignal(t *testing.T) {
	// 全部健康 Pod → 不注入
	snap := &cluster.ClusterSnapshot{
		Deployments: []cluster.Deployment{
			makeDeployment("default", "healthy-app", 2, 2),
		},
		ReplicaSets: []cluster.ReplicaSet{
			makeReplicaSet("default", "healthy-rs", "healthy-app"),
		},
		Pods: []cluster.Pod{
			makePodWithOwner("default", "healthy-1", "Running", 0, "ReplicaSet", "healthy-rs",
				makeContainer("app", "running", "", true, 0, ""),
			),
			makePodWithOwner("default", "healthy-2", "Running", 0, "ReplicaSet", "healthy-rs",
				makeContainer("app", "running", "", true, 0, ""),
			),
		},
	}

	results := ExtractDeterministicAnomalies(snap)
	impacts := findAllResults(results, "deployment_impact")
	if len(impacts) != 0 {
		t.Errorf("全健康 Deployment 不应注入, got %d 个信号", len(impacts))
	}
}

func TestExtractDeploymentImpact_NonRSPodSkipped(t *testing.T) {
	// Pod 不属于 ReplicaSet（直接创建或属于 DaemonSet）→ 不参与
	snap := &cluster.ClusterSnapshot{
		Deployments: []cluster.Deployment{
			makeDeployment("default", "web", 2, 0),
		},
		ReplicaSets: []cluster.ReplicaSet{
			makeReplicaSet("default", "web-rs", "web"),
		},
		Pods: []cluster.Pod{
			// DaemonSet Pod，不健康但 OwnerKind 不是 ReplicaSet
			makePodWithOwner("default", "ds-pod", "Running", 5, "DaemonSet", "my-ds",
				makeContainer("app", "waiting", "CrashLoopBackOff", false, 5, ""),
			),
			// 无 Owner 的 Pod
			makePodWithOwner("default", "standalone", "Running", 3, "", "",
				makeContainer("app", "waiting", "OOMKilled", false, 3, "OOMKilled"),
			),
		},
	}

	results := ExtractDeterministicAnomalies(snap)
	impacts := findAllResults(results, "deployment_impact")
	if len(impacts) != 0 {
		t.Errorf("非 RS 管理的 Pod 不应产生 deployment_impact, got %d", len(impacts))
	}
}

func TestExtractDeploymentImpact_IntegrationWithContainerAnomaly(t *testing.T) {
	// 集成测试：container_anomaly + deployment_impact 同时产生
	snap := &cluster.ClusterSnapshot{
		Deployments: []cluster.Deployment{
			makeDeployment("default", "app", 4, 1),
		},
		ReplicaSets: []cluster.ReplicaSet{
			makeReplicaSet("default", "app-rs", "app"),
		},
		Pods: []cluster.Pod{
			makePodWithOwner("default", "app-1", "Running", 10, "ReplicaSet", "app-rs",
				makeContainer("main", "waiting", "CrashLoopBackOff", false, 10, ""),
			),
			makePodWithOwner("default", "app-2", "Running", 5, "ReplicaSet", "app-rs",
				makeContainer("main", "waiting", "OOMKilled", false, 5, "OOMKilled"),
			),
			makePodWithOwner("default", "app-3", "Running", 3, "ReplicaSet", "app-rs",
				makeContainer("main", "running", "", false, 0, ""), // Ready=false
			),
			makePodWithOwner("default", "app-4", "Running", 0, "ReplicaSet", "app-rs",
				makeContainer("main", "running", "", true, 0, ""),
			),
		},
	}

	results := ExtractDeterministicAnomalies(snap)

	// 验证 container_anomaly：3 个不健康 Pod 应各有一个
	for _, podName := range []string{"app-1", "app-2", "app-3"} {
		key := "default/pod/" + podName
		ca := findResult(results, key, "container_anomaly")
		if ca == nil {
			t.Errorf("Pod %s 应有 container_anomaly", podName)
		}
	}

	// 验证 deployment_impact：75% → score=0.95，3 个 Pod 收到
	impacts := findAllResults(results, "deployment_impact")
	if len(impacts) != 3 {
		t.Fatalf("集成测试：应有 3 个 deployment_impact, got %d", len(impacts))
	}
	for _, r := range impacts {
		if r.Score != 0.95 {
			t.Errorf("集成测试 score 应为 0.95, got %.2f", r.Score)
		}
	}

	// 健康 Pod 无信号
	if findResult(results, "default/pod/app-4", "deployment_impact") != nil {
		t.Error("健康 Pod app-4 不应有 deployment_impact")
	}
	if findResult(results, "default/pod/app-4", "container_anomaly") != nil {
		t.Error("健康 Pod app-4 不应有 container_anomaly")
	}
}

func TestExtractIngressMetrics_ErrorRateRange(t *testing.T) {
	// IngressSLO.ErrorRate 已是 0-100 范围，不应再乘 100
	otel := &cluster.OTelSnapshot{
		SLOIngress: []slo.IngressSLO{
			{ServiceKey: "web@docker", ErrorRate: 2.5, AvgMs: 150},
			{ServiceKey: "api@docker", ErrorRate: 0.0, AvgMs: 50},
		},
	}

	points := extractIngressMetrics(otel)

	// web@docker: errorRate 应为 2.5（直接透传），不是 250
	webMetrics := indexPoints(points, "_cluster/ingress/web@docker")
	assertMetricApprox(t, webMetrics, "error_rate", 2.5)
	assertMetric(t, webMetrics, "avg_latency", 150)

	// api@docker: errorRate = 0
	apiMetrics := indexPoints(points, "_cluster/ingress/api@docker")
	assertMetricApprox(t, apiMetrics, "error_rate", 0.0)
}

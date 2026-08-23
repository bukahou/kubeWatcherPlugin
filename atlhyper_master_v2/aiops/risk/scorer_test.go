// atlhyper_master_v2/aiops/risk/scorer_test.go
// 风险评分引擎测试
package risk

import (
	"math"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// ==================== breadthBoost 测试 ====================

func TestBreadthBoost_Values(t *testing.T) {
	tests := []struct {
		n        int
		expected float64
	}{
		{0, 0.0},
		{1, 0.70},
		{2, 0.85},
		{3, 1.0},
		{5, 1.0},
	}
	for _, tt := range tests {
		got := breadthBoost(tt.n)
		if math.Abs(got-tt.expected) > 0.001 {
			t.Errorf("breadthBoost(%d) = %.3f, want %.3f", tt.n, got, tt.expected)
		}
	}
}

// ==================== Stage 1: 局部风险测试 ====================

func TestComputeLocalRisks_NoAnomalies(t *testing.T) {
	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "default/pod/api-1", MetricName: "restart_count", IsAnomaly: false, Score: 0.5},
	}
	config := DefaultRiskConfig()

	risks := ComputeLocalRisks(anomalies, config)
	if len(risks) != 0 {
		t.Errorf("expected no risks, got %d", len(risks))
	}
}

// Pod restart_count(both) score=0.8 → ch1=0.16, ch2=0.80×0.70=0.56 → max=0.56
func TestComputeLocalRisks_SingleAnomaly(t *testing.T) {
	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "default/pod/api-1", MetricName: "restart_count", IsAnomaly: true, Score: 0.8},
	}
	config := DefaultRiskConfig()

	risks := ComputeLocalRisks(anomalies, config)
	expected := 0.56 // both channel: max(0.20×0.8, 0.80×0.70) = max(0.16, 0.56)
	if diff := math.Abs(risks["default/pod/api-1"] - expected); diff > 0.001 {
		t.Errorf("expected %.3f, got %.3f", expected, risks["default/pod/api-1"])
	}
}

// Service: all statistical
func TestComputeLocalRisks_MultipleMetrics(t *testing.T) {
	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "default/service/api", MetricName: "error_rate", IsAnomaly: true, Score: 1.0},
		{EntityKey: "default/service/api", MetricName: "avg_latency", IsAnomaly: true, Score: 0.5},
		{EntityKey: "default/service/api", MetricName: "request_rate", IsAnomaly: false, Score: 0.3},
	}
	config := DefaultRiskConfig()

	risks := ComputeLocalRisks(anomalies, config)
	// error_rate: 0.10 × 1.0 = 0.10
	// avg_latency: 0.05 × 0.5 = 0.025
	// request_rate: not anomaly, skip
	expected := 0.125
	if diff := math.Abs(risks["default/service/api"] - expected); diff > 0.001 {
		t.Errorf("expected %.3f, got %.3f", expected, risks["default/service/api"])
	}
}

// Service: all statistical, clamped
func TestComputeLocalRisks_Clamped(t *testing.T) {
	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "default/service/api", MetricName: "error_rate", IsAnomaly: true, Score: 1.0},
		{EntityKey: "default/service/api", MetricName: "avg_latency", IsAnomaly: true, Score: 1.0},
		{EntityKey: "default/service/api", MetricName: "request_rate", IsAnomaly: true, Score: 1.0},
	}
	config := DefaultRiskConfig()

	risks := ComputeLocalRisks(anomalies, config)
	// 0.10 + 0.05 + 0.05 = 0.20
	expected := 0.20
	if diff := math.Abs(risks["default/service/api"] - expected); diff > 0.001 {
		t.Errorf("expected %.3f, got %.3f", expected, risks["default/service/api"])
	}
}

// Pod unknown_metric → 默认 statistical, weight=0.1 → ch1=0.1, ch2=0 → 0.1
func TestComputeLocalRisks_UnknownMetric(t *testing.T) {
	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "default/pod/api-1", MetricName: "unknown_metric", IsAnomaly: true, Score: 1.0},
	}
	config := DefaultRiskConfig()

	risks := ComputeLocalRisks(anomalies, config)
	// 未配置指标默认 statistical, weight=0.1 → ch1=0.1, ch2=0 → 0.1
	expected := 0.1
	if diff := math.Abs(risks["default/pod/api-1"] - expected); diff > 0.001 {
		t.Errorf("expected %.3f, got %.3f", expected, risks["default/pod/api-1"])
	}
}

// ==================== 回归测试 (Node/Service/Ingress 行为不变) ====================

func TestComputeLocalRisks_NodeStatistical(t *testing.T) {
	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "_cluster/node/node1", MetricName: "cpu_usage", IsAnomaly: true, Score: 0.85},
		{EntityKey: "_cluster/node/node1", MetricName: "memory_usage", IsAnomaly: true, Score: 0.70},
	}
	config := DefaultRiskConfig()

	risks := ComputeLocalRisks(anomalies, config)
	// 全 statistical → ch1 = 0.20×0.85 + 0.20×0.70 = 0.31, ch2 = 0
	expected := 0.31
	if diff := math.Abs(risks["_cluster/node/node1"] - expected); diff > 0.001 {
		t.Errorf("expected %.4f, got %.4f", expected, risks["_cluster/node/node1"])
	}
}

func TestComputeLocalRisks_ServiceStatistical(t *testing.T) {
	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "default/service/api", MetricName: "error_rate", IsAnomaly: true, Score: 1.0},
		{EntityKey: "default/service/api", MetricName: "avg_latency", IsAnomaly: true, Score: 0.5},
	}
	config := DefaultRiskConfig()

	risks := ComputeLocalRisks(anomalies, config)
	// 全 statistical → ch1 = 0.10×1.0 + 0.05×0.5 = 0.125
	expected := 0.125
	if diff := math.Abs(risks["default/service/api"] - expected); diff > 0.001 {
		t.Errorf("expected %.3f, got %.3f", expected, risks["default/service/api"])
	}
}

func TestComputeLocalRisks_IngressStatistical_Unchanged(t *testing.T) {
	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "default/ingress/api", MetricName: "error_rate", IsAnomaly: true, Score: 0.9},
		{EntityKey: "default/ingress/api", MetricName: "avg_latency", IsAnomaly: true, Score: 0.8},
	}
	config := DefaultRiskConfig()

	risks := ComputeLocalRisks(anomalies, config)
	// 全 statistical → ch1 = 0.50×0.9 + 0.50×0.8 = 0.85
	expected := 0.85
	if diff := math.Abs(risks["default/ingress/api"] - expected); diff > 0.001 {
		t.Errorf("expected %.3f, got %.3f", expected, risks["default/ingress/api"])
	}
}

// ==================== Pod 确定性通道测试 ====================

func TestComputeLocalRisks_PodSingleDeterministic(t *testing.T) {
	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "default/pod/api-1", MetricName: "container_anomaly", IsAnomaly: true, Score: 0.90},
	}
	config := DefaultRiskConfig()

	risks := ComputeLocalRisks(anomalies, config)
	// deterministic only → ch1=0 (det 不参与 ch1), ch2=0.90×0.70(n=1)=0.63
	expected := 0.63
	if diff := math.Abs(risks["default/pod/api-1"] - expected); diff > 0.001 {
		t.Errorf("expected %.3f, got %.3f", expected, risks["default/pod/api-1"])
	}
}

func TestComputeLocalRisks_PodTwoDeterministic(t *testing.T) {
	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "default/pod/api-1", MetricName: "container_anomaly", IsAnomaly: true, Score: 0.90},
		{EntityKey: "default/pod/api-1", MetricName: "critical_event", IsAnomaly: true, Score: 0.85},
	}
	config := DefaultRiskConfig()

	risks := ComputeLocalRisks(anomalies, config)
	// ch1=0, ch2=max(0.90,0.85)×0.85(n=2)=0.90×0.85=0.765
	expected := 0.765
	if diff := math.Abs(risks["default/pod/api-1"] - expected); diff > 0.001 {
		t.Errorf("expected %.3f, got %.3f", expected, risks["default/pod/api-1"])
	}
}

func TestComputeLocalRisks_PodThreeDeterministic(t *testing.T) {
	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "default/pod/api-1", MetricName: "container_anomaly", IsAnomaly: true, Score: 0.90},
		{EntityKey: "default/pod/api-1", MetricName: "critical_event", IsAnomaly: true, Score: 0.85},
		{EntityKey: "default/pod/api-1", MetricName: "deployment_impact", IsAnomaly: true, Score: 0.80},
	}
	config := DefaultRiskConfig()

	risks := ComputeLocalRisks(anomalies, config)
	// ch1=0, ch2=max(0.90,0.85,0.80)×1.0(n=3)=0.90
	expected := 0.90
	if diff := math.Abs(risks["default/pod/api-1"] - expected); diff > 0.001 {
		t.Errorf("expected %.3f, got %.3f", expected, risks["default/pod/api-1"])
	}
}

// ==================== Pod 混合通道测试 ====================

func TestComputeLocalRisks_PodMixed_Channel2Wins(t *testing.T) {
	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "default/pod/api-1", MetricName: "restart_count", IsAnomaly: true, Score: 0.80},     // both
		{EntityKey: "default/pod/api-1", MetricName: "container_anomaly", IsAnomaly: true, Score: 0.90}, // deterministic
	}
	config := DefaultRiskConfig()

	risks := ComputeLocalRisks(anomalies, config)
	// ch1 = 0.20×0.80 = 0.16 (only both participates in ch1)
	// ch2: restart(both)+container(det) → n=2, max=0.90 → 0.90×0.85=0.765
	expected := 0.765
	if diff := math.Abs(risks["default/pod/api-1"] - expected); diff > 0.001 {
		t.Errorf("expected %.3f, got %.3f", expected, risks["default/pod/api-1"])
	}
}

func TestComputeLocalRisks_PodBothOnly(t *testing.T) {
	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "default/pod/api-1", MetricName: "restart_count", IsAnomaly: true, Score: 0.80}, // both
	}
	config := DefaultRiskConfig()

	risks := ComputeLocalRisks(anomalies, config)
	// ch1 = 0.20×0.80 = 0.16
	// ch2: n=1, max=0.80 → 0.80×0.70=0.56
	// R_local = max(0.16, 0.56) = 0.56
	expected := 0.56
	if diff := math.Abs(risks["default/pod/api-1"] - expected); diff > 0.001 {
		t.Errorf("expected %.3f, got %.3f", expected, risks["default/pod/api-1"])
	}
}

// ==================== Stage 2: 时序权重测试 ====================

func TestApplyTemporalWeights_NoHistory(t *testing.T) {
	localRisks := map[string]float64{
		"default/pod/api-1": 0.5,
	}
	firstAnomalyTimes := map[string]int64{}

	weighted := ApplyTemporalWeights(localRisks, firstAnomalyTimes, time.Now().Unix(), 300)

	// 无历史记录时 wTime=1.0
	if diff := math.Abs(weighted["default/pod/api-1"] - 0.5); diff > 0.001 {
		t.Errorf("expected 0.5, got %.3f", weighted["default/pod/api-1"])
	}
}

func TestApplyTemporalWeights_RecentAnomaly(t *testing.T) {
	now := time.Now().Unix()
	localRisks := map[string]float64{
		"default/pod/api-1": 0.5,
	}
	firstAnomalyTimes := map[string]int64{
		"default/pod/api-1": now, // 刚刚出现
	}

	weighted := ApplyTemporalWeights(localRisks, firstAnomalyTimes, now, 300)

	// Δt=0, WTime = floor = 0.7, weighted = 0.5 × 0.7 = 0.35
	expected := 0.5 * TemporalFloor
	if diff := math.Abs(weighted["default/pod/api-1"] - expected); diff > 0.001 {
		t.Errorf("expected %.3f, got %.3f", expected, weighted["default/pod/api-1"])
	}
}

func TestApplyTemporalWeights_OldAnomaly(t *testing.T) {
	now := time.Now().Unix()
	localRisks := map[string]float64{
		"default/pod/api-1": 0.5,
	}
	firstAnomalyTimes := map[string]int64{
		"default/pod/api-1": now - 600, // 10 分钟前
	}

	weighted := ApplyTemporalWeights(localRisks, firstAnomalyTimes, now, 300)

	// Δt=600, τ=300, W = floor + (1-floor) × (1-exp(-2))
	// = 0.7 + 0.3 × 0.8647 = 0.959
	expectedW := TemporalFloor + (1-TemporalFloor)*(1-math.Exp(-2.0))
	expected := 0.5 * expectedW
	if diff := math.Abs(weighted["default/pod/api-1"] - expected); diff > 0.001 {
		t.Errorf("expected %.3f, got %.3f", expected, weighted["default/pod/api-1"])
	}
}

// ==================== Stage 3: 图传播测试 ====================

func TestPropagate_NoEdges(t *testing.T) {
	graph := aiops.NewDependencyGraph("test")
	graph.AddNode("default/node/node1", "node", "", "node1", nil)

	weightedRisks := map[string]float64{
		"default/node/node1": 0.8,
	}

	finalRisks, paths := Propagate(graph, weightedRisks, 0.6)

	// 无依赖: max(R_weighted, α×R_weighted) = R_weighted = 0.8
	expected := 0.8
	if diff := math.Abs(finalRisks["default/node/node1"] - expected); diff > 0.001 {
		t.Errorf("expected %.3f, got %.3f", expected, finalRisks["default/node/node1"])
	}
	if len(paths) != 0 {
		t.Errorf("expected no paths, got %d", len(paths))
	}
}

func TestPropagate_SimpleChain(t *testing.T) {
	// Pod → Node (pod runs_on node)
	graph := aiops.NewDependencyGraph("test")
	graph.AddNode("_cluster/node/node1", "node", "_cluster", "node1", nil)
	graph.AddNode("default/pod/api-1", "pod", "default", "api-1", nil)
	graph.AddEdge("default/pod/api-1", "_cluster/node/node1", "runs_on", 1.0)
	graph.RebuildIndex()

	weightedRisks := map[string]float64{
		"_cluster/node/node1": 0.8,
		"default/pod/api-1":   0.3,
	}

	finalRisks, _ := Propagate(graph, weightedRisks, 0.6)

	// Node (layer=0) 先计算: 无依赖
	// max(0.8, 0.6×0.8) = 0.8
	nodeExpected := 0.8
	if diff := math.Abs(finalRisks["_cluster/node/node1"] - nodeExpected); diff > 0.001 {
		t.Errorf("node: expected %.3f, got %.3f", nodeExpected, finalRisks["_cluster/node/node1"])
	}

	// Pod (layer=1) 后计算: 有下游依赖 node
	// mixed = α × R_weighted(pod) + (1-α) × R_final(node)
	//       = 0.6 × 0.3 + 0.4 × 0.8 = 0.18 + 0.32 = 0.50
	// max(0.3, 0.50) = 0.50 (传播提升了 Pod 风险)
	podMixed := 0.6*0.3 + 0.4*nodeExpected
	podExpected := math.Max(0.3, podMixed)
	if diff := math.Abs(finalRisks["default/pod/api-1"] - podExpected); diff > 0.001 {
		t.Errorf("pod: expected %.3f, got %.3f", podExpected, finalRisks["default/pod/api-1"])
	}
}

func TestPropagate_PropagationPaths(t *testing.T) {
	graph := aiops.NewDependencyGraph("test")
	graph.AddNode("_cluster/node/node1", "node", "_cluster", "node1", nil)
	graph.AddNode("default/pod/api-1", "pod", "default", "api-1", nil)
	graph.AddEdge("default/pod/api-1", "_cluster/node/node1", "runs_on", 1.0)
	graph.RebuildIndex()

	weightedRisks := map[string]float64{
		"_cluster/node/node1": 0.8,
		"default/pod/api-1":   0.3,
	}

	_, paths := Propagate(graph, weightedRisks, 0.6)

	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}
	if paths[0].From != "_cluster/node/node1" || paths[0].To != "default/pod/api-1" {
		t.Errorf("unexpected path: %s → %s", paths[0].From, paths[0].To)
	}
}

// ==================== ClusterRisk 聚合测试 ====================

func TestAggregate_BasicClusterRisk(t *testing.T) {
	entityRisks := map[string]*aiops.EntityRisk{
		"default/service/api": {
			EntityKey: "default/service/api",
			RLocal:    0.5,
			RFinal:    0.8,
		},
		"default/pod/api-1": {
			EntityKey: "default/pod/api-1",
			RLocal:    0.3,
			RFinal:    0.3,
		},
	}
	finalRisks := map[string]float64{
		"default/service/api": 0.8,
		"default/pod/api-1":   0.3,
	}

	config := DefaultRiskConfig()
	now := time.Now().Unix()

	clusterRisk := Aggregate("test", entityRisks, finalRisks, nil, config, now)

	// Risk = w1 × max(R_final) × 100 = 0.5 × 0.8 × 100 = 40
	expectedRisk := 40.0
	if diff := math.Abs(clusterRisk.Risk - expectedRisk); diff > 0.2 {
		t.Errorf("expected risk %.1f, got %.1f", expectedRisk, clusterRisk.Risk)
	}
	if clusterRisk.Level != "low" {
		t.Errorf("expected level 'low', got '%s'", clusterRisk.Level)
	}
	// anomalyCount 统计 rLocal > 0 的实体
	if clusterRisk.AnomalyCount != 2 {
		t.Errorf("expected 2 anomalies, got %d", clusterRisk.AnomalyCount)
	}
}

func TestAggregate_WithSLOContext(t *testing.T) {
	entityRisks := map[string]*aiops.EntityRisk{
		"default/service/api": {
			EntityKey: "default/service/api",
			RFinal:    0.6,
		},
	}
	finalRisks := map[string]float64{
		"default/service/api": 0.6,
	}

	sloCtx := &SLOContext{
		MaxBurnRate:     2.5,
		ErrorGrowthRate: 0.8,
	}
	config := DefaultRiskConfig()
	now := time.Now().Unix()

	clusterRisk := Aggregate("test", entityRisks, finalRisks, sloCtx, config, now)

	// Risk = 0.5×0.6×100 + 0.3×1.0×100 + 0.2×sigmoid(0.8)×100
	// = 30 + 30 + 0.2×sigmoid×100
	// BurnRate>=2.0 → sloBurnFactor=1.0
	// sigmoid(-2*(0.8-0.5)) = sigmoid(-0.6) ≈ 0.354
	if clusterRisk.Risk < 55 || clusterRisk.Risk > 75 {
		t.Errorf("expected risk in [55, 75], got %.1f", clusterRisk.Risk)
	}
}

// ==================== Scorer 端到端测试 ====================

func TestScorer_Calculate_E2E(t *testing.T) {
	// 构建图: Node → Pod → Service
	graph := aiops.NewDependencyGraph("test")
	graph.AddNode("_cluster/node/node1", "node", "_cluster", "node1", nil)
	graph.AddNode("default/pod/api-1", "pod", "default", "api-1", nil)
	graph.AddNode("default/service/api", "service", "default", "api", nil)
	graph.AddEdge("default/pod/api-1", "_cluster/node/node1", "runs_on", 1.0)
	graph.AddEdge("default/service/api", "default/pod/api-1", "selects", 1.0)
	graph.RebuildIndex()

	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "_cluster/node/node1", MetricName: "cpu_usage", IsAnomaly: true, Score: 0.9},
		{EntityKey: "default/pod/api-1", MetricName: "restart_count", IsAnomaly: true, Score: 0.5},
		{EntityKey: "default/service/api", MetricName: "error_rate", IsAnomaly: true, Score: 0.7},
	}

	scorer := NewScorer(nil)
	clusterRisk := scorer.Calculate("test", graph, anomalies, nil)

	if clusterRisk == nil {
		t.Fatal("expected non-nil cluster risk")
	}

	// 验证基本属性
	if clusterRisk.ClusterID != "test" {
		t.Errorf("expected cluster ID 'test', got '%s'", clusterRisk.ClusterID)
	}
	if clusterRisk.Risk < 0 || clusterRisk.Risk > 100 {
		t.Errorf("cluster risk out of range: %.1f", clusterRisk.Risk)
	}
	if clusterRisk.TotalEntities != 3 {
		t.Errorf("expected 3 entities, got %d", clusterRisk.TotalEntities)
	}

	// 验证实体风险
	entities := scorer.GetEntityRisks("test", "r_final", 10)
	if len(entities) != 3 {
		t.Fatalf("expected 3 entity risks, got %d", len(entities))
	}

	// 第一个应该是风险最高的
	if entities[0].RFinal < entities[1].RFinal {
		t.Error("entities not sorted by r_final descending")
	}

	// 所有风险值应该在 [0, 1]
	for _, e := range entities {
		if e.RFinal < 0 || e.RFinal > 1 {
			t.Errorf("entity %s r_final out of range: %.3f", e.EntityKey, e.RFinal)
		}
		if e.RLocal < 0 || e.RLocal > 1 {
			t.Errorf("entity %s r_local out of range: %.3f", e.EntityKey, e.RLocal)
		}
	}
}

func TestScorer_GetEntityRisk(t *testing.T) {
	graph := aiops.NewDependencyGraph("test")
	graph.AddNode("default/pod/api-1", "pod", "default", "api-1", nil)
	graph.RebuildIndex()

	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "default/pod/api-1", MetricName: "restart_count", IsAnomaly: true, Score: 0.8},
	}

	scorer := NewScorer(nil)
	scorer.Calculate("test", graph, anomalies, nil)

	risk := scorer.GetEntityRisk("test", "default/pod/api-1")
	if risk == nil {
		t.Fatal("expected non-nil entity risk")
	}
	if risk.EntityType != "pod" {
		t.Errorf("expected type 'pod', got '%s'", risk.EntityType)
	}
	if risk.RLocal <= 0 {
		t.Error("expected positive r_local")
	}
}

func TestScorer_NonExistentCluster(t *testing.T) {
	scorer := NewScorer(nil)

	if r := scorer.GetClusterRisk("unknown"); r != nil {
		t.Error("expected nil for unknown cluster")
	}
	if r := scorer.GetEntityRisks("unknown", "r_final", 10); r != nil {
		t.Error("expected nil for unknown cluster")
	}
}

func TestScorer_UpdateFirstAnomalyTimes(t *testing.T) {
	graph := aiops.NewDependencyGraph("test")
	graph.AddNode("default/pod/api-1", "pod", "default", "api-1", nil)
	graph.RebuildIndex()

	scorer := NewScorer(nil)

	// 第一次: 有异常
	anomalies1 := []*aiops.AnomalyResult{
		{EntityKey: "default/pod/api-1", MetricName: "restart_count", IsAnomaly: true, Score: 0.5},
	}
	scorer.Calculate("test", graph, anomalies1, nil)

	// 获取首次异常时间
	entityRisk1 := scorer.GetEntityRisk("test", "default/pod/api-1")
	if entityRisk1 == nil || entityRisk1.FirstAnomaly == 0 {
		t.Fatal("expected first anomaly time to be set")
	}
	firstTime := entityRisk1.FirstAnomaly

	// 第二次: 仍然异常，首次时间不变
	anomalies2 := []*aiops.AnomalyResult{
		{EntityKey: "default/pod/api-1", MetricName: "restart_count", IsAnomaly: true, Score: 0.6},
	}
	scorer.Calculate("test", graph, anomalies2, nil)
	entityRisk2 := scorer.GetEntityRisk("test", "default/pod/api-1")
	if entityRisk2.FirstAnomaly != firstTime {
		t.Error("first anomaly time should not change")
	}

	// 第三次: 恢复正常
	anomalies3 := []*aiops.AnomalyResult{
		{EntityKey: "default/pod/api-1", MetricName: "restart_count", IsAnomaly: false, Score: 0.0},
	}
	scorer.Calculate("test", graph, anomalies3, nil)
	entityRisk3 := scorer.GetEntityRisk("test", "default/pod/api-1")
	if entityRisk3 != nil && entityRisk3.FirstAnomaly != 0 {
		t.Error("first anomaly time should be cleared after recovery")
	}
}

// ==================== RiskLevel 映射测试 ====================

func TestRiskLevel(t *testing.T) {
	tests := []struct {
		rFinal   float64
		expected string
	}{
		{0.0, "healthy"},
		{0.1, "healthy"},
		{0.2, "low"},
		{0.4, "medium"},
		{0.6, "high"},
		{0.8, "critical"},
		{1.0, "critical"},
	}

	for _, tt := range tests {
		got := aiops.RiskLevel(tt.rFinal)
		if got != tt.expected {
			t.Errorf("RiskLevel(%.1f) = %s, want %s", tt.rFinal, got, tt.expected)
		}
	}
}

func TestClusterRiskLevel(t *testing.T) {
	tests := []struct {
		risk     float64
		expected string
	}{
		{0, "healthy"},
		{10, "healthy"},
		{20, "low"},
		{50, "warning"},
		{80, "critical"},
		{100, "critical"},
	}

	for _, tt := range tests {
		got := aiops.ClusterRiskLevel(tt.risk)
		if got != tt.expected {
			t.Errorf("ClusterRiskLevel(%.0f) = %s, want %s", tt.risk, got, tt.expected)
		}
	}
}

// ==================== Enhanced: 权重配置测试 ====================

func TestServiceWeights_Enhanced(t *testing.T) {
	config := DefaultRiskConfig()
	serviceConfigs := config.GetMetricConfigs("service")

	// Enhanced 后 service 应包含 APM 和 Log 指标
	requiredMetrics := []string{
		"error_rate", "avg_latency", "request_rate", // Basic SLO
		"apm_error_rate", "apm_p99_latency", // Enhanced APM
		"log_error_count", // Enhanced Log
	}
	for _, name := range requiredMetrics {
		if _, ok := serviceConfigs[name]; !ok {
			t.Errorf("service 权重配置缺少指标: %s", name)
		}
	}

	// 权重之和 = 1.0
	var sum float64
	for _, cfg := range serviceConfigs {
		sum += cfg.Weight
	}
	if math.Abs(sum-1.0) > 0.001 {
		t.Errorf("service 权重之和应为 1.0, got %.3f", sum)
	}
}

func TestNodeWeights_Enhanced(t *testing.T) {
	config := DefaultRiskConfig()
	nodeConfigs := config.GetMetricConfigs("node")

	// Enhanced 后 node 应包含磁盘和 PSI 指标
	requiredMetrics := []string{
		"cpu_usage", "memory_usage", // Basic
		"disk_usage",                      // Enhanced
		"psi_cpu", "psi_memory", "psi_io", // Enhanced
	}
	for _, name := range requiredMetrics {
		if _, ok := nodeConfigs[name]; !ok {
			t.Errorf("node 权重配置缺少指标: %s", name)
		}
	}

	// 权重之和 = 1.0
	var sum float64
	for _, cfg := range nodeConfigs {
		sum += cfg.Weight
	}
	if math.Abs(sum-1.0) > 0.001 {
		t.Errorf("node 权重之和应为 1.0, got %.3f", sum)
	}
}

func TestLogsEntityWeights(t *testing.T) {
	config := DefaultRiskConfig()
	logsConfigs := config.GetMetricConfigs("logs")

	if len(logsConfigs) == 0 {
		t.Fatal("logs 实体类型权重配置不存在")
	}

	// 应包含 log_error_count
	if _, ok := logsConfigs["log_error_count"]; !ok {
		t.Error("logs 权重配置缺少 log_error_count")
	}

	// 权重之和 = 1.0
	var sum float64
	for _, cfg := range logsConfigs {
		sum += cfg.Weight
	}
	if math.Abs(sum-1.0) > 0.001 {
		t.Errorf("logs 权重之和应为 1.0, got %.3f", sum)
	}
}

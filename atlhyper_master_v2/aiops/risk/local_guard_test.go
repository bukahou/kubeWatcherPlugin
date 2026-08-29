package risk

import (
	"testing"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// ──────────────────────────────────────────────────────────────
// 绝对阈值越界必须走确定性通道
// ──────────────────────────────────────────────────────────────
//
// 统计通道是加权和，node 各指标权重合计仅 0.80；硬线基准分 0.6 相乘后
// 天花板是 0.48 < incident 线 0.5 —— 即便所有硬线指标同时越界也升不上去，
// 等于硬线永远无法独立产生 incident。
//
// 确定性通道 max(score)×breadthBoost 才有正确的分级语义：
// 单指标越界 = warning（该看一眼），两个以上 = 节点真出事（incident）。
// 语义上也吻合：该通道本就是给「确定是坏的」信号用的（CrashLoopBackOff、
// OOMKilled），硬线越界属同一类。

func breachResult(metric string, score float64) *aiops.AnomalyResult {
	return &aiops.AnomalyResult{
		EntityKey: "_cluster/node/desk-zero", MetricName: metric,
		Score: score, IsAnomaly: true, AbsoluteBreach: true,
	}
}

func TestComputeLocalRisk_AbsoluteBreachRoutesToDeterministic(t *testing.T) {
	// 单指标越界：0.636 × breadthBoost(1)=0.70 = 0.445 → warning 档
	single := ComputeLocalRisks([]*aiops.AnomalyResult{
		breachResult("cpu_usage", 0.636),
	}, DefaultRiskConfig())
	got := single["_cluster/node/desk-zero"]
	if got <= 0.2 {
		t.Errorf("单指标越界 rLocal=%.3f，应 > 0.2（至少进 warning）——统计通道只给 0.127", got)
	}
	if got > 0.5 {
		t.Errorf("单指标越界 rLocal=%.3f，不应直接到 incident（0.5）——避免告警疲劳", got)
	}

	// 两指标越界：0.636 × 0.85 = 0.540 → incident
	double := ComputeLocalRisks([]*aiops.AnomalyResult{
		breachResult("cpu_usage", 0.636),
		breachResult("memory_usage", 0.622),
	}, DefaultRiskConfig())
	got2 := double["_cluster/node/desk-zero"]
	if got2 <= 0.5 {
		t.Errorf("两指标同时越界 rLocal=%.3f，应 > 0.5（节点确已出事）", got2)
	}
}

// 非越界的统计异常不得被改道 —— 原有行为必须保持
func TestComputeLocalRisk_StatisticalUnchanged(t *testing.T) {
	r := &aiops.AnomalyResult{
		EntityKey: "_cluster/node/desk-zero", MetricName: "cpu_usage",
		Score: 0.9, IsAnomaly: true, AbsoluteBreach: false,
	}
	got := ComputeLocalRisks([]*aiops.AnomalyResult{r}, DefaultRiskConfig())["_cluster/node/desk-zero"]
	want := 0.20 * 0.9 // 统计通道加权
	if got < want-0.001 || got > want+0.001 {
		t.Errorf("普通统计异常 rLocal=%.4f, want %.4f（应仍走统计通道）", got, want)
	}
}

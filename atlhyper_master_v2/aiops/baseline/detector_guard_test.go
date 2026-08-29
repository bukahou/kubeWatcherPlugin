package baseline

import (
	"math"
	"testing"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// ──────────────────────────────────────────────────────────────
// A：偏离度口径一致性
// ──────────────────────────────────────────────────────────────
//
// 方差用 oldEMA、偏离度却用更新后的 EMA —— 同一函数两种口径，
// 系统性低估偏离 (1-α)=3.3%。两者都应以「本次观测前的基线」为准。

func TestDetect_DeviationUsesPreUpdateEMA(t *testing.T) {
	// 构造已就绪状态：EMA=100，方差=25（σ=5）
	st := &aiops.BaselineState{
		EntityKey: "ns/pod/p", MetricName: "cpu_usage",
		EMA: 100, Variance: 25, Count: aiops.ColdStartMinCount + 1,
	}
	_, res := Detect(st, 120, 0) // 偏离应为 (120-100)/5 = 4.0σ
	if res == nil {
		t.Fatal("就绪状态应产出结果")
	}
	if math.Abs(res.Deviation-4.0) > 0.01 {
		t.Errorf("Deviation = %.3f, want 4.0 —— 若先更新再比较会得到 3.27（方差自我毒化）", res.Deviation)
	}
	// Baseline 字段同样应报告「比较时用的基线」，而非已污染的新值
	if math.Abs(res.Baseline-100) > 0.01 {
		t.Errorf("Baseline = %.3f, want 100（本次观测前的基线）", res.Baseline)
	}
}

// ──────────────────────────────────────────────────────────────
// B：绝对阈值兜底
// ──────────────────────────────────────────────────────────────
//
// 2026-08-29 压测实证：desk-zero CPU 打到 98%、gateway p99 到 2094ms，
// AIOps 全程判 healthy —— 负载数小时内渐进爬升，每一步都在 3σ 内，
// 而每一步又同时抬高 EMA 与 σ（温水煮青蛙）。
// 统计检测必须有静态阈值兜底：某些值无论「常态」如何都是坏的。

func TestDetect_AbsoluteGuard_FiresRegardlessOfBaseline(t *testing.T) {
	// 最恶劣情形：基线已被压测养到 60%，σ 极大 —— 统计上 98% 毫不异常
	st := &aiops.BaselineState{
		EntityKey: "_cluster/node/desk-zero", MetricName: "cpu_usage",
		EMA: 60, Variance: 400, Count: aiops.ColdStartMinCount + 1, // σ=20
	}
	_, res := Detect(st, 98, 0)
	if res == nil {
		t.Fatal("应产出结果")
	}
	if res.Deviation >= aiops.AnomalyThreshold {
		t.Fatalf("前置条件失效：偏离 %.2fσ 已超阈值，测不出兜底作用", res.Deviation)
	}
	if !res.IsAnomaly {
		t.Errorf("CPU 98%% 必须判异常（绝对阈值兜底），当前 IsAnomaly=false 偏离=%.2fσ", res.Deviation)
	}
	if !res.AbsoluteBreach {
		t.Error("应标记 AbsoluteBreach，供上层区分「统计异常」与「触碰硬线」")
	}
}

func TestDetect_AbsoluteGuard_Boundaries(t *testing.T) {
	tests := []struct {
		metric string
		value  float64
		want   bool
	}{
		{"cpu_usage", 89, false},
		{"cpu_usage", 90, true},  // 边界：达到即触发
		{"memory_usage", 95, true},
		{"memory_usage", 80, false},
		{"disk_usage", 90, true},
		{"psi_cpu", 60, true},
		{"apm_error_rate", 5, true}, // 错误率 5%
		{"apm_error_rate", 1, false},
		{"apm_rps", 99999, false},   // 无硬线的指标不受影响
		{"restart_count", 3, false}, // 计数类由统计通道负责
	}
	for _, tt := range tests {
		st := &aiops.BaselineState{
			EntityKey: "e", MetricName: tt.metric,
			EMA: tt.value, Variance: 1e6, // 方差巨大 → 统计通道必然判正常
			Count: aiops.ColdStartMinCount + 1,
		}
		_, res := Detect(st, tt.value, 0)
		if res.AbsoluteBreach != tt.want {
			t.Errorf("%s=%v → AbsoluteBreach=%v, want %v", tt.metric, tt.value, res.AbsoluteBreach, tt.want)
		}
	}
}

// 冷启动期不得因绝对阈值告警 —— 冷启动的语义是「还没资格判断」
func TestDetect_AbsoluteGuard_SilentDuringColdStart(t *testing.T) {
	st := &aiops.BaselineState{
		EntityKey: "e", MetricName: "cpu_usage", Count: 5,
	}
	_, res := Detect(st, 99, 0)
	if res != nil {
		t.Errorf("冷启动期应只学习不告警，得到 %+v", res)
	}
}

// 先验方差为 0 时的语义：任何变化都无穷显著，不得判为「无偏离」。
// 2026-08-29 改为「先验判定后更新」时一度打破零值快速通道（restart_count
// 由 0 跳到 5 不再告警），此测试钉住该语义。
func TestDetect_ZeroPriorVariance(t *testing.T) {
	base := func() *aiops.BaselineState {
		return &aiops.BaselineState{
			EntityKey: "e", MetricName: "restart_count",
			EMA: 0, Variance: 0, Count: aiops.ColdStartMinCount + 1,
		}
	}
	// 值发生变化 → 无穷显著
	_, res := Detect(base(), 5, 0)
	if !res.IsAnomaly || res.Deviation < aiops.AnomalyThreshold {
		t.Errorf("零方差下值由 0 变 5 应判异常, deviation=%.2f isAnomaly=%v", res.Deviation, res.IsAnomaly)
	}
	// 值未变 → 确实无偏离，不得误报
	_, res2 := Detect(base(), 0, 0)
	if res2.IsAnomaly || res2.Deviation != 0 {
		t.Errorf("零方差且值未变不应告警, deviation=%.2f", res2.Deviation)
	}
}

// 绝对阈值不仅要「判异常」，还必须给出与严重性匹配的 Score ——
// 否则它只是个标签：风险评分用 Score 加权（rLocal += weight × score），
// 而硬线场景的统计偏离天然很低，压测实测 CPU 98% 只得 0.0998 分，
// × 权重 0.20 = 0.02，永远够不到 incident 触发线 0.5。
func TestDetect_AbsoluteGuard_ScoreReflectsSeverity(t *testing.T) {
	mk := func(metric string, ema, variance, value float64) *aiops.AnomalyResult {
		st := &aiops.BaselineState{
			EntityKey: "e", MetricName: metric,
			EMA: ema, Variance: variance, Count: aiops.ColdStartMinCount + 1,
		}
		_, res := Detect(st, value, 0)
		return res
	}

	// 压测实况：CPU 98%，基线已被养到 60%，σ=20 → 统计上仅 1.9σ
	res := mk("cpu_usage", 60, 400, 98)
	if !res.AbsoluteBreach {
		t.Fatal("前置条件：应触发硬线")
	}
	if res.Score < aiops.AbsoluteBreachScore {
		t.Errorf("硬线场景 Score=%.4f，应至少为 %.2f —— 否则加权后够不到 incident 线",
			res.Score, aiops.AbsoluteBreachScore)
	}

	// 超得越多分越高
	mild := mk("cpu_usage", 60, 400, 90)   // 恰好触线
	severe := mk("cpu_usage", 60, 400, 100) // 满载
	if severe.Score <= mild.Score {
		t.Errorf("超线越多分应越高：90%%→%.4f, 100%%→%.4f", mild.Score, severe.Score)
	}

	// 统计分数更高时不得被硬线拉低（取两者较大值）
	bigDev := mk("cpu_usage", 90, 1, 98) // 8σ，统计分数接近 1
	if bigDev.Score < 0.9 {
		t.Errorf("统计上的极端异常不应被硬线基准拉低，Score=%.4f", bigDev.Score)
	}
}

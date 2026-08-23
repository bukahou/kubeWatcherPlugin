package slo

import (
	"math"
	"testing"
)

func almost(a, b float64) bool { return math.Abs(a-b) < 0.01 }

// 燃烧率 = 实际错误率 ÷ 允许错误率。
// 归一化后跨服务可比：1× 表示正好在窗口结束时用完预算，14.4× 表示 1 小时烧掉 2% 月度预算。
func TestCalculateBurnRate(t *testing.T) {
	cases := []struct {
		name      string
		errorRate float64 // 实际错误率（%）
		target    float64 // 目标可用率（%）
		want      float64
	}{
		{"错误率正好等于预算", 0.5, 99.5, 1},
		{"烧得比预算快 3 倍", 1.5, 99.5, 3},
		{"零错误", 0, 99.5, 0},
		{"14.4 倍（Google 呼叫线）", 7.2, 99.5, 14.4},
		{"目标 99%，错误 2%", 2, 99, 2},
		// 目标 100% 时不存在错误预算：任何错误都是无穷大的燃烧率，
		// 但返回 Inf 会污染前端渲染，用 0 错误 → 0，有错误 → 一个明确的上限值
		{"目标 100% 且无错误", 0, 100, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CalculateBurnRate(c.errorRate, c.target); !almost(got, c.want) {
				t.Errorf("CalculateBurnRate(%v, %v) = %v, want %v", c.errorRate, c.target, got, c.want)
			}
		})
	}
}

func TestCalculateBurnRate_ZeroBudget(t *testing.T) {
	// 目标 100% + 有错误：没有预算可烧，返回上限值而非 +Inf
	got := CalculateBurnRate(0.1, 100)
	if math.IsInf(got, 0) || got <= 0 {
		t.Errorf("目标 100%% 有错误时应返回有限的上限值，得到 %v", got)
	}
	if got != maxBurnRate {
		t.Errorf("= %v，期望上限 %v", got, maxBurnRate)
	}
}

// 错误预算改用事件计数口径：允许错多少个、已经错了多少个。
// 比「剩余 82.3%」直观得多，且分子分母都能显示在面板上。
func TestCalculateErrorBudget(t *testing.T) {
	cases := []struct {
		name             string
		total, bad       int64
		target           float64
		wantAllowed      int64
		wantConsumed     int64
		wantRemainingPct float64
	}{
		{"1420 请求 / 目标 99.5% / 错 1 个", 1420, 1, 99.5, 7, 1, 85.71},
		{"预算刚好用完", 1000, 5, 99.5, 5, 5, 0},
		{"超支", 1000, 10, 99.5, 5, 10, -100},
		{"零错误", 1000, 0, 99.5, 5, 0, 100},
		{"无流量", 0, 0, 99.5, 0, 0, 100},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := CalculateErrorBudget(c.total, c.bad, c.target)
			if b.AllowedEvents != c.wantAllowed {
				t.Errorf("AllowedEvents = %d, want %d", b.AllowedEvents, c.wantAllowed)
			}
			if b.ConsumedEvents != c.wantConsumed {
				t.Errorf("ConsumedEvents = %d, want %d", b.ConsumedEvents, c.wantConsumed)
			}
			if !almost(b.RemainingPct, c.wantRemainingPct) {
				t.Errorf("RemainingPct = %v, want %v", b.RemainingPct, c.wantRemainingPct)
			}
		})
	}
}

// 超支不应无限放大：-100% 是显示下限，否则进度条没法画。
func TestCalculateErrorBudget_OverspendClamped(t *testing.T) {
	b := CalculateErrorBudget(1000, 500, 99.5)
	if b.RemainingPct != -100 {
		t.Errorf("严重超支应钳到 -100%%，得到 %v", b.RemainingPct)
	}
	// 但真实计数不钳 —— 面板要显示「允许 5 个，实际错了 500 个」
	if b.ConsumedEvents != 500 || b.AllowedEvents != 5 {
		t.Errorf("事件计数不应被钳: allowed=%d consumed=%d", b.AllowedEvents, b.ConsumedEvents)
	}
}

// 多窗口判定：每个窗口有自己的阈值，超过即标记。
func TestBurnRateStatus(t *testing.T) {
	cases := []struct {
		window string
		rate   float64
		want   string
	}{
		{"1h", 20, "crit"}, // > 14.4
		{"1h", 10, "good"}, // 1h 窗口只有 crit 一档
		{"6h", 8, "warn"},  // > 6
		{"6h", 3, "good"},
		{"24h", 4, "warn"},  // > 3
		{"3d", 1.5, "warn"}, // > 1
		{"3d", 0.5, "good"},
	}
	for _, c := range cases {
		t.Run(c.window, func(t *testing.T) {
			if got := burnRateStatus(c.window, c.rate); got != c.want {
				t.Errorf("burnRateStatus(%s, %v) = %q, want %q", c.window, c.rate, got, c.want)
			}
		})
	}
}

// 预算耗尽预估：按当前燃烧率，还有多久烧完剩余预算。
func TestEstimateExhaustHours(t *testing.T) {
	// 窗口 7 天 = 168 小时，剩 50% 预算，燃烧率 2× → 168 × 0.5 / 2 = 42 小时
	if got := EstimateExhaustHours(50, 2, 7); !almost(got, 42) {
		t.Errorf("= %v, want 42", got)
	}
	// 燃烧率 <= 1 时窗口内烧不完，返回 0 表示「不会耗尽」
	if got := EstimateExhaustHours(50, 0.5, 7); got != 0 {
		t.Errorf("燃烧率低于 1 应返回 0（不会耗尽），得到 %v", got)
	}
	// 预算已用完
	if got := EstimateExhaustHours(0, 5, 7); got != 0 {
		t.Errorf("预算已耗尽应返回 0，得到 %v", got)
	}
}

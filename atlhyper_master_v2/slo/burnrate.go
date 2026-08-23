// burnrate.go 燃烧率与错误预算
//
// 燃烧率是 SLO 面板里唯一能跨服务横向比较的指标：
// 一个目标 99.9% 的服务错 0.05%，和一个目标 99% 的服务错 0.05%，
// 原始错误率一样，但前者只烧了半格预算、后者连零头都算不上。
// 归一化之后「谁最危险」才有意义。
//
// 参考 Google SRE Workbook《Alerting on SLOs》的多窗口多燃烧率模型。
package slo

import "math"

// maxBurnRate 目标定为 100%（没有错误预算）时的燃烧率上限。
// 数学上应该是 +Inf，但那会让 JSON 序列化和前端进度条都出问题，
// 用一个明确大于所有告警阈值的有限值代替。
const maxBurnRate = 999

// 多窗口阈值。来自 Google SRE Workbook 的推荐值：
// 1h 超 14.4× 意味着 1 小时烧掉 2% 的月度预算，该立刻呼叫；
// 3d 超 1× 只是说明照这个速度会超支，放周会上看即可。
var burnRateThresholds = map[string]float64{
	"1h":  14.4,
	"6h":  6,
	"24h": 3,
	"3d":  1,
}

// BurnRateWindow 一个窗口的燃烧率判定结果
type BurnRateWindow struct {
	Window    string  `json:"window"`
	Rate      float64 `json:"rate"`
	Threshold float64 `json:"threshold"`
	Status    string  `json:"status"` // good / warn / crit
}

// ErrorBudget 错误预算（事件计数口径）
//
// 用「允许错 7 个，已经错了 1 个」而不是「剩余 85.7%」——
// 后者在低流量时极具误导性：总共 10 个请求错了 1 个，百分比很难看，
// 但实际上什么都没发生。分子分母摆出来，读的人自己能判断。
type ErrorBudget struct {
	AllowedEvents  int64   `json:"allowedEvents"`
	ConsumedEvents int64   `json:"consumedEvents"`
	RemainingPct   float64 `json:"remainingPct"`
}

// CalculateBurnRate 燃烧率 = 实际错误率 ÷ 允许错误率。
// errorRate 与 targetAvailability 都是百分数（如 0.5 表示 0.5%、99.5 表示 99.5%）。
func CalculateBurnRate(errorRate, targetAvailability float64) float64 {
	allowed := 100 - targetAvailability
	if allowed <= 0 {
		// 目标 100%：没有预算可烧
		if errorRate <= 0 {
			return 0
		}
		return maxBurnRate
	}
	rate := errorRate / allowed
	if rate > maxBurnRate {
		return maxBurnRate
	}
	return rate
}

// CalculateErrorBudget 按事件计数算预算。
// totalRequests 为 0 时返回满预算 —— 没有流量不等于出了问题。
func CalculateErrorBudget(totalRequests, badRequests int64, targetAvailability float64) ErrorBudget {
	allowedRatio := (100 - targetAvailability) / 100
	allowed := int64(math.Floor(float64(totalRequests) * allowedRatio))

	b := ErrorBudget{
		AllowedEvents:  allowed,
		ConsumedEvents: badRequests,
		RemainingPct:   100,
	}
	if allowed <= 0 {
		// 流量太少还凑不满一个允许出错的名额：此时没错就是满预算，
		// 有错就是超支 —— 但别把它算成 -100%，低流量下一个错误不代表服务坏了。
		if badRequests > 0 {
			b.RemainingPct = 0
		}
		return b
	}

	remaining := float64(allowed-badRequests) / float64(allowed) * 100
	if remaining < -100 {
		remaining = -100 // 显示下限，否则进度条没法画；真实计数保留在 ConsumedEvents
	}
	b.RemainingPct = remaining
	return b
}

// burnRateStatus 按窗口阈值判定。1h 窗口只有 crit 一档（超了就该呼叫），
// 其余窗口超阈值算 warn。
func burnRateStatus(window string, rate float64) string {
	threshold, ok := burnRateThresholds[window]
	if !ok || rate <= threshold {
		return "good"
	}
	if window == "1h" {
		return "crit"
	}
	return "warn"
}

// BuildBurnRateWindow 组装单个窗口的判定结果
func BuildBurnRateWindow(window string, errorRate, targetAvailability float64) BurnRateWindow {
	rate := CalculateBurnRate(errorRate, targetAvailability)
	return BurnRateWindow{
		Window:    window,
		Rate:      round2(rate),
		Threshold: burnRateThresholds[window],
		Status:    burnRateStatus(window, rate),
	}
}

// EstimateExhaustHours 按当前燃烧率，剩余预算还能撑多少小时。
// 返回 0 表示「窗口内不会耗尽」（燃烧率 ≤ 1）或「已经耗尽」。
func EstimateExhaustHours(remainingPct, burnRate float64, windowDays int) float64 {
	if remainingPct <= 0 || burnRate <= 1 || windowDays <= 0 {
		return 0
	}
	windowHours := float64(windowDays) * 24
	return round2(windowHours * (remainingPct / 100) / burnRate)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

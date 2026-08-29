// atlhyper_master_v2/aiops/baseline/detector.go
// EMA + 3σ 异常检测算法
package baseline

import (
	"math"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// Detect 对单个指标执行异常检测
// 返回更新后的状态和异常结果（冷启动期间 result 为 nil）
func Detect(state *aiops.BaselineState, value float64, now int64) (*aiops.BaselineState, *aiops.AnomalyResult) {
	state.Count++
	alpha := aiops.DefaultAlpha

	// 冷启动：只学习，不告警
	if state.Count <= aiops.ColdStartMinCount {
		// 零值计数器快速通道：
		// restart_count / not_ready_containers 等指标正常值恒为 0，
		// 连续 10 个零值建立基线后，首个非零值立即触发正常检测
		// 注意：先检查快速通道条件，再更新 ConsecutiveZero
		if state.ConsecutiveZero >= int64(aiops.ColdStartZeroFastTrack) && value > 0 {
			// 快速通道：基线已确定为 0，跳到正常检测（不在此处更新 EMA，由下方统一处理）
			state.Count = int64(aiops.ColdStartMinCount) + 1
		} else {
			if value == 0 {
				state.ConsecutiveZero++
			} else {
				state.ConsecutiveZero = 0
			}
			// 标准冷启动：更新 EMA 但不产出结果
			if state.Count == 1 {
				state.EMA = value
				state.Variance = 0
			} else {
				state.EMA = alpha*value + (1-alpha)*state.EMA
				diff := value - state.EMA
				state.Variance = alpha*diff*diff + (1-alpha)*state.Variance
			}
			state.UpdatedAt = now
			return state, nil
		}
	}

	// 先用「本次观测之前」的分布做判定，再更新状态。
	//
	// 顺序很关键：旧实现先更新 EMA 与方差、再比较，等于让观测参与
	// 定义自己的正常范围 —— 一个大偏离会先抬高自身的 σ，再用被抬高的
	// σ 去衡量自己，偏离被系统性压低（EMA 侧 3.3%，方差侧更多：
	// 上例 4.00σ 被压到 3.27σ，少 18%）。这正是「方差自我毒化」，
	// 也是 2026-08-29 压测中渐进劣化始终测不出的机制之一。
	// EWMA 控制图的标准做法：以先验分布判定，判定后再吸收观测。
	oldEMA := state.EMA
	sigma := math.Sqrt(state.Variance)
	var deviation float64
	switch {
	case sigma > 1e-9:
		deviation = math.Abs(value-oldEMA) / sigma
	case math.Abs(value-oldEMA) > 1e-9:
		// 先验方差为 0：该指标从未波动过，任何变化都无穷显著。
		// 不可简单返回 0 —— 那会让 restart_count 从 0 跳到 5 也判正常。
		deviation = aiops.MaxDeviation
	default:
		deviation = 0 // 方差为 0 且值未变：确实无偏离
	}

	// 判定完成，更新状态
	state.EMA = alpha*value + (1-alpha)*state.EMA
	diff := value - oldEMA
	state.Variance = alpha*diff*diff + (1-alpha)*state.Variance
	state.UpdatedAt = now

	// 归一化到 [0, 1]
	score := sigmoid(deviation, aiops.AnomalyThreshold, aiops.SigmoidK)

	// 绝对阈值兜底：渐进劣化会被 EMA 学成常态（温水煮青蛙），
	// 统计通道对此结构性失明，须由静态硬线补上。详见 AbsoluteThresholds。
	absoluteBreach := false
	if limit, ok := aiops.AbsoluteThresholds[state.MetricName]; ok && value >= limit {
		absoluteBreach = true
	}

	result := &aiops.AnomalyResult{
		EntityKey:      state.EntityKey,
		MetricName:     state.MetricName,
		CurrentValue:   value,
		Baseline:       oldEMA,
		Deviation:      deviation,
		Score:          score,
		IsAnomaly:      deviation > aiops.AnomalyThreshold || absoluteBreach,
		AbsoluteBreach: absoluteBreach,
		DetectedAt:     now,
	}

	return state, result
}

// sigmoid 归一化函数
// score = 1 / (1 + exp(-k * (deviation - threshold)))
func sigmoid(deviation, threshold, k float64) float64 {
	return 1.0 / (1.0 + math.Exp(-k*(deviation-threshold)))
}

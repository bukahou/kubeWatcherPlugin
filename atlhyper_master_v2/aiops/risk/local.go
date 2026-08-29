// atlhyper_master_v2/aiops/risk/local.go
// Stage 1: 局部风险计算 — 双通道评分
//
// 通道 1 (统计): channel1 = Σ(w_i × score_i)          — statistical + both 参与
// 通道 2 (确定性): channel2 = max(score_i) × breadthBoost(n) — deterministic + both 参与
// R_local = max(channel1, channel2)
package risk

import "AtlHyper/atlhyper_master_v2/aiops"

// breadthBoost 返回确定性通道的广度因子
//
//	n=0 → 0.0, n=1 → 0.70, n=2 → 0.85, n≥3 → 1.0
func breadthBoost(n int) float64 {
	switch {
	case n <= 0:
		return 0.0
	case n == 1:
		return 0.70
	case n == 2:
		return 0.85
	default:
		return 1.0
	}
}

// ComputeLocalRisks 计算每个实体的局部风险分数 (双通道)
func ComputeLocalRisks(anomalies []*aiops.AnomalyResult, config *RiskConfig) map[string]float64 {
	// 按 entityKey 分组
	byEntity := map[string][]*aiops.AnomalyResult{}
	for _, a := range anomalies {
		byEntity[a.EntityKey] = append(byEntity[a.EntityKey], a)
	}

	localRisks := make(map[string]float64, len(byEntity))
	for entityKey, results := range byEntity {
		entityType := aiops.ExtractEntityType(entityKey)
		metricConfigs := config.GetMetricConfigs(entityType)

		var channel1 float64 // 统计通道: Σ(w_i × score_i)
		var maxScore float64 // 确定性通道: max(score_i)
		var detCount int     // 确定性通道参与指标数

		for _, r := range results {
			if !r.IsAnomaly {
				continue
			}

			mc, ok := metricConfigs[r.MetricName]
			if !ok {
				// 未配置的指标: 默认 statistical, weight=0.1
				mc = MetricConfig{Weight: 0.1, Channel: ChannelStatistical}
			}

			// 绝对阈值越界一律走确定性通道，不论该指标平时属于哪个通道。
			//
			// 理由（2026-08-29）：统计通道是加权和，node 各指标权重合计仅 0.80，
			// 硬线基准分 0.6 相乘后天花板 0.48 < incident 线 0.5 ——
			// 即便所有硬线指标同时越界也升不上去，硬线等于永远无法独立成事件。
			// 确定性通道 max(score)×breadthBoost 才有正确分级：
			// 单指标越界=warning（该看一眼），两个以上=节点真出事（incident）。
			// 语义上也吻合：该通道本就是给「确定是坏的」信号用的
			// （CrashLoopBackOff / OOMKilled），硬线越界属同一类。
			if r.AbsoluteBreach {
				detCount++
				if r.Score > maxScore {
					maxScore = r.Score
				}
				continue
			}

			// 通道 1: statistical + both 参与
			if mc.Channel == ChannelStatistical || mc.Channel == ChannelBoth {
				channel1 += mc.Weight * r.Score
			}

			// 通道 2: deterministic + both 参与
			if mc.Channel == ChannelDeterministic || mc.Channel == ChannelBoth {
				detCount++
				if r.Score > maxScore {
					maxScore = r.Score
				}
			}
		}

		channel2 := maxScore * breadthBoost(detCount)
		rLocal := channel1
		if channel2 > rLocal {
			rLocal = channel2
		}

		// 截断到 [0, 1]
		if rLocal > 1.0 {
			rLocal = 1.0
		}
		if rLocal > 0 {
			localRisks[entityKey] = rLocal
		}
	}

	return localRisks
}

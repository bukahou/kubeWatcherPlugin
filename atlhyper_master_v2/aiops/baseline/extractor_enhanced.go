// atlhyper_master_v2/aiops/baseline/extractor_enhanced.go
// Enhanced 层：从 OTelSnapshot 提取 APM / 日志 / 深度 Node 指标 + 确定性异常
// Basic 层函数在 extractor.go 中，本文件不依赖也不修改 Basic 层逻辑
package baseline

import (
	"fmt"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/model_v3/cluster"
)

// ==================== 确定性异常阈值 ====================

const (
	apmErrorRateThreshold  = 0.15 // APM error rate > 15%
	apmP99LatencyThreshold = 5000 // APM P99 > 5000ms
	logErrorCountThreshold = 500  // 全局 ERROR 日志 > 500 条/5min
)

// ==================== 指标提取 ====================

// extractAPMMetrics 从 APMServices 提取 APM 指标
func extractAPMMetrics(otel *cluster.OTelSnapshot) []aiops.MetricDataPoint {
	if otel == nil || len(otel.APMServices) == 0 {
		return nil
	}

	points := make([]aiops.MetricDataPoint, 0, len(otel.APMServices)*3)
	for _, svc := range otel.APMServices {
		key := aiops.EntityKey(svc.Namespace, "service", svc.Name)
		errorRate := 1 - svc.SuccessRate

		points = append(points,
			aiops.MetricDataPoint{EntityKey: key, MetricName: "apm_error_rate", Value: errorRate},
			aiops.MetricDataPoint{EntityKey: key, MetricName: "apm_p99_latency", Value: svc.P99Ms},
			aiops.MetricDataPoint{EntityKey: key, MetricName: "apm_rps", Value: svc.RPS},
		)
	}
	return points
}

// extractLogMetrics 从 LogsSummary + RecentLogs 提取日志指标
func extractLogMetrics(otel *cluster.OTelSnapshot) []aiops.MetricDataPoint {
	if otel == nil {
		return nil
	}
	if otel.LogsSummary == nil && len(otel.RecentLogs) == 0 {
		return nil
	}

	var points []aiops.MetricDataPoint

	// 全局日志计数（来自 Summary）
	if otel.LogsSummary != nil && len(otel.LogsSummary.SeverityCounts) > 0 {
		globalKey := aiops.EntityKey("_cluster", "logs", "global")
		if errCount, ok := otel.LogsSummary.SeverityCounts["ERROR"]; ok && errCount > 0 {
			points = append(points, aiops.MetricDataPoint{
				EntityKey: globalKey, MetricName: "log_error_count", Value: float64(errCount),
			})
		}
		if warnCount, ok := otel.LogsSummary.SeverityCounts["WARN"]; ok && warnCount > 0 {
			points = append(points, aiops.MetricDataPoint{
				EntityKey: globalKey, MetricName: "log_warn_count", Value: float64(warnCount),
			})
		}
	}

	// 服务级日志计数（从 RecentLogs 聚合）
	if len(otel.RecentLogs) > 0 {
		// service+severity → count
		type svcSev struct {
			service   string
			namespace string
			severity  string
		}
		counts := make(map[svcSev]float64)

		for i := range otel.RecentLogs {
			entry := &otel.RecentLogs[i]
			if entry.ServiceName == "" {
				continue
			}
			// 只统计 ERROR 和 WARN
			if entry.Severity != "ERROR" && entry.Severity != "WARN" {
				continue
			}
			ns := entry.Resource["k8s.namespace.name"]
			if ns == "" {
				ns = "default"
			}
			counts[svcSev{service: entry.ServiceName, namespace: ns, severity: entry.Severity}]++
		}

		for k, count := range counts {
			key := aiops.EntityKey(k.namespace, "service", k.service)
			metricName := "log_error_count"
			if k.severity == "WARN" {
				metricName = "log_warn_count"
			}
			points = append(points, aiops.MetricDataPoint{
				EntityKey: key, MetricName: metricName, Value: count,
			})
		}
	}

	return points
}

// extractEnhancedNodeMetrics 从 OTel MetricsNodes 提取深度 Node 指标（磁盘 + PSI）
func extractEnhancedNodeMetrics(otel *cluster.OTelSnapshot) []aiops.MetricDataPoint {
	if otel == nil || len(otel.MetricsNodes) == 0 {
		return nil
	}

	var points []aiops.MetricDataPoint
	for i := range otel.MetricsNodes {
		node := &otel.MetricsNodes[i]
		key := aiops.EntityKey("_cluster", "node", node.NodeName)

		// 磁盘使用率（主磁盘）
		if disk := node.GetPrimaryDisk(); disk != nil {
			points = append(points, aiops.MetricDataPoint{
				EntityKey: key, MetricName: "disk_usage", Value: disk.UsagePct,
			})
		}

		// PSI 指标
		points = append(points,
			aiops.MetricDataPoint{EntityKey: key, MetricName: "psi_cpu", Value: node.PSI.CPUSomePct},
			aiops.MetricDataPoint{EntityKey: key, MetricName: "psi_memory", Value: node.PSI.MemSomePct},
			aiops.MetricDataPoint{EntityKey: key, MetricName: "psi_io", Value: node.PSI.IOSomePct},
		)
	}
	return points
}

// ==================== 确定性异常 ====================

// ExtractOTelDeterministicAnomalies 从 OTelSnapshot 提取确定性异常
// 独立于 Basic 层的 ExtractDeterministicAnomalies，由 engine.go 分别调用
func ExtractOTelDeterministicAnomalies(otel *cluster.OTelSnapshot) []*aiops.AnomalyResult {
	if otel == nil {
		return nil
	}

	now := time.Now().Unix()
	var results []*aiops.AnomalyResult

	// APM 确定性异常
	for _, svc := range otel.APMServices {
		key := aiops.EntityKey(svc.Namespace, "service", svc.Name)
		errorRate := 1 - svc.SuccessRate

		// 高错误率 > 15%
		if errorRate > apmErrorRateThreshold {
			results = append(results, &aiops.AnomalyResult{
				EntityKey:    key,
				MetricName:   "apm_high_error_rate",
				CurrentValue: errorRate,
				Baseline:     apmErrorRateThreshold,
				Deviation:    (errorRate - apmErrorRateThreshold) / apmErrorRateThreshold * 10,
				Score:        apmErrorRateScore(errorRate),
				IsAnomaly:    true,
				DetectedAt:   now,
			})
		}

		// 高 P99 延迟 > 5000ms
		if svc.P99Ms > apmP99LatencyThreshold {
			results = append(results, &aiops.AnomalyResult{
				EntityKey:    key,
				MetricName:   "apm_high_p99_latency",
				CurrentValue: svc.P99Ms,
				Baseline:     apmP99LatencyThreshold,
				Deviation:    (svc.P99Ms - apmP99LatencyThreshold) / apmP99LatencyThreshold * 10,
				Score:        apmP99Score(svc.P99Ms),
				IsAnomaly:    true,
				DetectedAt:   now,
			})
		}
	}

	// 全局日志错误尖峰
	if otel.LogsSummary != nil {
		if errCount, ok := otel.LogsSummary.SeverityCounts["ERROR"]; ok && errCount > logErrorCountThreshold {
			results = append(results, &aiops.AnomalyResult{
				EntityKey:    aiops.EntityKey("_cluster", "logs", "global"),
				MetricName:   "log_error_spike",
				CurrentValue: float64(errCount),
				Baseline:     logErrorCountThreshold,
				Deviation:    float64(errCount-logErrorCountThreshold) / logErrorCountThreshold * 10,
				Score:        logErrorScore(errCount),
				IsAnomaly:    true,
				DetectedAt:   now,
			})
		}
	}

	return results
}

// ==================== 评分函数 ====================

// apmErrorRateScore error rate → 风险分数
func apmErrorRateScore(errorRate float64) float64 {
	switch {
	case errorRate >= 0.50:
		return 0.95
	case errorRate >= 0.30:
		return 0.85
	default:
		return 0.75
	}
}

// apmP99Score P99 延迟 → 风险分数
func apmP99Score(p99Ms float64) float64 {
	switch {
	case p99Ms >= 10000:
		return 0.90
	case p99Ms >= 7000:
		return 0.80
	default:
		return 0.75
	}
}

// logErrorScore 日志 ERROR 数 → 风险分数
func logErrorScore(count int64) float64 {
	switch {
	case count >= 2000:
		return 0.90
	case count >= 1000:
		return 0.80
	default:
		return 0.75
	}
}

// dumpAnomalies 调试用
func dumpAnomalies(results []*aiops.AnomalyResult) {
	for _, r := range results {
		fmt.Printf("  entity=%s metric=%s score=%.2f value=%.2f\n",
			r.EntityKey, r.MetricName, r.Score, r.CurrentValue)
	}
}

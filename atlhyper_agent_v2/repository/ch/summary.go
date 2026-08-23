// Package ch ClickHouse 仓库实现
package ch

import (
	"context"
	"math"

	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/repository/ch/query"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/model_v3/apm"
)

// summaryRepository OTel 概览仓库（ClickHouse 聚合查询）
type summaryRepository struct {
	client sdk.ClickHouseClient
}

// NewOTelSummaryRepository 创建 OTel 概览仓库
func NewOTelSummaryRepository(client sdk.ClickHouseClient) repository.OTelSummaryRepository {
	return &summaryRepository{client: client}
}

// GetAPMSummary APM 概览（从 otel_traces 聚合）
func (r *summaryRepository) GetAPMSummary(ctx context.Context) (totalServices, healthyServices int, totalRPS, avgSuccessRate, avgP99Ms float64, err error) {
	query := `
		SELECT count(DISTINCT ServiceName)       AS total_services,
		       count(DISTINCT if(err_rate < 0.05, ServiceName, NULL)) AS healthy_services,
		       sum(span_cnt) / 300               AS total_rps,
		       avg(1 - err_rate) * 100           AS avg_success_rate,
		       avg(p99_ns) / 1e6                 AS avg_p99_ms
		FROM (
		    SELECT ServiceName,
		           count()                       AS span_cnt,
		           countIf(StatusCode = ` + apm.SQLStatusCodeError + `) / count() AS err_rate,
		           quantile(0.99)(Duration)       AS p99_ns
		    FROM otel_traces
		    WHERE SpanKind = ` + apm.SQLSpanKindServer + ` AND Timestamp >= now() - INTERVAL 5 MINUTE
		    GROUP BY ServiceName
		)
	`

	err = r.client.QueryRow(ctx, query).Scan(
		&totalServices, &healthyServices, &totalRPS, &avgSuccessRate, &avgP99Ms,
	)
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}

	totalRPS = roundF(totalRPS, 2)
	avgSuccessRate = roundF(avgSuccessRate, 2)
	avgP99Ms = roundF(avgP99Ms, 2)
	return
}

// GetSLOSummary SLO 概览（从 Traefik sum + Linkerd gauge 聚合）
func (r *summaryRepository) GetSLOSummary(ctx context.Context) (ingressServices int, ingressAvgRPS float64, err error) {
	// Ingress（契约 ingress_request_total，与具体 ingress 实现解耦）
	ingressQuery := `
		SELECT count(DISTINCT svc) AS ingress_services, avg(rate_val) AS avg_rps
		FROM (
		    SELECT Attributes['service'] AS svc, ` + query.CounterRateExpr + ` AS rate_val
		    FROM otel_metrics_sum
		    WHERE MetricName = 'ingress_request_total'
		      AND TimeUnix >= now() - INTERVAL 5 MINUTE
		    GROUP BY svc HAVING count() >= 2
		)
	`
	_ = r.client.QueryRow(ctx, ingressQuery).Scan(&ingressServices, &ingressAvgRPS)
	ingressAvgRPS = roundF(ingressAvgRPS, 2)

	// 服务网格概览已移除：SLO 只做 ingress 外部视角，
	// 服务间调用质量由 APM 承担（见 slo-ingress-contract-design.md）。
	return ingressServices, ingressAvgRPS, nil
}

// GetMetricsSummary 基础设施指标概览（从 node_cpu + node_memory 聚合）
func (r *summaryRepository) GetMetricsSummary(ctx context.Context) (monitoredNodes int, avgCPUPct, avgMemPct, maxCPUPct, maxMemPct float64, err error) {
	// CPU usage (rate calculation)
	cpuQuery := `
		WITH cpu_rate AS (
		    SELECT ResourceAttributes['net.host.name'] AS ip, Attributes['mode'] AS mode, ` + query.CounterRateExpr + ` AS rate
		    FROM otel_metrics_sum
		    WHERE MetricName = 'node_cpu_seconds_total' AND TimeUnix >= now() - INTERVAL 5 MINUTE
		    GROUP BY ip, Attributes['cpu'], mode HAVING count() >= 2
		)
		SELECT count(DISTINCT ip), avg(usage) * 100, max(usage) * 100
		FROM (
		    SELECT ip, 1 - sumIf(rate, mode='idle') / if(sum(rate) = 0, 1, sum(rate)) AS usage
		    FROM cpu_rate GROUP BY ip
		)
	`
	_ = r.client.QueryRow(ctx, cpuQuery).Scan(&monitoredNodes, &avgCPUPct, &maxCPUPct)

	// Memory usage
	memQuery := `
		SELECT avg(1 - avail / if(total = 0, 1, total)) * 100 AS avg_mem,
		       max(1 - avail / if(total = 0, 1, total)) * 100 AS max_mem
		FROM (
		    SELECT ResourceAttributes['net.host.name'] AS ip,
		           argMaxIf(Value, TimeUnix, MetricName='node_memory_MemTotal_bytes') AS total,
		           argMaxIf(Value, TimeUnix, MetricName='node_memory_MemAvailable_bytes') AS avail
		    FROM otel_metrics_gauge
		    WHERE MetricName IN ('node_memory_MemTotal_bytes', 'node_memory_MemAvailable_bytes')
		      AND TimeUnix >= now() - INTERVAL 2 MINUTE
		    GROUP BY ip
		)
	`
	_ = r.client.QueryRow(ctx, memQuery).Scan(&avgMemPct, &maxMemPct)

	avgCPUPct = roundF(avgCPUPct, 2)
	avgMemPct = roundF(avgMemPct, 2)
	maxCPUPct = roundF(maxCPUPct, 2)
	maxMemPct = roundF(maxMemPct, 2)

	return monitoredNodes, avgCPUPct, avgMemPct, maxCPUPct, maxMemPct, nil
}

// roundF 四舍五入到指定小数位（NaN/Inf → 0）
func roundF(v float64, decimals int) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	pow := math.Pow(10, float64(decimals))
	return math.Round(v*pow) / pow
}

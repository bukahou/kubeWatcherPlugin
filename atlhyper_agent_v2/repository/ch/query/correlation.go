package query

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"AtlHyper/model_v3/apm"
)

// correlation.go — 相关性分析（Correlations）
//
// 回答「慢/错的请求，和正常请求相比，什么属性显著超标」。
// 对齐 Elastic APM Correlations 的 significant terms 思路，用 ClickHouse
// 条件聚合实现：一次查询同时得到每个属性值的前景计数（countIf）与
// 背景计数（count），Go 侧做占比 / 提升度 / 打分 / 分级。
//
// 打分与分级放 Agent 而非 Master：APM 域的 Master 是纯透传 + 缓存
// （无 convert 层），Agent 直出前端形态是该域的既定架构。

// correlationFields 候选字段白名单（v1 固定）。
// user_agent.original 基数过高，SQL 内归一化为 OS/设备家族。
var correlationFieldExprs = []struct {
	name string
	expr string // ClickHouse 表达式，产出该字段的归一化取值
}{
	{"url.path", "SpanAttributes['url.path']"},
	{"http.request.method", "SpanAttributes['http.request.method']"},
	{"http.response.status_code", "SpanAttributes['http.response.status_code']"},
	{"client.address", "SpanAttributes['client.address']"},
	{"user_agent.family", `multiIf(
	        SpanAttributes['user_agent.original'] LIKE '%iPhone%' OR SpanAttributes['user_agent.original'] LIKE '%iPad%', 'iOS',
	        SpanAttributes['user_agent.original'] LIKE '%Android%', 'Android',
	        SpanAttributes['user_agent.original'] LIKE '%Windows%', 'Windows',
	        SpanAttributes['user_agent.original'] LIKE '%Macintosh%', 'macOS',
	        positionCaseInsensitive(SpanAttributes['user_agent.original'], 'bot') > 0
	            OR positionCaseInsensitive(SpanAttributes['user_agent.original'], 'spider') > 0, 'bot',
	        SpanAttributes['user_agent.original'] != '', 'other',
	        '')`},
	{"k8s.pod.name", "ResourceAttributes['k8s.pod.name']"},
	{"k8s.node.name", "ResourceAttributes['k8s.node.name']"},
	{"service.version", "ResourceAttributes['service.version']"},
	{"service.instance.id", "ResourceAttributes['service.instance.id']"},
}

// buildCorrelationQuery 构造相关性聚合查询。
//
// where 只圈定统计范围（时间窗 + 服务 + 可选操作）；入口 span 限定
// （SpanKind = Server）在此统一追加 —— 前景/背景都必须基于入口 span
// （≈ Transaction），把内部 span 混进来会稀释每个请求的权重。
//
// mode = failure：前景 = StatusCode Error；
// mode = latency：前景 = Duration > thresholdNs（调用方先算好 P95）。
// 缺失的属性归一化为 '(none)' —— 缺失本身可能就是相关项。
func buildCorrelationQuery(where, mode string, thresholdNs int64) string {
	fgCond := "StatusCode = " + apm.SQLStatusCodeError
	if mode == apm.CorrelationModeLatency {
		fgCond = fmt.Sprintf("Duration > %d", thresholdNs)
	}

	pairs := ""
	for i, f := range correlationFieldExprs {
		if i > 0 {
			pairs += ",\n\t\t    "
		}
		pairs += fmt.Sprintf("('%s', if(%s = '', '(none)', %s))", f.name, f.expr, f.expr)
	}

	return fmt.Sprintf(`
		SELECT fv.1 AS field, fv.2 AS value,
		       countIf(%s) AS fgCount,
		       count() AS bgCount
		FROM otel_traces
		ARRAY JOIN [%s] AS fv
		WHERE %s AND SpanKind = %s
		GROUP BY field, value
	`, fgCond, pairs, where, apm.SQLSpanKindServer)
}

// correlationRow 聚合查询的一行
type correlationRow struct {
	Field   string
	Value   string
	FgCount int64
	BgCount int64
}

// scoreCorrelations 打分、分级、排序、截断（Top 10）。
//
//	score = fgRatio × ln(lift)   —— 覆盖率 × 提升度：
//	  只有 lift 高（提升度）没有 fgRatio（覆盖率）的低频值刷不了分；
//	  全集值 lift ≈ 1 → ln ≈ 0 → 自然沉底。
func scoreCorrelations(rows []correlationRow, fgTotal, bgTotal int64) []apm.CorrelationItem {
	if fgTotal <= 0 || bgTotal <= 0 {
		return []apm.CorrelationItem{}
	}
	items := make([]apm.CorrelationItem, 0, len(rows))
	for _, r := range rows {
		if r.FgCount <= 0 || r.BgCount <= 0 {
			continue // 与前景无关的值不进结果
		}
		fgRatio := float64(r.FgCount) / float64(fgTotal)
		bgRatio := float64(r.BgCount) / float64(bgTotal)
		lift := fgRatio / bgRatio
		score := fgRatio * math.Log(lift)
		if score < 0 {
			score = 0 // 前景占比低于背景的值不是嫌疑项
		}

		impact := apm.CorrelationImpactLow
		switch {
		case score > 0.5 && fgRatio > 0.3:
			impact = apm.CorrelationImpactHigh
		case score > 0.2:
			impact = apm.CorrelationImpactMedium
		}

		items = append(items, apm.CorrelationItem{
			Field: r.Field, Value: r.Value,
			FgCount: r.FgCount, BgCount: r.BgCount,
			FgRatio: roundTo(fgRatio, 4), BgRatio: roundTo(bgRatio, 4),
			Lift: roundTo(lift, 2), Score: roundTo(score, 4),
			Impact: impact,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Score > items[j].Score })
	if len(items) > 10 {
		items = items[:10]
	}
	return items
}

// GetTraceCorrelations 执行相关性分析。
func (r *traceRepository) GetTraceCorrelations(
	ctx context.Context, service, operation, mode string,
	since time.Duration, startTime, endTime string,
) (*apm.CorrelationResult, error) {
	timeConds, args := traceTimeCondition("Timestamp", since, startTime, endTime)
	conds := timeConds
	if service != "" {
		conds = append(conds, "ServiceName = ?")
		args = append(args, service)
	}
	if operation != "" {
		conds = append(conds, "SpanName = ?")
		args = append(args, operation)
	}
	where := ""
	for i, c := range conds {
		if i > 0 {
			where += " AND "
		}
		where += c
	}

	entryWhere := where + " AND SpanKind = " + apm.SQLSpanKindServer

	result := &apm.CorrelationResult{Mode: mode, Items: []apm.CorrelationItem{}}

	// 1. 总量（前景 / 背景）——latency 模式先算 P95 阈值
	var thresholdNs int64
	if mode == apm.CorrelationModeLatency {
		q := fmt.Sprintf("SELECT quantile(0.95)(Duration) FROM otel_traces WHERE %s", entryWhere)
		var p95 float64
		if err := r.client.QueryRow(ctx, q, args...).Scan(&p95); err != nil {
			return nil, fmt.Errorf("correlation p95: %w", err)
		}
		thresholdNs = int64(p95)
		result.ThresholdMs = roundTo(p95/1e6, 2)
	}

	fgCond := "StatusCode = " + apm.SQLStatusCodeError
	if mode == apm.CorrelationModeLatency {
		fgCond = fmt.Sprintf("Duration > %d", thresholdNs)
	}
	totalsQ := fmt.Sprintf("SELECT countIf(%s), count() FROM otel_traces WHERE %s", fgCond, entryWhere)
	if err := r.client.QueryRow(ctx, totalsQ, args...).Scan(&result.ForegroundCount, &result.BackgroundCount); err != nil {
		return nil, fmt.Errorf("correlation totals: %w", err)
	}
	result.LowSample = result.ForegroundCount < 5
	if result.ForegroundCount == 0 {
		return result, nil // 无可分析样本：空结果 + count=0，前端据此提示
	}

	// 2. 按字段值聚合前景/背景计数
	rows, err := r.client.Query(ctx, buildCorrelationQuery(where, mode, thresholdNs), args...)
	if err != nil {
		return nil, fmt.Errorf("correlation aggregate: %w", err)
	}
	defer rows.Close()

	var crows []correlationRow
	for rows.Next() {
		var cr correlationRow
		if err := rows.Scan(&cr.Field, &cr.Value, &cr.FgCount, &cr.BgCount); err != nil {
			continue
		}
		crows = append(crows, cr)
	}

	result.Items = scoreCorrelations(crows, result.ForegroundCount, result.BackgroundCount)
	return result, rows.Err()
}

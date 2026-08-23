package query

import (
	"context"
	"fmt"
	"time"

	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/common/logger"
	"AtlHyper/model_v3/slo"
)

var sloLog = logger.Module("SLO-CH")

// gaugeCounterDelta ClickHouse SQL: counter-reset-safe delta for Linkerd gauge counters.
// Reset detection: if peak > latest, a counter reset occurred within the window.
// Normal: latest - earliest; Reset: (peak - earliest) + latest.
const gaugeCounterDelta = `if(max(Value) > argMax(Value, TimeUnix),
    (max(Value) - argMin(Value, TimeUnix)) + argMax(Value, TimeUnix),
    argMax(Value, TimeUnix) - argMin(Value, TimeUnix))`

// seriesIsolation 用于 GROUP BY 隔离独立计数器系列的列（原始表查询用）。
// Linkerd gauge 指标按 (pod, status_code, route_name, target_addr) 产生独立计数器。
const seriesIsolation = `Attributes['pod'], Attributes['status_code'],
             Attributes['route_name'], Attributes['target_addr']`

// gaugeCounterDeltaMerge 是 gaugeCounterDelta 的 AggregatingMergeTree 版本。
// 操作预聚合 MV 中的 -State 列，使用 -Merge 组合器最终化。
const gaugeCounterDeltaMerge = `if(maxMerge(peak_val) > argMaxMerge(latest_val),
    (maxMerge(peak_val) - argMinMerge(earliest_val)) + argMaxMerge(latest_val),
    argMaxMerge(latest_val) - argMinMerge(earliest_val))`

// CounterRateExpr 按 counter-reset-safe delta 除以时间跨度，得到 per-second rate。
// 使用场景：5min 等短窗口，reset 概率极低，单次 reset 处理足够。
// 对于长窗口（1d/7d/30d），应改用 lagInFrame 的 Prometheus 完整算法
// （参见 queryIngressSLO 的 countQuery）。
// 导出为包外可用，供 ch 包内 summary.go 等其他位置共享同一算法。
const CounterRateExpr = gaugeCounterDelta + ` /
    (toUnixTimestamp(argMax(TimeUnix, TimeUnix)) - toUnixTimestamp(argMin(TimeUnix, TimeUnix)))`

// sloRepository SLO 查询仓库
type sloRepository struct {
	client sdk.ClickHouseClient
}

// NewSLOQueryRepository 创建 SLO 查询仓库
func NewSLOQueryRepository(client sdk.ClickHouseClient) repository.SLOQueryRepository {
	return &sloRepository{client: client}
}

// ──────────────────────────────────────────────────────────────
// Ingress 契约 —— 与具体 ingress 实现解耦的唯一接口
// ──────────────────────────────────────────────────────────────
//
// 本文件的查询只允许出现下列契约名，不得出现任何 ingress 实现
// （Traefik / Envoy / Nginx / Cilium …）的指标名或 label 名。
// 实现差异一律由 Collector 的 transform 归一化处理。
// 契约定义见 docs/design/active/slo-ingress-contract-design.md。
//
// 历史教训：Ingress 一路曾直接查 traefik_service_requests_total +
// Attributes['code']，Traefik 从集群移除后 SLO 页长期空白且无人察觉；
// 而 Mesh 一路因为做了统一契约，Istio 装了又拆都未伤及代码。
// slo_test.go 的 TestIngressSQL_NoVendorNames 会强制守护这条边界。
const (
	metricIngressRequestTotal   = "ingress_request_total"
	metricIngressDurationBucket = "ingress_request_duration_bucket"

	// 契约 label
	labelNamespace   = "namespace"
	labelService     = "service"
	labelStatusClass = "status_class"
)

// ⚠️ 单位约定：ingress_request_duration_bucket 的桶边界单位是【毫秒】。
//
// 契约本想统一为秒，但 OTTL 无法改写 histogram 的 ExplicitBounds 数组，
// Collector 侧做不了换算。若要求各实现自报单位，Agent 就得知道底下是谁 ——
// 那等于放弃抽象。因此契约直接规定桶为毫秒，由各实现在 Collector 侧对齐，
// Agent 一律按毫秒处理（正好是 IngressSLO.P*Ms 字段所需单位，无需换算）。

// ingressServiceKey 由契约 label 组合出服务标识。
//
// 用 namespace/service 而非某实现的内部 service 名（如 Traefik 的
// "geass-v3-geass-web@kubernetes"）—— K8s 原生概念，换实现不变。
func ingressServiceKey(namespace, service string) string {
	if service == "" {
		return ""
	}
	if namespace == "" {
		return service
	}
	return namespace + "/" + service
}

// isErrorStatusClass 判定某状态码类别是否计入 SLO 违约。
//
// 只有 5xx 计入：4xx 是客户端问题（参数错误、鉴权失败、请求不存在的资源），
// 属于业务正常流程，不构成服务质量违约。
func isErrorStatusClass(class string) bool {
	return class == "5"
}

// buildIngressCountQuery 构造请求计数查询。
//
// 按 {namespace, service, status_class} 分组 —— 每个组合是独立的累积计数器，
// 必须在最细粒度算 delta。算法等价于 Prometheus rate() 的 counterCorrection：
// v[i] >= v[i-1] 取差值，否则视为 counter reset 取 v[i]。
func buildIngressCountQuery(timeFilter string) string {
	return fmt.Sprintf(`
		SELECT ns, svc, class,
		       sum(if(Value >= prevValue, Value - prevValue, Value)) AS delta
		FROM (
		    SELECT Attributes['%s'] AS ns,
		           Attributes['%s'] AS svc,
		           Attributes['%s'] AS class,
		           Value, TimeUnix,
		           lagInFrame(Value, 1, Value) OVER
		               (PARTITION BY Attributes['%s'], Attributes['%s'], Attributes['%s']
		                ORDER BY TimeUnix) AS prevValue
		    FROM otel_metrics_sum
		    WHERE MetricName = '%s'
		      %s
		)
		GROUP BY ns, svc, class
		HAVING delta > 0
	`, labelNamespace, labelService, labelStatusClass,
		labelNamespace, labelService, labelStatusClass,
		metricIngressRequestTotal, timeFilter)
}

// buildIngressLatencyQuery 构造延迟直方图查询。
// argMax/argMin(BucketCounts) 取窗口内最新/最旧快照做差。
func buildIngressLatencyQuery(timeFilter string) string {
	return fmt.Sprintf(`
		SELECT Attributes['%s'] AS ns,
		       Attributes['%s'] AS svc,
		       argMax(ExplicitBounds, TimeUnix) AS bounds,
		       argMax(BucketCounts, TimeUnix) AS latest,
		       argMin(BucketCounts, TimeUnix) AS earliest
		FROM otel_metrics_histogram
		WHERE MetricName = '%s'
		  %s
		GROUP BY ns, svc
		HAVING count() >= 2
	`, labelNamespace, labelService, metricIngressDurationBucket, timeFilter)
}

// buildIngressHistoryQuery 构造按时间桶聚合的历史查询。
func buildIngressHistoryQuery(timeFilter string, bucketSec int64) string {
	return fmt.Sprintf(`
		SELECT toStartOfInterval(TimeUnix, INTERVAL %d SECOND) AS ts,
		       ns, svc, class,
		       sum(if(Value >= prevValue, Value - prevValue, Value)) AS delta
		FROM (
		    SELECT Attributes['%s'] AS ns,
		           Attributes['%s'] AS svc,
		           Attributes['%s'] AS class,
		           Value, TimeUnix,
		           lagInFrame(Value, 1, Value) OVER
		               (PARTITION BY Attributes['%s'], Attributes['%s'], Attributes['%s'],
		                             toStartOfInterval(TimeUnix, INTERVAL %d SECOND)
		                ORDER BY TimeUnix) AS prevValue
		    FROM otel_metrics_sum
		    WHERE MetricName = '%s'
		      %s
		)
		GROUP BY ts, ns, svc, class
		HAVING delta > 0
		ORDER BY ts
	`, bucketSec,
		labelNamespace, labelService, labelStatusClass,
		labelNamespace, labelService, labelStatusClass, bucketSec,
		metricIngressRequestTotal, timeFilter)
}

// buildIngressSummaryQuery 构造概览用的服务数与总量查询。
func buildIngressSummaryQuery() string {
	return fmt.Sprintf(`
		SELECT uniqExact(concat(Attributes['%s'], '/', Attributes['%s'])) AS svc_count
		FROM otel_metrics_sum
		WHERE MetricName = '%s'
		  AND TimeUnix >= now() - INTERVAL 5 MINUTE
	`, labelNamespace, labelService, metricIngressRequestTotal)
}

// ──────────────────────────────────────────────────────────────
// 共用辅助：从 ClickHouse histogram delta 行汇聚到 per-svc 桶
// ──────────────────────────────────────────────────────────────

// svcHistogram 聚合后的 per-service histogram (delta counts)
type svcHistogram struct {
	bounds []float64
	counts []uint64
}

// addHistogramDelta 把一行 {latest, earliest} BucketCounts 的 delta 累加到 hist
func addHistogramDelta(hist *svcHistogram, bounds []float64, latest, earliest []uint64) {
	// 首次：初始化 bounds 和 counts
	if len(hist.bounds) == 0 {
		hist.bounds = bounds
		hist.counts = make([]uint64, len(latest))
	}
	for i := 0; i < len(latest) && i < len(hist.counts); i++ {
		if i < len(earliest) && latest[i] >= earliest[i] {
			hist.counts[i] += latest[i] - earliest[i]
		} else {
			hist.counts[i] += latest[i] // counter reset
		}
	}
}

// histToLatency 从聚合后的 histogram 提取分位数和桶列表
func histToLatency(hist *svcHistogram) (p50, p90, p95, p99 float64, buckets []slo.LatencyBucket) {
	// 契约桶单位已是毫秒，直接取用 —— 不再 ×1000。
	// （Traefik 时代桶为秒，故旧实现有换算；换算残留会让 P99 放大一千倍，
	//   且不产生任何报错，只是数字离谱。见 TestHistToLatency_BucketsAreMilliseconds）
	p50 = roundTo(histogramPercentile(hist.bounds, hist.counts, 0.50), 2)
	p90 = roundTo(histogramPercentile(hist.bounds, hist.counts, 0.90), 2)
	p95 = roundTo(histogramPercentile(hist.bounds, hist.counts, 0.95), 2)
	p99 = roundTo(histogramPercentile(hist.bounds, hist.counts, 0.99), 2)

	for i, b := range hist.bounds {
		var cnt int64
		if i < len(hist.counts) {
			cnt = int64(hist.counts[i])
		}
		buckets = append(buckets, slo.LatencyBucket{
			LE:    roundTo(b, 2), // 契约单位毫秒
			Count: cnt,
		})
	}
	if len(hist.counts) > len(hist.bounds) {
		buckets = append(buckets, slo.LatencyBucket{
			LE:    0, // +Inf
			Count: int64(hist.counts[len(hist.bounds)]),
		})
	}
	return
}

// ──────────────────────────────────────────────────────────────
// ListIngressSLO — 当前窗口
// ──────────────────────────────────────────────────────────────

// ListIngressSLO 查询 Traefik 入口 SLO
func (r *sloRepository) ListIngressSLO(ctx context.Context, since time.Duration) ([]slo.IngressSLO, error) {
	sec := sinceSeconds(since)
	return r.queryIngressSLO(ctx, fmt.Sprintf(
		"AND TimeUnix >= now() - INTERVAL %d SECOND", sec), sec)
}

// ListIngressSLOPrevious 查询上一周期的 Traefik 入口 SLO
// since 表示窗口大小，查询 [now-2*since, now-since) 的数据
func (r *sloRepository) ListIngressSLOPrevious(ctx context.Context, since time.Duration) ([]slo.IngressSLO, error) {
	sec := sinceSeconds(since)
	return r.queryIngressSLO(ctx, fmt.Sprintf(
		"AND TimeUnix >= now() - INTERVAL %d SECOND AND TimeUnix < now() - INTERVAL %d SECOND",
		2*sec, sec), sec)
}

// queryIngressSLO 通用 Ingress SLO 查询（当前窗口 / 上一周期共用）
// timeFilter: 时间范围条件（不含 WHERE 前缀）
// windowSec: 窗口秒数（用于 RPS 计算）
func (r *sloRepository) queryIngressSLO(ctx context.Context, timeFilter string, windowSec int64) ([]slo.IngressSLO, error) {
	// ── 请求计数：按 {svc, code, method} 三维分组 ──
	// 每个组合是独立的累积计数器，必须在最细粒度计算 delta。
	//
	// 算法等价于 Prometheus rate()/increase() 的 counterCorrection 逻辑：
	// 用 lagInFrame 取相邻前一样本，对每对相邻样本：
	//   - v[i] >= v[i-1]: 正常递增，delta = v[i] - v[i-1]
	//   - v[i] <  v[i-1]: counter reset，delta = v[i]（reset 后从 0 开始）
	// 最后 sum 得到窗口内的总增量，正确处理任意次数的 reset。
	// lagInFrame 第三参数 Value 让首行 prevValue = 自身，首行 delta = 0。
	countQuery := buildIngressCountQuery(timeFilter)

	rows, err := r.client.Query(ctx, countQuery)
	if err != nil {
		return nil, fmt.Errorf("query ingress counts: %w", err)
	}
	defer rows.Close()

	type svcData struct {
		totalReqs   int64
		totalErrors int64
		codes       map[string]int64
		methods     map[string]int64
	}
	svcMap := make(map[string]*svcData)

	for rows.Next() {
		var ns, svc, class string
		var delta float64
		if err := rows.Scan(&ns, &svc, &class, &delta); err != nil {
			continue
		}
		svcKey := ingressServiceKey(ns, svc)
		if svcKey == "" {
			continue
		}
		d, ok := svcMap[svcKey]
		if !ok {
			d = &svcData{codes: make(map[string]int64), methods: make(map[string]int64)}
			svcMap[svcKey] = d
		}
		cnt := int64(delta)
		if cnt <= 0 {
			continue
		}
		d.totalReqs += cnt
		// 契约用 status_class（"2"/"4"/"5"…）而非精确码：各 ingress 实现
		// 的粒度不同，契约取最小公分母。展示层把它渲染为 "2xx"/"5xx"。
		d.codes[class+"xx"] += cnt
		if isErrorStatusClass(class) {
			d.totalErrors += cnt
		}
	}

	// ── Histogram: 按 {svc, code, method} 分组，计算 delta 桶 ──
	// argMax/argMin(BucketCounts, TimeUnix) 取窗口内最新/最旧快照做差
	latencyQuery := buildIngressLatencyQuery(timeFilter)

	histMap := make(map[string]*svcHistogram)
	latencyRows, lerr := r.client.Query(ctx, latencyQuery)
	if lerr == nil && latencyRows != nil {
		defer latencyRows.Close()
		for latencyRows.Next() {
			var ns, svc string
			var bounds []float64
			var latest, earliest []uint64
			if err := latencyRows.Scan(&ns, &svc, &bounds, &latest, &earliest); err != nil {
				continue
			}
			svcKey := ingressServiceKey(ns, svc)
			if svcKey == "" {
				continue
			}
			hist, ok := histMap[svcKey]
			if !ok {
				hist = &svcHistogram{}
				histMap[svcKey] = hist
			}
			addHistogramDelta(hist, bounds, latest, earliest)
		}
	}

	// ── 组装结果 ──
	duration := float64(windowSec)
	var result []slo.IngressSLO
	for key, d := range svcMap {
		item := slo.IngressSLO{
			ServiceKey:    key,
			DisplayName:   key,
			TotalRequests: d.totalReqs,
			TotalErrors:   d.totalErrors,
			RPS:           roundRPS(float64(d.totalReqs) / duration),
		}
		if d.totalReqs > 0 {
			item.SuccessRate = roundTo(float64(d.totalReqs-d.totalErrors)/float64(d.totalReqs)*100, 2)
			item.ErrorRate = roundTo(float64(d.totalErrors)/float64(d.totalReqs)*100, 2)
		}
		for code, cnt := range d.codes {
			item.StatusCodes = append(item.StatusCodes, slo.StatusCodeCount{Code: code, Count: cnt})
		}
		for method, cnt := range d.methods {
			item.Methods = append(item.Methods, slo.MethodCount{Method: method, Count: cnt})
		}
		if hist, ok := histMap[key]; ok {
			item.P50Ms, item.P90Ms, item.P95Ms, item.P99Ms, item.LatencyBuckets = histToLatency(hist)
		}
		result = append(result, item)
	}
	if result == nil {
		result = []slo.IngressSLO{}
	}
	return result, nil
}

// ──────────────────────────────────────────────────────────────
// GetSLOTimeSeries — 单个 ingress 服务的 SLO 时序
// ──────────────────────────────────────────────────────────────

// GetSLOTimeSeries 查询 SLO 时序数据
func (r *sloRepository) GetSLOTimeSeries(ctx context.Context, name string, since time.Duration) (*slo.TimeSeries, error) {
	sec := sinceSeconds(since)

	// name 为契约 serviceKey（"namespace/service"）。按 5 分钟窗口聚合，
	// 每个 status_class 系列独立做 counter-reset-safe delta。
	query := fmt.Sprintf(`
		SELECT ts, class, sum(if(Value >= prevValue, Value - prevValue, Value)) AS delta
		FROM (
			SELECT toStartOfInterval(TimeUnix, INTERVAL 300 SECOND) AS ts,
			       Attributes['%s'] AS class,
			       Value, TimeUnix,
			       lagInFrame(Value, 1, Value) OVER
			           (PARTITION BY Attributes['%s'],
			                         toStartOfInterval(TimeUnix, INTERVAL 300 SECOND)
			            ORDER BY TimeUnix) AS prevValue
			FROM otel_metrics_sum
			WHERE MetricName = '%s'
			  AND concat(Attributes['%s'], '/', Attributes['%s']) = ?
			  AND TimeUnix >= now() - INTERVAL %d SECOND
		)
		GROUP BY ts, class
		HAVING delta > 0
		ORDER BY ts
	`, labelStatusClass, labelStatusClass, metricIngressRequestTotal,
		labelNamespace, labelService, sec)

	rows, err := r.client.Query(ctx, query, name)
	if err != nil {
		return nil, fmt.Errorf("query SLO time series: %w", err)
	}
	defer rows.Close()

	type tsData struct {
		total, success float64
	}
	tsMap := make(map[time.Time]*tsData)

	for rows.Next() {
		var ts time.Time
		var class string
		var delta float64
		if err := rows.Scan(&ts, &class, &delta); err != nil {
			continue
		}
		if delta <= 0 {
			continue
		}
		d, ok := tsMap[ts]
		if !ok {
			d = &tsData{}
			tsMap[ts] = d
		}
		d.total += delta
		if !isErrorStatusClass(class) {
			d.success += delta // 仅 5xx 计为错误
		}
	}

	ts := &slo.TimeSeries{Name: name}
	for t, d := range tsMap {
		dp := slo.DataPoint{
			Timestamp: t,
			RPS:       roundTo(d.total/300, 2), // 5 分钟窗口
		}
		if d.total > 0 {
			dp.SuccessRate = roundTo(d.success/d.total*100, 2)
		}
		ts.Points = append(ts.Points, dp)
	}
	if ts.Points == nil {
		ts.Points = []slo.DataPoint{}
	}
	return ts, nil
}

// ──────────────────────────────────────────────────────────────
// GetSLOSummary — 仪表盘摘要
// ──────────────────────────────────────────────────────────────

// GetSLOSummary 获取 SLO 仪表盘摘要
func (r *sloRepository) GetSLOSummary(ctx context.Context) (*slo.SLOSummary, error) {
	since := 5 * time.Minute

	// 同时获取 Ingress 和 Service SLO
	type ingressResult struct {
		data []slo.IngressSLO
		err  error
	}

	// SLO 只覆盖 ingress（外部视角）。服务间调用质量由 APM 承担，
	// 详见 docs/design/active/slo-ingress-contract-design.md 的范围决策。
	ingData, ingErr := r.ListIngressSLO(ctx, since)

	summary := &slo.SLOSummary{}

	// 合并统计
	var totalSuccRate, totalRPS, totalP99 float64
	var count int

	if ingErr == nil {
		for _, s := range ingData {
			count++
			totalSuccRate += s.SuccessRate
			totalRPS += s.RPS
			totalP99 += s.P99Ms

			if s.SuccessRate >= 99.9 {
				summary.HealthyServices++
			} else if s.SuccessRate >= 99.0 {
				summary.WarningServices++
			} else {
				summary.CriticalServices++
			}
		}
	}

	summary.TotalServices = count
	if count > 0 {
		summary.AvgSuccessRate = roundTo(totalSuccRate/float64(count), 2)
		summary.TotalRPS = roundTo(totalRPS, 2)
		summary.AvgP99Ms = roundTo(totalP99/float64(count), 2)
	}

	return summary, nil
}

// ──────────────────────────────────────────────────────────────
// GetIngressSLOHistory — Ingress SLO 时序数据
// ──────────────────────────────────────────────────────────────

// GetIngressSLOHistory 查询 Ingress SLO 时序数据
// since: 总时间范围, bucket: 每个桶的时间跨度
func (r *sloRepository) GetIngressSLOHistory(ctx context.Context, since, bucket time.Duration) ([]slo.SLOHistoryPoint, error) {
	sec := sinceSeconds(since)
	bucketSec := sinceSeconds(bucket)

	// ── 请求计数时序：按 {ts, namespace, service, status_class} 分组 ──
	// 算法同 queryIngressSLO 的 Prometheus rate() 逻辑，但分区键加上桶 ts，
	// 每个 (ns, svc, class, bucket) 内独立做 counter-reset-safe 累加。
	countQuery := buildIngressHistoryQuery(
		fmt.Sprintf("AND TimeUnix >= now() - INTERVAL %d SECOND", sec), bucketSec)

	rows, err := r.client.Query(ctx, countQuery)
	if err != nil {
		return nil, fmt.Errorf("query ingress history counts: %w", err)
	}
	defer rows.Close()

	type bucketKey struct {
		ts  time.Time
		svc string
	}
	type bucketData struct {
		totalReqs   int64
		totalErrors int64
	}
	dataMap := make(map[bucketKey]*bucketData)

	for rows.Next() {
		var ts time.Time
		var ns, svc, class string
		var delta float64
		if err := rows.Scan(&ts, &ns, &svc, &class, &delta); err != nil {
			continue
		}
		svcKey := ingressServiceKey(ns, svc)
		if svcKey == "" {
			continue
		}
		key := bucketKey{ts: ts, svc: svcKey}
		d, ok := dataMap[key]
		if !ok {
			d = &bucketData{}
			dataMap[key] = d
		}
		cnt := int64(delta)
		if cnt <= 0 {
			continue
		}
		d.totalReqs += cnt
		if isErrorStatusClass(class) {
			d.totalErrors += cnt
		}
	}

	// ── 延迟时序：按 {svc, ts, code, method} 分组，计算 delta 桶 ──
	latencyQuery := fmt.Sprintf(`
		SELECT Attributes['%s'] AS ns,
		       Attributes['%s'] AS svc,
		       toStartOfInterval(TimeUnix, INTERVAL %d SECOND) AS ts,
		       argMax(ExplicitBounds, TimeUnix) AS bounds,
		       argMax(BucketCounts, TimeUnix) AS latest,
		       argMin(BucketCounts, TimeUnix) AS earliest
		FROM otel_metrics_histogram
		WHERE MetricName = '%s'
		  AND TimeUnix >= now() - INTERVAL %d SECOND
		GROUP BY ns, svc, ts
		HAVING count() >= 2
		ORDER BY ns, svc, ts
	`, labelNamespace, labelService, bucketSec, metricIngressDurationBucket, sec)

	latencyByBucket := make(map[bucketKey]*svcHistogram)
	latRows, latErr := r.client.Query(ctx, latencyQuery)
	if latErr == nil && latRows != nil {
		defer latRows.Close()
		for latRows.Next() {
			var ns, svc string
			var ts time.Time
			var bounds []float64
			var latest, earliest []uint64
			if err := latRows.Scan(&ns, &svc, &ts, &bounds, &latest, &earliest); err != nil {
				continue
			}
			svcKey := ingressServiceKey(ns, svc)
			if svcKey == "" {
				continue
			}
			key := bucketKey{ts: ts, svc: svcKey}
			hist, ok := latencyByBucket[key]
			if !ok {
				hist = &svcHistogram{}
				latencyByBucket[key] = hist
			}
			addHistogramDelta(hist, bounds, latest, earliest)
		}
	}

	// ── 组装结果 ──
	bucketDuration := float64(bucketSec)
	var result []slo.SLOHistoryPoint
	for key, d := range dataMap {
		point := slo.SLOHistoryPoint{
			Timestamp:     key.ts,
			ServiceKey:    key.svc,
			TotalRequests: d.totalReqs,
			RPS:           roundRPS(float64(d.totalReqs) / bucketDuration),
		}
		if d.totalReqs > 0 {
			point.Availability = roundTo(float64(d.totalReqs-d.totalErrors)/float64(d.totalReqs)*100, 2)
			point.ErrorRate = roundTo(float64(d.totalErrors)/float64(d.totalReqs)*100, 2)
		}
		if hist, ok := latencyByBucket[key]; ok {
			point.P95Ms = roundTo(histogramPercentile(hist.bounds, hist.counts, 0.95)*1000, 2)
			point.P99Ms = roundTo(histogramPercentile(hist.bounds, hist.counts, 0.99)*1000, 2)
		}
		result = append(result, point)
	}
	if result == nil {
		result = []slo.SLOHistoryPoint{}
	}
	return result, nil
}

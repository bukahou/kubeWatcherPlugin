// contract.go 实现 ClickHouse 枚举契约自检。
//
// 存在理由:
//
//	otel_traces 的 SpanKind / StatusCode、otel_logs 的 SeverityText 等字符串取值由
//	OTel Collector exporter 或应用侧 SDK 决定，不受本项目控制。2026-08 的一次 Collector
//	升级把 SpanKind 从 protobuf 全名改成短名，导致所有 APM 查询的 WHERE 条件匹配不到
//	任何行。
//
//	这类故障最难缠的地方在于【它不报错】—— SQL 执行成功、返回 0 行、日志无异常，
//	表现只是"页面上没数据"，很容易被误判成"还没有流量"。该故障因此潜伏了 27 天。
//
// 本自检把静默失败变成一条 ERROR 日志。除启动时检查一次外还周期性重跑：
// 被观测应用是之后才开始上报的（如 geass 接入日志），启动时表为空查不到，
// 周期检查保证数据到达后 interval 内能发现漂移。
package query

import (
	"context"
	"fmt"
	"strings"
	"time"

	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/common/logger"
	"AtlHyper/model_v3/apm"
	logmodel "AtlHyper/model_v3/log"
)

// enumContract 描述一张表的一列及其预期取值。
type enumContract struct {
	table    string
	column   string
	expected []string
	// filter 是可选的额外 WHERE 条件（不含 WHERE 关键字）。
	//
	// 大表必须带：otel_metrics_sum 有千万行级数据，裸 SELECT DISTINCT 会全表扫描
	// 直至超时（实测 i/o timeout）。用指标名 + 时间窗口把扫描范围收敛到最近数据，
	// 既避免超时，也让检查更精确（只看目标指标的取值）。
	filter string
}

// enumContracts 是全部受检契约。新增信号源时在此登记。
var enumContracts = []enumContract{
	{table: "otel_traces", column: "SpanKind", expected: apm.ExpectedSpanKinds},
	{table: "otel_traces", column: "StatusCode", expected: apm.ExpectedStatusCodes},
	{table: "otel_logs", column: "SeverityText", expected: logmodel.ExpectedSeverityTexts},
	// Ingress SLO 契约: status_class 由 Collector 从各 ingress 实现归一化而来
	// (Envoy 的 envoy_response_code_class / Traefik 的 code 首字符 / ...)。
	// 实现换代时若 transform 规则遗漏, 这里会第一时间报出来。
	{
		table:    "otel_metrics_sum",
		column:   "Attributes['status_class']",
		expected: expectedStatusClasses,
		filter:   "MetricName = 'ingress_request_total' AND TimeUnix > now() - INTERVAL 15 MINUTE",
	},
}

// expectedStatusClasses 是 ingress_request_total 的 status_class 取值域。
var expectedStatusClasses = []string{"1", "2", "3", "4", "5"}

// RunEnumContractChecks 立即执行一次契约自检，之后每 interval 重跑一次，直到 ctx 结束。
// 供 Agent 启动时以 goroutine 调用；不阻塞、不返回错误 —— 可观测性缺失属于降级，
// 不该让 Agent 起不来。
func RunEnumContractChecks(ctx context.Context, client sdk.ClickHouseClient, interval time.Duration) {
	if client == nil {
		return
	}
	VerifyEnumContracts(ctx, client)
	VerifyMetricsCollected(ctx, client)

	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			VerifyEnumContracts(ctx, client)
			VerifyMetricsCollected(ctx, client)
		}
	}
}

// VerifyEnumContracts 检查每个契约列实际出现的取值是否都在预期集合内。
//
// 表/列为空时跳过（没有数据不代表契约错误，可能只是还没有服务上报）。
func VerifyEnumContracts(ctx context.Context, client sdk.ClickHouseClient) {
	for _, c := range enumContracts {
		actual, err := distinctValues(ctx, client, c)
		if err != nil {
			logger.Warn("[契约自检] 查询失败", "table", c.table, "column", c.column, "err", err)
			continue
		}
		if len(actual) == 0 {
			logger.Info("[契约自检] 跳过：暂无数据", "table", c.table, "column", c.column)
			continue
		}

		if unknown := diff(actual, c.expected); len(unknown) > 0 {
			logger.Error("[契约漂移] ClickHouse 枚举取值与预期不符，相关查询将返回空或计数错误",
				"table", c.table,
				"column", c.column,
				"实际", strings.Join(actual, ","),
				"预期", strings.Join(c.expected, ","),
				"未知值", strings.Join(unknown, ","),
				"处理", "检查 Collector 版本 / 上报方 SDK，并更新 model_v3 对应 enum.go")
			continue
		}
		logger.Info("[契约自检] 通过", "table", c.table, "column", c.column, "实际", strings.Join(actual, ","))
	}
}

// distinctValues 取某表某列的去重取值（上限 20 个，防止异常数据撑爆）。
func distinctValues(ctx context.Context, client sdk.ClickHouseClient, c enumContract) ([]string, error) {
	// table / column / filter 均来自本文件内的白名单常量，非外部输入，无注入风险。
	q := fmt.Sprintf("SELECT DISTINCT %s FROM %s", c.column, c.table)
	if c.filter != "" {
		q += " WHERE " + c.filter
	}
	q += " LIMIT 20"
	rows, err := client.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var values []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			continue
		}
		if v != "" {
			values = append(values, v)
		}
	}
	return values, rows.Err()
}

// diff 返回 actual 中不在 expected 里的值。
func diff(actual, expected []string) []string {
	set := make(map[string]struct{}, len(expected))
	for _, e := range expected {
		set[e] = struct{}{}
	}
	var unknown []string
	for _, a := range actual {
		if _, ok := set[a]; !ok {
			unknown = append(unknown, a)
		}
	}
	return unknown
}

// ──────────────────────────────────────────────────────────────
// 指标采集契约：Agent 查的 node_* 指标，Collector 是否都采到了
// ──────────────────────────────────────────────────────────────

// metricsCollectedWindow 是判定"已采集"的时间窗口。node-exporter 每 15s 一轮，
// 15 分钟内没有任何一行即视为未采集（留足 Collector 重启 / 网络抖动的余量）。
const metricsCollectedWindow = "15 MINUTE"

// VerifyMetricsCollected 核对 NodeExporterMetrics 清单里的每个指标近期在 ClickHouse 是否有数据。
//
// 缺失的指标意味着 Collector 的 keep regex 与 Agent 查询漂移 —— 查询会静默返回空，
// 页面上对应卡片空白且无任何报错。2026-08 实测有 39 个指标处于此状态达数月。
//
// node-exporter 指标分布在 gauge 与 sum 两张表（counter 进 sum，瞬时值进 gauge），两张都查。
func VerifyMetricsCollected(ctx context.Context, client sdk.ClickHouseClient) {
	if client == nil {
		return
	}
	collected := make(map[string]struct{})
	for _, table := range []string{"otel_metrics_gauge", "otel_metrics_sum"} {
		names, err := recentMetricNames(ctx, client, table)
		if err != nil {
			logger.Warn("[契约自检] 指标采集核对查询失败", "table", table, "err", err)
			return // 一张表查不了就没法下结论，避免误报大批"未采集"
		}
		for _, n := range names {
			collected[n] = struct{}{}
		}
	}

	var missing []string
	for _, m := range NodeExporterMetrics {
		if _, ok := collected[m]; !ok {
			missing = append(missing, m)
		}
	}
	if len(missing) > 0 {
		logger.Error("[契约漂移] Agent 查询的 node-exporter 指标未被采集，对应卡片将空白",
			"缺失数", len(missing),
			"缺失", strings.Join(missing, ","),
			"处理", "用 NodeExporterKeepRegex() 重新生成 collector.yaml 的 keep regex")
		return
	}
	logger.Info("[契约自检] 通过", "table", "otel_metrics", "column", "MetricName",
		"清单", len(NodeExporterMetrics), "缺失", 0)
}

// recentMetricNames 取某表近期出现过的 node_* 指标名。
func recentMetricNames(ctx context.Context, client sdk.ClickHouseClient, table string) ([]string, error) {
	// table 来自本文件白名单，非外部输入。
	q := fmt.Sprintf(
		"SELECT DISTINCT MetricName FROM %s WHERE MetricName LIKE 'node_%%' AND TimeUnix > now() - INTERVAL %s",
		table, metricsCollectedWindow)
	rows, err := client.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			continue
		}
		names = append(names, n)
	}
	return names, rows.Err()
}

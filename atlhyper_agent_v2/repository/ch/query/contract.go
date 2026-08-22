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
}

// enumContracts 是全部受检契约。新增信号源时在此登记。
var enumContracts = []enumContract{
	{"otel_traces", "SpanKind", apm.ExpectedSpanKinds},
	{"otel_traces", "StatusCode", apm.ExpectedStatusCodes},
	{"otel_logs", "SeverityText", logmodel.ExpectedSeverityTexts},
}

// RunEnumContractChecks 立即执行一次契约自检，之后每 interval 重跑一次，直到 ctx 结束。
// 供 Agent 启动时以 goroutine 调用；不阻塞、不返回错误 —— 可观测性缺失属于降级，
// 不该让 Agent 起不来。
func RunEnumContractChecks(ctx context.Context, client sdk.ClickHouseClient, interval time.Duration) {
	if client == nil {
		return
	}
	VerifyEnumContracts(ctx, client)

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
		}
	}
}

// VerifyEnumContracts 检查每个契约列实际出现的取值是否都在预期集合内。
//
// 表/列为空时跳过（没有数据不代表契约错误，可能只是还没有服务上报）。
func VerifyEnumContracts(ctx context.Context, client sdk.ClickHouseClient) {
	for _, c := range enumContracts {
		actual, err := distinctValues(ctx, client, c.table, c.column)
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
func distinctValues(ctx context.Context, client sdk.ClickHouseClient, table, column string) ([]string, error) {
	// table / column 来自本文件内的白名单常量，非外部输入，不存在注入风险。
	q := fmt.Sprintf("SELECT DISTINCT %s FROM %s LIMIT 20", column, table)
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

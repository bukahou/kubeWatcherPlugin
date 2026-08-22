// contract.go 实现 ClickHouse 枚举契约自检。
//
// 存在理由:
//
//	otel_traces 的 SpanKind / StatusCode 取值由 OTel Collector 的 ClickHouse
//	exporter 决定，不受本项目控制。2026-08 的一次 Collector 升级把枚举从
//	protobuf 全名改成短名，导致所有 APM 查询的 WHERE 条件匹配不到任何行。
//
//	这类故障最难缠的地方在于【它不报错】—— SQL 执行成功、返回 0 行、日志无异常，
//	表现只是"页面上没数据"，很容易被误判成"还没有流量"。该故障因此潜伏了 27 天。
//
// 本自检把这个静默失败变成启动后 10 秒内的一条 ERROR 日志。
package query

import (
	"context"
	"fmt"
	"strings"

	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/common/logger"
	"AtlHyper/model_v3/apm"
)

// VerifyTraceEnumContract 检查 otel_traces 中实际出现的 SpanKind / StatusCode
// 是否都在预期集合内，不符则打 ERROR 日志。
//
// 表为空时跳过检查（没有数据不代表契约错误，可能只是还没有服务上报）。
// 本函数不返回错误、不阻断启动 —— 可观测性缺失属于降级，不该让 Agent 起不来。
func VerifyTraceEnumContract(ctx context.Context, client sdk.ClickHouseClient) {
	if client == nil {
		return
	}

	checks := []struct {
		column   string
		expected []string
	}{
		{"SpanKind", apm.ExpectedSpanKinds},
		{"StatusCode", apm.ExpectedStatusCodes},
	}

	for _, c := range checks {
		actual, err := distinctValues(ctx, client, c.column)
		if err != nil {
			logger.Warn("[契约自检] 查询失败", "column", c.column, "err", err)
			continue
		}
		if len(actual) == 0 {
			logger.Info("[契约自检] 跳过：otel_traces 暂无数据", "column", c.column)
			continue
		}

		if unknown := diff(actual, c.expected); len(unknown) > 0 {
			logger.Error("[契约漂移] ClickHouse 枚举取值与预期不符，APM 查询将返回空行",
				"column", c.column,
				"实际", strings.Join(actual, ","),
				"预期", strings.Join(c.expected, ","),
				"未知值", strings.Join(unknown, ","),
				"处理", "检查 OTel Collector 版本，并更新 model_v3/apm/enum.go")
			continue
		}
		logger.Info("[契约自检] 通过", "column", c.column, "实际", strings.Join(actual, ","))
	}
}

// distinctValues 取某列的去重取值（上限 20 个，防止异常数据撑爆）。
func distinctValues(ctx context.Context, client sdk.ClickHouseClient, column string) ([]string, error) {
	// column 来自本文件内的白名单常量，非外部输入，不存在注入风险。
	q := fmt.Sprintf("SELECT DISTINCT %s FROM otel_traces LIMIT 20", column)
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

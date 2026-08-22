// enum.go 定义 ClickHouse otel_traces 表中 SpanKind / StatusCode 的取值。
//
// ⚠️ 这些字符串不是 OTLP 协议规定的，而是由 OTel Collector 的 ClickHouse
// exporter 在写库时决定的 —— 应用侧用什么 SDK（Go / Java / Python）都不影响它。
//
// 历史教训（2026-08）:
//
//	Collector 升级到 0.151.0 后，exporter 把枚举从 protobuf 全名
//	（SPAN_KIND_SERVER / STATUS_CODE_ERROR）改成了短名（Server / Error）。
//	所有 APM 查询的 WHERE 条件因此匹配不到任何行 —— 查询成功、返回空、零报错，
//	该故障潜伏 27 天无人察觉，直到有服务重新开始上报 trace 才暴露。
//
// 因此：升级 OTel Collector 后必须重新验证本文件的取值。Agent 启动时会做
// 契约自检（见 repository/ch/query/contract.go），实际值与此处不符会打 ERROR 日志。
//
// 查询真实取值:
//
//	SELECT DISTINCT SpanKind FROM otel_traces;
//	SELECT DISTINCT StatusCode FROM otel_traces;
package apm

// SpanKind 取值 —— 对应 OTLP 的 span kind 枚举。
const (
	SpanKindServer   = "Server"
	SpanKindClient   = "Client"
	SpanKindInternal = "Internal"
	SpanKindProducer = "Producer"
	SpanKindConsumer = "Consumer"
)

// StatusCode 取值 —— 对应 OTLP 的 span status 枚举。
const (
	StatusCodeUnset = "Unset"
	StatusCodeOk    = "Ok"
	StatusCodeError = "Error"
)

// ExpectedSpanKinds 是契约自检时的预期集合（见 Agent 启动自检）。
var ExpectedSpanKinds = []string{
	SpanKindServer, SpanKindClient, SpanKindInternal,
	SpanKindProducer, SpanKindConsumer,
}

// ExpectedStatusCodes 是契约自检时的预期集合。
var ExpectedStatusCodes = []string{
	StatusCodeUnset, StatusCodeOk, StatusCodeError,
}

// ============================================================
// SQL 字面量 —— 供 ClickHouse 查询语句拼接使用（含单引号）
// ============================================================
//
// 查询语句里不要直接写 'Server' / 'Error' 字面量，一律引用这些常量。
// 这样 Collector 再次变更枚举格式时，只需改本文件顶部的取值定义。

const (
	SQLSpanKindServer  = "'" + SpanKindServer + "'"
	SQLSpanKindClient  = "'" + SpanKindClient + "'"
	SQLStatusCodeError = "'" + StatusCodeError + "'"
)

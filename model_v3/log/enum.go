// enum.go 定义 ClickHouse otel_logs 表中 SeverityText 的预期取值。
//
// 来源: OTLP 日志的 severity_text 由应用侧 SDK 决定。
//   - Go otelslog bridge 写 slog 的级别名: DEBUG / INFO / WARN / ERROR
//   - Java agent (logback) 写: TRACE / DEBUG / INFO / WARN / ERROR
//   - 个别运行时会出现 WARNING / SEVERE / FATAL 的别名
//
// Agent 的日志聚合 SQL (repository/ch/query/log.go) 按这些字符串计数。
// 若某个 SDK 写入了集合之外的值 (如小写 "info"), 该级别在概览里会被静默归零 ——
// 与 APM 枚举漂移 (见 model_v3/apm/enum.go) 同类问题, 因此纳入 Agent 启动/周期契约自检。
package log

// 标准级别名 (与 Agent SQL 的计数分组一致)。
const (
	SeverityTrace   = "TRACE"
	SeverityDebug   = "DEBUG"
	SeverityInfo    = "INFO"
	SeverityWarn    = "WARN"
	SeverityWarning = "WARNING" // WARN 别名
	SeverityError   = "ERROR"
	SeveritySevere  = "SEVERE" // ERROR 别名 (java.util.logging)
	SeverityFatal   = "FATAL"
)

// ExpectedSeverityTexts 是契约自检的预期集合。空字符串 (无级别) 不在此列, 自检跳过空值。
var ExpectedSeverityTexts = []string{
	SeverityTrace, SeverityDebug, SeverityInfo,
	SeverityWarn, SeverityWarning,
	SeverityError, SeveritySevere, SeverityFatal,
}

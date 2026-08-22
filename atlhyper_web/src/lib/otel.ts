/**
 * OTel span 枚举工具（单一信任源）
 *
 * 取值契约（与后端 model_v3/apm/enum.go 对齐）：
 *   ClickHouse otel_traces 表的 SpanKind / StatusCode 字段。
 *   这些字符串由 OTel Collector 的 ClickHouse exporter 决定，
 *   既不是 OTLP 协议规定的，也与应用侧用什么 SDK 无关。
 *
 * 前端禁止再直接比较 `span.statusCode === "..."` 这类字面量，
 * 一律通过本模块导出的判定函数消费。
 *
 * 历史背景：2026-08，Collector 升级到 0.151.0 后 exporter 把枚举从
 * protobuf 全名（SPAN_KIND_SERVER / STATUS_CODE_ERROR）改成短名
 * （Server / Error）。后端 24 处 SQL 与前端 4 处判定同时失效——
 * 查询静默返回空、错误计数恒为 0，且潜伏 27 天无人察觉。本模块即为根治手段。
 */

/** SpanKind 取值 */
export const SPAN_KIND = {
  server: "Server",
  client: "Client",
  internal: "Internal",
  producer: "Producer",
  consumer: "Consumer",
} as const;

/** StatusCode 取值 */
export const STATUS_CODE = {
  unset: "Unset",
  ok: "Ok",
  error: "Error",
} as const;

/** 最小 span 形状——只约束本模块用到的字段，避免与各处 Span 类型耦合 */
interface SpanLike {
  spanKind?: string;
  statusCode?: string;
  parentSpanId?: string;
}

/** 是否为错误 span */
export function isErrorSpan(span: SpanLike): boolean {
  return span.statusCode === STATUS_CODE.error;
}

/** 是否为入站 span（服务端） */
export function isServerSpan(span: SpanLike): boolean {
  return span.spanKind === SPAN_KIND.server;
}

/** 是否为出站 span（客户端）——跨服务拓扑边推导依赖它 */
export function isClientSpan(span: SpanLike): boolean {
  return span.spanKind === SPAN_KIND.client;
}

/** 是否为根 span */
export function isRootSpan(span: SpanLike): boolean {
  return !span.parentSpanId;
}

/**
 * 展示用格式化——直接返回原值。
 *
 * 旧版枚举需要剥掉 `SPAN_KIND_` / `STATUS_CODE_` 前缀才能显示，
 * 新版本身就是短名。保留此函数是为了让调用点不必再关心格式变迁。
 */
export function formatSpanKind(kind: string): string {
  return kind.replace(/^SPAN_KIND_/, "");
}

export function formatStatusCode(code: string): string {
  return code.replace(/^STATUS_CODE_/, "");
}

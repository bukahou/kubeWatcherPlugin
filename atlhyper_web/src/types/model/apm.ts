/**
 * APM 数据模型 — 对齐 model_v3/apm/trace.go
 *
 * 数据源: ClickHouse otel_traces 表
 * JSON tag 统一 camelCase，前端直接使用后端字段名。
 */

// ============================================================
// Span — otel_traces 行的领域模型
// ============================================================

export interface SpanHTTP {
  method: string;
  route?: string;
  url?: string;
  statusCode?: number;
  server?: string;
  serverPort?: number;
}

export interface SpanDB {
  system: string;
  name?: string;
  operation?: string;
  table?: string;
  statement?: string;
}

export interface SpanResource {
  serviceVersion?: string;
  instanceId?: string;
  podName?: string;
  nodeName?: string;
  deploymentName?: string;
  namespaceName?: string;
  clusterName?: string;
}

export interface SpanEvent {
  timestamp: string; // ISO 8601
  name: string;
  attributes?: Record<string, string>;
}

export type SpanErrorSource = "span_event" | "status_message" | "trace_log";

export interface SpanError {
  type: string;
  message: string;
  stacktrace?: string;
  /** 证据来源：span 自带事件 / StatusMessage / 同 trace 的 ERROR 日志回填 */
  source?: SpanErrorSource;
  /** source=trace_log 时证据日志所属服务（可能是上游，不同于 span 自身） */
  sourceService?: string;
}

export interface Span {
  timestamp: string;      // ISO 8601
  traceId: string;
  spanId: string;
  parentSpanId: string;
  spanName: string;
  spanKind: string;       // "Server" | "Client" | "Internal" ... 见 @/lib/otel
  serviceName: string;
  duration: number;       // nanoseconds
  durationMs: number;     // milliseconds
  statusCode: string;     // "Unset" | "Ok" | "Error" 见 @/lib/otel
  statusMessage: string;

  http?: SpanHTTP;
  db?: SpanDB;
  resource: SpanResource;
  events: SpanEvent[];
  error?: SpanError;
}

// ============================================================
// TraceSummary — Trace 列表项
// ============================================================

export interface TraceSummary {
  traceId: string;
  rootService: string;
  rootOperation: string;
  services: string[];
  durationMs: number;
  spanCount: number;
  serviceCount: number;
  hasError: boolean;
  errorType?: string;
  errorMessage?: string;
  timestamp: string; // ISO 8601
}

// ============================================================
// TraceDetail — 完整 Trace（瀑布图）
// ============================================================

export interface TraceDetail {
  traceId: string;
  durationMs: number;
  serviceCount: number;
  spanCount: number;
  spans: Span[];
}

// ============================================================
// APMService — 服务级聚合统计
// ============================================================

export interface APMService {
  name: string;
  namespace: string;
  environment?: string;
  spanCount: number;
  errorCount: number;
  successRate: number;    // 0-1
  avgDurationMs: number;
  p50Ms: number;
  p99Ms: number;
  rps: number;
}

// ============================================================
// OperationStats — 操作级聚合统计
// ============================================================

export interface OperationStats {
  serviceName: string;
  operationName: string;
  spanCount: number;
  errorCount: number;
  successRate: number;  // 0-1
  avgDurationMs: number;
  p50Ms: number;
  p99Ms: number;
  rps: number;
}

// ============================================================
// Topology — 服务拓扑
// ============================================================

export type HealthStatus = "healthy" | "warning" | "critical" | "unknown";

export interface TopologyNode {
  id: string;
  name: string;
  namespace: string;
  type: string;           // "service" | "database" | "external"
  rps: number;
  successRate: number;    // 0-1
  p99Ms: number;
  status: HealthStatus;
}

export interface TopologyEdge {
  source: string;
  target: string;
  callCount: number;
  avgMs: number;
  errorRate: number;      // 0-1
}

export interface Topology {
  nodes: TopologyNode[];
  edges: TopologyEdge[];
}

// ============================================================
// Computed / Chart Types
// ============================================================

export interface LatencyBucket {
  rangeStart: number;     // milliseconds
  rangeEnd: number;
  count: number;
}

export interface Dependency {
  name: string;
  type: string;           // "service" | "database" | "external"
  callCount: number;
  avgMs: number;
  errorRate: number;
  impact: number;         // 0-1
}

export interface SpanTypeBreakdown {
  type: string;           // "HTTP" | "DB" | "Other"
  percentage: number;
}

// ============================================================
// Correlations — 相关性分析（对齐 model_v3/apm/correlation.go）
// ============================================================

export type CorrelationMode = "latency" | "failure";
export type CorrelationImpact = "high" | "medium" | "low";

export interface CorrelationItem {
  field: string;
  value: string;          // "(none)" = 字段缺失
  fgCount: number;
  bgCount: number;
  fgRatio: number;        // 0-1
  bgRatio: number;        // 0-1
  lift: number;
  score: number;
  impact: CorrelationImpact;
}

export interface CorrelationResult {
  mode: CorrelationMode;
  foregroundCount: number;
  backgroundCount: number;
  lowSample: boolean;     // 前景样本 < 5，仅供参考
  thresholdMs?: number;   // latency 模式的 P95 阈值
  items: CorrelationItem[];
}

// ============================================================
// APMTimePoint — 服务时序趋势数据点（Concentrator 预聚合）
// ============================================================

export interface APMTimePoint {
  timestamp: string;   // ISO 8601
  rps: number;
  successRate: number; // 0-1
  avgMs: number;
  p99Ms: number;
  errorCount: number;
}

export interface APMServiceSeriesResponse {
  service: string;
  namespace: string;
  points: APMTimePoint[];
}

// ============================================================
// HTTP 状态码分布统计
// ============================================================

export interface HTTPStats {
  statusCode: number;
  method: string;
  operation: string;
  count: number;
}

// ============================================================
// 数据库操作统计
// ============================================================

export interface DBOperationStats {
  dbSystem: string;
  dbName: string;
  operation: string;
  table: string;
  callCount: number;
  avgMs: number;
  p99Ms: number;
  errorRate: number;
}

// ============================================================
// Helper functions
// ============================================================

// span 判定函数统一由 @/lib/otel 提供（单一信任源）。
// 此处曾有 isSpanError / isServerSpan 的重复实现，硬编码了旧版枚举字面量，
// Collector 升级后恒为 false，已移除。

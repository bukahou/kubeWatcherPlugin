/**
 * SLO 监控相关类型定义
 * 与后端 atlhyper_master_v2/model/slo.go 保持一致
 */

// SLO 状态
export type SLOStatus = "healthy" | "warning" | "critical" | "unknown";

// 趋势方向
export type SLOTrend = "up" | "down" | "stable";

// SLO 指标（对应 model.SLOMetrics）
export interface SLOMetrics {
  availability: number;       // 可用性 (0-100)
  p95Latency: number;         // P95 延迟 (ms)
  p99Latency: number;         // P99 延迟 (ms)
  errorRate: number;           // 错误率 (0-100)
  requestsPerSec: number;     // 每秒请求数
  totalRequests: number;
  goodRequests: number;
  badRequests: number;       // 总请求数
}

// SLO 目标规格（对应 model.SLOTargetSpec）
// 一个域名一组目标：固定滚动窗口 + 可用率 + P95。
// 页面上的时间范围切换只影响图表画多长，不改变目标。
export interface SLOTargetSpec {
  availability: number;       // 目标可用性
  p95Latency: number;         // 目标 P95 延迟 (ms)
  windowDays: number;         // 滚动窗口天数
}

// 单个窗口的燃烧率判定（对应 slo.BurnRateWindow）
export interface BurnRateWindow {
  window: string;             // 1h / 6h / 24h / 3d
  rate: number;               // 燃烧率倍数
  threshold: number;          // 该窗口的告警阈值
  status: "good" | "warn" | "crit";
}

// 错误预算与燃烧率（对应 model.SLOBudget）
// 判定全部在后端完成，前端只渲染颜色与进度条。
export interface SLOBudget {
  remainingPct: number;
  allowedEvents: number;      // 窗口内允许出错的请求数
  consumedEvents: number;     // 已经出错的请求数
  burnRates: BurnRateWindow[];
  exhaustHours: number;       // 按当前燃烧率还能撑多少小时；0 = 窗口内不会耗尽
}

// 域名 SLO（对应 model.DomainSLO）
export interface DomainSLO {
  host: string;
  ingressName: string;
  ingressClass: string;
  namespace: string;
  tls: boolean;
  target: SLOTargetSpec;
  budget?: SLOBudget;
  current: SLOMetrics | null;
  previous?: SLOMetrics | null;
  errorBudgetRemaining: number;
  status: SLOStatus;
  trend: SLOTrend;
}

// SLO 汇总（对应 model.SLOSummary）
export interface SLOSummary {
  totalServices: number;
  totalDomains: number;
  healthyCount: number;
  warningCount: number;
  criticalCount: number;
  avgAvailability: number;
  avgErrorBudget: number;
  totalRps: number;
}

// 域名 SLO 列表响应（对应 model.SLODomainsResponse）
export interface DomainSLOListResponse {
  domains: DomainSLO[];
  summary: SLOSummary;
}

// 域名 SLO 详情（复用 DomainSLO）
export type DomainSLODetail = DomainSLO;

// 历史数据点（对应 model.SLODomainHistoryItem）
export interface SLOHistoryPoint {
  timestamp: string;
  availability: number;
  p95Latency: number;
  p99Latency: number;
  rps: number;
  errorRate: number;
  errorBudget: number;
}

// 域名历史数据响应（对应 model.SLODomainHistoryResponse）
export interface DomainSLOHistoryResponse {
  host: string;
  history: SLOHistoryPoint[];
}

// SLO 目标配置（camelCase，匹配后端 model.SLOTargetResponse）
export interface SLOTarget {
  id?: number;
  clusterId: string;
  host: string;
  windowDays: number;
  availabilityTarget: number;
  p95LatencyTarget: number;
  createdAt?: string;
  updatedAt?: string;
}

// 状态变更历史项
export interface SLOStatusHistoryItem {
  host: string;
  timeRange: string;
  oldStatus: SLOStatus;
  newStatus: SLOStatus;
  availability: number;
  p95Latency: number;
  errorBudgetRemaining: number;
  changedAt: string;
}

// 状态历史响应
export type SLOStatusHistoryResponse = SLOStatusHistoryItem[];

// ==================== V2 API 类型（按真实域名分组） ====================

// 后端服务级别 SLO（对应 model.ServiceSLO）
// 注意：Metrics 数据是按 service 级别聚合的，不是按 path 级别
export interface ServiceSLO {
  serviceKey: string;              // Traefik service key (namespace-name-port@kubernetes)
  serviceName: string;             // 服务名称
  servicePort: number;             // 服务端口
  namespace: string;               // 命名空间
  paths: string[];                 // 使用该服务的路径列表（仅展示用，共享同一份 metrics）
  ingressName: string;             // IngressRoute/Ingress 名称
  current: SLOMetrics | null;      // 当前周期指标
  previous?: SLOMetrics | null;    // 上一周期指标
  target: SLOTargetSpec;          // 该服务的 SLO 目标
  budget?: SLOBudget;             // 错误预算与燃烧率
  status: SLOStatus;               // 状态
  errorBudgetRemaining: number;    // 剩余错误预算
}

// 域名级别 SLO（对应 model.DomainSLOResponseV2）
export interface DomainSLOV2 {
  domain: string;                  // 真实域名（如 example.com）
  tls: boolean;                    // 是否启用 TLS
  services: ServiceSLO[];          // 该域名下的所有后端服务
  summary: SLOMetrics | null;      // 域名级别汇总指标
  previous?: SLOMetrics | null;    // 上一周期汇总指标
  target: SLOTargetSpec;           // 该域名的 SLO 目标
  budget?: SLOBudget;              // 错误预算与燃烧率
  status: SLOStatus;               // 域名状态
  errorBudgetRemaining: number;    // 域名剩余错误预算
}

// 数据提供者信息（Ingress 和 Mesh 解耦）
export interface SLOProviders {
  ingress: "traefik" | "nginx" | null;   // Ingress 控制器类型
  mesh: "linkerd" | "istio" | null;       // 服务网格类型（null = 无网格）
  traces: boolean;                         // 是否有 OTel Traces 数据
}

// V2 域名列表响应（对应 model.SLODomainsResponseV2）
export interface DomainSLOListResponseV2 {
  domains: DomainSLOV2[];
  summary: SLOSummary;
  providers: SLOProviders;                // 当前集群的数据提供者
}

// ==================== 延迟分布 API 类型 ====================

// 延迟分布桶
export interface LatencyBucket {
  le: number;       // 上界 (ms)
  count: number;    // 该桶内的请求数
}

// HTTP 方法分布
export interface MethodBreakdown {
  method: string;   // GET, POST, PUT, DELETE, OTHER
  count: number;
}

// 状态码分布
export interface StatusCodeBreakdown {
  code: string;     // "2xx", "3xx", "4xx", "5xx"
  count: number;
}

// 延迟分布响应
export interface LatencyDistributionResponse {
  domain: string;
  totalRequests: number;
  p50LatencyMs: number;
  p95LatencyMs: number;
  p99LatencyMs: number;
  avgLatencyMs: number;
  buckets: LatencyBucket[];
  methods: MethodBreakdown[];
  statusCodes: StatusCodeBreakdown[];
}

// 延迟分布请求参数
export interface SLOLatencyParams {
  clusterId: string;
  domain: string;
  timeRange?: string;
}

// API 请求参数
export interface SLODomainsParams {
  clusterId?: string;
  timeRange?: string;  // 查看范围：1h | 6h | 24h | 3d | 7d（不影响 SLO 目标窗口）
}

export interface SLODomainDetailParams {
  clusterId: string;
  host: string;
  timeRange?: string;
}

export interface SLODomainHistoryParams {
  clusterId: string;
  host: string;
  timeRange?: string;
}

export interface SLOTargetCreateParams {
  clusterId: string;
  host: string;
  windowDays: number;
  availabilityTarget: number;
  p95LatencyTarget: number;
}

export interface SLOStatusHistoryParams {
  clusterId?: string;
  host?: string;
  limit?: number;
}

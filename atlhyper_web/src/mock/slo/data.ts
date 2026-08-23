/**
 * SLO Mock 数据
 *
 * 严格基于 docs/design/active/clickhouse-otel-data-reference.md 真实数据
 * 2 个域名: geass (Linkerd mesh + Traces) / atlhyper (Traefik ingress only)
 */

import type {
  DomainSLOV2,
  ServiceSLO,
  SLOMetrics,
  SLOSummary,
  SLOProviders,
  SLOHistoryPoint,
  LatencyBucket,
  MethodBreakdown,
  StatusCodeBreakdown,
} from "@/types/slo";

// ============================================================================
// Providers — Traefik + Linkerd + OTel Traces (集群 zgmf-x10a 真实配置)
// ============================================================================

export const MOCK_PROVIDERS: SLOProviders = {
  ingress: "traefik",
  mesh: "linkerd",
  traces: true,
};

// ============================================================================
// 辅助函数
// ============================================================================

function mkMetrics(
  availability: number,
  p95: number,
  p99: number,
  errorRate: number,
  rps: number,
  total: number,
): SLOMetrics {
  const bad = Math.round(total * errorRate / 100);
  return {
    availability, p95Latency: p95, p99Latency: p99, errorRate,
    requestsPerSec: rps, totalRequests: total,
    goodRequests: total - bad, badRequests: bad,
  };
}

function mkService(
  serviceKey: string,
  serviceName: string,
  port: number,
  namespace: string,
  paths: string[],
  ingressName: string,
  current: SLOMetrics,
  status: "healthy" | "warning" | "critical",
  errorBudget: number,
  previous?: SLOMetrics,
): ServiceSLO {
  return {
    serviceKey, serviceName, servicePort: port, namespace, paths, ingressName,
    current, previous: previous ?? null,
    target: { availability: 99.9, p95Latency: 200, windowDays: 7 },
    status, errorBudgetRemaining: errorBudget,
  };
}

// ============================================================================
// 真实数据常量 (from clickhouse-otel-data-reference.md)
// ============================================================================

// Traefik Histogram ExplicitBounds (秒→毫秒)
const TRAEFIK_BOUNDS_MS = [5, 10, 25, 50, 75, 100, 150, 200, 300, 500, 750, 1000, 2500, 5000, 10000];

// atlhyper 域真实 BucketCounts (§8.2: service=atlhyper-atlhyper-web-3000@kubernetes, code=404)
// Count=8762, Sum=47.408s, Avg=5.41ms
const ATLHYPER_BUCKET_COUNTS = [6768, 1592, 282, 51, 22, 21, 13, 6, 3, 4, 0, 0, 0, 0, 0, 0];

// Linkerd le 桶 (§6.3): [1, 2, 4, 10, 40, 40000] ms
// geass 域 6 个微服务, 大多请求 < 10ms (Trace 样本: auth 2ms, history 32ms, media 26ms, DB 1-3ms)

// ============================================================================
// 域名 1: geass.zgmf-x10a.local — Healthy
// Traefik service: geass-geass-web-3000@kubernetes
// 后端: 6 个 Linkerd mesh 微服务 (gateway, media, auth, favorites, history, user)
// ============================================================================

const geassServices: ServiceSLO[] = [
  mkService(
    "geass-geass-web-3000@kubernetes", "geass-web", 3000, "geass",
    ["/", "/api/*"], "geass-ingressroute",
    mkMetrics(99.95, 12, 35, 0.05, 85, 7344000),
    "healthy", 92.0,
    mkMetrics(99.93, 14, 38, 0.07, 80, 6912000),
  ),
];

const domainGeass: DomainSLOV2 = {
  domain: "geass.zgmf-x10a.local",
  tls: true,
  services: geassServices,
  summary: mkMetrics(99.95, 12, 35, 0.05, 85, 7344000),
  previous: mkMetrics(99.93, 14, 38, 0.07, 80, 6912000),
  target: { availability: 99.9, p95Latency: 200, windowDays: 7 },
  status: "healthy",
  errorBudgetRemaining: 92.0,
};

// ============================================================================
// 域名 2: atlhyper.zgmf-x10a.local — Warning
// Traefik service: atlhyper-atlhyper-web-3000@kubernetes
// Next.js 前端 SPA, 大量静态资源 404
// Histogram: Count=8762, Sum=47.408s, Avg=5.41ms, p95≈25ms, p99≈75ms
// ============================================================================

const atlhyperServices: ServiceSLO[] = [
  mkService(
    "atlhyper-atlhyper-web-3000@kubernetes", "atlhyper-web", 3000, "atlhyper",
    ["/", "/api/*", "/_next/*"], "atlhyper-ingressroute",
    mkMetrics(99.42, 25, 75, 0.58, 6, 518400),
    "warning", 28.0,
    mkMetrics(99.65, 20, 60, 0.35, 7, 604800),
  ),
];

const domainAtlhyper: DomainSLOV2 = {
  domain: "atlhyper.zgmf-x10a.local",
  tls: true,
  services: atlhyperServices,
  summary: mkMetrics(99.42, 25, 75, 0.58, 6, 518400),
  previous: mkMetrics(99.65, 20, 60, 0.35, 7, 604800),
  target: { availability: 99.9, p95Latency: 200, windowDays: 7 },
  status: "warning",
  errorBudgetRemaining: 28.0,
};

// ============================================================================
// 汇总
// ============================================================================

export const MOCK_DOMAINS: DomainSLOV2[] = [domainGeass, domainAtlhyper];

export const MOCK_SUMMARY: SLOSummary = {
  totalServices: MOCK_DOMAINS.reduce((s, d) => s + d.services.length, 0),
  totalDomains: MOCK_DOMAINS.length,
  healthyCount: MOCK_DOMAINS.filter((d) => d.status === "healthy").length,
  warningCount: MOCK_DOMAINS.filter((d) => d.status === "warning").length,
  criticalCount: MOCK_DOMAINS.filter((d) => d.status === "critical").length,
  avgAvailability:
    MOCK_DOMAINS.reduce((s, d) => s + (d.summary?.availability ?? 0), 0) / MOCK_DOMAINS.length,
  avgErrorBudget:
    MOCK_DOMAINS.reduce((s, d) => s + d.errorBudgetRemaining, 0) / MOCK_DOMAINS.length,
  totalRps: MOCK_DOMAINS.reduce((s, d) => s + (d.summary?.requestsPerSec ?? 0), 0),
};

// ============================================================================
// 历史数据生成器
// ============================================================================

export function generateHistory(domain: string, timeRange: string): SLOHistoryPoint[] {
  const d = MOCK_DOMAINS.find((dd) => dd.domain === domain);
  if (!d || !d.summary) return [];

  const hours = timeRange === "30d" ? 720 : timeRange === "7d" ? 168 : 24;
  const interval = timeRange === "30d" ? 6 : timeRange === "7d" ? 2 : 1;
  const points: SLOHistoryPoint[] = [];
  const now = Date.now();
  const base = d.summary;

  for (let i = hours; i >= 0; i -= interval) {
    const ts = new Date(now - i * 3600000).toISOString();
    const jitter = () => (Math.random() - 0.5) * 2;
    const avail = Math.min(100, Math.max(95, base.availability + jitter() * 0.3));
    const p95 = Math.max(5, base.p95Latency + jitter() * base.p95Latency * 0.15);
    const p99 = Math.max(10, base.p99Latency + jitter() * base.p99Latency * 0.15);
    const rps = Math.max(0, base.requestsPerSec + jitter() * base.requestsPerSec * 0.2);
    const errRate = Math.max(0, base.errorRate + jitter() * 0.1);
    const budget = Math.max(0, Math.min(100, d.errorBudgetRemaining + jitter() * 5));

    points.push({
      timestamp: ts,
      availability: +avail.toFixed(3),
      p95Latency: +p95.toFixed(1),
      p99Latency: +p99.toFixed(1),
      rps: +rps.toFixed(1),
      errorRate: +errRate.toFixed(3),
      errorBudget: +budget.toFixed(1),
    });
  }
  return points;
}

// ============================================================================
// 延迟分布数据生成器
// ============================================================================

export function generateLatencyDistribution(domain: string): {
  buckets: LatencyBucket[];
  methods: MethodBreakdown[];
  statusCodes: StatusCodeBreakdown[];
  p50: number;
  p95: number;
  p99: number;
  avg: number;
  total: number;
} {
  const d = MOCK_DOMAINS.find((dd) => dd.domain === domain);
  const m = d?.summary;
  const total = m?.totalRequests ?? 100000;

  // atlhyper 域: 直接使用真实 Traefik histogram 数据
  if (domain === "atlhyper.zgmf-x10a.local") {
    return generateAtlhyperLatency(total);
  }

  // geass 域: 基于 Linkerd 延迟特征构造分布 (大多 < 10ms)
  return generateGeassLatency(total);
}

/** atlhyper 域: 真实 Traefik histogram (§8.2) */
function generateAtlhyperLatency(total: number) {
  // 真实 BucketCounts 比例 (Count=8762)
  const realTotal = ATLHYPER_BUCKET_COUNTS.reduce((a, b) => a + b, 0);
  const buckets: LatencyBucket[] = TRAEFIK_BOUNDS_MS.map((le, i) => ({
    le,
    count: Math.round((ATLHYPER_BUCKET_COUNTS[i] / realTotal) * total),
  }));
  // +Inf 桶的 count 归入最后一个显式桶
  if (ATLHYPER_BUCKET_COUNTS.length > TRAEFIK_BOUNDS_MS.length) {
    const infCount = ATLHYPER_BUCKET_COUNTS[TRAEFIK_BOUNDS_MS.length];
    buckets[buckets.length - 1].count += Math.round((infCount / realTotal) * total);
  }

  // HTTP 方法: 真实 Traefik 方法 (GET 为主, 少量 POST/PUT/HEAD)
  const methods: MethodBreakdown[] = [
    { method: "GET", count: Math.floor(total * 0.72) },
    { method: "POST", count: Math.floor(total * 0.18) },
    { method: "PUT", count: Math.floor(total * 0.06) },
    { method: "HEAD", count: Math.floor(total * 0.04) },
  ];

  // 状态码: 真实 Traefik 状态码 — atlhyper 以 404 为主 (静态资源未找到)
  const statusCodes: StatusCodeBreakdown[] = [
    { code: "200", count: Math.floor(total * 0.88) },
    { code: "404", count: Math.floor(total * 0.08) },
    { code: "302", count: Math.floor(total * 0.03) },
    { code: "401", count: Math.floor(total * 0.01) },
  ];

  return {
    buckets, methods, statusCodes,
    p50: 3.8,    // 从 BucketCounts 推算: 77.2% < 5ms → p50 ≈ 3.8ms
    p95: 25,     // 98.5% < 25ms → p95 ≈ 25ms
    p99: 75,     // 从尾部分布推算
    avg: 5.41,   // 真实值: Sum/Count = 47.408/8762
    total,
  };
}

/** geass 域: 基于 Linkerd le 桶 [1, 2, 4, 10, 40, 40000] ms 构造分布 */
function generateGeassLatency(total: number) {
  // Linkerd mesh 延迟特征: 大多请求 < 10ms
  // Trace 样本: auth 2ms, history 32ms, media 26ms, gateway 87ms(聚合), DB 1-3ms
  // 按 Traefik ExplicitBounds 展示, 但值基于 Linkerd 特征
  const bucketRatios = [
    0.35,  // < 5ms   — DB 查询 (1-3ms) + auth (2ms)
    0.30,  // < 10ms  — 大部分业务逻辑
    0.18,  // < 25ms  — media/history 服务
    0.08,  // < 50ms  — 复杂聚合
    0.04,  // < 75ms  — 网关聚合 (gateway 87ms 样本)
    0.02,  // < 100ms — 尾部延迟
    0.015, // < 150ms
    0.005, // < 200ms
    0.003, // < 300ms
    0.002, // < 500ms
    0.001, // < 750ms
    0.001, // < 1000ms
    0.001, // < 2500ms
    0.001, // < 5000ms
    0.001, // < 10000ms
  ];

  const buckets: LatencyBucket[] = TRAEFIK_BOUNDS_MS.map((le, i) => ({
    le,
    count: Math.round(bucketRatios[i] * total),
  }));

  // HTTP 方法: geass 以 POST 为主 (API 调用, Trace 数据中 POST 占多数)
  const methods: MethodBreakdown[] = [
    { method: "POST", count: Math.floor(total * 0.65) },
    { method: "GET", count: Math.floor(total * 0.28) },
    { method: "PUT", count: Math.floor(total * 0.05) },
    { method: "HEAD", count: Math.floor(total * 0.02) },
  ];

  // 状态码: geass 几乎全 200 (Linkerd status_code: 200, 少量 404/500)
  const statusCodes: StatusCodeBreakdown[] = [
    { code: "200", count: Math.floor(total * 0.9980) },
    { code: "404", count: Math.floor(total * 0.0012) },
    { code: "500", count: Math.floor(total * 0.0005) },
    { code: "302", count: Math.floor(total * 0.0003) },
  ];

  return {
    buckets, methods, statusCodes,
    p50: 4.2,   // 65% < 10ms → p50 ≈ 4.2ms
    p95: 12,    // 基于 Linkerd 延迟特征
    p99: 35,
    avg: 7.8,   // 加权平均
    total,
  };
}

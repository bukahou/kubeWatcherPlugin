/**
 * APM Mock — 计算型数据 (延迟分布、依赖、Span类型分布)
 */

import type {
  TraceSummary,
  LatencyBucket,
  Dependency,
  SpanTypeBreakdown,
} from "@/types/model/apm";
import { mockTraces } from "./data";

/**
 * 延迟分布直方图 (毫秒单位, Kibana 风格)
 */
export function mockGetLatencyDistribution(
  traces: TraceSummary[]
): LatencyBucket[] {
  // Kibana-style dense boundaries in ms
  const boundaries = [
    // 1-10ms
    1, 1.5, 2, 2.5, 3, 3.5, 4, 5, 6, 7, 8, 9,
    // 10-100ms
    10, 12, 15, 18, 20, 25, 30, 35, 40, 50, 60, 70, 80, 90,
    // 100ms-1s
    100, 120, 150, 200, 250, 300, 400, 500, 600, 700, 800, 900,
    // 1-10s
    1000, 1200, 1500, 2000, 3000, 4000, 5000, 6000, 8000,
    // 10-50s
    10000, 15000, 20000, 30000, 40000, 50000,
  ];

  const buckets: LatencyBucket[] = boundaries.slice(0, -1).map((start, i) => ({
    rangeStart: start,
    rangeEnd: boundaries[i + 1],
    count: 0,
  }));

  for (const trace of traces) {
    for (let i = buckets.length - 1; i >= 0; i--) {
      if (trace.durationMs >= buckets[i].rangeStart) {
        buckets[i].count++;
        break;
      }
    }
  }

  return buckets;
}

/**
 * 服务依赖列表
 */
export function mockGetDependencies(serviceName: string): Dependency[] {
  const depMap = new Map<string, {
    type: string;
    durations: number[];
    errorCount: number;
  }>();

  for (const trace of mockTraces) {
    const spanMap = new Map(trace.spans.map((s) => [s.spanId, s]));

    for (const span of trace.spans) {
      if (!span.parentSpanId) continue;
      const parent = spanMap.get(span.parentSpanId);
      if (!parent || parent.serviceName !== serviceName) continue;
      if (span.serviceName === serviceName) continue;

      // Determine dependency type
      let type = "service";
      if (span.db) type = "database";

      const depName = span.db ? `mysql:${span.db.name || "geass_v2"}` : span.serviceName;
      if (!depMap.has(depName)) {
        depMap.set(depName, { type, durations: [], errorCount: 0 });
      }
      const dep = depMap.get(depName)!;
      dep.durations.push(span.durationMs);
      if (span.statusCode === "Error") dep.errorCount++;
    }
  }

  const allTotals = Array.from(depMap.values()).map((d) =>
    d.durations.reduce((a, b) => a + b, 0)
  );
  const maxTotal = Math.max(...allTotals, 1);

  return Array.from(depMap.entries()).map(([name, data]) => {
    const count = data.durations.length;
    const totalDuration = data.durations.reduce((a, b) => a + b, 0);
    return {
      name,
      type: data.type,
      callCount: count,
      avgMs: count > 0 ? Math.round((totalDuration / count) * 100) / 100 : 0,
      errorRate: count > 0 ? Math.round((data.errorCount / count) * 10000) / 10000 : 0,
      impact: totalDuration / maxTotal,
    };
  });
}

/**
 * Span 类型分布 — 基于结构化属性精确分类
 */
export function mockGetSpanTypeBreakdown(
  serviceName: string
): SpanTypeBreakdown[] {
  let httpTime = 0;
  let dbTime = 0;
  let otherTime = 0;

  for (const trace of mockTraces) {
    for (const span of trace.spans) {
      if (span.serviceName !== serviceName) continue;

      if (span.http) {
        httpTime += span.durationMs;
      } else if (span.db) {
        dbTime += span.durationMs;
      } else {
        otherTime += span.durationMs;
      }
    }
  }

  const total = httpTime + dbTime + otherTime;
  if (total === 0) return [{ type: "Other", percentage: 100 }];

  const result: SpanTypeBreakdown[] = [];
  if (httpTime > 0) result.push({ type: "HTTP", percentage: Math.round((httpTime / total) * 1000) / 10 });
  if (dbTime > 0) result.push({ type: "DB", percentage: Math.round((dbTime / total) * 1000) / 10 });
  if (otherTime > 0) result.push({ type: "Other", percentage: Math.round((otherTime / total) * 1000) / 10 });
  return result;
}

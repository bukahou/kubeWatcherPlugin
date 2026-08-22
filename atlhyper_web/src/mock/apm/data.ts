/**
 * APM Mock 静态数据 — 基于 ClickHouse 真实数据特征生成
 *
 * 6 个 geass 服务, ~80 traces, ~800 spans
 * 参考: docs/design/active/clickhouse-otel-data-reference.md
 */

import type { Span, TraceSummary, TraceDetail, APMService, Topology } from "@/types/model/apm";

// ============================================================
// Deterministic random
// ============================================================

function seededRandom(seed: string): () => number {
  let h = 0;
  for (let i = 0; i < seed.length; i++) {
    h = (Math.imul(31, h) + seed.charCodeAt(i)) | 0;
  }
  return () => {
    h = (Math.imul(h, 1103515245) + 12345) | 0;
    return ((h >>> 16) & 0x7fff) / 0x7fff;
  };
}

function hexId(rand: () => number, len: number): string {
  const hex = "0123456789abcdef";
  let result = "";
  for (let i = 0; i < len; i++) {
    result += hex[Math.floor(rand() * 16)];
  }
  return result;
}

// ============================================================
// Service definitions (based on ClickHouse 3.1)
// ============================================================

interface ServiceDef {
  name: string;
  namespace: string;
  port: number;
  endpoints: string[];
  dbTables: string[];
  podSuffix: string;
}

const SERVICES: ServiceDef[] = [
  {
    name: "geass-gateway",
    namespace: "default",
    port: 8080,
    endpoints: [
      "POST /api/history/list",
      "POST /api/ura-anime/sort/release",
      "POST /public/anime/sort/release",
      "POST /api/favorites/list",
      "GET /api/user/info",
      "POST /api/media/search",
    ],
    dbTables: [],
    podSuffix: "ff5988887-fmztl",
  },
  {
    name: "geass-media",
    namespace: "default",
    port: 8081,
    endpoints: [
      "POST /batch/fetch",
      "POST /ura-anime/sort/release",
      "POST /anime/v2/sort/release",
      "POST /media/search",
    ],
    dbTables: ["anime", "av", "ura_anime", "drama"],
    podSuffix: "6d8b9c7f5-k2m3n",
  },
  {
    name: "geass-auth",
    namespace: "default",
    port: 8085,
    endpoints: ["POST /token/verify"],
    dbTables: [],
    podSuffix: "4c5d6e7f8-p9q0r",
  },
  {
    name: "geass-favorites",
    namespace: "default",
    port: 8082,
    endpoints: ["POST /favorites/v2/list", "POST /favorites/add"],
    dbTables: ["favorites"],
    podSuffix: "3b4c5d6e7-s1t2u",
  },
  {
    name: "geass-history",
    namespace: "default",
    port: 8083,
    endpoints: ["POST /history/v2/list"],
    dbTables: ["history"],
    podSuffix: "2a3b4c5d6-v3w4x",
  },
  {
    name: "geass-user",
    namespace: "default",
    port: 8084,
    endpoints: ["POST /auth/login", "GET /user/profile"],
    dbTables: ["user_logins"],
    podSuffix: "1z2a3b4c5-y5z6a",
  },
];

const CLUSTER_NAME = "zgmf-x10a";

// ============================================================
// Trace generation patterns (based on ClickHouse 4.2)
// ============================================================

interface TracePattern {
  gatewayEndpoint: string;
  downstream: { service: ServiceDef; endpoint: string }[];
  weight: number; // relative frequency
}

function svc(name: string): ServiceDef {
  return SERVICES.find((s) => s.name === name)!;
}

const TRACE_PATTERNS: TracePattern[] = [
  {
    gatewayEndpoint: "POST /api/history/list",
    downstream: [
      { service: svc("geass-history"), endpoint: "POST /history/v2/list" },
      { service: svc("geass-media"), endpoint: "POST /batch/fetch" },
    ],
    weight: 20,
  },
  {
    gatewayEndpoint: "POST /api/ura-anime/sort/release",
    downstream: [
      { service: svc("geass-media"), endpoint: "POST /ura-anime/sort/release" },
    ],
    weight: 15,
  },
  {
    gatewayEndpoint: "POST /public/anime/sort/release",
    downstream: [
      { service: svc("geass-media"), endpoint: "POST /anime/v2/sort/release" },
    ],
    weight: 12,
  },
  {
    gatewayEndpoint: "POST /api/favorites/list",
    downstream: [
      { service: svc("geass-favorites"), endpoint: "POST /favorites/v2/list" },
      { service: svc("geass-media"), endpoint: "POST /batch/fetch" },
    ],
    weight: 15,
  },
  {
    gatewayEndpoint: "GET /api/user/info",
    downstream: [
      { service: svc("geass-user"), endpoint: "GET /user/profile" },
    ],
    weight: 8,
  },
  {
    gatewayEndpoint: "POST /api/media/search",
    downstream: [
      { service: svc("geass-media"), endpoint: "POST /media/search" },
    ],
    weight: 10,
  },
];

// ============================================================
// Generate spans for a single trace
// ============================================================

function makeResource(svcDef: ServiceDef) {
  return {
    serviceVersion: "0.0.1-SNAPSHOT",
    podName: `${svcDef.name}-${svcDef.podSuffix}`,
    clusterName: CLUSTER_NAME,
    instanceId: `${svcDef.name}-instance-1`,
  };
}

function generateTrace(
  rand: () => number,
  pattern: TracePattern,
  baseTime: Date,
  isError: boolean,
): { spans: Span[]; traceId: string } {
  const traceId = hexId(rand, 32);
  const spans: Span[] = [];
  const gateway = svc("geass-gateway");
  const auth = svc("geass-auth");

  // Total gateway duration: 50-150ms
  const gatewayDurationMs = 50 + rand() * 100;
  const gatewayStart = baseTime.toISOString();
  const gatewaySpanId = hexId(rand, 16);

  // Gateway SERVER span (root)
  spans.push({
    timestamp: gatewayStart,
    traceId,
    spanId: gatewaySpanId,
    parentSpanId: "",
    spanName: pattern.gatewayEndpoint,
    spanKind: "Server",
    serviceName: gateway.name,
    duration: Math.round(gatewayDurationMs * 1e6),
    durationMs: Math.round(gatewayDurationMs * 100) / 100,
    statusCode: isError ? "Error" : "Unset",
    statusMessage: isError ? "Internal Server Error" : "",
    http: {
      method: pattern.gatewayEndpoint.split(" ")[0],
      route: pattern.gatewayEndpoint.split(" ")[1],
      statusCode: isError ? 500 : 200,
      server: gateway.name,
      serverPort: gateway.port,
    },
    resource: makeResource(gateway),
    events: isError
      ? [{
          timestamp: new Date(baseTime.getTime() + gatewayDurationMs * 0.8).toISOString(),
          name: "exception",
          attributes: {
            "exception.type": "java.lang.RuntimeException",
            "exception.message": "Downstream service timeout",
            "exception.stacktrace": "java.lang.RuntimeException: Downstream service timeout\n\tat com.geass.gateway.handler.ProxyHandler.invoke(ProxyHandler.java:142)\n\tat com.geass.gateway.filter.AuthFilter.doFilter(AuthFilter.java:58)\n\tat org.springframework.web.servlet.FrameworkServlet.service(FrameworkServlet.java:897)\n\tat javax.servlet.http.HttpServlet.service(HttpServlet.java:750)",
          },
        }]
      : [],
    ...(isError ? {
      error: {
        type: "java.lang.RuntimeException",
        message: "Downstream service timeout",
        stacktrace: "java.lang.RuntimeException: Downstream service timeout\n\tat com.geass.gateway.handler.ProxyHandler.invoke(ProxyHandler.java:142)\n\tat com.geass.gateway.filter.AuthFilter.doFilter(AuthFilter.java:58)\n\tat org.springframework.web.servlet.FrameworkServlet.service(FrameworkServlet.java:897)\n\tat javax.servlet.http.HttpServlet.service(HttpServlet.java:750)",
      },
    } : {}),
  });

  let timeOffset = 2; // ms offset from gateway start

  // Auth call (always present): gateway CLIENT → auth SERVER
  const authClientId = hexId(rand, 16);
  const authServerId = hexId(rand, 16);
  const authDurationMs = 2 + rand() * 6; // 2-8ms

  spans.push({
    timestamp: new Date(baseTime.getTime() + timeOffset).toISOString(),
    traceId,
    spanId: authClientId,
    parentSpanId: gatewaySpanId,
    spanName: "POST",
    spanKind: "Client",
    serviceName: gateway.name,
    duration: Math.round(authDurationMs * 1e6),
    durationMs: Math.round(authDurationMs * 100) / 100,
    statusCode: "Unset",
    statusMessage: "",
    http: {
      method: "POST",
      statusCode: 200,
      url: `http://${auth.name}:${auth.port}/token/verify`,
      server: auth.name,
      serverPort: auth.port,
    },
    resource: makeResource(gateway),
    events: [],
  });

  spans.push({
    timestamp: new Date(baseTime.getTime() + timeOffset + 1).toISOString(),
    traceId,
    spanId: authServerId,
    parentSpanId: authClientId,
    spanName: "POST /token/verify",
    spanKind: "Server",
    serviceName: auth.name,
    duration: Math.round((authDurationMs - 1) * 1e6),
    durationMs: Math.round((authDurationMs - 1) * 100) / 100,
    statusCode: "Unset",
    statusMessage: "",
    http: {
      method: "POST",
      route: "/token/verify",
      statusCode: 200,
      server: auth.name,
      serverPort: auth.port,
    },
    resource: makeResource(auth),
    events: [],
  });

  timeOffset += authDurationMs + 1;

  // Downstream calls
  for (const ds of pattern.downstream) {
    const clientId = hexId(rand, 16);
    const serverId = hexId(rand, 16);
    const dsDurationMs = 10 + rand() * 40; // 10-50ms
    const dsIsError = isError && rand() > 0.7;

    // Gateway CLIENT span
    spans.push({
      timestamp: new Date(baseTime.getTime() + timeOffset).toISOString(),
      traceId,
      spanId: clientId,
      parentSpanId: gatewaySpanId,
      spanName: pattern.gatewayEndpoint.split(" ")[0],
      spanKind: "Client",
      serviceName: gateway.name,
      duration: Math.round(dsDurationMs * 1e6),
      durationMs: Math.round(dsDurationMs * 100) / 100,
      statusCode: dsIsError ? "Error" : "Unset",
      statusMessage: "",
      http: {
        method: pattern.gatewayEndpoint.split(" ")[0],
        statusCode: dsIsError ? 500 : 200,
        url: `http://${ds.service.name}:${ds.service.port}${ds.endpoint.split(" ")[1]}`,
        server: ds.service.name,
        serverPort: ds.service.port,
      },
      resource: makeResource(gateway),
      events: [],
    });

    // Downstream SERVER span
    const serverDurationMs = dsDurationMs - 2 - rand() * 3;
    spans.push({
      timestamp: new Date(baseTime.getTime() + timeOffset + 1).toISOString(),
      traceId,
      spanId: serverId,
      parentSpanId: clientId,
      spanName: ds.endpoint,
      spanKind: "Server",
      serviceName: ds.service.name,
      duration: Math.round(serverDurationMs * 1e6),
      durationMs: Math.round(serverDurationMs * 100) / 100,
      statusCode: dsIsError ? "Error" : "Unset",
      statusMessage: dsIsError ? "Database query failed" : "",
      http: {
        method: ds.endpoint.split(" ")[0],
        route: ds.endpoint.split(" ")[1],
        statusCode: dsIsError ? 500 : 200,
        server: ds.service.name,
        serverPort: ds.service.port,
      },
      resource: makeResource(ds.service),
      events: dsIsError
        ? [{
            timestamp: new Date(baseTime.getTime() + timeOffset + serverDurationMs * 0.9).toISOString(),
            name: "exception",
            attributes: {
              "exception.type": "java.sql.SQLException",
              "exception.message": "Database query failed",
              "exception.stacktrace": "java.sql.SQLException: Database query failed\n\tat com.geass.repository.BaseRepository.execute(BaseRepository.java:87)\n\tat com.geass.service.DataService.query(DataService.java:45)",
            },
          }]
        : [],
      ...(dsIsError ? {
        error: {
          type: "java.sql.SQLException",
          message: "Database query failed",
          stacktrace: "java.sql.SQLException: Database query failed\n\tat com.geass.repository.BaseRepository.execute(BaseRepository.java:87)\n\tat com.geass.service.DataService.query(DataService.java:45)",
        },
      } : {}),
    });

    // DB queries (if service has dbTables)
    let dbOffset = 2;
    for (const table of ds.service.dbTables.slice(0, 2 + Math.floor(rand() * 2))) {
      const dbSpanId = hexId(rand, 16);
      const dbDurationMs = 1 + rand() * 4; // 1-5ms

      spans.push({
        timestamp: new Date(baseTime.getTime() + timeOffset + dbOffset).toISOString(),
        traceId,
        spanId: dbSpanId,
        parentSpanId: serverId,
        spanName: `SELECT geass_v2.${table}`,
        spanKind: "Client",
        serviceName: ds.service.name,
        duration: Math.round(dbDurationMs * 1e6),
        durationMs: Math.round(dbDurationMs * 100) / 100,
        statusCode: "Unset",
        statusMessage: "",
        db: {
          system: "mysql",
          name: "geass_v2",
          operation: "SELECT",
          table,
          statement: `SELECT * FROM ${table} WHERE id = ? ORDER BY created_at DESC LIMIT ?`,
        },
        resource: makeResource(ds.service),
        events: [],
      });
      dbOffset += dbDurationMs + 0.5;
    }

    timeOffset += dsDurationMs + 2;
  }

  return { spans, traceId };
}

// ============================================================
// Build all mock data
// ============================================================

function buildMockData() {
  const rand = seededRandom("apm-mock-v3-stable");
  const now = new Date();
  const traces: TraceDetail[] = [];
  const summaries: TraceSummary[] = [];

  // Generate ~80 traces based on pattern weights
  const totalWeight = TRACE_PATTERNS.reduce((s, p) => s + p.weight, 0);
  const TRACE_COUNT = 80;

  for (let i = 0; i < TRACE_COUNT; i++) {
    // Select pattern by weight
    let r = rand() * totalWeight;
    let pattern = TRACE_PATTERNS[0];
    for (const p of TRACE_PATTERNS) {
      r -= p.weight;
      if (r <= 0) { pattern = p; break; }
    }

    // Spread traces over last 15 days
    const offsetMs = Math.floor(rand() * 15 * 24 * 3600 * 1000);
    const baseTime = new Date(now.getTime() - offsetMs);
    const isError = rand() < 0.04; // ~4% error rate

    const { spans, traceId } = generateTrace(rand, pattern, baseTime, isError);
    const services = new Set(spans.map((s) => s.serviceName));
    const rootSpan = spans[0];
    const totalDurationMs = rootSpan.durationMs;

    traces.push({
      traceId,
      durationMs: totalDurationMs,
      serviceCount: services.size,
      spanCount: spans.length,
      spans,
    });

    summaries.push({
      traceId,
      rootService: rootSpan.serviceName,
      rootOperation: rootSpan.spanName,
      services: [...services],
      durationMs: totalDurationMs,
      spanCount: spans.length,
      serviceCount: services.size,
      hasError: isError,
      ...(isError ? {
        errorType: "java.lang.RuntimeException",
        errorMessage: "Downstream service timeout",
      } : {}),
      timestamp: rootSpan.timestamp,
    });
  }

  // Sort summaries by timestamp desc
  summaries.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());

  return { traces, summaries };
}

const { traces: ALL_TRACES, summaries: ALL_SUMMARIES } = buildMockData();

// ============================================================
// Exported data
// ============================================================

export const mockTraces: TraceDetail[] = ALL_TRACES;
export const mockTraceSummaries: TraceSummary[] = ALL_SUMMARIES;
export const mockTraceDetailsMap: Record<string, TraceDetail> = Object.fromEntries(
  ALL_TRACES.map((t) => [t.traceId, t])
);

// ============================================================
// APM Services — aggregated from traces
// ============================================================

function buildServiceStats(): APMService[] {
  const svcMap = new Map<string, { durations: number[]; errors: number }>();

  for (const svcDef of SERVICES) {
    svcMap.set(svcDef.name, { durations: [], errors: 0 });
  }

  for (const trace of ALL_TRACES) {
    for (const span of trace.spans) {
      if (span.spanKind !== "Server") continue;
      const entry = svcMap.get(span.serviceName);
      if (!entry) continue;
      entry.durations.push(span.durationMs);
      if (span.statusCode === "Error") entry.errors++;
    }
  }

  // Time window for RPS: 15 days in seconds
  const timeWindowS = 15 * 24 * 3600;

  return SERVICES.map((svcDef) => {
    const data = svcMap.get(svcDef.name)!;
    const count = data.durations.length;
    const sorted = [...data.durations].sort((a, b) => a - b);
    const avg = count > 0 ? sorted.reduce((a, b) => a + b, 0) / count : 0;
    const p50 = count > 0 ? sorted[Math.floor(count * 0.5)] : 0;
    const p99 = count > 0 ? sorted[Math.floor(count * 0.99)] : 0;

    return {
      name: svcDef.name,
      namespace: svcDef.namespace,
      spanCount: count,
      errorCount: data.errors,
      successRate: count > 0 ? 1 - data.errors / count : 1,
      avgDurationMs: Math.round(avg * 100) / 100,
      p50Ms: Math.round(p50 * 100) / 100,
      p99Ms: Math.round(p99 * 100) / 100,
      rps: count > 0 ? Math.round((count / timeWindowS) * 1000) / 1000 : 0,
    };
  });
}

export const mockAPMServices: APMService[] = buildServiceStats();

// ============================================================
// Topology — built from cross-service calls
// ============================================================

function buildTopology(): Topology {
  const nodeMap = new Map<string, {
    rpsCount: number;
    errors: number;
    p99Durations: number[];
    type: string;
    namespace: string;
  }>();
  const edgeMap = new Map<string, {
    source: string;
    target: string;
    callCount: number;
    totalMs: number;
    errorCount: number;
  }>();

  for (const trace of ALL_TRACES) {
    const spanMap = new Map(trace.spans.map((s) => [s.spanId, s]));

    for (const span of trace.spans) {
      // Ensure node exists
      if (!nodeMap.has(span.serviceName)) {
        nodeMap.set(span.serviceName, {
          rpsCount: 0, errors: 0, p99Durations: [], type: "service", namespace: "default",
        });
      }
      const node = nodeMap.get(span.serviceName)!;
      if (span.spanKind === "Server") {
        node.rpsCount++;
        node.p99Durations.push(span.durationMs);
        if (span.statusCode === "Error") node.errors++;
      }

      // DB spans create database nodes
      if (span.db) {
        const dbNodeId = `mysql:${span.db.name || "geass_v2"}`;
        if (!nodeMap.has(dbNodeId)) {
          nodeMap.set(dbNodeId, {
            rpsCount: 0, errors: 0, p99Durations: [], type: "database", namespace: "default",
          });
        }
        const dbNode = nodeMap.get(dbNodeId)!;
        dbNode.rpsCount++;
        dbNode.p99Durations.push(span.durationMs);

        const dbEdgeKey = `${span.serviceName}>${dbNodeId}`;
        if (!edgeMap.has(dbEdgeKey)) {
          edgeMap.set(dbEdgeKey, {
            source: span.serviceName, target: dbNodeId, callCount: 0, totalMs: 0, errorCount: 0,
          });
        }
        const dbEdge = edgeMap.get(dbEdgeKey)!;
        dbEdge.callCount++;
        dbEdge.totalMs += span.durationMs;
      }

      // Cross-service edges
      if (!span.parentSpanId) continue;
      const parent = spanMap.get(span.parentSpanId);
      if (!parent || parent.serviceName === span.serviceName) continue;

      const edgeKey = `${parent.serviceName}>${span.serviceName}`;
      if (!edgeMap.has(edgeKey)) {
        edgeMap.set(edgeKey, {
          source: parent.serviceName, target: span.serviceName,
          callCount: 0, totalMs: 0, errorCount: 0,
        });
      }
      const edge = edgeMap.get(edgeKey)!;
      edge.callCount++;
      edge.totalMs += span.durationMs;
      if (span.statusCode === "Error") edge.errorCount++;
    }
  }

  const timeWindowS = 15 * 24 * 3600;

  const nodes = Array.from(nodeMap.entries()).map(([name, data]) => {
    const sorted = [...data.p99Durations].sort((a, b) => a - b);
    const p99 = sorted.length > 0 ? sorted[Math.floor(sorted.length * 0.99)] : 0;
    const successRate = data.rpsCount > 0 ? 1 - data.errors / data.rpsCount : 1;

    let status: "healthy" | "warning" | "critical" | "unknown" = "healthy";
    if (successRate < 0.95) status = "critical";
    else if (successRate < 0.99) status = "warning";

    return {
      id: name,
      name,
      namespace: data.namespace,
      type: data.type,
      rps: Math.round((data.rpsCount / timeWindowS) * 1000) / 1000,
      successRate: Math.round(successRate * 10000) / 10000,
      p99Ms: Math.round(p99 * 100) / 100,
      status,
    };
  });

  const edges = Array.from(edgeMap.values()).map((e) => ({
    source: e.source,
    target: e.target,
    callCount: e.callCount,
    avgMs: e.callCount > 0 ? Math.round((e.totalMs / e.callCount) * 100) / 100 : 0,
    errorRate: e.callCount > 0 ? Math.round((e.errorCount / e.callCount) * 10000) / 10000 : 0,
  }));

  return { nodes, edges };
}

export const mockTopology: Topology = buildTopology();

/**
 * APM 数据源 — 直接调用 Master observe API 查询 ClickHouse
 */

import {
  mockGetLatencyDistribution,
} from "@/mock/apm";
import type { MockTraceQueryParams } from "@/mock/apm";
import type { APMService, TraceSummary, TraceDetail, Topology, OperationStats, Dependency, APMTimePoint, HTTPStats, DBOperationStats, CorrelationResult, CorrelationMode } from "@/types/model/apm";
import * as observeApi from "@/api/observe-apm";

export type { MockTraceQueryParams } from "@/mock/apm";

/** 时间参数：since（相对时间）+ startTime/endTime（绝对时间） */
export interface TimeParams {
  since?: string;
  startTime?: string;
  endTime?: string;
}

export async function getAPMServices(clusterId?: string, time?: TimeParams): Promise<APMService[]> {
  if (!clusterId) return [];
  try {
    const response = await observeApi.getTracesServices(clusterId, time?.since, time?.startTime, time?.endTime);
    return response.data.data || [];
  } catch {
    return [];
  }
}

export async function queryTraces(clusterId?: string, params?: MockTraceQueryParams, time?: TimeParams): Promise<{ traces: TraceSummary[]; total: number }> {
  if (!clusterId) return { traces: [], total: 0 };
  try {
    const response = await observeApi.getTracesList(clusterId, {
      service: params?.service,
      operation: params?.operation,
      min_duration: params?.minDurationMs ? String(params.minDurationMs) : undefined,
      limit: params?.limit,
      status_code: params?.status_code,
      method: params?.method,
      time_range: time?.since,
      start_time: time?.startTime,
      end_time: time?.endTime,
    });
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = response.data.data as any;
    // 快照直读返回 { traces, total }，Command 返回数组
    const traces: TraceSummary[] = Array.isArray(raw) ? raw : (raw?.traces || []);
    const total: number = Array.isArray(raw) ? traces.length : (raw?.total ?? traces.length);
    return { traces, total };
  } catch {
    return { traces: [], total: 0 };
  }
}

export async function getTraceDetail(traceId: string, clusterId?: string): Promise<TraceDetail | null> {
  if (!clusterId) return null;
  const response = await observeApi.getTraceDetail(clusterId, traceId);
  return response.data.data || null;
}

export async function getTopology(clusterId?: string, time?: TimeParams): Promise<Topology> {
  if (!clusterId) return { nodes: [], edges: [] };
  try {
    const response = await observeApi.getTracesTopology(clusterId, time?.since, time?.startTime, time?.endTime);
    const data = response.data.data;
    return { nodes: data?.nodes || [], edges: data?.edges || [] };
  } catch {
    return { nodes: [], edges: [] };
  }
}

export async function getOperations(clusterId?: string, time?: TimeParams): Promise<OperationStats[]> {
  if (!clusterId) return [];
  try {
    const response = await observeApi.getTracesOperations(clusterId, time?.since, time?.startTime, time?.endTime);
    return response.data.data || [];
  } catch {
    return [];
  }
}

/** timeRange 字符串 → 分钟数 */
function timeRangeToMinutes(timeRange?: string): number {
  if (!timeRange) return 60;
  const match = timeRange.match(/^(\d+)(m|h|d)$/);
  if (!match) return 60;
  const v = parseInt(match[1], 10);
  switch (match[2]) {
    case "m": return v;
    case "h": return v * 60;
    case "d": return v * 24 * 60;
    default: return 60;
  }
}

export async function getServiceTimeSeries(clusterId?: string, serviceName?: string, timeRange?: string): Promise<APMTimePoint[]> {
  if (!clusterId || !serviceName) return [];
  try {
    const minutes = timeRangeToMinutes(timeRange);
    const response = await observeApi.getAPMServiceSeries(clusterId, serviceName, minutes);
    return response.data.data?.points || [];
  } catch {
    return [];
  }
}

export async function getHTTPStats(clusterId?: string, serviceName?: string, time?: TimeParams): Promise<HTTPStats[]> {
  if (!clusterId || !serviceName) return [];
  try {
    const response = await observeApi.getTracesHTTPStats(clusterId, {
      service: serviceName,
      ...(time?.since && time.since !== "15m" ? { time_range: time.since } : {}),
      ...(time?.startTime ? { start_time: time.startTime } : {}),
      ...(time?.endTime ? { end_time: time.endTime } : {}),
    });
    return response.data.data || [];
  } catch {
    return [];
  }
}

export async function getDBStats(clusterId?: string, serviceName?: string, time?: TimeParams): Promise<DBOperationStats[]> {
  if (!clusterId || !serviceName) return [];
  try {
    const response = await observeApi.getTracesDBStats(clusterId, {
      service: serviceName,
      ...(time?.since && time.since !== "15m" ? { time_range: time.since } : {}),
      ...(time?.startTime ? { start_time: time.startTime } : {}),
      ...(time?.endTime ? { end_time: time.endTime } : {}),
    });
    return response.data.data || [];
  } catch {
    return [];
  }
}

export async function getCorrelations(
  clusterId: string | undefined,
  serviceName: string,
  mode: CorrelationMode,
  time?: TimeParams,
  operation?: string,
): Promise<CorrelationResult | null> {
  if (!clusterId) return null;
  try {
    const response = await observeApi.getTracesCorrelations(clusterId, {
      service: serviceName,
      operation,
      mode,
      ...(time?.since ? { time_range: time.since } : {}),
      ...(time?.startTime ? { start_time: time.startTime } : {}),
      ...(time?.endTime ? { end_time: time.endTime } : {}),
    });
    return response.data.data || null;
  } catch {
    return null;
  }
}

// ============================================================
// 客户端计算函数 — 从已加载的真实数据派生
// ============================================================

/** 延迟分布直方图 — 纯计算，入参为已加载的 traces */
export function getLatencyDistribution(traces: TraceSummary[]) {
  return mockGetLatencyDistribution(traces);
}

/** 从 topology 数据派生某服务的依赖列表 */
export function getDependenciesFromTopology(serviceName: string, topology: Topology | null): Dependency[] {
  if (!topology) return [];

  const nodeMap = new Map(topology.nodes.map((n) => [n.id, n]));

  // 找出该服务作为 source 的所有 edges
  const deps: Dependency[] = [];
  for (const edge of topology.edges) {
    // source 格式: "namespace/serviceName"，匹配 serviceName
    const sourceService = edge.source.split("/").pop() ?? "";
    if (sourceService !== serviceName) continue;

    const targetNode = nodeMap.get(edge.target);
    const name = targetNode?.name ?? edge.target;
    const type = targetNode?.type ?? "service";

    deps.push({
      name,
      type,
      callCount: edge.callCount,
      avgMs: edge.avgMs,
      errorRate: edge.errorRate,
      impact: 0, // 后面统一计算
    });
  }

  // 计算 impact
  const maxTotal = Math.max(...deps.map((d) => d.callCount * d.avgMs), 1);
  for (const dep of deps) {
    dep.impact = (dep.callCount * dep.avgMs) / maxTotal;
  }

  return deps;
}

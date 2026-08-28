/**
 * APM (Traces) 信号域 API
 */

import { get } from "./request";
import type { ObserveResponse } from "./observe-common";
import type { TraceSummary, TraceDetail, APMService, Topology, OperationStats, APMServiceSeriesResponse, HTTPStats, DBOperationStats, CorrelationResult } from "@/types/model/apm";

/** 查询 Trace 列表 */
export function getTracesList(clusterId: string, params?: {
  service?: string;
  operation?: string;
  min_duration?: string;
  max_duration?: string;
  limit?: number;
  offset?: number;
  start_time?: string;
  end_time?: string;
  time_range?: string;
  status_code?: string;
  method?: string;
}) {
  return get<ObserveResponse<TraceSummary[]>>("/api/v2/observe/traces", {
    cluster_id: clusterId,
    ...params,
  });
}

/** 获取服务列表 */
export function getTracesServices(clusterId: string, timeRange?: string, startTime?: string, endTime?: string) {
  return get<ObserveResponse<APMService[]>>("/api/v2/observe/traces/services", {
    cluster_id: clusterId,
    ...(timeRange ? { time_range: timeRange } : {}),
    ...(startTime ? { start_time: startTime } : {}),
    ...(endTime ? { end_time: endTime } : {}),
  });
}

/** 获取服务拓扑 */
export function getTracesTopology(clusterId: string, timeRange?: string, startTime?: string, endTime?: string) {
  return get<ObserveResponse<Topology>>("/api/v2/observe/traces/topology", {
    cluster_id: clusterId,
    ...(timeRange ? { time_range: timeRange } : {}),
    ...(startTime ? { start_time: startTime } : {}),
    ...(endTime ? { end_time: endTime } : {}),
  });
}

/** 获取操作级聚合统计 */
export function getTracesOperations(clusterId: string, timeRange?: string, startTime?: string, endTime?: string) {
  return get<ObserveResponse<OperationStats[]>>("/api/v2/observe/traces/operations", {
    cluster_id: clusterId,
    ...(timeRange ? { time_range: timeRange } : {}),
    ...(startTime ? { start_time: startTime } : {}),
    ...(endTime ? { end_time: endTime } : {}),
  });
}

/** 获取服务时序趋势 (Concentrator 预聚合) */
export function getAPMServiceSeries(clusterId: string, serviceName: string, minutes?: number) {
  return get<ObserveResponse<APMServiceSeriesResponse>>(
    `/api/v2/observe/traces/services/${encodeURIComponent(serviceName)}/series`,
    { cluster_id: clusterId, ...(minutes ? { minutes: String(minutes) } : {}) },
  );
}

/** 相关性分析：慢/错请求中显著超标的属性（对齐 ES APM Correlations） */
export function getTracesCorrelations(clusterId: string, params: {
  service: string;
  operation?: string;
  mode: "latency" | "failure";
  time_range?: string;
  start_time?: string;
  end_time?: string;
}) {
  return get<ObserveResponse<CorrelationResult>>("/api/v2/observe/traces/correlations", {
    cluster_id: clusterId,
    ...params,
  });
}

/** 获取 HTTP 状态码分布 */
export function getTracesHTTPStats(clusterId: string, params: {
  service: string;
  time_range?: string;
  start_time?: string;
  end_time?: string;
}) {
  return get<ObserveResponse<HTTPStats[]>>("/api/v2/observe/traces/stats", {
    cluster_id: clusterId,
    sub_action: "http_stats",
    ...params,
  });
}

/** 获取数据库操作统计 */
export function getTracesDBStats(clusterId: string, params: {
  service: string;
  time_range?: string;
  start_time?: string;
  end_time?: string;
}) {
  return get<ObserveResponse<DBOperationStats[]>>("/api/v2/observe/traces/stats", {
    cluster_id: clusterId,
    sub_action: "db_stats",
    ...params,
  });
}

/** 获取 Trace 详情 */
export function getTraceDetail(clusterId: string, traceId: string) {
  return get<ObserveResponse<TraceDetail>>(
    `/api/v2/observe/traces/${encodeURIComponent(traceId)}`,
    { cluster_id: clusterId }
  );
}

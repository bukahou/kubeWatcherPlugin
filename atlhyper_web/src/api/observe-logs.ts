/**
 * Logs 信号域 API
 */

import { get, post } from "./request";
import type { ObserveResponse } from "./observe-common";
import type { LogEntry, LogFacets, LogHistogramResult } from "@/types/model/log";

/** 日志查询结果（不含 histogram） */
export interface LogQueryResponse {
  logs: LogEntry[];
  total: number;
  facets: LogFacets;
}

/** 查询日志 (POST) */
export function queryLogs(params: {
  cluster_id: string;
  query?: string;
  service?: string;
  level?: string;
  scope?: string;
  trace_id?: string;
  span_id?: string;
  limit?: number;
  offset?: number;
  since?: string;
  start_time?: string;
  end_time?: string;
}) {
  return post<ObserveResponse<LogQueryResponse>>("/api/v2/observe/logs/query", params);
}

/** 查询日志直方图 (GET, ClickHouse 聚合) */
export function getLogsHistogram(params: {
  cluster_id: string;
  since?: string;
  service?: string;
  level?: string;
  scope?: string;
  query?: string;
  start_time?: string;
  end_time?: string;
}) {
  return get<ObserveResponse<LogHistogramResult>>("/api/v2/observe/logs/histogram", params);
}

// ── G5 接线（2026-08-29 能力盘点）────────────────────────────
// 快照级全局摘要，与 query/facets 的「当前过滤窗口」口径不同：
// latestAt 是日志流新鲜度信号（采集断流时最先体现在这里），
// severityCounts 是未过滤的全局分布（过滤到别处时 ERROR 也不会漏看）。

export interface LogsSummary {
  totalEntries: number;
  severityCounts: Record<string, number>;
  topServices: { service: string; count: number }[];
  latestAt: string;
}

export function getLogsSummary(clusterId: string) {
  return get<ObserveResponse<LogsSummary>>("/api/v2/observe/logs/summary", {
    cluster_id: clusterId,
  });
}

/**
 * Metrics 数据源代理
 *
 * 根据中心配置自动切换 mock / api
 * - 实时数据（summary + nodes）使用 observe/metrics 端点（ClickHouse）
 * - 历史数据使用 node-metrics 端点（SQLite，快速直读）
 */

import { getDataSourceMode } from "@/config/data-source";
import * as mock from "@/mock/metrics";
import * as observe from "@/api/observe-metrics";
import * as nodeMetrics from "@/api/node-metrics";

export type { Summary } from "@/types/node-metrics";
export type { MockClusterNodeMetricsResult, MockNodeMetricsHistoryResult } from "@/mock/metrics";

export async function getClusterNodeMetrics(clusterId: string) {
  if (getDataSourceMode("metrics") === "mock") return mock.mockGetClusterNodeMetrics();

  // 并行请求 summary 和 nodes（ClickHouse 实时数据）
  const [summaryRes, nodesRes] = await Promise.all([
    observe.getMetricsSummary(clusterId),
    observe.getMetricsNodes(clusterId),
  ]);

  return {
    summary: summaryRes.data.data,
    nodes: nodesRes.data.data,
  };
}

export async function getHardwareHealth(clusterId: string) {
  if (getDataSourceMode("metrics") === "mock") return mock.mockGetHardwareHealth();
  const res = await observe.getMetricsHardware(clusterId);
  return res.data.data;
}

export async function getNodeMetricsHistory(clusterId: string, nodeName: string, hours?: number) {
  if (getDataSourceMode("metrics") === "mock") return mock.mockGetNodeMetricsHistory(nodeName, hours);
  // 历史数据使用 SQLite 直读（快速，无需 Command 机制）
  return nodeMetrics.getNodeMetricsHistory(clusterId, nodeName, hours);
}

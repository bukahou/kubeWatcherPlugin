/**
 * Metrics Mock — 统一导出
 */

export { mockGetClusterNodeMetrics, mockGetNodeMetricsHistory } from "./queries";
export type { MockClusterNodeMetricsResult, MockNodeMetricsHistoryResult } from "./queries";
export { mockGetHardwareHealth, MOCK_HARDWARE } from "./hardware";

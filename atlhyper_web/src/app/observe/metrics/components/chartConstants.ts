import type { NodeMetrics } from "@/types/node-metrics";

// ==================== Types ====================

/** Global time window — total data range */
export type TimeWindow = "1h" | "6h" | "1d" | "7d";

/** Per-chart aggregation interval (seconds) */
export type Interval = number;

/** Sort function: extract the "current" value from a snapshot for ranking */
export type SnapshotSortFn = (n: NodeMetrics) => number;

// ==================== Constants ====================

export const PALETTE = ["#F97316", "#10B981", "#3B82F6", "#8B5CF6", "#EF4444"];

export const TIME_WINDOWS: { key: TimeWindow; label: string; hours: number }[] = [
  { key: "1h", label: "1h", hours: 1 },
  { key: "6h", label: "6h", hours: 6 },
  { key: "1d", label: "1d", hours: 24 },
  { key: "7d", label: "7d", hours: 168 },
];

/** Available intervals per window. label = display text, seconds = bucket size */
export const INTERVALS_BY_WINDOW: Record<TimeWindow, { label: string; seconds: number }[]> = {
  "1h":  [{ label: "1m", seconds: 60 }, { label: "5m", seconds: 300 }, { label: "15m", seconds: 900 }],
  "6h":  [{ label: "5m", seconds: 300 }, { label: "15m", seconds: 900 }, { label: "30m", seconds: 1800 }],
  "1d":  [{ label: "15m", seconds: 900 }, { label: "1h", seconds: 3600 }, { label: "3h", seconds: 10800 }],
  "7d":  [{ label: "1h", seconds: 3600 }, { label: "6h", seconds: 21600 }, { label: "1d", seconds: 86400 }],
};

/** Metric key matching the keys returned by mock/api history */
export const METRICS: {
  key: string;
  unit: string;
  sortSnapshot: SnapshotSortFn;
}[] = [
  { key: "cpu", unit: "%", sortSnapshot: (n) => n.cpu.usagePct },
  { key: "memory", unit: "%", sortSnapshot: (n) => n.memory.usagePct },
  { key: "disk", unit: "%", sortSnapshot: (n) => (n.disks?.find((d) => d.mountPoint === "/") ?? n.disks?.[0])?.usagePct ?? 0 },
  { key: "temp", unit: "°C", sortSnapshot: (n) => n.temperature.cpuTempC },
];

/** Minimum data points for an interval to be auto-selected */
export const MIN_AUTO_POINTS = 3;

/** 页面级时间范围（小时）→ 内部窗口档位。P3 统一后由 TimeRangePicker 推导，无手动选择 */
export function windowForHours(hours: number): TimeWindow {
  if (hours <= 1) return "1h";
  if (hours <= 6) return "6h";
  if (hours <= 24) return "1d";
  return "7d";
}

// types/hardware.ts
// 硬件健康矩阵 — 1:1 对齐 atlhyper_master_v2/model/metrics_hardware.go
//
// 与 types/node-metrics.ts 的区别：那边是节点上报的原始读数，
// 这边是 Master 判定完的结果，每格自带 status。前端不做任何阈值比较。
// 格为 null = 该节点没有这个传感器 → 显示「无数据」，既不是正常也不是故障。

export type HardwareStatus = "good" | "warn" | "crit";

export interface HardwareTempCell {
  value: number;
  max: number;
  crit: number;
  label?: string;
  status: HardwareStatus;
}

export interface HardwareUndervoltCell {
  alarm: boolean;
  status: HardwareStatus;
}

export interface HardwareFanCell {
  rpm: number | null;   // null = 只有调速档位，没有转速传感器
  state: number;
  maxState: number;
  status: HardwareStatus;
}

export interface HardwareFreqCell {
  currentGHz: number;
  maxGHz: number;
  ratioPct: number;
  status: HardwareStatus;
}

export interface HardwareAwaitCell {
  valueMs: number;
  device: string;
  status: HardwareStatus;
}

/** 单个温度传感器的判定结果（节点详情温度卡用；矩阵那三格是每类取最热的一个）*/
export interface HardwareSensorCell {
  label: string;
  sensor: string;
  class: "cpu" | "disk" | "other";
  value: number;
  max: number;
  crit: number;
  status: HardwareStatus;
}

/** 资源使用率格。0% 是真实读数，所以这一格不会是 null */
export interface HardwareUsageCell {
  value: number;
  status: HardwareStatus;
}

export interface HardwareRow {
  nodeName: string;
  profile: string;
  profileLabel: string;
  cpuUsage: HardwareUsageCell | null;
  memUsage: HardwareUsageCell | null;
  diskUsage: HardwareUsageCell | null;
  cpuTemp: HardwareTempCell | null;
  diskTemp: HardwareTempCell | null;
  otherTemp: HardwareTempCell | null;
  undervolt: HardwareUndervoltCell | null;
  fan: HardwareFanCell | null;
  cpuFreq: HardwareFreqCell | null;
  diskAwait: HardwareAwaitCell | null;
  sensors: HardwareSensorCell[];
  overall: HardwareStatus;
}

export interface HardwareMaxTemp {
  value: number;
  nodeName: string;
  sensor: string;
  status: HardwareStatus;
}

export interface HardwareMaxAwait {
  valueMs: number;
  nodeName: string;
  device: string;
  status: HardwareStatus;
}

export interface HardwareSummary {
  maxTemp: HardwareMaxTemp | null;
  maxDiskTemp: HardwareMaxTemp | null;
  undervoltNodes: number;
  throttledNodes: number;
  maxDiskAwait: HardwareMaxAwait | null;
}

export interface HardwareHealth {
  rows: HardwareRow[];
  summary: HardwareSummary;
}

// ──────────────────────────────────────────────────────────────
// 节点对比表 — 对齐 model/metrics_hardware.go 的 NodeComparisonResponse
// ──────────────────────────────────────────────────────────────

/** 一格。value 为 null = 该节点取不到这个信号；text 只含数字与单位，文案由前端 i18n 出 */
export interface CompareCell {
  value: number | null;
  text?: string;
  status: HardwareStatus;
}

export interface CompareRow {
  nodeName: string;
  profile: string;
  cells: Record<string, CompareCell>;
  overall: HardwareStatus;
}

export interface NodeComparison {
  /** 列顺序由后端固定（硬件优先），前端不重排 */
  columns: string[];
  rows: CompareRow[];
}

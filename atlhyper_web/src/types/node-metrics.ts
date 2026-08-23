// types/node-metrics.ts
// 节点硬件指标类型定义 — 1:1 对齐 model_v3/metrics/node_metrics.go JSON tag

// ============================================================================
// CPU 指标
// ============================================================================
export interface NodeCPU {
  usagePct: number;
  userPct: number;
  systemPct: number;
  iowaitPct: number;
  load1: number;
  load5: number;
  load15: number;
  cores: number;
  freqHz?: number[];
  freqMaxHz?: number;   // 标称最高频率（热降频判定用）
}

// ============================================================================
// 内存指标
// ============================================================================
export interface NodeMemory {
  totalBytes: number;
  availableBytes: number;
  freeBytes: number;
  cachedBytes: number;
  buffersBytes: number;
  usagePct: number;
  swapTotalBytes: number;
  swapFreeBytes: number;
  swapUsagePct: number;
}

// ============================================================================
// 磁盘指标
// ============================================================================
export interface NodeDisk {
  device: string;
  mountPoint: string;
  fsType: string;
  totalBytes: number;
  availBytes: number;
  usagePct: number;
  readBytesPerSec: number;
  writeBytesPerSec: number;
  readIOPS: number;
  writeIOPS: number;
  ioUtilPct: number;
  awaitReadMs: number;  // 平均读延迟
  awaitWriteMs: number; // 平均写延迟
  queueDepth: number;   // 平均在途请求数
}

// ============================================================================
// 网络指标
// ============================================================================
export interface NodeNetwork {
  interface: string;
  up: boolean;
  speedBps: number;
  mtu: number;
  rxBytesPerSec: number;
  txBytesPerSec: number;
  rxPktPerSec: number;
  txPktPerSec: number;
  rxErrPerSec: number;
  txErrPerSec: number;
  rxDropPerSec: number;
  txDropPerSec: number;
}

// ============================================================================
// 温度指标
// ============================================================================
export interface NodeTemperature {
  cpuTempC: number;
  cpuMaxC: number;
  cpuCritC: number;
  sensors: TempSensor[];
}

export interface TempSensor {
  chip: string;
  chipName?: string;    // hwmon 可读名：coretemp / nvme / rp1_adc ...
  sensor: string;
  currentC: number;
  maxC: number;
  critC: number;
}

// ============================================================================
// PSI 压力信息
// ============================================================================
export interface NodePSI {
  cpuSomePct: number;
  memSomePct: number;
  memFullPct: number;
  ioSomePct: number;
  ioFullPct: number;
}

// ============================================================================
// TCP 连接状态
// ============================================================================
export interface NodeTCP {
  currEstab: number;
  alloc: number;
  inUse: number;
  timeWait: number;
  socketsUsed: number;
}

// ============================================================================
// 系统资源指标
// ============================================================================
export interface NodeSystem {
  conntrackEntries: number;
  conntrackLimit: number;
  filefdAllocated: number;
  filefdMax: number;
  entropyBits: number;
}

// ============================================================================
// 虚拟内存统计
// ============================================================================
export interface NodeVMStat {
  pgFaultPerSec: number;
  pgMajFaultPerSec: number;
  pswpInPerSec: number;
  pswpOutPerSec: number;
}

// ============================================================================
// 软中断统计
// ============================================================================
export interface NodeSoftnet {
  droppedPerSec: number;
  squeezedPerSec: number;
}

// ============================================================================
// 节点指标快照 (聚合) — 对齐 Go NodeMetrics
// ============================================================================
export interface NodeMetrics {
  nodeName: string;
  nodeIP: string;
  timestamp: string;

  cpu: NodeCPU;
  memory: NodeMemory;
  disks: NodeDisk[];
  networks: NodeNetwork[];
  temperature: NodeTemperature;

  psi: NodePSI;
  tcp: NodeTCP;
  system: NodeSystem;
  vmstat: NodeVMStat;
  softnet: NodeSoftnet;

  kernel?: string;
  uptime?: number;

  hardwareProfile?: string;   // desk / raspi5 / raspi4 / unknown
  hardware?: NodeHardware;    // 缺失 = 上报方不采集，按「无数据」处理
}

// ============================================================================
// 硬件传感器（风扇 / 散热 / 电压）
// ============================================================================
export interface NodeHardware {
  undervoltAlarm?: boolean;   // 缺失 = 无此传感器
  fans: FanSensor[];
  cooling: CoolingDevice[];
}

export interface FanSensor {
  chip: string;
  sensor: string;
  rpm: number;
}

export interface CoolingDevice {
  name: string;
  type: string;
  curState: number;
  maxState: number;
}

// ============================================================================
// 时序数据（趋势图用） — 对齐 Go Point / Series
// ============================================================================
export interface Point {
  timestamp: string;   // ISO 8601
  value: number;
}

export interface Series {
  metric: string;
  labels?: Record<string, string>;
  points: Point[];
}

// ============================================================================
// 集群节点指标概览 — 对齐 Go Summary
// ============================================================================
export interface Summary {
  totalNodes: number;
  onlineNodes: number;
  avgCpuPct: number;
  avgMemPct: number;
  maxCpuPct: number;
  maxMemPct: number;
  maxCpuTemp: number;
}

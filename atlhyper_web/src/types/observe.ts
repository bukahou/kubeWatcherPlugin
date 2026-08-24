// types/observe.ts
// 观测模块跨信号类型 —— 对齐 atlhyper_master_v2/model/observe.go

/**
 * 信号新鲜度状态。判定在后端完成，前端只渲染文案和颜色。
 *
 * idle 与 stale 的区别是这个特性存在的全部理由：两者在页面上都表现为空白，
 * 但 idle 不用管（没人访问），stale 要救火（采集链路断了）。
 */
export type FreshnessStatus = "live" | "idle" | "stale" | "absent";

export interface SignalFreshnessItem {
  signal: "metrics" | "traces" | "logs";
  lastDataAt?: string;
  lagSeconds: number;
  status: FreshnessStatus;
}

export interface FreshnessResponse {
  signals: SignalFreshnessItem[];
  collectorHealthy: boolean;
}

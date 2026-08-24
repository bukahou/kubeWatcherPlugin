/**
 * 时间范围选择器工具函数
 */

import type { PresetKey, RelativeUnit, TimeRangeSelection } from "@/types/time-range";

/** 全部预设 key，顺序即选择器展示顺序 */
export const PRESET_KEYS = ["15min", "1h", "24h", "7d", "15d", "30d"] as const;

/** 预设 key → Go duration string (用于 API since 参数) */
const PRESET_SINCE: Record<PresetKey, string> = {
  "15min": "15m",
  "1h": "1h",
  "24h": "24h",
  "7d": "168h",
  "15d": "360h",
  "30d": "720h",
};

/** 预设 key → 毫秒数 */
const PRESET_MS: Record<PresetKey, number> = {
  "15min": 15 * 60_000,
  "1h": 3_600_000,
  "24h": 86_400_000,
  "7d": 7 * 86_400_000,
  "15d": 15 * 86_400_000,
  "30d": 30 * 86_400_000,
};

/** 单位 → 毫秒 */
const UNIT_MS: Record<RelativeUnit, number> = {
  m: 60_000,
  h: 3_600_000,
  d: 86_400_000,
};

/**
 * 转换为 Go duration 字符串 (since 参数)
 * absolute 模式返回 undefined（使用 start_time/end_time 代替）
 */
export function toSince(sel: TimeRangeSelection): string | undefined {
  switch (sel.mode) {
    case "preset":
      return PRESET_SINCE[sel.preset];
    case "custom": {
      // 统一转换为分钟（Go duration 格式）
      const totalMs = sel.value * UNIT_MS[sel.unit];
      const totalMinutes = Math.round(totalMs / 60_000);
      if (totalMinutes >= 60 && totalMinutes % 60 === 0) {
        return `${totalMinutes / 60}h`;
      }
      return `${totalMinutes}m`;
    }
    case "absolute":
      return undefined;
  }
}

/**
 * 转换为绝对时间参数 (start_time/end_time ISO 字符串)
 * 非 absolute 模式返回空对象
 */
export function toAbsoluteParams(sel: TimeRangeSelection): { startTime?: string; endTime?: string } {
  if (sel.mode !== "absolute") return {};
  return {
    startTime: new Date(sel.start).toISOString(),
    endTime: new Date(sel.end).toISOString(),
  };
}

/**
 * 转换为时间跨度毫秒数（用于直方图 X 轴标签精度判断）
 */
export function toSpanMs(sel: TimeRangeSelection): number {
  switch (sel.mode) {
    case "preset":
      return PRESET_MS[sel.preset];
    case "custom":
      return sel.value * UNIT_MS[sel.unit];
    case "absolute":
      return sel.end - sel.start;
  }
}

/**
 * 生成显示标签
 */
export function toDisplayLabel(
  sel: TimeRangeSelection,
  presetLabels: Record<PresetKey, string>,
  unitLabels: { minutes: string; hours: string; days: string },
): string {
  switch (sel.mode) {
    case "preset":
      return presetLabels[sel.preset];
    case "custom": {
      const unitLabel =
        sel.unit === "m" ? unitLabels.minutes :
        sel.unit === "h" ? unitLabels.hours :
        unitLabels.days;
      return `${sel.value} ${unitLabel}`;
    }
    case "absolute": {
      const fmt = (ts: number) => {
        const d = new Date(ts);
        const month = d.getMonth() + 1;
        const day = d.getDate();
        const h = String(d.getHours()).padStart(2, "0");
        const m = String(d.getMinutes()).padStart(2, "0");
        return `${month}/${day} ${h}:${m}`;
      };
      return `${fmt(sel.start)} — ${fmt(sel.end)}`;
    }
  }
}

// ──────────────────────────────────────────────────────────────
// SLO 窗口贴合
// ──────────────────────────────────────────────────────────────

/**
 * SLO 可用的预聚合窗口，与 Agent 的 sloWindowConfigs 一一对应
 * （atlhyper_agent_v2/service/snapshot/slo_collector.go）。
 *
 * SLO 不能按任意时间范围查询 —— 这些窗口是 Agent 预先算好缓存的，
 * 传一个不在列表里的 key 会让后端静默 fallback 到 5 分钟数据。
 * 两边必须同步改。
 */
export const SLO_WINDOWS = ["1h", "6h", "24h", "3d", "7d"] as const;
export type SLOWindow = (typeof SLO_WINDOWS)[number];

const SLO_WINDOW_MS: Record<SLOWindow, number> = {
  "1h": 3_600_000,
  "6h": 6 * 3_600_000,
  "24h": 86_400_000,
  "3d": 3 * 86_400_000,
  "7d": 7 * 86_400_000,
};

export interface SLOWindowResult {
  window: SLOWindow;
  /** 是否发生了贴合。降级必须让用户看见，不能默默换掉窗口 */
  degraded: boolean;
}

/**
 * 把任意时间范围贴合到最近的 SLO 预聚合窗口。
 *
 * 向上取而非向下取：向下取会让统计窗口比用户要求的短，可用率与预算都会失真；
 * 向上取只是范围更宽，语义上是安全的。
 * 超过最大窗口时钳到 7d —— ClickHouse 只保留 7 天数据。
 */
export function toSLOWindow(sel: TimeRangeSelection): SLOWindowResult {
  const spanMs = toSpanMs(sel);
  for (const w of SLO_WINDOWS) {
    if (spanMs <= SLO_WINDOW_MS[w]) {
      return { window: w, degraded: spanMs !== SLO_WINDOW_MS[w] };
    }
  }
  const largest = SLO_WINDOWS[SLO_WINDOWS.length - 1];
  return { window: largest, degraded: spanMs !== SLO_WINDOW_MS[largest] };
}

// ──────────────────────────────────────────────────────────────
// URL 参数编解码 —— 让时间轴可分享、刷新不丢
// ──────────────────────────────────────────────────────────────

const CUSTOM_PATTERN = /^(\d+)([mhd])$/;

/** 序列化为 URL query 片段（不含 ?）。preset 与 custom 都用可读形式，人能看懂 */
export function formatRangeParam(sel: TimeRangeSelection): string {
  switch (sel.mode) {
    case "preset":
      return `range=${sel.preset}`;
    case "custom":
      return `range=${sel.value}${sel.unit}`;
    case "absolute":
      return `from=${sel.start}&to=${sel.end}`;
  }
}

/**
 * 从 URL 参数解析时间范围。
 *
 * 返回 null 表示「URL 里没有或不合法」，由调用方决定回退到 localStorage 还是默认值。
 * URL 是用户可以随手编辑的，任何非法输入都返回 null 而不是抛异常。
 */
export function parseRangeParam(params: URLSearchParams): TimeRangeSelection | null {
  const from = params.get("from");
  const to = params.get("to");
  if (from !== null && to !== null) {
    const start = Number(from);
    const end = Number(to);
    if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) return null;
    return { mode: "absolute", start, end };
  }

  const range = params.get("range");
  if (!range) return null;

  if ((PRESET_KEYS as readonly string[]).includes(range)) {
    return { mode: "preset", preset: range as PresetKey };
  }

  const m = CUSTOM_PATTERN.exec(range);
  if (!m) return null;
  const value = Number(m[1]);
  if (value <= 0) return null;
  return { mode: "custom", value, unit: m[2] as RelativeUnit };
}

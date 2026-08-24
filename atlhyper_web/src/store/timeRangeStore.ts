/**
 * 观测模块的全局时间轴
 *
 * 四个信号页（Metrics / APM / Logs / SLO）共享同一个时间范围 ——
 * 排查故障时「同一时刻四个信号都发生了什么」是核心操作，
 * 每切一次页面就要重选时间，等于每次都重新开始调查。
 *
 * 持久化优先级：URL 参数 > localStorage > 默认 1h。
 * URL 优先是为了让分享出去的链接生效；localStorage 是为了刷新不丢。
 */

import { create } from "zustand";
import type { TimeRangeSelection } from "@/types/time-range";
import { parseRangeParam } from "@/lib/time-range";

const STORAGE_KEY = "observeTimeRange";
const DEFAULT_SELECTION: TimeRangeSelection = { mode: "preset", preset: "1h" };

interface TimeRangeStore {
  selection: TimeRangeSelection;
  /** 是否已经从 URL / localStorage 恢复过，避免 hydrate 覆盖用户已做的选择 */
  hydrated: boolean;
  setSelection: (s: TimeRangeSelection) => void;
  hydrate: (params: URLSearchParams) => void;
}

/** 从 localStorage 读取上次的选择；解析失败一律当作没有 */
function readStored(): TimeRangeSelection | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as TimeRangeSelection;
    // 只认识这三种模式，其余（旧版本遗留、手工篡改）丢弃
    if (parsed?.mode === "preset" || parsed?.mode === "custom" || parsed?.mode === "absolute") {
      return parsed;
    }
    return null;
  } catch {
    return null;
  }
}

function persist(s: TimeRangeSelection) {
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(s));
  } catch {
    // 隐私模式下 localStorage 会抛异常，时间轴照常工作，只是刷新后回到默认值
  }
}

export const useTimeRangeStore = create<TimeRangeStore>((set, get) => ({
  selection: DEFAULT_SELECTION,
  hydrated: false,

  setSelection: (s) => {
    persist(s);
    set({ selection: s, hydrated: true });
  },

  hydrate: (params) => {
    if (get().hydrated) return;
    const fromUrl = parseRangeParam(params);
    const next = fromUrl ?? readStored() ?? DEFAULT_SELECTION;
    // URL 里带了范围时也写回 localStorage —— 打开别人分享的链接后，
    // 下次自己进来看到的还是这段时间，符合直觉
    if (fromUrl) persist(fromUrl);
    set({ selection: next, hydrated: true });
  },
}));

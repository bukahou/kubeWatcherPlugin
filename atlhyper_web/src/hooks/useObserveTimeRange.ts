/**
 * 观测页读取全局时间轴 —— 带能力声明与显式降级
 *
 * 统一时间轴不是「一个值广播给所有人」：
 *   - APM / Logs 能按任意范围查 ClickHouse
 *   - SLO 只有五个预聚合窗口（Agent 侧算好缓存的）
 *   - Metrics 的卡片与硬件矩阵永远是当前快照，范围只作用于趋势图
 *
 * 所以是「全局值 + 每页声明能力 + 自动降级」，且降级必须让用户看见。
 * 静默降级正是上一版「标签写 1 天、数据其实是 5 分钟」那个 bug 的成因。
 */

"use client";

import { useCallback, useEffect, useMemo } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useTimeRangeStore } from "@/store/timeRangeStore";
import { formatRangeParam, toSLOWindow, toSpanMs } from "@/lib/time-range";
import type { TimeRangeSelection } from "@/types/time-range";
import type { SLOWindow } from "@/lib/time-range";

/** 页面对时间范围的支持能力 */
export type RangeCapability =
  /** 任意范围（APM / Logs） */
  | "full"
  /** 只支持 SLO 的五个预聚合窗口 */
  | "sloWindows"
  /** 范围只作用于趋势图，主体数据是当前快照（Metrics） */
  | "trendOnly";

export interface ObserveTimeRange {
  /** 全局原值，用于选择器回显 —— 即使当前页用不了这个范围也照原样显示 */
  selection: TimeRangeSelection;
  /** 当前页实际使用的范围 */
  effective: TimeRangeSelection;
  /** SLO 页专用：贴合后的窗口 key */
  sloWindow: SLOWindow;
  /** 是否发生降级；true 时页面必须显示 degradeNote */
  degraded: boolean;
  /** 降级说明的 i18n 参数（窗口名），页面自己拼文案 */
  degradeTo?: string;
  setSelection: (s: TimeRangeSelection) => void;
}

export function useObserveTimeRange(capability: RangeCapability): ObserveTimeRange {
  const searchParams = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();
  const { selection, setSelection: setStoreSelection, hydrate } = useTimeRangeStore();

  // 选择变化时同步到地址栏，这样复制 URL 就能把当前时间段分享出去。
  // 用 replace 而不是 push —— 调时间范围不该塞满浏览器的前进后退历史。
  const setSelection = useCallback(
    (s: TimeRangeSelection) => {
      setStoreSelection(s);
      const params = new URLSearchParams(searchParams?.toString() ?? "");
      params.delete("range");
      params.delete("from");
      params.delete("to");
      for (const [k, v] of new URLSearchParams(formatRangeParam(s))) {
        params.set(k, v);
      }
      router.replace(`${pathname}?${params.toString()}`, { scroll: false });
    },
    [searchParams, router, pathname, setStoreSelection],
  );

  // 首次挂载时从 URL / localStorage 恢复
  useEffect(() => {
    hydrate(new URLSearchParams(searchParams?.toString() ?? ""));
  }, [searchParams, hydrate]);

  return useMemo(() => {
    const base: ObserveTimeRange = {
      selection,
      effective: selection,
      sloWindow: "1h",
      degraded: false,
      setSelection,
    };

    if (capability === "sloWindows") {
      const { window, degraded } = toSLOWindow(selection);
      return { ...base, sloWindow: window, degraded, degradeTo: degraded ? window : undefined };
    }

    // trendOnly / full 都不改变 effective —— trendOnly 的限制体现在页面怎么用它，
    // 而不是这里改值（趋势图照常用全局范围，卡片本来就不看它）
    return { ...base, sloWindow: toSLOWindow(selection).window };
  }, [capability, selection, setSelection]);
}

/** 时间跨度（毫秒），供直方图桶宽等计算使用 */
export function useTimeSpanMs(): number {
  const { selection } = useTimeRangeStore();
  return useMemo(() => toSpanMs(selection), [selection]);
}

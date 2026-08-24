/**
 * 拉取当前集群的信号新鲜度
 *
 * 四个观测页共用：空数据时用它说清楚是「没有流量」还是「采集异常」。
 * 判定在 Master 完成，这里只负责取数与挑出本页关心的那个信号。
 */

"use client";

import { useCallback, useEffect, useState } from "react";
import { useClusterStore } from "@/store/clusterStore";
import { getSignalFreshness } from "@/datasource/metrics";
import type { SignalFreshnessItem } from "@/types/observe";

/** 新鲜度变化很慢，60 秒一次足够，不跟着页面数据的刷新频率走 */
const REFRESH_MS = 60_000;

export function useSignalFreshness(signal: SignalFreshnessItem["signal"]) {
  const { currentClusterId } = useClusterStore();
  const [item, setItem] = useState<SignalFreshnessItem | null>(null);

  const load = useCallback(async () => {
    if (!currentClusterId) return;
    try {
      const res = await getSignalFreshness(currentClusterId);
      setItem(res?.signals.find((s) => s.signal === signal) ?? null);
    } catch {
      // 取不到就显示「未知」，不影响主体数据 —— 新鲜度是辅助信息，不该让页面报错
      setItem(null);
    }
  }, [currentClusterId, signal]);

  useEffect(() => {
    load();
    const id = setInterval(load, REFRESH_MS);
    return () => clearInterval(id);
  }, [load]);

  return item;
}

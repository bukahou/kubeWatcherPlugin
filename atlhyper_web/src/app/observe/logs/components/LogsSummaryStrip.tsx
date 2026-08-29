"use client";

import { useState, useEffect } from "react";
import { Activity } from "lucide-react";
import { getLogsSummary } from "@/api/observe-logs";
import type { LogsSummary } from "@/api/observe-logs";
import { useI18n } from "@/i18n/context";

// ──────────────────────────────────────────────────────────────
// 日志全局摘要条（G5 接线）
// ──────────────────────────────────────────────────────────────
//
// 与下方 Facets 不重复：Facets 统计的是「当前过滤窗口」，这里是
// 快照级全局视角。核心是 latestAt —— 日志流是否新鲜。压测期间
// collector 队列满丢数据时，最先能暴露问题的就是这个时间戳。

function freshness(latestAt: string): { text: string; stale: boolean } {
  const d = new Date(latestAt);
  if (isNaN(d.getTime()) || d.getFullYear() < 2000) return { text: "—", stale: true };
  const sec = Math.max(0, Math.round((Date.now() - d.getTime()) / 1000));
  if (sec < 60) return { text: `${sec}s`, stale: false };
  const min = Math.floor(sec / 60);
  // 快照本身约 10s 一轮，落后超过 5 分钟即视为断流征兆
  return { text: `${min}m`, stale: min >= 5 };
}

export function LogsSummaryStrip({ clusterId }: { clusterId: string }) {
  const { t } = useI18n();
  const [summary, setSummary] = useState<LogsSummary | null>(null);

  useEffect(() => {
    let cancelled = false;
    getLogsSummary(clusterId)
      .then((res) => { if (!cancelled) setSummary(res.data.data); })
      .catch(() => { /* 摘要是辅助信息，失败静默隐藏而非报错打断主功能 */ });
    return () => { cancelled = true; };
  }, [clusterId]);

  // 无数据时不渲染骨架 —— 这是辅助条，不值得占位闪烁
  if (!summary) return null;

  const err = summary.severityCounts?.["ERROR"] ?? 0;
  const warn = summary.severityCounts?.["WARN"] ?? 0;
  const fresh = freshness(summary.latestAt);

  return (
    <div className="flex items-center gap-4 text-xs text-muted px-1 flex-wrap">
      <span className="flex items-center gap-1.5">
        <Activity className={`w-3.5 h-3.5 ${fresh.stale ? "text-red-500" : "text-emerald-500"}`} />
        <span className={fresh.stale ? "text-red-500 font-medium" : ""}>
          {t.logs.latestLog} {fresh.text}
        </span>
      </span>
      <span className="tabular-nums">{t.logs.totalGlobal}: {summary.totalEntries.toLocaleString()}</span>
      <span className="tabular-nums">
        <span className={err > 0 ? "text-red-500 font-medium" : ""}>ERROR {err}</span>
        {" / "}
        <span className={warn > 0 ? "text-yellow-500" : ""}>WARN {warn}</span>
      </span>
      {summary.topServices?.[0] && (
        <span className="hidden md:inline">
          {t.logs.topService}: <span className="font-mono">{summary.topServices[0].service}</span>
        </span>
      )}
    </div>
  );
}

"use client";

import { memo } from "react";
import { Activity, MoonStar, AlertTriangle, HelpCircle } from "lucide-react";
import { useI18n } from "@/i18n/context";
import type { FreshnessStatus, SignalFreshnessItem } from "@/types/observe";

/**
 * 信号新鲜度徽章
 *
 * 解决的问题：页面空白时分不清是「没有流量」还是「采集挂了」。
 * 实测遇到过 traces 停了 78 分钟，查了 ClickHouse 和 ingress 计数才确认
 * 是凌晨没人访问 —— 这个判断本该由页面直接给出。
 */

const ICONS: Record<FreshnessStatus, typeof Activity> = {
  live: Activity,
  idle: MoonStar,
  stale: AlertTriangle,
  absent: HelpCircle,
};

const TONE: Record<FreshnessStatus, string> = {
  live: "text-emerald-500",
  // idle 用中性色而不是警告色 —— 没人访问不是问题，标黄会制造无谓的焦虑
  idle: "text-muted",
  stale: "text-red-500",
  absent: "text-muted opacity-60",
};

/** 把秒数说成人话：92 秒 → 1 分钟前 */
function formatLag(seconds: number, t: { justNow: string; minutesAgo: string; hoursAgo: string }): string {
  if (seconds < 60) return t.justNow;
  if (seconds < 3600) return t.minutesAgo.replace("{n}", String(Math.round(seconds / 60)));
  return t.hoursAgo.replace("{n}", String(Math.round(seconds / 3600)));
}

export const SignalFreshnessBadge = memo(function SignalFreshnessBadge({
  item,
}: {
  /** 为 null 时（接口未就绪）渲染占位而不是消失 —— 组件不隐藏 */
  item: SignalFreshnessItem | null;
}) {
  const { t } = useI18n();
  const f = t.observe.freshness;

  if (!item) {
    return (
      <span className="inline-flex items-center gap-1 text-[10px] sm:text-xs text-muted opacity-60">
        <HelpCircle className="w-3 h-3" />
        {f.unknown}
      </span>
    );
  }

  const Icon = ICONS[item.status];
  const label =
    item.status === "live"
      ? f.live
      : item.status === "idle"
        ? f.idle
        : item.status === "stale"
          ? f.stale
          : f.absent;

  const detail =
    item.status === "absent"
      ? undefined
      : formatLag(item.lagSeconds, { justNow: f.justNow, minutesAgo: f.minutesAgo, hoursAgo: f.hoursAgo });

  return (
    <span
      className={`inline-flex items-center gap-1 text-[10px] sm:text-xs ${TONE[item.status]}`}
      title={item.status === "idle" ? f.idleHint : item.status === "stale" ? f.staleHint : undefined}
    >
      <Icon className="w-3 h-3 flex-shrink-0" />
      <span className="whitespace-nowrap">
        {label}
        {detail && <span className="text-muted"> · {detail}</span>}
      </span>
    </span>
  );
});

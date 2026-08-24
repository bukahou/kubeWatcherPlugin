"use client";

import { memo } from "react";
import type { SLOBudget, SLOMetrics } from "@/types/slo";

export interface GoodBadCountTranslations {
  eventCount: string;
  good: string;
  bad: string;
  allowed: string;
  overspent: string;
  noData: string;
}

/**
 * Good / Bad 事件计数
 *
 * 「允许错 8 个、实际错了 123 个」比任何百分比都直接。
 * 百分比在低流量下会骗人：10 个请求错 1 个，可用率 90% 看着像事故，其实什么都没发生。
 */
export const GoodBadCount = memo(function GoodBadCount({
  summary,
  budget,
  t,
}: {
  summary: SLOMetrics | null;
  budget?: SLOBudget;
  t: GoodBadCountTranslations;
}) {
  if (!summary) {
    return (
      <div className="rounded-lg bg-[var(--hover-bg)] p-3 text-center text-xs text-muted">
        {t.noData}
      </div>
    );
  }

  const overspent = budget ? budget.consumedEvents > budget.allowedEvents : false;

  return (
    <div className="rounded-lg bg-[var(--hover-bg)] p-3">
      <div className="text-xs font-medium text-default mb-2">{t.eventCount}</div>
      <div className="grid grid-cols-3 gap-3">
        <div>
          <div className="text-[10px] text-muted">{t.good}</div>
          <div className="text-sm font-semibold text-emerald-500 tabular-nums">
            {summary.goodRequests.toLocaleString()}
          </div>
        </div>
        <div>
          <div className="text-[10px] text-muted">{t.bad}</div>
          <div className={`text-sm font-semibold tabular-nums ${overspent ? "text-red-500" : "text-default"}`}>
            {summary.badRequests.toLocaleString()}
          </div>
        </div>
        <div>
          <div className="text-[10px] text-muted">{t.allowed}</div>
          <div className="text-sm font-semibold text-muted tabular-nums">
            {budget ? budget.allowedEvents.toLocaleString() : "—"}
          </div>
        </div>
      </div>
      {overspent && budget && (
        <div className="mt-2 text-[10px] text-red-500">
          {t.overspent.replace("{n}", String(budget.consumedEvents - budget.allowedEvents))}
        </div>
      )}
    </div>
  );
});

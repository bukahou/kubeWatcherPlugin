"use client";

import { Fragment, memo } from "react";
import { Flame, ChevronRight, Activity, ScrollText } from "lucide-react";
import Link from "next/link";
import { apmLinkForService, logsLinkForService } from "@/lib/signal-link";
import type { DomainSLOV2, BurnRateWindow } from "@/types/slo";
import { DomainDetail } from "./DomainDetail";
import type { DomainDetailTranslations } from "./detail-translations";

export interface SLOListTableTranslations {
  domain: string;
  viewTraces: string;
  viewErrorLogs: string;
  target: string;
  currentSli: string;
  errorBudget: string;
  burnRate: string;
  p95Latency: string;
  status: string;
  healthy: string;
  warning: string;
  critical: string;
  unknown: string;
  noData: string;
  goodBad: string;
  exhaustIn: string;
  hours: string;
}

/** 燃烧率上色。阈值来自后端（Google SRE 多窗口模型），前端只负责渲染。 */
const burnColor = (b: BurnRateWindow | undefined) => {
  if (!b) return "text-muted";
  switch (b.status) {
    case "crit":
      return "text-red-500 font-semibold";
    case "warn":
      return "text-amber-500 font-semibold";
    default:
      return "text-default";
  }
};

const statusChip = (s: string) => {
  switch (s) {
    case "critical":
      return "bg-red-500/10 text-red-500";
    case "warning":
      return "bg-amber-500/10 text-amber-500";
    case "healthy":
      return "bg-emerald-500/10 text-emerald-500";
    default:
      return "bg-[var(--hover-bg)] text-muted";
  }
};

/** 错误预算条。低于 20% 变红 —— 这时候该停止发布了 */
function BudgetBar({ pct }: { pct: number }) {
  const clamped = Math.max(0, Math.min(100, pct));
  const color = pct > 50 ? "bg-emerald-500" : pct > 20 ? "bg-amber-500" : "bg-red-500";
  return (
    <div className="flex items-center gap-2">
      <div className="h-1.5 w-16 rounded-full bg-[var(--hover-bg)] overflow-hidden">
        <div className={`h-full rounded-full ${color}`} style={{ width: `${clamped}%` }} />
      </div>
      <span className={`text-xs tabular-nums ${pct <= 20 ? "text-red-500" : "text-muted"}`}>
        {pct.toFixed(0)}%
      </span>
    </div>
  );
}

/**
 * SLO 清单表 —— 一行一个域名，一眼看出谁在烧预算。
 *
 * 燃烧率是这张表存在的理由：原始错误率没法跨服务比较（目标 99.9% 的服务
 * 错 0.05% 已经烧掉半格预算，目标 99% 的同样错误率连零头都不算），
 * 归一化之后「谁最危险」才有意义。
 */
export const SLOListTable = memo(function SLOListTable({
  domains,
  expandedId,
  onSelect,
  timeRange,
  clusterId,
  onRefresh,
  t,
  detailT,
}: {
  domains: DomainSLOV2[];
  expandedId: string | null;
  onSelect: (domain: string) => void;
  timeRange: string;
  clusterId: string;
  onRefresh: () => void;
  t: SLOListTableTranslations;
  detailT: DomainDetailTranslations;
}) {
  const statusLabel = (s: string) =>
    s === "healthy" ? t.healthy : s === "warning" ? t.warning : s === "critical" ? t.critical : t.unknown;

  const findBurn = (d: DomainSLOV2, window: string) =>
    d.budget?.burnRates?.find((b) => b.window === window);

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full min-w-[880px] text-left">
          <thead>
            <tr className="border-b border-[var(--border-color)]">
              <th className="py-2.5 pl-4 pr-3 text-[11px] font-medium text-muted">{t.domain}</th>
              <th className="py-2.5 px-3 text-[11px] font-medium text-muted">{t.target}</th>
              <th className="py-2.5 px-3 text-[11px] font-medium text-muted">{t.currentSli}</th>
              <th className="py-2.5 px-3 text-[11px] font-medium text-muted">{t.errorBudget}</th>
              <th className="py-2.5 px-3 text-[11px] font-medium text-muted">
                <span className="inline-flex items-center gap-1">
                  <Flame className="w-3 h-3" />
                  {t.burnRate} 1h
                </span>
              </th>
              <th className="py-2.5 px-3 text-[11px] font-medium text-muted">6h</th>
              <th className="py-2.5 px-3 text-[11px] font-medium text-muted">{t.p95Latency}</th>
              <th className="py-2.5 px-3 text-[11px] font-medium text-muted">{t.status}</th>
              <th className="py-2.5 pr-4 w-8"></th>
            </tr>
          </thead>
          <tbody>
            {domains.map((d) => {
              const sli = d.summary?.availability;
              const b1h = findBurn(d, "1h");
              const b6h = findBurn(d, "6h");
              const budget = d.budget;
              const isOpen = expandedId === d.domain;
              return (
                <Fragment key={d.domain}>
                <tr
                  onClick={() => onSelect(d.domain)}
                  className={`border-b border-[var(--border-color)] last:border-0 cursor-pointer transition-colors ${
                    isOpen ? "bg-[var(--hover-bg)]" : "hover:bg-[var(--hover-bg)]"
                  }`}
                >
                  <td className="py-2.5 pl-4 pr-3">
                    <div className="text-sm text-default font-medium">{d.domain}</div>
                    {budget && (
                      <div className="text-[10px] text-muted tabular-nums">
                        {t.goodBad}: {budget.consumedEvents} / {budget.allowedEvents}
                      </div>
                    )}
                    {/* 跨信号入口：SLO 说这里超支了，下一步必然是「哪些请求错了」。
                        stopPropagation 是因为整行点击会展开详情，这两个链接要各走各的 */}
                    <div className="flex items-center gap-2 mt-1">
                      {(d.services ?? []).slice(0, 1).map((svc) => (
                        <span key={svc.serviceKey} className="flex items-center gap-2">
                          <Link
                            href={apmLinkForService(svc.serviceKey)}
                            onClick={(e) => e.stopPropagation()}
                            className="inline-flex items-center gap-0.5 text-[10px] text-blue-500 hover:underline"
                          >
                            <Activity className="w-2.5 h-2.5" />
                            {t.viewTraces}
                          </Link>
                          <Link
                            href={logsLinkForService(svc.serviceKey)}
                            onClick={(e) => e.stopPropagation()}
                            className="inline-flex items-center gap-0.5 text-[10px] text-blue-500 hover:underline"
                          >
                            <ScrollText className="w-2.5 h-2.5" />
                            {t.viewErrorLogs}
                          </Link>
                        </span>
                      ))}
                    </div>
                  </td>
                  <td className="py-2.5 px-3 text-xs text-muted tabular-nums whitespace-nowrap">
                    {d.target ? `${d.target.availability}% · ${d.target.windowDays}d` : t.noData}
                  </td>
                  <td className="py-2.5 px-3 text-sm text-default tabular-nums">
                    {sli === undefined || sli === null ? t.noData : `${sli.toFixed(3)}%`}
                  </td>
                  <td className="py-2.5 px-3">
                    {budget ? <BudgetBar pct={budget.remainingPct} /> : <span className="text-xs text-muted">{t.noData}</span>}
                  </td>
                  <td className={`py-2.5 px-3 text-sm tabular-nums ${burnColor(b1h)}`}>
                    {b1h ? `${b1h.rate.toFixed(1)}×` : t.noData}
                  </td>
                  <td className={`py-2.5 px-3 text-sm tabular-nums ${burnColor(b6h)}`}>
                    {b6h ? `${b6h.rate.toFixed(1)}×` : t.noData}
                  </td>
                  <td className="py-2.5 px-3 text-sm text-default tabular-nums">
                    {d.summary?.p95Latency ? `${d.summary.p95Latency}ms` : t.noData}
                  </td>
                  <td className="py-2.5 px-3">
                    <span className={`px-2 py-0.5 rounded-md text-[11px] font-medium ${statusChip(d.status)}`}>
                      {statusLabel(d.status)}
                    </span>
                    {budget && budget.exhaustHours > 0 && (
                      <div className="text-[10px] text-amber-500 mt-0.5 tabular-nums">
                        {t.exhaustIn} {budget.exhaustHours.toFixed(0)}{t.hours}
                      </div>
                    )}
                  </td>
                  <td className="py-2.5 pr-4">
                    <ChevronRight
                      className={`w-4 h-4 text-muted transition-transform ${isOpen ? "rotate-90" : ""}`}
                    />
                  </td>
                </tr>
                {/* 详情就地展开：域名与它的细节在同一处，不用在页面上找第二遍 */}
                {isOpen && (
                  <tr>
                    <td colSpan={9} className="p-0">
                      <DomainDetail
                        domain={d}
                        timeRange={timeRange}
                        clusterId={clusterId}
                        onRefresh={onRefresh}
                        t={detailT}
                      />
                    </td>
                  </tr>
                )}
                </Fragment>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
});

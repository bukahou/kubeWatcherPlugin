"use client";

import { BarChart3, Layers } from "lucide-react";
import type { LatencyDistributionResponse } from "@/types/slo";
import { LatencyHistogram } from "./LatencyHistogram";

export interface LatencyTabTranslations {
  latencyDistribution: string;
  methodBreakdown: string;
  statusCodeBreakdown: string;
  requests: string;
  clearSelection: string;
  noData: string;
}

// 窗口 key 本身就是可读标签（1h / 6h / 24h / 3d / 7d），不需要再映射一层
function timeRangeLabel(tr: string): string {
  return tr;
}

export function LatencyTab({ data, timeRange, t }: {
  data: LatencyDistributionResponse | null;
  timeRange: string;
  t: LatencyTabTranslations;
}) {
  if (!data || !data.buckets || data.buckets.length === 0) {
    return (
      <div className="text-center py-8 text-muted text-sm">
        {t.noData}
      </div>
    );
  }

  const badgeLabel = `Ingress \u00b7 ${timeRangeLabel(timeRange)}`;
  const hasMethods = data.methods && data.methods.length > 0;
  const hasStatusCodes = data.statusCodes && data.statusCodes.length > 0;

  return (
    <div className="flex flex-col lg:flex-row gap-4">
      {/* Left: Histogram (60%) */}
      <div className="lg:w-[60%] flex-shrink-0 flex">
        <LatencyHistogram
          buckets={data.buckets}
          p50={data.p50LatencyMs}
          p95={data.p95LatencyMs}
          p99={data.p99LatencyMs}
          badgeLabel={badgeLabel}
          t={t}
        />
      </div>
      {/* Right: Method + StatusCode (40%) */}
      {(hasMethods || hasStatusCodes) && (
        <div className="flex-1 min-w-0 space-y-4">
          {hasMethods && (
            <MethodChart
              methods={data.methods}
              totalRequests={data.totalRequests}
              badgeLabel={badgeLabel}
              t={t}
            />
          )}
          {hasStatusCodes && (
            <StatusCodeChart
              statusCodes={data.statusCodes}
              totalRequests={data.totalRequests}
              badgeLabel={badgeLabel}
              t={t}
            />
          )}
        </div>
      )}
    </div>
  );
}

// ==================== Method Chart ====================

const methodColors: Record<string, string> = {
  GET: "bg-blue-500",
  POST: "bg-emerald-500",
  PUT: "bg-amber-500",
  DELETE: "bg-red-500",
  OTHER: "bg-slate-500",
  PATCH: "bg-violet-500",
};

function MethodChart({ methods, totalRequests, badgeLabel, t }: {
  methods: { method: string; count: number }[];
  totalRequests: number;
  badgeLabel: string;
  t: LatencyTabTranslations;
}) {
  const maxCount = Math.max(...methods.map(m => m.count), 1);

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="px-4 py-3 border-b border-[var(--border-color)] flex items-center gap-3">
        <Layers className="w-4 h-4 text-primary" />
        <span className="text-sm font-medium text-default">{t.methodBreakdown}</span>
        <span className="px-1.5 py-0.5 rounded text-[9px] font-medium bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400">{badgeLabel}</span>
        <span className="text-[10px] text-muted">
          {totalRequests.toLocaleString()} {t.requests}
        </span>
      </div>
      <div className="p-4 space-y-2.5">
        {methods.map((m) => {
          const percent = totalRequests > 0 ? (m.count / totalRequests) * 100 : 0;
          const barWidth = (m.count / maxCount) * 100;
          return (
            <div key={m.method} className="flex items-center gap-3">
              <span className={`text-[10px] font-mono px-1.5 py-0.5 rounded font-medium w-14 text-center text-white ${methodColors[m.method] || "bg-slate-500"}`}>
                {m.method}
              </span>
              <div className="flex-1 h-5 bg-[var(--hover-bg)] rounded-sm overflow-hidden relative">
                <div
                  className={`h-full rounded-sm ${methodColors[m.method] || "bg-slate-500"} opacity-80`}
                  style={{ width: `${barWidth}%` }}
                />
              </div>
              <div className="w-28 text-right flex items-center gap-2 justify-end">
                <span className="text-xs font-medium text-default">{m.count.toLocaleString()}</span>
                <span className="text-[10px] text-muted">({percent.toFixed(0)}%)</span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

// ==================== Status Code Chart ====================

const statusColorMap: Record<string, { bar: string; bg: string; text: string }> = {
  "2": { bar: "bg-emerald-500", bg: "bg-emerald-50 dark:bg-emerald-900/20", text: "text-emerald-700 dark:text-emerald-400" },
  "3": { bar: "bg-blue-500", bg: "bg-blue-50 dark:bg-blue-900/20", text: "text-blue-700 dark:text-blue-400" },
  "4": { bar: "bg-amber-500", bg: "bg-amber-50 dark:bg-amber-900/20", text: "text-amber-700 dark:text-amber-400" },
  "5": { bar: "bg-red-500", bg: "bg-red-50 dark:bg-red-900/20", text: "text-red-700 dark:text-red-400" },
};
const defaultStatusColor = statusColorMap["2"];

function getStatusColor(code: string) {
  return statusColorMap[code[0]] || defaultStatusColor;
}

function StatusCodeChart({ statusCodes, totalRequests, badgeLabel, t }: {
  statusCodes: { code: string; count: number }[];
  totalRequests: number;
  badgeLabel: string;
  t: LatencyTabTranslations;
}) {
  const sorted = [...statusCodes].sort((a, b) => a.code.localeCompare(b.code));
  const maxCount = Math.max(...sorted.map(s => s.count), 1);

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="px-4 py-3 border-b border-[var(--border-color)] flex items-center gap-3">
        <BarChart3 className="w-4 h-4 text-primary" />
        <span className="text-sm font-medium text-default">{t.statusCodeBreakdown}</span>
        <span className="px-1.5 py-0.5 rounded text-[9px] font-medium bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400">{badgeLabel}</span>
        <span className="text-[10px] text-muted">
          {totalRequests.toLocaleString()} {t.requests}
        </span>
      </div>
      <div className="p-4 space-y-2.5">
        {sorted.map((s) => {
          const percent = totalRequests > 0 ? (s.count / totalRequests) * 100 : 0;
          const barWidth = (s.count / maxCount) * 100;
          const colors = getStatusColor(s.code);
          return (
            <div key={s.code} className="flex items-center gap-3">
              <span className={`text-[10px] font-mono px-1.5 py-0.5 rounded font-semibold w-10 text-center ${colors.text} ${colors.bg}`}>
                {s.code}
              </span>
              <div className="flex-1 h-5 bg-[var(--hover-bg)] rounded-sm overflow-hidden relative">
                <div
                  className={`h-full rounded-sm ${colors.bar} opacity-80`}
                  style={{ width: `${barWidth}%` }}
                />
              </div>
              <div className="w-32 text-right flex items-center gap-2 justify-end">
                <span className="text-xs font-medium text-default">{percent.toFixed(1)}%</span>
                <span className="text-[10px] text-muted">{s.count.toLocaleString()}</span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

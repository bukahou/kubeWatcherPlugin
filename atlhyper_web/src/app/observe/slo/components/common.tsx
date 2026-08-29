"use client";

import {
  Activity,
  AlertTriangle,
  TrendingUp,
  TrendingDown,
  Minus,
  ArrowUpRight,
  ArrowDownRight,
} from "lucide-react";

// ==================== Namespace Colors ====================

const namespaceColors: Record<string, { fill: string; stroke: string; light: string }> = {
  "kube-system": { fill: "#7c3aed", stroke: "#6d28d9", light: "#a78bfa" },
  "geass":       { fill: "#0891b2", stroke: "#0e7490", light: "#22d3ee" },
  "elastic":     { fill: "#d97706", stroke: "#b45309", light: "#fbbf24" },
  "atlhyper":    { fill: "#059669", stroke: "#047857", light: "#34d399" },
  "default":     { fill: "#4b5563", stroke: "#374151", light: "#9ca3af" },
};

export function getNamespaceColor(ns: string) {
  return namespaceColors[ns] || namespaceColors["default"];
}

// ==================== Status Badge ====================

export function StatusBadge({ status, labels }: {
  status: string;
  labels: { healthy: string; warning: string; critical: string; unknown: string };
}) {
  const config: Record<string, { bg: string; text: string; dot: string }> = {
    healthy:  { bg: "bg-emerald-500/10", text: "text-emerald-500", dot: "bg-emerald-500" },
    warning:  { bg: "bg-amber-500/10",   text: "text-amber-500",   dot: "bg-amber-500" },
    critical: { bg: "bg-red-500/10",     text: "text-red-500",     dot: "bg-red-500" },
    unknown:  { bg: "bg-gray-500/10",    text: "text-gray-500",    dot: "bg-gray-500" },
  };
  const c = config[status] || config.unknown;
  const label = (labels as Record<string, string>)[status] || labels.unknown;
  return (
    <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium ${c.bg} ${c.text}`}>
      <span className={`w-1.5 h-1.5 rounded-full ${c.dot}`} />
      {label}
    </span>
  );
}

// ==================== Error Budget Bar ====================

export function ErrorBudgetBar({ percent }: { percent: number }) {
  const isHealthy = percent > 50;
  const isWarning = percent > 20 && percent <= 50;
  return (
    <div className="flex items-center gap-2">
      <div className="flex-1 h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
        <div
          className={`h-full rounded-full transition-all ${
            isHealthy ? "bg-emerald-500" : isWarning ? "bg-amber-500" : "bg-red-500"
          }`}
          style={{ width: `${Math.max(0, Math.min(100, percent))}%` }}
        />
      </div>
      <span className={`text-xs font-medium w-10 text-right ${
        isHealthy ? "text-emerald-500" : isWarning ? "text-amber-500" : "text-red-500"
      }`}>
        {percent.toFixed(0)}%
      </span>
    </div>
  );
}

// ==================== Trend Icon ====================

export function TrendIcon({ trend }: { trend?: string }) {
  if (trend === "up") return <TrendingUp className="w-4 h-4 text-emerald-500" />;
  if (trend === "down") return <TrendingDown className="w-4 h-4 text-red-500" />;
  return <Minus className="w-4 h-4 text-gray-400" />;
}

// ==================== Compare Metric ====================

export function CompareMetric({ label, current, previous, unit, inverse = false, previousPeriodLabel }: {
  label: string;
  current: number;
  previous: number;
  unit: string;
  inverse?: boolean;
  previousPeriodLabel: string;
}) {
  const diff = current - previous;
  const percentDiff = previous !== 0 ? (diff / previous) * 100 : 0;
  const isImproved = inverse ? diff < 0 : diff > 0;
  const isWorsened = inverse ? diff > 0 : diff < 0;

  return (
    <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
      <div className="text-xs text-muted mb-1">{label}</div>
      <div className="flex items-end gap-2">
        <span className="text-lg font-bold text-default">{current.toFixed(2)}{unit}</span>
        <div className={`flex items-center text-xs ${isImproved ? "text-emerald-500" : isWorsened ? "text-red-500" : "text-gray-400"}`}>
          {isImproved ? (
            <ArrowUpRight className="w-3 h-3" />
          ) : isWorsened ? (
            <ArrowDownRight className="w-3 h-3" />
          ) : (
            <Minus className="w-3 h-3" />
          )}
          <span>{Math.abs(percentDiff).toFixed(1)}%</span>
        </div>
      </div>
      <div className="text-xs text-muted mt-0.5">{previousPeriodLabel} {previous.toFixed(2)}{unit}</div>
    </div>
  );
}

// ==================== Summary Card ====================

export function SummaryCard({
  icon: Icon,
  label,
  value,
  subValue,
  color,
}: {
  icon: typeof Activity;
  label: string;
  value: string;
  subValue?: string;
  color: string;
}) {
  return (
    <div className="p-3 sm:p-4 rounded-xl bg-card border border-[var(--border-color)]">
      <div className="flex items-center gap-2 sm:gap-3">
        <div className={`p-1.5 sm:p-2 rounded-lg flex-shrink-0 ${color}`}>
          <Icon className="w-4 h-4 sm:w-5 sm:h-5" />
        </div>
        <div className="min-w-0">
          <div className="text-[10px] sm:text-xs text-muted truncate">{label}</div>
          <div className="text-lg sm:text-xl font-bold text-default">{value}</div>
          {subValue && <div className="text-[10px] sm:text-xs text-muted truncate">{subValue}</div>}
        </div>
      </div>
    </div>
  );
}

// ==================== Utilities ====================

export function formatLatency(ms: number): string {
  if (ms === 0) return "0ms";
  if (ms < 1) return `${ms.toFixed(2)}ms`;
  if (ms < 10) return `${ms.toFixed(1)}ms`;
  if (ms < 1000) return `${Math.round(ms)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}min`;
}

export function formatRPS(rps: number): string {
  if (rps === 0) return "0";
  if (rps >= 1000) return `${(rps / 1000).toFixed(1)}K`;
  if (rps >= 1) return rps.toFixed(0);
  return rps.toFixed(2);
}

export function formatNumber(num: number): string {
  if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`;
  if (num >= 1000) return `${(num / 1000).toFixed(1)}K`;
  return num.toLocaleString();
}

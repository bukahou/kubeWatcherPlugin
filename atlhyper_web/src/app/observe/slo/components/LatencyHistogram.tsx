"use client";

import { BarChart3 } from "lucide-react";
import type { LatencyTabTranslations } from "./LatencyTab";

// 1-2-5 log-scale tick series (evenly spaced on log axis)
const STANDARD_TICKS = [1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000, 10000];

// Log-scale mapping: ms value -> 0-100% position within [lo, hi]
function logPos(ms: number, lo: number, hi: number): number {
  const v = Math.log10(Math.max(ms, 0.1));
  const a = Math.log10(Math.max(lo, 0.1));
  const b = Math.log10(Math.max(hi, 1));
  if (b <= a) return 50;
  return Math.min(100, Math.max(0, ((v - a) / (b - a)) * 100));
}

// Determine visible axis range, snapped to standard ticks with padding
function axisRange(les: number[], p99Hint?: number): [number, number] {
  if (les.length === 0) return [1, 1000];
  const minLe = Math.min(...les);
  const maxLe = Math.max(...les);
  const effectiveMax = p99Hint ? Math.min(maxLe, Math.max(p99Hint * 3, maxLe * 0.3)) : maxLe;
  let lo = STANDARD_TICKS[0];
  for (const t of STANDARD_TICKS) { if (t <= minLe * 0.6) lo = t; else break; }
  let hi = STANDARD_TICKS[STANDARD_TICKS.length - 1];
  for (let i = STANDARD_TICKS.length - 1; i >= 0; i--) { if (STANDARD_TICKS[i] >= effectiveMax * 1.4) hi = STANDARD_TICKS[i]; else break; }
  return [Math.min(lo, minLe * 0.5), Math.max(hi, effectiveMax * 1.5)];
}

// Select visible tick labels within axis range
function visibleTicks(lo: number, hi: number): number[] {
  return STANDARD_TICKS.filter(t => t >= lo && t <= hi);
}

function tickLabel(ms: number): string { return ms >= 1000 ? `${ms / 1000}s` : `${ms}ms`; }

export function LatencyHistogram({ buckets, p50, p95, p99, badgeLabel, t }: {
  buckets: { le: number; count: number }[];
  p50: number;
  p95: number;
  p99: number;
  badgeLabel: string;
  t: LatencyTabTranslations;
}) {
  const sorted = [...buckets].sort((a, b) => a.le - b.le);
  const active = sorted.filter(b => b.count > 0);
  if (active.length === 0) return null;

  const maxCount = Math.max(...active.map(b => b.count), 1);
  const [lo, hi] = axisRange(active.map(b => b.le), p99);
  const ticks = visibleTicks(lo, hi);

  const visibleBuckets = active.filter(b => b.le <= hi * 1.1);
  const bars = visibleBuckets.map((b) => {
    const idx = sorted.indexOf(b);
    const prev = idx > 0 ? sorted[idx - 1].le : lo;
    const left = logPos(prev, lo, hi);
    const right = logPos(b.le, lo, hi);
    const color = b.le > p99
      ? "bg-red-400/80 hover:bg-red-500"
      : b.le > p95
        ? "bg-amber-400/90 hover:bg-amber-500"
        : b.le > p50
          ? "bg-teal-400/80 hover:bg-teal-500 dark:bg-teal-500/70 dark:hover:bg-teal-400"
          : "bg-blue-400/80 hover:bg-blue-500 dark:bg-blue-500/70 dark:hover:bg-blue-400";
    const prevLe = idx > 0 ? sorted[idx - 1].le : 0;
    const barW = 1.8;
    const center = (left + right) / 2;
    return { ...b, prevLe, left: center - barW / 2, width: barW, color };
  });

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden w-full h-full flex flex-col">
      <div className="px-4 py-3 border-b border-[var(--border-color)] flex items-center justify-between">
        <div className="flex items-center gap-3">
          <BarChart3 className="w-4 h-4 text-primary" />
          <span className="text-sm font-medium text-default">{t.latencyDistribution}</span>
          <span className="px-1.5 py-0.5 rounded text-[9px] font-medium bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400">{badgeLabel}</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="px-2 py-0.5 bg-blue-100 dark:bg-blue-900/30 rounded text-[10px] text-blue-700 dark:text-blue-400 font-medium">P50 {p50}ms</span>
          <span className="px-2 py-0.5 bg-amber-100 dark:bg-amber-900/30 rounded text-[10px] text-amber-700 dark:text-amber-400 font-medium">P95 {p95}ms</span>
          <span className="px-2 py-0.5 bg-red-100 dark:bg-red-900/30 rounded text-[10px] text-red-700 dark:text-red-400 font-medium">P99 {p99}ms</span>
        </div>
      </div>

      <div className="px-4 py-3 flex-1 flex flex-col min-h-0">
        <div className="flex gap-2 flex-1 min-h-[9rem]">
          {/* Y axis */}
          <div className="flex flex-col justify-between text-[9px] text-muted text-right w-8 flex-shrink-0">
            <span>{maxCount.toLocaleString()}</span>
            <span>{Math.round(maxCount / 2).toLocaleString()}</span>
            <span>0</span>
          </div>

          {/* Chart column */}
          <div className="flex-1 flex flex-col min-w-0">
            {/* Bars area */}
            <div className="relative flex-1">
              {/* Horizontal grid */}
              <div className="absolute inset-0 pointer-events-none">
                <div className="absolute top-0 left-0 right-0 border-t border-[var(--border-color)]" />
                <div className="absolute top-1/2 left-0 right-0 border-t border-dashed border-[var(--border-color)]" />
                <div className="absolute bottom-0 left-0 right-0 border-t border-[var(--border-color)]" />
              </div>

              {/* Vertical tick grid */}
              {ticks.map(tick => (
                <div key={tick} className="absolute top-0 bottom-0 w-px bg-[var(--border-color)] opacity-30 pointer-events-none"
                  style={{ left: `${logPos(tick, lo, hi)}%` }} />
              ))}

              {/* P50/P95/P99 lines */}
              {[
                { value: p50, c: "blue", label: "P50" },
                { value: p95, c: "amber", label: "P95" },
                { value: p99, c: "red", label: "P99" },
              ].map(({ value, c, label }) => (
                <div key={label} className={`absolute top-0 bottom-0 w-px bg-${c}-500/70 z-20 pointer-events-none`}
                  style={{ left: `${logPos(value, lo, hi)}%` }}>
                  <div className={`absolute -top-1 left-1/2 -translate-x-1/2 px-1 py-0.5 bg-${c}-500 text-white text-[8px] font-medium whitespace-nowrap rounded`}>
                    {label}
                  </div>
                </div>
              ))}

              {/* Bars */}
              {bars.map((bar, idx) => {
                const hPct = (bar.count / maxCount) * 100;
                return (
                  <div key={idx} className="absolute bottom-0 group z-10"
                    style={{ left: `${bar.left}%`, width: `${bar.width}%`, height: "100%" }}>
                    <div className="h-full flex items-end px-px">
                      <div className={`w-full rounded-t-sm transition-all duration-150 ${bar.color}`}
                        style={{ height: `${Math.max(hPct, 2)}%` }} />
                    </div>
                    <div className="absolute bottom-full mb-2 left-1/2 -translate-x-1/2 hidden group-hover:block z-30 pointer-events-none">
                      <div className="bg-slate-900 text-white text-[10px] px-2.5 py-1.5 rounded-md shadow-xl whitespace-nowrap border border-slate-700">
                        <div className="font-medium">{bar.prevLe > 0 ? `${bar.prevLe}\u2013${bar.le}ms` : `0\u2013${bar.le}ms`}</div>
                        <div className="text-slate-300">{bar.count.toLocaleString()} {t.requests}</div>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>

            {/* X axis */}
            <div className="relative h-5 mt-1 flex-shrink-0 text-[9px] text-muted">
              {ticks.map(tick => (
                <span key={tick} className="absolute -translate-x-1/2 whitespace-nowrap"
                  style={{ left: `${logPos(tick, lo, hi)}%` }}>
                  {tickLabel(tick)}
                </span>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

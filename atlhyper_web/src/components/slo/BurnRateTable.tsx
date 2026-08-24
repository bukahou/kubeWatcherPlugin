"use client";

import { memo } from "react";
import { Flame } from "lucide-react";
import type { BurnRateWindow } from "@/types/slo";

export interface BurnRateTableTranslations {
  burnRate: string;
  burnRateHint: string;
  window: string;
  rate: string;
  threshold: string;
  status: string;
  good: string;
  warn: string;
  crit: string;
  noData: string;
}

/**
 * 多窗口燃烧率表（Google SRE 的多窗口模型）
 *
 * 一个窗口看不出问题：1 小时烧得猛可能只是一次发布抖动，3 天持续超 1× 才是真的会超支。
 * 四个窗口并排，才能区分「刚出事」「正在恶化」「长期欠账」。
 * 阈值与判定都来自后端，这里只上色。
 */
export const BurnRateTable = memo(function BurnRateTable({
  burnRates,
  t,
}: {
  burnRates: BurnRateWindow[];
  t: BurnRateTableTranslations;
}) {
  const tone = (s: string) =>
    s === "crit" ? "text-red-500" : s === "warn" ? "text-amber-500" : "text-emerald-500";
  const label = (s: string) => (s === "crit" ? t.crit : s === "warn" ? t.warn : t.good);

  return (
    <div className="rounded-lg bg-[var(--hover-bg)] p-3">
      <div className="flex items-baseline gap-2 mb-2">
        <span className="inline-flex items-center gap-1 text-xs font-medium text-default">
          <Flame className="w-3.5 h-3.5" />
          {t.burnRate}
        </span>
        <span className="text-[10px] text-muted">{t.burnRateHint}</span>
      </div>

      {burnRates.length === 0 ? (
        <div className="py-3 text-center text-xs text-muted">{t.noData}</div>
      ) : (
        <table className="w-full text-left">
          <thead>
            <tr className="text-[10px] text-muted">
              <th className="py-1 pr-3 font-medium">{t.window}</th>
              <th className="py-1 px-3 font-medium">{t.rate}</th>
              <th className="py-1 px-3 font-medium">{t.threshold}</th>
              <th className="py-1 pl-3 font-medium">{t.status}</th>
            </tr>
          </thead>
          <tbody>
            {burnRates.map((b) => (
              <tr key={b.window} className="border-t border-[var(--border-color)]">
                <td className="py-1.5 pr-3 text-xs text-default tabular-nums">{b.window}</td>
                <td className={`py-1.5 px-3 text-xs tabular-nums font-medium ${tone(b.status)}`}>
                  {b.rate.toFixed(2)}×
                </td>
                <td className="py-1.5 px-3 text-xs text-muted tabular-nums">{b.threshold}×</td>
                <td className={`py-1.5 pl-3 text-xs ${tone(b.status)}`}>{label(b.status)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
});

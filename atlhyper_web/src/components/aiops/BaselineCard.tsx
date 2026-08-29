"use client";

import { useI18n } from "@/i18n/context";
import type { EntityBaseline } from "@/api/aiops";

// ──────────────────────────────────────────────────────────────
// 指标基线卡片
// ──────────────────────────────────────────────────────────────
//
// AIOps 引擎一直在为每个实体学习 EMA 基线，但此前界面上没有任何入口
// 能看到它（2026-08-29 能力盘点 G1）。
//
// 展示的核心不是「EMA 是多少」，而是「这个基线能不能信」：
// 采样数未达冷启动阈值时引擎只学习、不判异常，此时 ema=0 意味着
// 「还没学够」而非「常态就是 0」。两者混淆会让人误以为基线已生效。
// 故 ready 状态与学习进度是卡片的第一信息，数值其次。

interface BaselineCardProps {
  baseline: EntityBaseline | null;
}

// 基线值可能是 0/1 的布尔型指标，也可能是计数，量级差异大
function formatValue(v: number): string {
  if (v === 0) return "0";
  if (Math.abs(v) >= 100) return v.toFixed(0);
  if (Math.abs(v) >= 1) return v.toFixed(2);
  return v.toFixed(3);
}

export function BaselineCard({ baseline }: BaselineCardProps) {
  const { t } = useI18n();

  if (!baseline || baseline.states.length === 0) {
    return (
      <div>
        <h4 className="text-xs font-medium text-muted mb-2">{t.aiops.baselineTitle}</h4>
        <div className="py-3 text-center text-xs text-muted">{t.aiops.baselineUnavailable}</div>
      </div>
    );
  }

  const threshold = baseline.coldStartMinCount || 0;

  return (
    <div>
      <div className="flex items-baseline gap-2 mb-2">
        <h4 className="text-xs font-medium text-muted">{t.aiops.baselineTitle}</h4>
        <span className="text-[10px] text-muted/70">{t.aiops.baselineDesc}</span>
      </div>

      <div className="space-y-1.5">
        {baseline.states.map((s) => {
          const pct = threshold > 0 ? Math.min(100, (s.count / threshold) * 100) : 100;
          return (
            <div key={s.metricName} className="flex items-center gap-3 text-xs">
              <span className="font-mono text-default w-40 truncate" title={s.metricName}>
                {s.metricName}
              </span>

              <span className="text-muted">
                {t.aiops.baselineNormal}:{" "}
                <span className="text-default font-mono tabular-nums">{formatValue(s.ema)}</span>
              </span>

              <span className="text-muted">
                {t.aiops.baselineFluctuation}:{" "}
                <span className="text-default font-mono tabular-nums">±{formatValue(s.stdDev)}</span>
              </span>

              {/* 学习进度：未就绪时该基线不参与异常判定，必须让人看出来 */}
              <span className="flex items-center gap-1.5 ml-auto">
                <span className="text-muted tabular-nums text-[11px]">
                  {t.aiops.baselineSamples} {s.count}
                  {threshold > 0 && !s.ready && `/${threshold}`}
                </span>
                {!s.ready && threshold > 0 && (
                  <span className="w-12 h-1 rounded-full bg-border overflow-hidden" aria-hidden>
                    <span className="block h-full bg-amber-500" style={{ width: `${pct}%` }} />
                  </span>
                )}
                <span
                  className={`text-[10px] px-1.5 py-0.5 rounded font-medium ${
                    s.ready
                      ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
                      : "bg-amber-500/10 text-amber-600 dark:text-amber-400"
                  }`}
                >
                  {s.ready ? t.aiops.baselineReady : t.aiops.baselineLearning}
                </span>
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

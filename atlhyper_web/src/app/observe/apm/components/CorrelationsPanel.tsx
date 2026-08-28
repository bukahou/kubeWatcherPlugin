"use client";

import { useState, useEffect } from "react";
import { AlertTriangle, Loader2 } from "lucide-react";
import type { ApmTranslations } from "@/types/i18n";
import type { CorrelationResult, CorrelationMode, CorrelationImpact } from "@/types/model/apm";
import { getCorrelations, type TimeParams } from "@/datasource/apm";
import { formatDurationMs } from "@/lib/format";

// ============================================================
// CorrelationsPanel — 相关性分析（对齐 Elastic APM Correlations）
//
// 回答「慢/错的请求相比正常请求，什么属性显著超标」。
// 打分、排序、impact 分级全部由后端完成，此处只渲染。
// 低样本告警必须展示 —— 2 条失败里 100% 是 iPhone 是线索，不是结论。
// ============================================================

interface CorrelationsPanelProps {
  t: ApmTranslations;
  clusterId?: string;
  serviceName: string;
  timeParams?: TimeParams;
  operation?: string;
}

const IMPACT_STYLE: Record<CorrelationImpact, string> = {
  high: "bg-red-500/15 text-red-400",
  medium: "bg-amber-500/15 text-amber-400",
  low: "bg-gray-500/10 text-muted",
};

function RatioBar({ ratio, color }: { ratio: number; color: string }) {
  return (
    <div className="flex items-center gap-2 min-w-[130px]">
      <div className="flex-1 h-1.5 rounded-full bg-[var(--hover-bg)] overflow-hidden">
        <div className="h-full rounded-full" style={{ width: `${Math.min(ratio * 100, 100)}%`, backgroundColor: color }} />
      </div>
      <span className="text-[11px] font-mono tabular-nums w-[38px] text-right text-default">
        {Math.round(ratio * 100)}%
      </span>
    </div>
  );
}

export function CorrelationsPanel({ t, clusterId, serviceName, timeParams, operation }: CorrelationsPanelProps) {
  const [mode, setMode] = useState<CorrelationMode>("failure");
  const [result, setResult] = useState<CorrelationResult | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    getCorrelations(clusterId, serviceName, mode, timeParams, operation)
      .then(setResult)
      .finally(() => setLoading(false));
  }, [clusterId, serviceName, mode, timeParams, operation]);

  const fgLabel = mode === "failure" ? t.corrFgRatioFailure : t.corrFgRatioLatency;

  return (
    <div className="border border-[var(--border-color)] rounded-xl bg-card overflow-hidden">
      <div className="flex items-center gap-3 px-4 py-3 border-b border-[var(--border-color)]">
        <div>
          <h4 className="text-sm font-semibold text-default">{t.correlations}</h4>
          <p className="text-[11px] text-muted mt-0.5">{t.correlationsDesc}</p>
        </div>
        <div className="ml-auto flex items-center gap-1 text-xs">
          {(["failure", "latency"] as const).map((m) => (
            <button
              key={m}
              onClick={() => setMode(m)}
              className={`px-2.5 py-1 rounded-full border transition-colors ${
                mode === m
                  ? "border-primary text-primary"
                  : "border-[var(--border-color)] text-muted hover:text-default"
              }`}
            >
              {m === "failure" ? t.correlationModeFailure : t.correlationModeLatency}
            </button>
          ))}
        </div>
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-10 text-sm text-muted">
          <Loader2 className="w-4 h-4 animate-spin mr-2" />
          {t.loading}
        </div>
      ) : !result || result.foregroundCount === 0 || result.items.length === 0 ? (
        <div className="flex items-center justify-center py-10 text-sm text-muted">{t.corrNoSamples}</div>
      ) : (
        <>
          <div className="flex items-center gap-3 px-4 py-2 text-[11px] text-muted border-b border-[var(--border-color)]/40">
            <span className="font-mono">
              {result.foregroundCount} / {result.backgroundCount}
            </span>
            {result.thresholdMs !== undefined && result.thresholdMs > 0 && (
              <span>{t.corrThreshold}: {formatDurationMs(result.thresholdMs)}</span>
            )}
            {result.lowSample && (
              <span className="ml-auto flex items-center gap-1 text-amber-400 bg-amber-500/10 border border-amber-500/30 rounded-md px-2 py-0.5">
                <AlertTriangle className="w-3 h-3" />
                {t.corrLowSample}
              </span>
            )}
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="text-[10px] text-muted">
                  <th className="text-left font-medium px-4 py-2">{t.corrAttrValue}</th>
                  <th className="text-left font-medium px-4 py-2">{fgLabel}</th>
                  <th className="text-left font-medium px-4 py-2">{t.corrBgRatio}</th>
                  <th className="text-right font-medium px-4 py-2">{t.corrLift}</th>
                  <th className="text-left font-medium px-4 py-2">{t.corrImpact}</th>
                </tr>
              </thead>
              <tbody>
                {result.items.map((item) => (
                  <tr key={`${item.field}=${item.value}`} className="border-t border-[var(--border-color)]/30">
                    <td className="px-4 py-2 font-mono">
                      <div className="text-default break-all">{item.field} = {item.value}</div>
                      <div className="text-[10px] text-muted">{item.fgCount} / {result.foregroundCount}</div>
                    </td>
                    <td className="px-4 py-2"><RatioBar ratio={item.fgRatio} color="#ef4444" /></td>
                    <td className="px-4 py-2"><RatioBar ratio={item.bgRatio} color="var(--text-muted)" /></td>
                    <td className="px-4 py-2 text-right font-mono tabular-nums text-default">{item.lift.toFixed(1)}×</td>
                    <td className="px-4 py-2">
                      <span className={`inline-flex px-2 py-0.5 rounded-full text-[10px] font-semibold ${IMPACT_STYLE[item.impact]}`}>
                        ● {item.impact}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </div>
  );
}

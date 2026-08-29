"use client";

import { useState, useEffect } from "react";
import { X, Loader2 } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { RiskBadge } from "@/components/aiops/RiskBadge";
import { EntityLink } from "@/components/aiops/EntityLink";
import { AIAnalysisSection } from "@/components/aiops/AIAnalysisSection";
import { RootCauseCard } from "./RootCauseCard";
import { TimelineView } from "./TimelineView";
import { getIncidentDetail, getIncidentPatterns } from "@/api/aiops";
import { useClusterStore } from "@/store/clusterStore";
import type { IncidentDetail, IncidentPattern } from "@/api/aiops";
import { formatRiskScore } from "@/lib/risk";

const STATE_COLORS: Record<string, string> = {
  warning: "bg-yellow-500/15 text-yellow-600 dark:text-yellow-400",
  incident: "bg-red-500/15 text-red-600 dark:text-red-400",
  recovery: "bg-blue-500/15 text-blue-600 dark:text-blue-400",
  stable: "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400",
};

interface IncidentDetailModalProps {
  incidentId: string | null;
  open: boolean;
  onClose: () => void;
}

function formatDuration(seconds: number, minuteLabel: string): string {
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes} ${minuteLabel}`;
  const hours = Math.floor(minutes / 60);
  const mins = minutes % 60;
  return `${hours}h ${mins}m`;
}

const ROLE_COLORS: Record<string, string> = {
  root_cause: "text-red-500",
  affected: "text-yellow-600 dark:text-yellow-400",
  symptom: "text-blue-500",
};


export function IncidentDetailModal({ incidentId, open, onClose }: IncidentDetailModalProps) {
  const { t } = useI18n();
  const { currentClusterId } = useClusterStore();
  const [detail, setDetail] = useState<IncidentDetail | null>(null);
  const [loading, setLoading] = useState(false);
  // 历史模式（G2）：拿到详情后按 rootCause 实体查同类事件史
  const [pattern, setPattern] = useState<IncidentPattern | null | undefined>(undefined);

  useEffect(() => {
    if (!incidentId || !open) {
      setDetail(null);
      return;
    }

    setLoading(true);
    setPattern(undefined);
    getIncidentDetail(incidentId)
      .then((d) => {
        setDetail(d);
        // rootCause 字段即实体 key；查 30 天内同实体的事件史
        if (d?.rootCause) {
          getIncidentPatterns(currentClusterId, d.rootCause)
            .then((ps) => setPattern(ps.find((p) => p.entityKey === d.rootCause) ?? null))
            .catch(() => setPattern(null));
        } else {
          setPattern(null);
        }
      })
      .catch((err) => console.error("Failed to load incident detail:", err))
      .finally(() => setLoading(false));
  }, [incidentId, open, currentClusterId]);

  if (!open) return null;

  const rootCauseEntity = detail?.entities.find((e) => e.role === "root_cause");

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* 背景遮罩 */}
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" onClick={onClose} />

      {/* 弹窗 */}
      <div className="relative bg-card border border-[var(--border-color)] rounded-2xl shadow-2xl w-full max-w-2xl max-h-[85vh] overflow-y-auto mx-4">
        {/* 头部 */}
        <div className="sticky top-0 bg-card border-b border-[var(--border-color)] px-6 py-4 flex items-center justify-between rounded-t-2xl">
          <div className="flex items-center gap-3">
            <h2 className="text-base font-bold text-default">
              {t.aiops.incidentId} {detail?.id ?? incidentId}
            </h2>
          </div>
          <button onClick={onClose} className="p-1 rounded-lg hover:bg-[var(--hover-bg)] text-muted hover:text-default transition-colors">
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* 内容 */}
        <div className="px-6 py-4 space-y-5">
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="w-6 h-6 animate-spin text-blue-500" />
            </div>
          ) : detail ? (
            <>
              {/* 状态行 */}
              <div className="flex flex-wrap items-center gap-3 text-sm">
                <span className={`px-2 py-1 rounded-full text-xs font-medium ${STATE_COLORS[detail.state] ?? ""}`}>
                  {t.aiops.state[detail.state as keyof typeof t.aiops.state] ?? detail.state}
                </span>
                <RiskBadge level={detail.severity} size="md" />
                <span className="text-muted">
                  {t.aiops.duration}: <span className="text-default font-medium">{formatDuration(detail.durationS, t.aiops.minutes)}</span>
                </span>
                <span className="text-muted">
                  {t.aiops.peakRisk}: <span className="text-default font-mono font-medium">{detail.peakRisk.toFixed(1)}</span>
                </span>
              </div>

              {/* 根因卡片 */}
              <RootCauseCard entity={rootCauseEntity} />

              {/* 受影响实体 */}
              {detail.entities.length > 0 && (
                <div>
                  <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">
                    {t.aiops.affectedEntities}
                  </h4>
                  <div className="space-y-1.5">
                    {detail.entities.map((e) => (
                      <div key={e.entityKey} className="flex items-center gap-3 text-sm">
                        <EntityLink entityKey={e.entityKey} />
                        <span className={`text-xs font-medium ${ROLE_COLORS[e.role] ?? "text-muted"}`}>{e.role}</span>
                        <span className="text-xs text-muted">
                          R={formatRiskScore(e.rFinal)}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* 时间线 */}
              <TimelineView timeline={detail.timeline} />

              {/* 历史模式（G2）：这个问题是不是老毛病 */}
              <div>
                <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">
                  {t.aiops.patternTitle}
                </h4>
                {pattern === undefined ? (
                  <div className="text-xs text-muted opacity-60">…</div>
                ) : pattern === null || pattern.patternCount <= 1 ? (
                  <div className="text-xs text-muted">{t.aiops.patternFirst}</div>
                ) : (
                  <div className="flex flex-wrap gap-x-5 gap-y-1 text-xs">
                    <span className="text-muted">
                      {t.aiops.patternRecur}:{" "}
                      <span className="text-default font-medium tabular-nums">{pattern.patternCount}</span>
                    </span>
                    <span className="text-muted">
                      {t.aiops.patternAvgDuration}:{" "}
                      <span className="text-default tabular-nums">
                        {formatDuration(pattern.avgDuration, t.aiops.minutes)}
                      </span>
                    </span>
                    <span className="text-muted">
                      {t.aiops.patternLast}:{" "}
                      <span className="text-default">{new Date(pattern.lastOccurrence).toLocaleString()}</span>
                    </span>
                  </div>
                )}
              </div>

              {/* AI 分析报告 */}
              <AIAnalysisSection incidentId={detail.id} incidentState={detail.state} />
            </>
          ) : (
            <div className="py-8 text-center text-sm text-muted">{t.aiops.noData}</div>
          )}
        </div>
      </div>
    </div>
  );
}

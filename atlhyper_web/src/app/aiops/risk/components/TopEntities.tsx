"use client";

import { useState, useCallback } from "react";
import { ChevronDown, ChevronRight, Loader2 } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { RiskBadge } from "@/components/aiops/RiskBadge";
import { EntityLink } from "@/components/aiops/EntityLink";
import { getEntityRiskDetail, getEntityBaseline } from "@/api/aiops";
import { BaselineCard } from "@/components/aiops/BaselineCard";
import { CausalTreeNodeView } from "@/components/aiops/CausalTreeNodeView";
import type { EntityRisk, EntityRiskDetail, EntityBaseline } from "@/api/aiops";

const LIMIT_OPTIONS = [20, 50, 100] as const;

interface TopEntitiesProps {
  entities: EntityRisk[];
  clusterId: string;
  limit: number;
  onLimitChange: (limit: number) => void;
}

function formatTimeAgo(ts: number, t: { minutesAgo: string; hoursAgo: string; justNow: string; noAnomaly: string }): string {
  if (!ts) return t.noAnomaly;
  const diffMs = Date.now() - ts * 1000; // ts 是 Unix 秒，转换为毫秒
  const diffMin = Math.floor(diffMs / 60000);
  if (diffMin < 1) return t.justNow;
  if (diffMin < 60) return `${diffMin} ${t.minutesAgo}`;
  return `${Math.floor(diffMin / 60)} ${t.hoursAgo}`;
}

export function TopEntities({ entities, clusterId, limit, onLimitChange }: TopEntitiesProps) {
  const { t } = useI18n();
  const [expandedKey, setExpandedKey] = useState<string | null>(null);
  const [detailCache, setDetailCache] = useState<Record<string, EntityRiskDetail>>({});
  const [baselineCache, setBaselineCache] = useState<Record<string, EntityBaseline | null>>({});
  const [loadingKey, setLoadingKey] = useState<string | null>(null);

  const handleToggle = useCallback(
    async (key: string) => {
      if (expandedKey === key) {
        setExpandedKey(null);
        return;
      }
      setExpandedKey(key);

      if (!detailCache[key]) {
        setLoadingKey(key);
        // 两者并行：基线失败不应挡住风险详情，反之亦然
        const [detailRes, baselineRes] = await Promise.allSettled([
          getEntityRiskDetail(clusterId, key),
          getEntityBaseline(clusterId, key),
        ]);
        if (detailRes.status === "fulfilled") {
          setDetailCache((prev) => ({ ...prev, [key]: detailRes.value }));
        } else {
          console.error("Failed to load entity detail:", detailRes.reason);
        }
        setBaselineCache((prev) => ({
          ...prev,
          [key]: baselineRes.status === "fulfilled" ? baselineRes.value : null,
        }));
        setLoadingKey(null);
      }
    },
    [expandedKey, clusterId, detailCache]
  );

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="px-5 py-3 border-b border-[var(--border-color)] flex items-center justify-between">
        <h3 className="text-sm font-semibold text-default">{t.aiops.topRiskEntities}</h3>
        <div className="flex items-center gap-1">
          {LIMIT_OPTIONS.map((n) => (
            <button
              key={n}
              onClick={() => onLimitChange(n)}
              className={`px-2 py-0.5 text-xs rounded transition-colors ${
                limit === n
                  ? "bg-blue-500/15 text-blue-500 font-medium"
                  : "text-muted hover:text-default"
              }`}
            >
              {n}
            </button>
          ))}
        </div>
      </div>

      {/* 表头 */}
      <div className="grid grid-cols-[2fr_80px_80px_80px_80px_100px] gap-2 px-5 py-2 text-[10px] text-muted uppercase tracking-wider border-b border-[var(--border-color)]/50">
        <span>{t.aiops.entityKey}</span>
        <span>{t.aiops.entityType}</span>
        <span>{t.aiops.rLocal}</span>
        <span>{t.aiops.rFinal}</span>
        <span>{t.aiops.riskLevelLabel}</span>
        <span>{t.aiops.firstAnomaly}</span>
      </div>

      {/* 行 */}
      {entities.length === 0 ? (
        <div className="px-5 py-8 text-center text-sm text-muted">{t.aiops.noData}</div>
      ) : (
        entities.map((entity) => {
          const isExpanded = expandedKey === entity.entityKey;
          const detail = detailCache[entity.entityKey];
          const isLoading = loadingKey === entity.entityKey;

          return (
            <div key={entity.entityKey}>
              <button
                onClick={() => handleToggle(entity.entityKey)}
                className="w-full grid grid-cols-[2fr_80px_80px_80px_80px_100px] gap-2 px-5 py-2.5 text-sm hover:bg-[var(--hover-bg)] transition-colors items-center"
              >
                <div className="flex items-center gap-2 min-w-0">
                  {isExpanded ? (
                    <ChevronDown className="w-3.5 h-3.5 text-muted flex-shrink-0" />
                  ) : (
                    <ChevronRight className="w-3.5 h-3.5 text-muted flex-shrink-0" />
                  )}
                  <EntityLink entityKey={entity.entityKey} />
                </div>
                <span className="text-xs text-muted">{entity.entityType}</span>
                <span className="text-xs font-mono text-default">{Math.round(entity.rLocal)}</span>
                <span className="text-xs font-mono font-semibold text-default">{Math.round(entity.rFinal)}</span>
                <RiskBadge level={entity.riskLevel} />
                <span className="text-xs text-muted">{formatTimeAgo(entity.firstAnomaly, t.aiops)}</span>
              </button>

              {/* 展开详情 */}
              {isExpanded && (
                <div className="px-5 pb-4 pt-1 bg-[var(--background)]/50 border-b border-[var(--border-color)]/30">
                  {isLoading ? (
                    <div className="flex items-center justify-center py-4">
                      <Loader2 className="w-5 h-5 animate-spin text-blue-500" />
                    </div>
                  ) : detail ? (
                    <div className="space-y-4">
                    {detail.metrics.length === 0 && detail.causalChain.length === 0 && (!detail.causalTree || detail.causalTree.length === 0) ? (
                      <div className="py-2 text-xs text-muted">
                        {t.aiops.noAnomalyDetail}
                      </div>
                    ) : (
                    <div className="space-y-3">
                      {/* 异常指标 */}
                      {detail.metrics.length > 0 && (
                        <div>
                          <h4 className="text-xs font-medium text-muted mb-2">{t.aiops.metricName}</h4>
                          <div className="space-y-1">
                            {detail.metrics.map((m, i) => (
                              <div key={i} className="flex items-center gap-3 text-xs">
                                <span className="font-mono text-default w-40 truncate">{m.metricName}</span>
                                <span className="text-muted">
                                  {t.aiops.currentValue}: <span className="text-default">{m.currentValue.toFixed(2)}</span>
                                </span>
                                <span className="text-muted">
                                  {t.aiops.baseline}: <span className="text-default">{m.baseline.toFixed(2)}</span>
                                </span>
                                <span className="text-muted">
                                  {t.aiops.deviation}: <span className="text-default">{m.deviation.toFixed(1)}σ</span>
                                </span>
                                {m.isAnomaly && (
                                  <span className="text-red-500 text-[10px] font-medium">{t.aiops.isAnomaly}</span>
                                )}
                              </div>
                            ))}
                          </div>
                        </div>
                      )}

                      {/* 因果树（优先）或因果链（降级） */}
                      {detail.causalTree && detail.causalTree.length > 0 ? (
                        <div>
                          <h4 className="text-xs font-medium text-muted mb-2">{t.aiops.causalTree}</h4>
                          <div className="space-y-1">
                            {detail.causalTree.map((node, i) => (
                              <CausalTreeNodeView key={i} node={node} depth={0} t={t.aiops} />
                            ))}
                          </div>
                        </div>
                      ) : detail.causalChain.length > 0 ? (
                        <div>
                          <h4 className="text-xs font-medium text-muted mb-2">{t.aiops.causalChain}</h4>
                          <div className="space-y-1">
                            {detail.causalChain.map((c, i) => (
                              <div key={i} className="flex items-center gap-2 text-xs">
                                <span className="text-muted">{i + 1}.</span>
                                <EntityLink entityKey={c.entityKey} showType={false} />
                                <span className="font-mono text-default">{c.metricName}</span>
                                <span className="text-muted">{c.deviation.toFixed(1)}σ</span>
                              </div>
                            ))}
                          </div>
                        </div>
                      ) : null}
                    </div>
                    )}

                    {/* 基线：无论是否有异常都展示 —— 健康实体此前展开只有一行
                        「暂无异常详情」，基线才是这里真正有信息量的内容 */}
                    <BaselineCard baseline={baselineCache[entity.entityKey] ?? null} />
                    </div>
                  ) : null}
                </div>
              )}
            </div>
          );
        })
      )}
    </div>
  );
}

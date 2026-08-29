"use client";

import { useState, useEffect } from "react";
import { Loader2 } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { RiskBadge } from "@/components/aiops/RiskBadge";
import { EntityLink } from "@/components/aiops/EntityLink";
import { CausalTreeNodeView } from "@/components/aiops/CausalTreeNodeView";
import { getEntityRiskDetail, getGraphTrace } from "@/api/aiops";
import type { EntityRiskDetail, GraphTraceResult } from "@/api/aiops";
import { formatRiskScore } from "@/lib/risk";

interface NodeDetailProps {
  entityKey: string;
  clusterId: string;
}

export function NodeDetail({ entityKey, clusterId }: NodeDetailProps) {
  const { t } = useI18n();
  const [detail, setDetail] = useState<EntityRiskDetail | null>(null);
  const [loading, setLoading] = useState(true);
  // 依赖追踪（G3 接线）：与风险详情并行拉取，任一失败不影响另一个
  const [upstream, setUpstream] = useState<GraphTraceResult | null>(null);
  const [downstream, setDownstream] = useState<GraphTraceResult | null>(null);

  useEffect(() => {
    setLoading(true);
    setUpstream(null);
    setDownstream(null);
    getEntityRiskDetail(clusterId, entityKey)
      .then(setDetail)
      .catch((err) => console.error("Failed to load entity detail:", err))
      .finally(() => setLoading(false));
    getGraphTrace(clusterId, entityKey, "upstream").then(setUpstream).catch(() => {});
    getGraphTrace(clusterId, entityKey, "downstream").then(setDownstream).catch(() => {});
  }, [clusterId, entityKey]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="w-5 h-5 animate-spin text-blue-500" />
      </div>
    );
  }

  if (!detail) {
    return <div className="text-sm text-muted text-center py-8">{t.aiops.noData}</div>;
  }

  return (
    <div className="space-y-4 overflow-y-auto max-h-full">
      {/* 实体信息 */}
      <div>
        <h3 className="text-sm font-bold text-default mb-2">{t.aiops.nodeDetail}</h3>
        <div className="space-y-1.5 text-xs">
          <div className="flex items-center gap-2">
            <EntityLink entityKey={detail.entityKey} />
          </div>
          <div className="text-muted">
            {t.aiops.entityType}: <span className="text-default">{detail.entityType}</span>
          </div>
          {detail.namespace && (
            <div className="text-muted">
              {t.common.namespace}: <span className="text-default">{detail.namespace}</span>
            </div>
          )}
        </div>
      </div>

      {/* 风险分数 */}
      <div className="bg-[var(--background)] rounded-lg p-3 space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-xs text-muted">{t.aiops.rLocal}</span>
          <span className="text-sm font-mono font-medium text-default">{detail.rLocal.toFixed(1)}</span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-xs text-muted">{t.aiops.rFinal}</span>
          <span className="text-sm font-mono font-bold text-default">{formatRiskScore(detail.rFinal)}</span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-xs text-muted">{t.aiops.riskLevelLabel}</span>
          <RiskBadge level={detail.riskLevel} size="md" />
        </div>
      </div>

      {/* 指标列表 */}
      {detail.metrics?.length > 0 && (
        <div>
          <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">{t.aiops.metricName}</h4>
          <div className="space-y-2">
            {detail.metrics.map((m, i) => (
              <div key={i} className="bg-[var(--background)] rounded-lg p-2.5 text-xs space-y-1">
                <div className="flex items-center justify-between">
                  <span className="font-mono text-default">{m.metricName}</span>
                  <span className="flex items-center gap-1">
                    {m.isAnomaly && (
                      <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-red-500/15 text-red-500 font-medium">
                        {t.aiops.isAnomaly}
                      </span>
                    )}
                    {m.absoluteBreach && (
                      <span
                        className="text-[10px] px-1.5 py-0.5 rounded-full bg-red-600/20 text-red-600 dark:text-red-400 font-medium"
                        title={t.aiops.absoluteBreachHint}
                      >
                        {t.aiops.absoluteBreach}
                      </span>
                    )}
                  </span>
                </div>
                <div className="flex gap-3 text-muted">
                  <span>
                    {t.aiops.currentValue}: <span className="text-default">{m.currentValue.toFixed(2)}</span>
                  </span>
                  <span>
                    {t.aiops.deviation}: <span className="text-default">{m.deviation.toFixed(1)}σ</span>
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* 因果树（优先）或因果链（降级） */}
      {detail.causalTree && detail.causalTree.length > 0 ? (
        <div>
          <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">{t.aiops.causalTree}</h4>
          <div className="space-y-1.5">
            {detail.causalTree.map((node, i) => (
              <CausalTreeNodeView key={i} node={node} depth={0} t={t.aiops} />
            ))}
          </div>
        </div>
      ) : detail.causalChain?.length > 0 ? (
        <div>
          <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">{t.aiops.causalChain}</h4>
          <div className="space-y-1.5">
            {detail.causalChain.map((c, i) => (
              <div key={i} className="flex items-center gap-2 text-xs">
                <span className="text-muted w-4 text-right">{i + 1}.</span>
                <EntityLink entityKey={c.entityKey} showType={false} />
                <span className="font-mono text-muted">{c.metricName}</span>
                <span className="text-default">{c.deviation.toFixed(1)}σ</span>
              </div>
            ))}
          </div>
        </div>
      ) : null}

      {/* 依赖追踪（G3）：沿依赖图实际遍历的上下游，区别于「传播路径」
          （后者是风险传播的贡献度计算，前者是拓扑结构本身） */}
      <div>
        <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">{t.aiops.depTrace}</h4>
        {[
          { label: t.aiops.depUpstream, res: upstream },
          { label: t.aiops.depDownstream, res: downstream },
        ].map(({ label, res }) => {
          // 遍历结果包含起点自身，展示时剔除
          const others = res?.nodes.filter((n) => n.key !== entityKey) ?? null;
          return (
            <div key={label} className="mb-2">
              <div className="text-[10px] text-muted mb-1">{label}</div>
              {others === null ? (
                <div className="text-xs text-muted opacity-60">…</div>
              ) : others.length === 0 ? (
                <div className="text-xs text-muted opacity-60">{t.aiops.depNone}</div>
              ) : (
                <div className="space-y-1">
                  {others.map((n) => (
                    <div key={n.key} className="text-xs">
                      <EntityLink entityKey={n.key} />
                    </div>
                  ))}
                </div>
              )}
            </div>
          );
        })}
      </div>

      {/* 传播路径 */}
      {detail.propagation?.length > 0 && (
        <div>
          <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">{t.aiops.propagation}</h4>
          <div className="space-y-1.5">
            {detail.propagation.map((p, i) => (
              <div key={i} className="flex items-center gap-2 text-xs">
                <EntityLink entityKey={p.from} showType={false} />
                <span className="text-muted">→</span>
                <EntityLink entityKey={p.to} showType={false} />
                <span className="text-muted">({(p.contribution * 100).toFixed(0)}%)</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

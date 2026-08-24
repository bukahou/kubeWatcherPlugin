"use client";

import { useState, useEffect, useCallback } from "react";
import { Activity, BarChart3, Settings2 } from "lucide-react";
import { HistoryChart } from "./HistoryChart";
import { ErrorBudgetBurnChart } from "./ErrorBudgetBurnChart";
import { BurnRateTable } from "./BurnRateTable";
import { GoodBadCount } from "./GoodBadCount";
import { LatencyTab } from "./LatencyTab";
import { SLOTargetModal } from "./SLOTargetModal";
import { getSLODomainHistory, getSLOLatencyDistribution } from "@/api/slo";
import type { DomainSLOV2, SLOHistoryPoint, LatencyDistributionResponse } from "@/types/slo";
import type { DomainDetailTranslations } from "./detail-translations";

/**
 * 域名 SLO 详情 —— 清单表行内展开的内容
 *
 * 只放清单表里没有的东西。域名、目标、当前 SLI、预算余量、燃烧率概要、P95、状态
 * 都在清单表那一行上了，这里再列一遍是三份重复（改版前正是如此）。
 *
 * 两个 tab 按「排查动作」而非「数据类型」划分：
 *   预算 —— 还能烧多久、烧得多快、错了多少个（SLO 的本职）
 *   延迟 —— 慢在哪里（分位数、方法、状态码）
 */
export function DomainDetail({
  domain,
  timeRange,
  clusterId,
  onRefresh,
  t,
}: {
  domain: DomainSLOV2;
  timeRange: string;
  clusterId: string;
  onRefresh: () => void;
  t: DomainDetailTranslations;
}) {
  const [activeTab, setActiveTab] = useState<"budget" | "latency">("budget");
  const [showTargetModal, setShowTargetModal] = useState(false);
  const [history, setHistory] = useState<SLOHistoryPoint[]>([]);
  const [latencyData, setLatencyData] = useState<LatencyDistributionResponse | null>(null);

  const targets = {
    availability: domain.target?.availability ?? 99,
    p95Latency: domain.target?.p95Latency ?? 300,
  };

  // 时间范围变了，缓存的数据作废
  useEffect(() => {
    setLatencyData(null);
    setHistory([]);
  }, [timeRange]);

  const loadHistory = useCallback(async () => {
    if (history.length > 0) return;
    try {
      const res = await getSLODomainHistory({ clusterId, host: domain.domain, timeRange });
      if (res.data?.history) setHistory(res.data.history);
    } catch (err) {
      console.warn("[SLO] 历史数据获取失败:", err);
    }
  }, [clusterId, domain.domain, timeRange, history.length]);

  const loadLatency = useCallback(async () => {
    if (latencyData) return;
    try {
      const res = await getSLOLatencyDistribution({ clusterId, domain: domain.domain, timeRange });
      if (res.data) setLatencyData(res.data);
    } catch {
      // 延迟分布可能还没有数据，静默降级为空状态
    }
  }, [clusterId, domain.domain, timeRange, latencyData]);

  // 按需加载：只在切到对应 tab 时才请求
  useEffect(() => {
    if (activeTab === "budget") loadHistory();
    if (activeTab === "latency") loadLatency();
  }, [activeTab, loadHistory, loadLatency]);

  const budgetHistory = history.map((h) => ({ timestamp: h.timestamp, errorBudget: h.errorBudget }));

  return (
    <div className="border-t border-[var(--border-color)] bg-[var(--background)]">
      {/* tab 切换 + 配置入口 */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-[var(--border-color)]">
        <div className="flex gap-1">
          {[
            { id: "budget" as const, label: t.tabBudget, icon: Activity },
            { id: "latency" as const, label: t.tabLatency, icon: BarChart3 },
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-lg transition-colors ${
                activeTab === tab.id
                  ? "bg-primary/10 text-primary"
                  : "text-muted hover:text-default hover:bg-[var(--hover-bg)]"
              }`}
            >
              <tab.icon className="w-3.5 h-3.5" />
              {tab.label}
            </button>
          ))}
        </div>
        <button
          onClick={() => setShowTargetModal(true)}
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-lg text-muted hover:text-default hover:bg-[var(--hover-bg)] transition-colors"
        >
          <Settings2 className="w-3.5 h-3.5" />
          <span className="hidden sm:inline">{t.configTarget}</span>
        </button>
      </div>

      {activeTab === "budget" && (
        <div className="p-4 space-y-4">
          {/* 数字先行：允许错几个、实际错了几个 —— 比任何图都直接 */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <GoodBadCount summary={domain.summary} budget={domain.budget} t={t} />
            <BurnRateTable burnRates={domain.budget?.burnRates ?? []} t={t} />
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <HistoryChart
              history={history}
              targets={targets}
              t={{
                p95Latency: t.p95Latency,
                errorRate: t.errorRate,
                target: t.target,
                sloTrend: t.sloTrend,
                noData: t.noData,
              }}
            />
            <ErrorBudgetBurnChart
              history={budgetHistory}
              errorBudgetRemaining={domain.budget?.remainingPct ?? domain.errorBudgetRemaining}
              t={{
                errorBudgetBurn: t.errorBudgetBurn,
                current: t.current,
                estimatedExhaust: t.estimatedExhaust,
                noData: t.noData,
              }}
            />
          </div>
        </div>
      )}

      {activeTab === "latency" && (
        <div className="p-4">
          <LatencyTab
            data={latencyData}
            timeRange={timeRange}
            t={{
              latencyDistribution: t.latencyDistribution,
              methodBreakdown: t.methodBreakdown,
              statusCodeBreakdown: t.statusCodeBreakdown,
              clearSelection: t.clearSelection,
              requests: t.requests,
              noData: t.noData,
            }}
          />
        </div>
      )}

      <SLOTargetModal
        isOpen={showTargetModal}
        onClose={() => setShowTargetModal(false)}
        domain={domain.domain}
        clusterId={clusterId}
        currentWindowDays={domain.target?.windowDays}
        onSaved={onRefresh}
        t={t}
      />
    </div>
  );
}

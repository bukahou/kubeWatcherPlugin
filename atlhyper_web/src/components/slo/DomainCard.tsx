"use client";

import { useState, useEffect, useCallback } from "react";
import {
  Settings2,
  Activity,
  Network,
  Calendar,
  BarChart3,
} from "lucide-react";
import { DomainSummaryRow } from "./DomainSummaryRow";
import { SLOTargetModal } from "./SLOTargetModal";
import { OverviewTab } from "./OverviewTab";
import { CompareTab } from "./CompareTab";
import { LatencyTab } from "./LatencyTab";
import { getSLODomainHistory, getSLOLatencyDistribution } from "@/datasource/slo";
import type { DomainSLOV2, LatencyDistributionResponse, SLOHistoryPoint } from "@/types/slo";

type TimeRange = "1d" | "7d" | "30d";

export interface DomainCardTranslations {
  services: string;
  availability: string;
  p95Latency: string;
  p99Latency: string;
  errorRate: string;
  errorBudget: string;
  throughput: string;
  // Tabs
  tabOverview: string;
  tabCompare: string;
  tabLatency: string;
  configTarget: string;
  // Overview tab
  totalRequests: string;
  target: string;
  sloTrend: string;
  errorBudgetBurn: string;
  current: string;
  serviceTopology: string;
  service: string;
  rps: string;
  mtls: string;
  status: string;
  healthy: string;
  warning: string;
  critical: string;
  unknown: string;
  inbound: string;
  outbound: string;
  noCallData: string;
  callRelation: string;
  p50Latency: string;
  avgLatency: string;
  // Compare tab
  currentVsPrevious: string;
  previousPeriod: string;
  // Modal
  configSloTarget: string;
  targetDomain: string;
  selectPeriod: string;
  day: string;
  week: string;
  month: string;
  targetAvailability: string;
  targetAvailabilityHint: string;
  targetP95: string;
  targetP95Hint: string;
  errorRateThreshold: string;
  errorRateAutoCalc: string;
  cancel: string;
  save: string;
  saving: string;
  estimatedExhaust: string;
  // Mesh detail + Latency tab (shared)
  statusCodeBreakdown: string;
  latencyDistribution: string;
  requests: string;
  loading: string;
  methodBreakdown: string;
  clearSelection: string;
}

export function DomainCard({ domain, expanded, onToggle, timeRange, clusterId, onRefresh, t }: {
  domain: DomainSLOV2;
  expanded: boolean;
  onToggle: () => void;
  timeRange: TimeRange;
  clusterId: string;
  onRefresh: () => void;
  t: DomainCardTranslations;
}) {
  const [activeTab, setActiveTab] = useState<"overview" | "compare" | "latency">("overview");
  const [showTargetModal, setShowTargetModal] = useState(false);
  const [history, setHistory] = useState<SLOHistoryPoint[]>([]);
  const [latencyData, setLatencyData] = useState<LatencyDistributionResponse | null>(null);

  const availability = domain.summary?.availability ?? 0;
  const p95Latency = domain.summary?.p95Latency ?? 0;
  const errorRate = domain.summary?.errorRate ?? 0;
  const rps = domain.summary?.requestsPerSec ?? 0;

  // 使用后端聚合的 previous 数据
  const prevAvailability = domain.previous?.availability ?? availability;
  const prevP95Latency = domain.previous?.p95Latency ?? p95Latency;
  const prevErrorRate = domain.previous?.errorRate ?? errorRate;

  const trend = availability > prevAvailability ? "up" : availability < prevAvailability ? "down" : "stable";
  const domainTargets = domain.targets?.[timeRange] || domain.targets?.["1d"] || { availability: 95, p95Latency: 300 };
  const targets = { availability: domainTargets.availability, p95Latency: domainTargets.p95Latency };

  // Reset cached data when timeRange changes
  useEffect(() => {
    setLatencyData(null);
    setHistory([]);
  }, [timeRange]);

  // Lazy-load latency data when latency tab is opened
  const loadLatencyData = useCallback(async () => {
    if (latencyData) return;
    try {
      const res = await getSLOLatencyDistribution({ clusterId, domain: domain.domain, timeRange });
      if (res.data) setLatencyData(res.data);
    } catch {
      // Latency data may not be available yet
    }
  }, [clusterId, domain.domain, timeRange, latencyData]);

  // Lazy-load history when overview tab is opened
  const loadHistory = useCallback(async () => {
    if (history.length > 0) return;
    try {
      const res = await getSLODomainHistory({ clusterId, host: domain.domain, timeRange });
      if (res.data?.history) setHistory(res.data.history);
    } catch (err) {
      console.warn("[SLO] History fetch error:", err);
    }
  }, [clusterId, domain.domain, timeRange, history.length]);

  useEffect(() => {
    if (!expanded) return;
    if (activeTab === "overview") loadHistory();
    if (activeTab === "latency") loadLatencyData();
  }, [expanded, activeTab, loadHistory, loadLatencyData]);

  return (
    <div className="border border-[var(--border-color)] rounded-xl overflow-hidden bg-card">
      <DomainSummaryRow
        domain={domain.domain}
        status={domain.status}
        tls={domain.tls}
        serviceCount={domain.services.length}
        availability={availability}
        p95Latency={p95Latency}
        errorRate={errorRate}
        totalRequests={domain.summary?.totalRequests ?? 0}
        rps={rps}
        errorBudgetRemaining={domain.errorBudgetRemaining}
        targets={targets}
        trend={trend}
        expanded={expanded}
        onToggle={onToggle}
        t={t}
      />

      {/* Expanded Details */}
      {expanded && (
        <div className="border-t border-[var(--border-color)]">
          {/* Tabs */}
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 px-3 sm:px-4 pt-3 pb-2 border-b border-[var(--border-color)]">
            <div className="flex items-center gap-1 overflow-x-auto">
              {[
                { id: "overview" as const, label: t.tabOverview, icon: Activity },
                { id: "latency" as const, label: t.tabLatency, icon: BarChart3 },
                { id: "compare" as const, label: t.tabCompare, icon: Calendar },
              ].map((tab) => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`flex items-center gap-1 sm:gap-1.5 px-2.5 sm:px-3 py-1.5 text-xs rounded-lg transition-colors whitespace-nowrap ${
                    activeTab === tab.id ? "bg-primary/10 text-primary" : "text-muted hover:text-default hover:bg-[var(--hover-bg)]"
                  }`}
                >
                  <tab.icon className="w-3.5 h-3.5" />
                  <span className="hidden sm:inline">{tab.label}</span>
                </button>
              ))}
            </div>
            <button
              onClick={() => setShowTargetModal(true)}
              className="flex items-center gap-1.5 px-2.5 sm:px-3 py-1.5 text-xs rounded-lg text-muted hover:text-default hover:bg-[var(--hover-bg)] transition-colors self-end sm:self-auto"
            >
              <Settings2 className="w-3.5 h-3.5" />
              <span className="hidden sm:inline">{t.configTarget}</span>
            </button>
          </div>

          <div className="bg-[var(--background)]">
            {activeTab === "overview" && (
              <div className="p-3 sm:p-4">
                <OverviewTab
                  summary={domain.summary}
                  errorBudgetRemaining={domain.errorBudgetRemaining}
                  targets={targets}
                  history={history.length > 0 ? history : undefined}
                  t={{
                    availability: t.availability,
                    p95Latency: t.p95Latency,
                    p99Latency: t.p99Latency,
                    errorRate: t.errorRate,
                    totalRequests: t.totalRequests,
                    errorBudget: t.errorBudget,
                    target: t.target,
                    throughput: t.throughput,
                    sloTrend: t.sloTrend,
                    errorBudgetBurn: t.errorBudgetBurn,
                    current: t.current,
                    estimatedExhaust: t.estimatedExhaust,
                    noData: t.noCallData,
                  }}
                />
              </div>
            )}

            {activeTab === "latency" && (
              <div className="p-3 sm:p-4">
                <LatencyTab
                  data={latencyData}
                  timeRange={timeRange}
                  t={{
                    latencyDistribution: t.latencyDistribution,
                    methodBreakdown: t.methodBreakdown,
                    statusCodeBreakdown: t.statusCodeBreakdown,
                    requests: t.totalRequests,
                    clearSelection: t.clearSelection,
                    noData: t.tabLatency,
                  }}
                />
              </div>
            )}

            {activeTab === "compare" && (
              <CompareTab
                current={{ availability, p95Latency, errorRate }}
                previous={{ availability: prevAvailability, p95Latency: prevP95Latency, errorRate: prevErrorRate }}
                t={{
                  currentVsPrevious: t.currentVsPrevious,
                  previousPeriod: t.previousPeriod,
                  availability: t.availability,
                  p95Latency: t.p95Latency,
                  errorRate: t.errorRate,
                }}
              />
            )}
          </div>
        </div>
      )}

      <SLOTargetModal
        isOpen={showTargetModal}
        onClose={() => setShowTargetModal(false)}
        domain={domain.domain}
        clusterId={clusterId}
        timeRange={timeRange}
        onSaved={onRefresh}
        t={{
          configSloTarget: t.configSloTarget,
          targetDomain: t.targetDomain,
          selectPeriod: t.selectPeriod,
          day: t.day,
          week: t.week,
          month: t.month,
          targetAvailability: t.targetAvailability,
          targetAvailabilityHint: t.targetAvailabilityHint,
          targetP95: t.targetP95,
          targetP95Hint: t.targetP95Hint,
          errorRateThreshold: t.errorRateThreshold,
          errorRateAutoCalc: t.errorRateAutoCalc,
          cancel: t.cancel,
          save: t.save,
          saving: t.saving,
        }}
      />
    </div>
  );
}

"use client";

import { useState, useMemo, useEffect, useCallback, useRef } from "react";
import { Layout } from "@/components/layout/Layout";
import { LoadingSpinner } from "@/components/common";
import { useI18n } from "@/i18n/context";
import { OTelGuard } from "@/components/observe/OTelGuard";
import { getSLODomainsV2 } from "@/datasource/slo";
import { useObserveTimeRange } from "@/hooks/useObserveTimeRange";
import { useSignalFreshness } from "@/hooks/useSignalFreshness";
import { SignalFreshnessBadge } from "@/components/observe/SignalFreshnessBadge";
import { TimeRangePicker } from "@/components/common";
import { getClusterList } from "@/api/cluster";
import { getDataSourceMode } from "@/config/data-source";
import {
  Activity,
  AlertTriangle,
  RefreshCw,
  Globe,
  Zap,
  Gauge,
  Server,
} from "lucide-react";
import { SummaryCard, formatNumber } from "@/components/slo/common";
import { SLOListTable } from "@/components/slo/SLOListTable";
import type { DomainSLOV2, SLOSummary } from "@/types/slo";



const REFRESH_INTERVAL = 30000;

export default function SLOPage() {
  return (
    <OTelGuard>
      <SLOPageContent />
    </OTelGuard>
  );
}

function SLOPageContent() {
  const { t } = useI18n();
  const sloT = t.slo;
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState("");
  const [domains, setDomains] = useState<DomainSLOV2[]>([]);
  const [summary, setSummary] = useState<SLOSummary | null>(null);
  const [clusterId, setClusterId] = useState("");
  const [expandedId, setExpandedId] = useState<string | null>(null);
  // 全局时间轴 + 贴合到 SLO 的五个预聚合窗口。
  // SLO 查不了任意范围（窗口是 Agent 预先算好缓存的），贴合后必须告诉用户实际用了哪个窗口。
  const {
    selection: timeSelection,
    setSelection: setTimeSelection,
    sloWindow: timeRange,
    degraded,
    degradeTo,
  } = useObserveTimeRange("sloWindows");
  // SLO 的数据源是 ingress 指标，跟着 metrics 的新鲜度走
  const freshness = useSignalFreshness("metrics");

  const isMountedRef = useRef(true);
  const isFirstLoadRef = useRef(true);

  const fetchData = useCallback(async (showRefreshing = false) => {
    if (showRefreshing) setRefreshing(true);
    try {
      let currentClusterId = clusterId;
      if (!currentClusterId && getDataSourceMode("slo") !== "mock") {
        const clusterRes = await getClusterList();
        const clusters = clusterRes.data?.clusters || [];
        if (clusters.length === 0) {
          if (isMountedRef.current && isFirstLoadRef.current) setError(sloT.noCluster);
          return;
        }
        currentClusterId = clusters[0].clusterId;
        setClusterId(currentClusterId);
      }
      const res = await getSLODomainsV2({ clusterId: currentClusterId, timeRange });
      if (isMountedRef.current) {
        setDomains(res.data?.domains || []);
        setSummary(res.data?.summary || null);
        setError("");
      }
    } catch (err) {
      if (isMountedRef.current) {
        console.warn("[SLO] Fetch error:", err);
        if (isFirstLoadRef.current) setError(err instanceof Error ? err.message : sloT.loadFailed);
      }
    } finally {
      if (isMountedRef.current) {
        setLoading(false);
        setRefreshing(false);
        isFirstLoadRef.current = false;
      }
    }
  }, [clusterId, timeRange]);

  useEffect(() => {
    isMountedRef.current = true;
    fetchData();
    const intervalId = setInterval(() => fetchData(true), REFRESH_INTERVAL);
    return () => { isMountedRef.current = false; clearInterval(intervalId); };
  }, [fetchData]);

  const handleRefresh = () => fetchData(true);

  const summaryData = useMemo(() => {
    const avgP95 = domains.length > 0
      ? domains.reduce((sum, d) => sum + (d.summary?.p95Latency || 0), 0) / domains.length
      : 0;
    if (summary) {
      return {
        totalServices: summary.totalServices || 0,
        totalDomains: summary.totalDomains,
        healthyCount: summary.healthyCount,
        warningCount: summary.warningCount,
        criticalCount: summary.criticalCount,
        totalRPS: summary.totalRps,
        avgAvailability: summary.avgAvailability,
        avgP95,
      };
    }
    const totalDomains = domains.length;
    const healthyCount = domains.filter(d => d.status === "healthy").length;
    const warningCount = domains.filter(d => d.status === "warning").length;
    const criticalCount = domains.filter(d => d.status === "critical").length;
    const totalRPS = domains.reduce((sum, d) => sum + (d.summary?.requestsPerSec || 0), 0);
    const avgAvailability = totalDomains > 0 ? domains.reduce((sum, d) => sum + (d.summary?.availability || 0), 0) / totalDomains : 0;
    const totalServices = domains.reduce((sum, d) => sum + d.services.length, 0);
    return { totalServices, totalDomains, healthyCount, warningCount, criticalCount, totalRPS, avgAvailability, avgP95 };
  }, [domains, summary]);

  // DomainDetail 及其子组件的翻译。清单表那一行已经承载了域名/目标/SLI/预算/燃烧率概要，
  // 这里只需要详情独有的部分。
  const domainDetailT = useMemo(() => ({
    tabBudget: sloT.tabOverview,
    tabLatency: sloT.tabLatency,
    configTarget: sloT.configTarget,
    // 燃烧率表
    burnRate: sloT.burnRate,
    burnRateHint: sloT.burnRateHint,
    window: sloT.window,
    rate: sloT.rate,
    threshold: sloT.threshold,
    status: sloT.sloStatus,
    good: sloT.healthy,
    warn: sloT.warning,
    crit: sloT.critical,
    // Good/Bad 计数
    eventCount: sloT.eventCount,
    bad: sloT.badEvents,
    allowed: sloT.allowedEvents,
    overspent: sloT.overspent,
    // 图表
    p95Latency: sloT.p95Latency,
    errorRate: sloT.errorRate,
    target: sloT.target,
    sloTrend: sloT.sloTrend,
    errorBudgetBurn: sloT.errorBudgetBurn,
    current: sloT.current,
    estimatedExhaust: sloT.estimatedExhaust,
    noData: sloT.noData,
    // 延迟 tab
    latencyDistribution: sloT.latencyDistribution,
    methodBreakdown: sloT.methodBreakdown,
    statusCodeBreakdown: sloT.statusCodeBreakdown,
    clearSelection: sloT.clearSelection,
    requests: sloT.requests,
    // 目标配置
    configSloTarget: sloT.configSloTarget,
    targetDomain: sloT.targetDomain,
    sloWindow: sloT.sloWindow,
    sloWindowHint: sloT.sloWindowHint,
    days: sloT.days,
    targetAvailability: sloT.targetAvailability,
    targetAvailabilityHint: sloT.targetAvailabilityHint,
    targetP95: sloT.targetP95,
    targetP95Hint: sloT.targetP95Hint,
    errorRateThreshold: sloT.errorRateThreshold,
    errorRateAutoCalc: sloT.errorRateAutoCalc,
    cancel: sloT.cancel,
    save: sloT.save,
    saving: sloT.saving,
  }), [sloT]);

  if (loading) {
    return <Layout><LoadingSpinner /></Layout>;
  }

  return (
    <Layout>
      <div className="-m-6 min-h-[calc(100vh-3.5rem)] bg-[var(--background)]">
        {/* Header */}
        <div className="px-4 sm:px-6 py-4 border-b border-[var(--border-color)] bg-card">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-xl bg-gradient-to-br from-violet-100 to-indigo-100 dark:from-violet-900/30 dark:to-indigo-900/30">
                <Activity className="w-5 h-5 sm:w-6 sm:h-6 text-violet-600 dark:text-violet-400" />
              </div>
              <div>
                <h1 className="text-base sm:text-lg font-semibold text-default">{sloT.pageTitle}</h1>
                <p className="text-xs text-muted hidden sm:block">{sloT.pageDescription}</p>
              </div>
            </div>
            <div className="flex items-center gap-2 sm:gap-3 self-end sm:self-auto">
              <div className="flex items-center gap-2">
                <SignalFreshnessBadge item={freshness} />
                <TimeRangePicker value={timeSelection} onChange={setTimeSelection} t={sloT} />
                {degraded && (
                  <span className="text-[10px] text-amber-500 whitespace-nowrap" title={sloT.windowDegradedHint}>
                    {sloT.windowDegraded.replace("{window}", degradeTo ?? "")}
                  </span>
                )}
              </div>
              <button onClick={handleRefresh} disabled={refreshing}
                className="p-2.5 sm:p-2 rounded-lg hover:bg-[var(--hover-bg)] text-muted hover:text-default transition-colors disabled:opacity-50">
                <RefreshCw className={`w-4 h-4 ${refreshing ? "animate-spin" : ""}`} />
              </button>
            </div>
          </div>
        </div>

        <div className="p-4 sm:p-6 space-y-4 sm:space-y-6">
          {/* Error */}
          {error && domains.length === 0 && (
            <div className="text-center py-12 bg-card rounded-xl border border-[var(--border-color)]">
              <AlertTriangle className="w-12 h-12 mx-auto mb-3 text-red-500" />
              <p className="text-red-500">{error}</p>
            </div>
          )}

          {/* Empty */}
          {/* Summary Cards — 始终显示 */}
          {!loading && !error && (
            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3 sm:gap-4">
              <SummaryCard icon={Server} label={sloT.totalServices} value={summaryData.totalServices.toString()} subValue={sloT.ingressBackends} color="bg-blue-500/10 text-blue-500" />
              <SummaryCard icon={Globe} label={sloT.monitoredDomains} value={summaryData.totalDomains.toString()} subValue={`${summaryData.healthyCount} ${sloT.healthy}`} color="bg-violet-500/10 text-violet-500" />
              <SummaryCard icon={Activity} label={sloT.avgAvailability} value={`${summaryData.avgAvailability.toFixed(2)}%`} color="bg-emerald-500/10 text-emerald-500" />
              <SummaryCard icon={Gauge} label={sloT.avgP95} value={`${Math.round(summaryData.avgP95)}ms`} color="bg-cyan-500/10 text-cyan-500" />
              <SummaryCard icon={Zap} label={sloT.totalRPS} value={formatNumber(summaryData.totalRPS)} subValue={sloT.reqPerSec} color="bg-amber-500/10 text-amber-500" />
              <SummaryCard icon={AlertTriangle} label={sloT.alertCount} value={(summaryData.warningCount + summaryData.criticalCount).toString()}
                subValue={`${summaryData.criticalCount} ${sloT.severe}`}
                color={summaryData.criticalCount > 0 ? "bg-red-500/10 text-red-500" : "bg-amber-500/10 text-amber-500"} />
            </div>
          )}

          {/* Empty — 无域名数据 */}
          {!error && !loading && domains.length === 0 && (
            <div className="text-center py-12 bg-card rounded-xl border border-[var(--border-color)]">
              <Server className="w-12 h-12 mx-auto mb-3 text-muted opacity-50" />
              <p className="text-default font-medium mb-2">{sloT.noData}</p>
              <p className="text-sm text-muted">{sloT.noDataHint}</p>
            </div>
          )}

          {domains.length > 0 && (
            <>

              {/* SLO 清单表 —— 一行一个域名，一眼看出谁在烧预算 */}
              <div>
                <div className="flex items-baseline justify-between mb-2">
                  <h2 className="text-sm font-semibold text-default">
                    {sloT.sloListTitle}
                    <span className="ml-2 text-xs font-normal text-muted">({summaryData.totalDomains})</span>
                  </h2>
                  <p className="text-[11px] text-muted hidden lg:block">{sloT.sloListHint}</p>
                </div>
                <SLOListTable
                  domains={domains}
                  expandedId={expandedId}
                  onSelect={(d) => setExpandedId(expandedId === d ? null : d)}
                  timeRange={timeRange}
                  clusterId={clusterId}
                  onRefresh={handleRefresh}
                  detailT={domainDetailT}
                  t={{
                    domain: sloT.monitoredDomains,
                    viewTraces: sloT.viewTraces,
                    viewErrorLogs: sloT.viewErrorLogs,
                    target: sloT.target,
                    currentSli: sloT.currentSli,
                    errorBudget: sloT.errorBudget,
                    burnRate: sloT.burnRate,
                    p95Latency: sloT.p95Latency,
                    status: sloT.sloStatus,
                    healthy: sloT.healthy,
                    warning: sloT.warning,
                    critical: sloT.critical,
                    unknown: sloT.unknown,
                    noData: sloT.noData,
                    goodBad: sloT.goodBad,
                    exhaustIn: sloT.exhaustIn,
                    hours: sloT.hours,
                  }}
                />
              </div>

              {/* Data Source Note */}
              <div className="p-4 rounded-xl bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800">
                <div className="flex items-start gap-3">
                  <div className="p-1.5 rounded-lg bg-blue-100 dark:bg-blue-900/50">
                    <Activity className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                  </div>
                  <div className="text-sm">
                    <p className="font-medium text-blue-800 dark:text-blue-200 mb-1">{sloT.dataSourceTitle}</p>
                    <p className="text-blue-700 dark:text-blue-300 text-xs leading-relaxed">{sloT.dataSourceDesc}</p>
                  </div>
                </div>
              </div>
            </>
          )}
        </div>
      </div>
    </Layout>
  );
}

"use client";

import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { useSearchParams } from "next/navigation";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { useClusterStore } from "@/store/clusterStore";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import { useObserveTimeRange } from "@/hooks/useObserveTimeRange";
import { useSignalFreshness } from "@/hooks/useSignalFreshness";
import { toSince, toAbsoluteParams } from "@/lib/time-range";
import { Loader2, WifiOff, AlertTriangle } from "lucide-react";
import { OTelGuard } from "@/components/observe/OTelGuard";

import type { TraceSummary, TraceDetail, APMService, Topology, OperationStats } from "@/types/model/apm";
import {
  getAPMServices,
  queryTraces,
  getTraceDetail,
  getTopology,
  getOperations,
  type TimeParams,
} from "@/datasource/apm";

import { ApmPageHeader } from "./components/ApmPageHeader";
import { ServiceList } from "./components/ServiceList";
import { ServiceOverview } from "./components/ServiceOverview";
import { TraceWaterfall } from "./components/TraceWaterfall";
import { ServiceTopology } from "./components/ServiceTopology";
import { filterTraceForService } from "./components/trace-utils";
import { isErrorSpan } from "@/lib/otel";

type ViewState =
  | { level: "services" }
  | { level: "service-detail"; serviceName: string }
  | { level: "trace-detail"; serviceName: string; operationName: string; traceId: string; traceIndex: number };

export default function ApmPage() {
  return (
    <OTelGuard>
      <ApmPageContent />
    </OTelGuard>
  );
}

function ApmPageContent() {
  const { t } = useI18n();
  const ta = t.apm;
  const { currentClusterId } = useClusterStore();

  const [view, setView] = useState<ViewState>({ level: "services" });
  const [traceDetail, setTraceDetail] = useState<TraceDetail | null>(null);
  const [serviceStats, setServiceStats] = useState<APMService[]>([]);
  const [topology, setTopology] = useState<Topology | null>(null);
  const [operations, setOperations] = useState<OperationStats[]>([]);
  const [operationTraces, setOperationTraces] = useState<TraceSummary[]>([]);
  // 时间轴是全局的 —— 切到 Logs 查同一时刻的日志时不用重选
  const { selection: timeSelection, setSelection: setTimeSelection } = useObserveTimeRange("full");
  const freshness = useSignalFreshness("traces");

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);

  // 跨信号关联：URL ?trace=xxx 自动进入 trace-detail
  const searchParams = useSearchParams();
  const traceParam = searchParams.get("trace");
  const handledTraceParam = useRef<string | null>(null);

  useEffect(() => {
    if (!traceParam || !currentClusterId || traceParam === handledTraceParam.current) return;
    handledTraceParam.current = traceParam;
    getTraceDetail(traceParam, currentClusterId).then((detail) => {
      if (detail && detail.spans.length > 0) {
        const rootSpan = detail.spans[0];
        setView({
          level: "trace-detail",
          serviceName: rootSpan.serviceName,
          operationName: rootSpan.spanName,
          traceId: traceParam,
          traceIndex: 0,
        });
        setTraceDetail(detail);
        const uniqueServices = [...new Set(detail.spans.map(s => s.serviceName))];
        setOperationTraces([{
          traceId: detail.traceId,
          rootService: rootSpan.serviceName,
          rootOperation: rootSpan.spanName,
          services: uniqueServices,
          durationMs: detail.durationMs,
          spanCount: detail.spanCount,
          serviceCount: detail.serviceCount,
          hasError: isErrorSpan(rootSpan),
          timestamp: rootSpan.timestamp,
        }]);
      }
    });
  }, [traceParam, currentClusterId]);

  /** 从 timeSelection 派生 API 时间参数 */
  const timeParams = useMemo((): TimeParams => {
    const since = toSince(timeSelection);
    const abs = toAbsoluteParams(timeSelection);
    return { since, startTime: abs.startTime, endTime: abs.endTime };
  }, [timeSelection]);

  const loadData = useCallback(async (showLoading = true) => {
    if (showLoading) setIsRefreshing(true);
    try {
      const [svcStats, topo, ops] = await Promise.all([
        getAPMServices(currentClusterId, timeParams),
        getTopology(currentClusterId, timeParams),
        getOperations(currentClusterId, timeParams),
      ]);
      setServiceStats(svcStats);
      setTopology(topo);
      setOperations(ops);
      setError(null);
    } catch {
      setError(ta.loadFailed);
    } finally {
      setLoading(false);
      setIsRefreshing(false);
    }
  }, [currentClusterId, timeParams, ta.loadFailed]);

  // 静默刷新（自动刷新用，不显示 loading 状态）
  const loadDataSilent = useCallback(() => {
    loadData(false);
  }, [loadData]);

  // 服务列表 / 服务详情自动刷新，trace 瀑布图禁用（历史数据，刷新可能丢失）
  useAutoRefresh(loadDataSilent, {
    enabled: view.level !== "trace-detail",
  });

  useEffect(() => {
    if (view.level !== "trace-detail") {
      setTraceDetail(null);
      return;
    }
    getTraceDetail(view.traceId, currentClusterId).then(setTraceDetail);
  }, [view, currentClusterId]);

  const goToServices = () => setView({ level: "services" });
  const goToService = (name: string) =>
    setView({ level: "service-detail", serviceName: name });

  // 点击事务行：查该 operation 的 traces，然后进入第三层
  const goToTraceForOperation = async (serviceName: string, operation: string) => {
    const result = await queryTraces(currentClusterId, {
      service: serviceName, operation, limit: 200,
    }, timeParams);
    if (result.traces.length > 0) {
      setOperationTraces(result.traces);
      setView({
        level: "trace-detail",
        serviceName,
        operationName: operation,
        traceId: result.traces[0].traceId,
        traceIndex: 0,
      });
    }
  };

  const navigateTrace = (index: number) => {
    if (view.level === "trace-detail" && index >= 0 && index < operationTraces.length) {
      setView({
        ...view,
        traceId: operationTraces[index].traceId,
        traceIndex: index,
      });
    }
  };

  // 从子服务进入追踪详情时，裁剪 Span 树：只保留该服务的入口 Span 及其所有后代
  const focusedTrace = useMemo(() => {
    if (!traceDetail || view.level !== "trace-detail") return null;
    return filterTraceForService(traceDetail, view.serviceName);
  }, [traceDetail, view]);

  if (!currentClusterId) {
    return (
      <Layout>
        <div className="flex flex-col items-center justify-center h-96 text-center">
          <WifiOff className="w-12 h-12 mb-4 text-muted" />
          <p className="text-default font-medium mb-2">{ta.noCluster}</p>
          <p className="text-sm text-muted">{ta.noClusterDesc}</p>
        </div>
      </Layout>
    );
  }

  if (loading) {
    return (
      <Layout>
        <div className="flex items-center justify-center h-96">
          <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
        </div>
      </Layout>
    );
  }

  if (error && serviceStats.length === 0) {
    return (
      <Layout>
        <div className="flex flex-col items-center justify-center h-96 text-center">
          <AlertTriangle className="w-12 h-12 mb-4 text-yellow-500" />
          <p className="text-default font-medium mb-2">{error}</p>
          <button
            onClick={() => loadData(true)}
            className="mt-4 px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors"
          >
            {t.common.retry}
          </button>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="space-y-4 sm:space-y-5">
        <ApmPageHeader
          ta={ta}
          tc={t.common}
          view={view}
          timeSelection={timeSelection}
          freshness={freshness}
          isRefreshing={isRefreshing}
          onTimeChange={setTimeSelection}
          onRefresh={() => loadData(true)}
          onGoToServices={goToServices}
          onGoToService={goToService}
        />

        {/* View content */}
        {view.level === "services" && (
          <>
            <ServiceTopology t={ta} topology={topology ?? { nodes: [], edges: [] }} onSelectService={goToService} />
            <ServiceList
              t={ta}
              tt={t.table}
              serviceStats={serviceStats}
              onSelectService={goToService}
            />
          </>
        )}

        {view.level === "service-detail" && (
          <ServiceOverview
            t={ta}
            serviceName={view.serviceName}
            operations={operations}
            topology={topology}
            clusterId={currentClusterId}
            timeParams={timeParams}
            onSelectOperation={(op) => goToTraceForOperation(view.serviceName, op)}
            onNavigateToService={goToService}
            onSelectTrace={(traceId) => {
              setView({
                level: "trace-detail",
                serviceName: view.serviceName,
                operationName: "",
                traceId,
                traceIndex: 0,
              });
            }}
          />
        )}

        {view.level === "trace-detail" && focusedTrace && (
          <TraceWaterfall
            t={ta}
            trace={focusedTrace}
            allTraces={operationTraces}
            currentTraceIndex={view.traceIndex}
            onNavigateTrace={navigateTrace}
          />
        )}

        {view.level === "trace-detail" && !traceDetail && (
          <div className="flex items-center justify-center h-48">
            <Loader2 className="w-6 h-6 animate-spin text-blue-500" />
          </div>
        )}
      </div>
    </Layout>
  );
}

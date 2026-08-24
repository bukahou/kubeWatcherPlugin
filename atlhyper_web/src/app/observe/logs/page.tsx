"use client";

import { useState, useEffect, useCallback } from "react";
import { useSearchParams } from "next/navigation";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { useClusterStore } from "@/store/clusterStore";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import {
  RefreshCw,
  WifiOff,
  ChevronRight,
} from "lucide-react";

import { OTelGuard } from "@/components/observe/OTelGuard";
import { queryLogs, queryLogHistogram } from "@/datasource/logs";
import { TimeRangePicker } from "@/components/common";
import { toSince, toAbsoluteParams, toSpanMs } from "@/lib/time-range";
import { useObserveTimeRange } from "@/hooks/useObserveTimeRange";
import { useSignalFreshness } from "@/hooks/useSignalFreshness";
import { SignalFreshnessBadge } from "@/components/observe/SignalFreshnessBadge";

import type { LogEntry, LogQueryResult, LogHistogramBucket } from "@/types/model/log";

import { LogToolbar } from "./components/LogToolbar";
import { LogFacets } from "./components/LogFacets";
import { LogList } from "./components/LogList";
import { LogHistogram } from "./components/LogHistogram";
import { LogDetailDrawer } from "./components/LogDetail";
import { LogFilterPills } from "./components/LogFilterPills";

export default function LogsPage() {
  return (
    <OTelGuard>
      <LogsPageContent />
    </OTelGuard>
  );
}

function LogsPageContent() {
  const { t } = useI18n();
  const tl = t.logs;
  const { currentClusterId } = useClusterStore();

  // URL ?traceId= / ?spanId= params
  const searchParams = useSearchParams();
  const urlTraceId = searchParams.get("traceId") || undefined;
  const urlSpanId = searchParams.get("spanId") || undefined;

  // Filter state
  const [search, setSearch] = useState("");
  const [selectedServices, setSelectedServices] = useState<string[]>([]);
  const [selectedSeverities, setSelectedSeverities] = useState<string[]>([]);
  const [selectedScopes, setSelectedScopes] = useState<string[]>([]);
  const PAGE_SIZE = 50;
  const [page, setPage] = useState(1);

  // Facets panel collapse
  const [facetsCollapsed, setFacetsCollapsed] = useState(false);

  // Histogram brush time range
  const [brushTimeRange, setBrushTimeRange] = useState<[number, number] | null>(null);

  // Log detail drawer
  const [selectedEntry, setSelectedEntry] = useState<LogEntry | null>(null);
  const [selectedIdx, setSelectedIdx] = useState<number | null>(null);

  const handleSelectEntry = (entry: LogEntry, idx: number) => {
    if (selectedIdx === idx) {
      setSelectedEntry(null);
      setSelectedIdx(null);
    } else {
      setSelectedEntry(entry);
      setSelectedIdx(idx);
    }
  };
  const handleCloseDrawer = () => {
    setSelectedEntry(null);
    setSelectedIdx(null);
  };

  // Time range selection
  // 时间轴是全局的 —— 从 APM 切过来时保持同一段时间，不用重选
  const { selection: timeSelection, setSelection: setTimeSelection } = useObserveTimeRange("full");
  const freshness = useSignalFreshness("logs");

  // Reset page when any filter changes (including brush)
  useEffect(() => {
    setPage(1);
  }, [search, selectedServices, selectedSeverities, selectedScopes, timeSelection, brushTimeRange]);

  // Clear brush when non-brush filters change
  useEffect(() => {
    setBrushTimeRange(null);
  }, [search, selectedServices, selectedSeverities, selectedScopes, timeSelection]);

  // Histogram: independent request
  const [histogramData, setHistogramData] = useState<LogHistogramBucket[]>([]);
  const [histogramIntervalMs, setHistogramIntervalMs] = useState(0);

  const loadHistogram = useCallback(async () => {
    const since = toSince(timeSelection);
    const abs = toAbsoluteParams(timeSelection);
    const data = await queryLogHistogram({
      clusterId: currentClusterId,
      search,
      services: selectedServices,
      severities: selectedSeverities,
      scopes: selectedScopes,
      since: since,
      startTime: abs.startTime,
      endTime: abs.endTime,
    });
    setHistogramData(data.buckets);
    setHistogramIntervalMs(data.intervalMs);
  }, [currentClusterId, search, selectedServices, selectedSeverities, selectedScopes, timeSelection]);

  useEffect(() => {
    loadHistogram();
  }, [loadHistogram]);

  // Query logs
  const emptyResult: LogQueryResult = { logs: [], total: 0, facets: { services: [], severities: [], scopes: [] } };
  const [result, setResult] = useState<LogQueryResult>(emptyResult);

  const loadLogs = useCallback(async () => {
    const since = toSince(timeSelection);
    const abs = toAbsoluteParams(timeSelection);
    const brushActive = brushTimeRange !== null;

    const data = await queryLogs({
      clusterId: currentClusterId,
      search,
      services: selectedServices,
      severities: selectedSeverities,
      scopes: selectedScopes,
      since,
      limit: PAGE_SIZE,
      offset: (page - 1) * PAGE_SIZE,
      startTime: brushActive ? brushTimeRange[0] : (abs.startTime ? new Date(abs.startTime).getTime() : undefined),
      endTime: brushActive ? brushTimeRange[1] + histogramIntervalMs : (abs.endTime ? new Date(abs.endTime).getTime() : undefined),
      traceId: urlTraceId,
      spanId: urlSpanId,
    });
    setResult(data);
  }, [currentClusterId, search, selectedServices, selectedSeverities, selectedScopes, timeSelection, page, brushTimeRange, histogramIntervalMs, urlTraceId, urlSpanId]);

  useEffect(() => {
    loadLogs();
  }, [loadLogs]);

  // 自动刷新（同时刷新日志列表和直方图）
  const refreshAll = useCallback(() => {
    loadLogs();
    loadHistogram();
  }, [loadLogs, loadHistogram]);

  const { refresh, intervalSeconds } = useAutoRefresh(refreshAll, {
    interval: 15000,
    immediate: false, // loadLogs/loadHistogram 已经由各自的 useEffect 触发初始加载
  });

  const handlePageChange = (p: number) => {
    setPage(p);
    setSelectedEntry(null);
    setSelectedIdx(null);
  };

  // Filter helpers
  const removeService = (v: string) => setSelectedServices((prev) => prev.filter((s) => s !== v));
  const removeSeverity = (v: string) => setSelectedSeverities((prev) => prev.filter((s) => s !== v));
  const removeScope = (v: string) => setSelectedScopes((prev) => prev.filter((s) => s !== v));
  const clearAllFilters = () => {
    setSelectedServices([]);
    setSelectedSeverities([]);
    setSelectedScopes([]);
    setBrushTimeRange(null);
  };

  // No cluster selected
  if (!currentClusterId) {
    return (
      <Layout>
        <div className="flex flex-col items-center justify-center h-96 text-center">
          <WifiOff className="w-12 h-12 mb-4 text-muted" />
          <p className="text-default font-medium mb-2">{tl.noCluster}</p>
          <p className="text-sm text-muted">{tl.noClusterDesc}</p>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="space-y-4 sm:space-y-5 overflow-hidden">
        {/* Header */}
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="text-xl sm:text-2xl font-bold text-default">{tl.pageTitle}</h1>
            <p className="text-xs text-muted mt-1">{tl.pageDescription}</p>
          </div>

          <div className="flex items-center gap-2">
            <SignalFreshnessBadge item={freshness} />
            <TimeRangePicker
              value={timeSelection}
              onChange={setTimeSelection}
              t={tl}
            />
            <span className="text-xs text-muted bg-[var(--background)] px-2 py-1 rounded">
              {intervalSeconds}s
            </span>
            <button
              onClick={refresh}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg bg-primary text-white hover:bg-primary/90 transition-colors"
            >
              <RefreshCw className="w-3.5 h-3.5" />
              {t.common.refresh}
            </button>
          </div>
        </div>

        {/* Search toolbar */}
        <LogToolbar
          search={search}
          onSearchChange={setSearch}
          page={page}
          pageSize={PAGE_SIZE}
          total={result.total}
          t={tl}
        />

        {/* Filter pills */}
        <LogFilterPills
          selectedSeverities={selectedSeverities}
          selectedServices={selectedServices}
          selectedScopes={selectedScopes}
          brushTimeRange={brushTimeRange}
          urlTraceId={urlTraceId}
          onRemoveSeverity={removeSeverity}
          onRemoveService={removeService}
          onRemoveScope={removeScope}
          onClearBrushTimeRange={() => setBrushTimeRange(null)}
          onClearAll={clearAllFilters}
          clearFiltersLabel={tl.clearFilters}
        />

        {/* Log volume histogram */}
        <LogHistogram
          data={histogramData}
          intervalMs={histogramIntervalMs}
          title={tl.logVolume}
          timeSpanMs={toSpanMs(timeSelection)}
          selectedTimeRange={brushTimeRange}
          onTimeRangeSelect={setBrushTimeRange}
        />

        {/* Main content: facets + log list */}
        <div className="flex gap-4">
          {facetsCollapsed ? (
            <button
              onClick={() => setFacetsCollapsed(false)}
              className="w-7 flex-shrink-0 flex flex-col items-center pt-1 rounded-lg border border-[var(--border-color)] bg-card hover:bg-[var(--hover-bg)] transition-colors cursor-pointer"
            >
              <ChevronRight className="w-3.5 h-3.5 text-muted" />
            </button>
          ) : (
            <LogFacets
              services={result.facets.services}
              severities={result.facets.severities}
              scopes={result.facets.scopes}
              selectedServices={selectedServices}
              selectedSeverities={selectedSeverities}
              selectedScopes={selectedScopes}
              onServicesChange={setSelectedServices}
              onSeveritiesChange={setSelectedSeverities}
              onScopesChange={setSelectedScopes}
              onCollapse={() => setFacetsCollapsed(true)}
              t={tl}
            />
          )}

          <div className="flex-1 min-w-0">
            <LogList
              logs={result.logs}
              total={result.total}
              page={page}
              pageSize={PAGE_SIZE}
              onPageChange={handlePageChange}
              onSelectEntry={handleSelectEntry}
              selectedIdx={selectedIdx}
              searchHighlight={search || undefined}
              t={tl}
            />
          </div>
        </div>
      </div>

      {/* Log detail drawer */}
      <LogDetailDrawer
        entry={selectedEntry}
        onClose={handleCloseDrawer}
        t={tl}
      />
    </Layout>
  );
}

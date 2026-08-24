"use client";

import { useState, useMemo, useCallback } from "react";
import {
  ChevronRight,
  ChevronLeft,
  ChevronsLeft,
  ChevronsRight,
  Copy,
  Check,
} from "lucide-react";
import type { TraceDetail, TraceSummary, Span } from "@/types/model/apm";
import Link from "next/link";
import { ScrollText } from "lucide-react";
import { logsLinkForTrace } from "@/lib/signal-link";
import type { ApmTranslations } from "@/types/i18n";
import { formatDurationMs, formatTimeAgo } from "@/lib/format";
import { getLatencyDistribution } from "@/datasource/apm";
import { LatencyDistribution } from "./LatencyDistribution";
import { SERVICE_COLORS, buildSpanTree, flattenTree } from "./waterfall-utils";
import { SpanRow } from "./SpanRow";
import { SpanDrawer } from "./SpanDrawer";
import { SpanLogs } from "./SpanLogs";
import { TraceMetadata } from "./TraceMetadata";

interface TraceWaterfallProps {
  t: ApmTranslations;
  trace: TraceDetail;
  allTraces: TraceSummary[];
  currentTraceIndex: number;
  onNavigateTrace: (index: number) => void;
}

export function TraceWaterfall({
  t,
  trace,
  allTraces,
  currentTraceIndex,
  onNavigateTrace,
}: TraceWaterfallProps) {
  const [selectedSpan, setSelectedSpan] = useState<Span | null>(null);
  const [collapsedSpans, setCollapsedSpans] = useState<Set<string>>(new Set());
  const [copiedId, setCopiedId] = useState(false);
  const [activeTab, setActiveTab] = useState(0);

  const serviceColorMap = useMemo(() => {
    const services = [...new Set(trace.spans.map((s) => s.serviceName))];
    const map = new Map<string, string>();
    services.forEach((svc, i) => {
      map.set(svc, SERVICE_COLORS[i % SERVICE_COLORS.length]);
    });
    return map;
  }, [trace.spans]);

  const tree = useMemo(() => buildSpanTree(trace.spans), [trace.spans]);
  const flatSpans = useMemo(
    () => flattenTree(tree, collapsedSpans),
    [tree, collapsedSpans]
  );

  // Convert ISO timestamps to ms for relative positioning
  const spanTimesMs = useMemo(() => {
    return trace.spans.map((s) => new Date(s.timestamp).getTime());
  }, [trace.spans]);

  const traceStartMs = useMemo(() => Math.min(...spanTimesMs), [spanTimesMs]);
  const traceEndMs = useMemo(() => {
    return Math.max(...trace.spans.map((s, i) => spanTimesMs[i] + s.durationMs));
  }, [trace.spans, spanTimesMs]);
  const traceDurationMs = traceEndMs - traceStartMs;

  const latencyBuckets = useMemo(
    () => getLatencyDistribution(allTraces),
    [allTraces]
  );

  const highlightBucket = useMemo(() => {
    if (allTraces.length === 0 || currentTraceIndex < 0) return undefined;
    const currentDuration = allTraces[currentTraceIndex]?.durationMs ?? 0;
    for (let i = latencyBuckets.length - 1; i >= 0; i--) {
      if (currentDuration >= latencyBuckets[i].rangeStart) return i;
    }
    return 0;
  }, [allTraces, currentTraceIndex, latencyBuckets]);

  const toggleCollapse = useCallback((spanId: string) => {
    setCollapsedSpans((prev) => {
      const next = new Set(prev);
      if (next.has(spanId)) next.delete(spanId);
      else next.add(spanId);
      return next;
    });
  }, []);

  const copyTraceId = () => {
    navigator.clipboard.writeText(trace.traceId);
    setCopiedId(true);
    setTimeout(() => setCopiedId(false), 2000);
  };

  const currentTraceSummary = allTraces[currentTraceIndex];

  const tickCount = 6;
  const ticks = Array.from({ length: tickCount }, (_, i) => (i / (tickCount - 1)) * traceDurationMs);

  return (
    <div className="space-y-4">
      {/* Latency Distribution */}
      <div className="border border-[var(--border-color)] rounded-xl p-4 bg-card">
        <LatencyDistribution
          title={t.latencyDistribution}
          totalTraces={allTraces.length}
          buckets={latencyBuckets}
          highlightBucket={highlightBucket}
        />
      </div>

      {/* Trace sample navigation */}
      <div className="flex items-center justify-between border border-[var(--border-color)] rounded-xl px-4 py-3 bg-card">
        <div className="flex items-center gap-3">
          <span className="text-sm font-medium text-default">{t.traceSample}</span>
          <div className="flex items-center gap-1">
            <button onClick={() => onNavigateTrace(0)} disabled={currentTraceIndex <= 0} className="p-1 rounded hover:bg-[var(--hover-bg)] disabled:opacity-30 transition-colors">
              <ChevronsLeft className="w-4 h-4 text-muted" />
            </button>
            <button onClick={() => onNavigateTrace(currentTraceIndex - 1)} disabled={currentTraceIndex <= 0} className="p-1 rounded hover:bg-[var(--hover-bg)] disabled:opacity-30 transition-colors">
              <ChevronLeft className="w-4 h-4 text-muted" />
            </button>
            <span className="text-sm text-default px-2 min-w-[60px] text-center">
              {currentTraceIndex + 1} / {allTraces.length}
            </span>
            <button onClick={() => onNavigateTrace(currentTraceIndex + 1)} disabled={currentTraceIndex >= allTraces.length - 1} className="p-1 rounded hover:bg-[var(--hover-bg)] disabled:opacity-30 transition-colors">
              <ChevronRight className="w-4 h-4 text-muted" />
            </button>
            <button onClick={() => onNavigateTrace(allTraces.length - 1)} disabled={currentTraceIndex >= allTraces.length - 1} className="p-1 rounded hover:bg-[var(--hover-bg)] disabled:opacity-30 transition-colors">
              <ChevronsRight className="w-4 h-4 text-muted" />
            </button>
          </div>
        </div>
        {currentTraceSummary && (
          <div className="text-xs text-muted">
            {formatTimeAgo(currentTraceSummary.timestamp)} | {formatDurationMs(currentTraceSummary.durationMs)}
          </div>
        )}
      </div>

      {/* Trace waterfall */}
      <div className="border border-[var(--border-color)] rounded-xl bg-card overflow-hidden">
        {/* Trace ID header */}
        <div className="flex items-center gap-3 px-4 py-3 border-b border-[var(--border-color)]">
          <div className="flex items-center gap-2">
            <span className="text-xs text-muted">Trace ID:</span>
            <code className="text-xs text-default font-mono bg-[var(--hover-bg)] px-2 py-0.5 rounded">
              {trace.traceId}
            </code>
            <button onClick={copyTraceId} className="p-1 rounded hover:bg-[var(--hover-bg)] transition-colors">
              {copiedId ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Copy className="w-3.5 h-3.5 text-muted" />}
            </button>
            {/* 反向链路：Logs 早就能跳到 APM，反过来一直缺 —— 看瀑布图时想确认
                某一段到底打了什么日志，只能自己去 Logs 页粘 traceId */}
            <Link
              href={logsLinkForTrace(trace.traceId)}
              className="inline-flex items-center gap-1 text-xs text-blue-500 hover:underline"
            >
              <ScrollText className="w-3 h-3" />
              {t.viewLogs}
            </Link>
          </div>
          <span className="text-xs text-muted">
            {trace.spanCount} {t.spans} | {trace.serviceCount} {t.serviceCount} | {formatDurationMs(trace.durationMs)}
          </span>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-[var(--border-color)] px-4">
          {[t.timeline, t.metadata, t.logs].map((label, i) => (
            <button
              key={label}
              onClick={() => setActiveTab(i)}
              className={`px-3 py-2 text-xs font-medium border-b-2 -mb-px transition-colors ${
                activeTab === i ? "text-primary border-primary" : "text-muted border-transparent hover:text-default"
              }`}
            >
              {label}
            </button>
          ))}
        </div>

        {/* Metadata tab content */}
        {activeTab === 1 && (
          <TraceMetadata t={t} trace={trace} />
        )}

        {/* Logs tab content */}
        {activeTab === 2 && (
          <SpanLogs t={t} traceId={trace.traceId} serviceName={selectedSpan?.serviceName} />
        )}

        {/* Timeline tab content */}
        {activeTab === 0 && <>
        {/* Service legend */}
        <div className="flex flex-wrap gap-3 px-4 py-2 border-b border-[var(--border-color)]">
          {[...serviceColorMap.entries()].map(([svc, color]) => (
            <div key={svc} className="flex items-center gap-1.5 text-xs">
              <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: color }} />
              <span className="text-muted">{svc}</span>
            </div>
          ))}
        </div>

        {/* Timeline header */}
        <div className="px-4 py-2 border-b border-[var(--border-color)]">
          <div className="flex">
            <div className="w-[80px] flex-shrink-0" />
            <div className="flex-1 flex justify-between text-[10px] text-muted">
              {ticks.map((tick, i) => (
                <span key={i}>{formatDurationMs(tick)}</span>
              ))}
            </div>
          </div>
        </div>

        {/* Waterfall rows */}
        <div className="overflow-auto">
          {flatSpans.map((node) => (
            <SpanRow
              key={node.span.spanId}
              node={node}
              serviceColorMap={serviceColorMap}
              traceStartMs={traceStartMs}
              traceDurationMs={traceDurationMs}
              isSelected={selectedSpan?.spanId === node.span.spanId}
              isCollapsed={collapsedSpans.has(node.span.spanId)}
              onSelect={setSelectedSpan}
              onToggleCollapse={toggleCollapse}
            />
          ))}
        </div>
        </>}
      </div>

      {/* Span detail drawer */}
      <SpanDrawer
        t={t}
        span={selectedSpan}
        trace={trace}
        serviceColorMap={serviceColorMap}
        traceStartMs={traceStartMs}
        onClose={() => setSelectedSpan(null)}
      />
    </div>
  );
}

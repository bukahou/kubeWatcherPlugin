"use client";

import { useMemo } from "react";
import { AlertTriangle } from "lucide-react";
import type { TraceDetail, Span } from "@/types/model/apm";
import type { ApmTranslations } from "@/types/i18n";
import { formatDurationMs } from "@/lib/format";
import { SERVICE_COLORS } from "./waterfall-utils";
import { isErrorSpan, STATUS_CODE } from "@/lib/otel";

interface TraceMetadataProps {
  t: ApmTranslations;
  trace: TraceDetail;
}

interface ServiceAgg {
  name: string;
  color: string;
  spanCount: number;
  totalMs: number;
  percent: number;
  errorCount: number;
  pods: Set<string>;
  nodes: Set<string>;
  namespaces: Set<string>;
  deployments: Set<string>;
  versions: Set<string>;
}

export function TraceMetadata({ t, trace }: TraceMetadataProps) {
  const { services, errorSpans, traceDurationMs, startTime } = useMemo(() => {
    const svcMap = new Map<string, ServiceAgg>();
    const svcOrder: string[] = [];
    let colorIdx = 0;

    for (const span of trace.spans) {
      let agg = svcMap.get(span.serviceName);
      if (!agg) {
        agg = {
          name: span.serviceName,
          color: SERVICE_COLORS[colorIdx++ % SERVICE_COLORS.length],
          spanCount: 0,
          totalMs: 0,
          percent: 0,
          errorCount: 0,
          pods: new Set(),
          nodes: new Set(),
          namespaces: new Set(),
          deployments: new Set(),
          versions: new Set(),
        };
        svcMap.set(span.serviceName, agg);
        svcOrder.push(span.serviceName);
      }

      agg.spanCount++;
      agg.totalMs += span.durationMs;
      if (isErrorSpan(span)) agg.errorCount++;

      const res = span.resource;
      if (res?.podName) agg.pods.add(res.podName);
      if (res?.nodeName) agg.nodes.add(res.nodeName);
      if (res?.namespaceName) agg.namespaces.add(res.namespaceName);
      if (res?.deploymentName) agg.deployments.add(res.deploymentName);
      if (res?.serviceVersion) agg.versions.add(res.serviceVersion);
    }

    const totalMs = Math.max(...svcOrder.map(n => svcMap.get(n)!.totalMs), 1);
    for (const agg of svcMap.values()) {
      agg.percent = (agg.totalMs / totalMs) * 100;
    }

    const errors = trace.spans.filter(isErrorSpan);

    const timestamps = trace.spans.map(s => new Date(s.timestamp).getTime());
    const start = new Date(Math.min(...timestamps)).toLocaleString();
    const duration = trace.durationMs;

    return {
      services: svcOrder.map(n => svcMap.get(n)!),
      errorSpans: errors,
      traceDurationMs: duration,
      startTime: start,
    };
  }, [trace]);

  return (
    <div className="p-4 space-y-5">
      {/* Basic Info */}
      <section>
        <SectionHeader title={t.overview} />
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
          <StatCard label={t.startTime} value={startTime} />
          <StatCard label={t.duration} value={formatDurationMs(traceDurationMs)} />
          <StatCard label={t.spans} value={String(trace.spanCount)} />
          <StatCard label={t.serviceCount} value={String(trace.serviceCount)} />
        </div>
      </section>

      {/* Service Breakdown */}
      <section>
        <SectionHeader title={t.serviceBreakdown} />
        <div className="border border-[var(--border-color)] rounded-lg overflow-hidden">
          {/* Table header */}
          <div className="grid grid-cols-[1fr_80px_140px_80px_80px] gap-2 px-3 py-2 text-[10px] font-semibold text-muted uppercase tracking-wider bg-[var(--hover-bg)]">
            <span>{t.serviceName}</span>
            <span className="text-right">{t.spans}</span>
            <span>{t.durationPercent}</span>
            <span className="text-right">{t.errorSpans}</span>
            <span className="text-right">{t.instanceCount}</span>
          </div>
          {services.map((svc) => (
            <ServiceRow key={svc.name} svc={svc} t={t} />
          ))}
        </div>
      </section>

      {/* Resource Overview */}
      <section>
        <SectionHeader title={t.resourceOverview} />
        <div className="space-y-2">
          {services.map((svc) => {
            const hasResource = svc.pods.size > 0 || svc.namespaces.size > 0 || svc.versions.size > 0;
            if (!hasResource) return null;
            return (
              <div key={svc.name} className="border border-[var(--border-color)] rounded-lg overflow-hidden">
                <div className="flex items-center gap-2 px-3 py-2 bg-[var(--hover-bg)]">
                  <span className="w-2.5 h-2.5 rounded-full flex-shrink-0" style={{ backgroundColor: svc.color }} />
                  <span className="text-xs font-medium text-default">{svc.name}</span>
                  {svc.versions.size > 0 && (
                    <span className="text-[10px] text-muted ml-auto">{[...svc.versions].join(", ")}</span>
                  )}
                </div>
                <div className="px-3 py-2 space-y-1">
                  {svc.namespaces.size > 0 && (
                    <ResourceRow label={t.namespaceName} values={svc.namespaces} />
                  )}
                  {svc.deployments.size > 0 && (
                    <ResourceRow label={t.deploymentName} values={svc.deployments} />
                  )}
                  {svc.pods.size > 0 && (
                    <ResourceRow label={t.podName} values={svc.pods} />
                  )}
                  {svc.nodes.size > 0 && (
                    <ResourceRow label={t.nodeName} values={svc.nodes} />
                  )}
                </div>
              </div>
            );
          })}
        </div>
      </section>

      {/* Error Summary */}
      {errorSpans.length > 0 && (
        <section>
          <SectionHeader title={`${t.errors} (${errorSpans.length})`} />
          <div className="space-y-2">
            {errorSpans.map((span) => (
              <ErrorCard key={span.spanId} span={span} t={t} />
            ))}
          </div>
        </section>
      )}
    </div>
  );
}

// ============================================================
// Sub-components
// ============================================================

function SectionHeader({ title }: { title: string }) {
  return <h4 className="text-[11px] font-semibold text-muted uppercase tracking-wider mb-2">{title}</h4>;
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="px-3 py-2 rounded-lg bg-[var(--hover-bg)]">
      <div className="text-[10px] text-muted mb-0.5">{label}</div>
      <div className="text-sm font-medium text-default truncate">{value}</div>
    </div>
  );
}

function ServiceRow({ svc, t }: { svc: ServiceAgg; t: ApmTranslations }) {
  return (
    <div className="grid grid-cols-[1fr_80px_140px_80px_80px] gap-2 px-3 py-2 text-xs border-t border-[var(--border-color)] hover:bg-[var(--hover-bg)] transition-colors">
      <div className="flex items-center gap-2 min-w-0">
        <span className="w-2.5 h-2.5 rounded-full flex-shrink-0" style={{ backgroundColor: svc.color }} />
        <span className="text-default truncate">{svc.name}</span>
      </div>
      <span className="text-right text-default tabular-nums">{svc.spanCount}</span>
      <div className="flex items-center gap-2">
        <div className="flex-1 h-1.5 rounded-full bg-[var(--hover-bg)] overflow-hidden">
          <div
            className="h-full rounded-full"
            style={{ width: `${Math.max(svc.percent, 2)}%`, backgroundColor: svc.color }}
          />
        </div>
        <span className="text-muted text-[10px] tabular-nums w-[36px] text-right">
          {svc.percent.toFixed(0)}%
        </span>
      </div>
      <span className={`text-right tabular-nums ${svc.errorCount > 0 ? "text-red-400" : "text-muted"}`}>
        {svc.errorCount > 0 ? svc.errorCount : "-"}
      </span>
      <span className="text-right text-muted tabular-nums">{svc.pods.size || "-"}</span>
    </div>
  );
}

function ResourceRow({ label, values }: { label: string; values: Set<string> }) {
  return (
    <div className="flex gap-2 text-xs">
      <span className="text-muted flex-shrink-0 min-w-[90px]">{label}</span>
      <span className="text-default font-mono text-[11px] truncate">{[...values].join(", ")}</span>
    </div>
  );
}

function ErrorCard({ span, t }: { span: Span; t: ApmTranslations }) {
  return (
    <div className="border border-red-500/30 rounded-lg p-3 bg-red-500/5">
      <div className="flex items-center gap-2 mb-1">
        <AlertTriangle className="w-3.5 h-3.5 text-red-400 flex-shrink-0" />
        <span className="text-xs font-medium text-default truncate">{span.serviceName}</span>
        <span className="text-[10px] text-muted">→ {span.spanName}</span>
      </div>
      {span.error && (
        <>
          <div className="text-xs font-semibold text-red-400">{span.error.type}</div>
          <div className="text-xs text-default mt-0.5 line-clamp-2">{span.error.message}</div>
        </>
      )}
      {!span.error && (
        <div className="text-xs text-muted">{t.statusCode}: {STATUS_CODE.error}</div>
      )}
    </div>
  );
}

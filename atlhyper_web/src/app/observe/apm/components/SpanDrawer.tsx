"use client";

import { X } from "lucide-react";
import type { Span, TraceDetail } from "@/types/model/apm";
import type { ApmTranslations } from "@/types/i18n";
import { formatDurationMs } from "@/lib/format";
import { SpanLogs } from "./SpanLogs";
import { formatSpanKind, formatStatusCode, isErrorSpan } from "@/lib/otel";

// ============================================================
// SpanDrawer — structured attribute display
// ============================================================

interface SpanDrawerProps {
  t: ApmTranslations;
  span: Span | null;
  trace: TraceDetail;
  serviceColorMap: Map<string, string>;
  traceStartMs: number;
  onClose: () => void;
}

export function SpanDrawer({
  t,
  span,
  trace,
  serviceColorMap,
  traceStartMs,
  onClose,
}: SpanDrawerProps) {
  if (!span) return null;

  const color = serviceColorMap.get(span.serviceName) ?? "#94a3b8";
  const spanStartMs = new Date(span.timestamp).getTime();

  // Self-time = span duration minus direct children duration
  const childDuration = trace.spans
    .filter((s) => s.parentSpanId === span.spanId)
    .reduce((sum, s) => sum + s.durationMs, 0);
  const selfTime = span.durationMs - childDuration;
  const startOffset = spanStartMs - traceStartMs;

  return (
    <>
      <div className="fixed inset-0 bg-black/30 z-40 transition-opacity" onClick={onClose} />
      <div className="fixed right-0 top-0 h-full w-[520px] max-w-[90vw] z-50 bg-card border-l border-[var(--border-color)] shadow-2xl overflow-y-auto">
        {/* Header */}
        <div className="sticky top-0 z-10 bg-card border-b border-[var(--border-color)] px-5 py-4 flex items-center justify-between">
          <div className="flex items-center gap-2 min-w-0">
            <span className="w-3 h-3 rounded-full flex-shrink-0" style={{ backgroundColor: color }} />
            <h3 className="text-sm font-semibold text-default truncate">{t.spanDetail}</h3>
          </div>
          <button onClick={onClose} className="p-1.5 rounded-lg hover:bg-[var(--hover-bg)] transition-colors flex-shrink-0">
            <X className="w-4 h-4 text-muted" />
          </button>
        </div>

        <div className="p-5 space-y-5">
          {/* Overview */}
          <section>
            <SectionHeader title={t.overview} />
            <div className="grid grid-cols-2 gap-2.5">
              <MetricCard label={t.serviceName}>
                <div className="flex items-center gap-1.5">
                  <span className="w-2 h-2 rounded-full flex-shrink-0" style={{ backgroundColor: color }} />
                  <span className="text-sm font-medium text-default truncate">{span.serviceName}</span>
                </div>
              </MetricCard>
              <MetricCard label={t.spanKind}>
                <span className="text-sm font-medium text-default">
                  {formatSpanKind(span.spanKind)}
                </span>
              </MetricCard>
              <MetricCard label={t.statusCode}>
                <span className={`text-sm font-medium ${isErrorSpan(span) ? "text-red-400" : "text-emerald-400"}`}>
                  {formatStatusCode(span.statusCode)}
                </span>
              </MetricCard>
              <MetricCard label={t.duration}>
                <span className="text-sm font-medium text-default">{formatDurationMs(span.durationMs)}</span>
              </MetricCard>
              <MetricCard label={t.selfTime}>
                <span className="text-sm font-medium text-default">{formatDurationMs(selfTime)}</span>
                <span className="text-[10px] text-muted ml-1">
                  ({span.durationMs > 0 ? Math.round((selfTime / span.durationMs) * 100) : 0}%)
                </span>
              </MetricCard>
              <MetricCard label={t.childTime}>
                <span className="text-sm font-medium text-default">{formatDurationMs(childDuration)}</span>
              </MetricCard>
              <MetricCard label={t.startOffset} fullWidth>
                <span className="text-sm font-mono text-default">+{formatDurationMs(startOffset)}</span>
              </MetricCard>
            </div>
          </section>

          {/* Error Info */}
          {span.error && (
            <section>
              <SectionHeader title={t.errorInfo} />
              <div className="border border-red-500/30 rounded-lg p-3 bg-red-500/5">
                <div className="text-sm font-semibold text-red-400">{span.error.type}</div>
                <div className="text-sm text-default mt-1">{span.error.message}</div>
                {span.error.stacktrace && (
                  <details className="mt-2">
                    <summary className="text-xs text-muted cursor-pointer hover:text-default transition-colors">{t.showStacktrace}</summary>
                    <pre className="mt-1 text-xs text-muted font-mono whitespace-pre-wrap break-all max-h-[300px] overflow-y-auto">
                      {span.error.stacktrace}
                    </pre>
                  </details>
                )}
              </div>
            </section>
          )}

          {/* SpanName */}
          <section>
            <SectionHeader title={t.operationName} />
            <div className="px-3 py-2.5 rounded-lg bg-[var(--hover-bg)] text-sm font-mono text-default break-all">
              {span.spanName}
            </div>
          </section>

          {/* Resource */}
          {span.resource && (span.resource.podName || span.resource.clusterName || span.resource.serviceVersion) && (
            <section>
              <SectionHeader title={t.resourceInfo} />
              <div className="space-y-1.5">
                {span.resource.podName && <KVRow label={t.podName} value={span.resource.podName} />}
                {span.resource.nodeName && <KVRow label={t.nodeName} value={span.resource.nodeName} />}
                {span.resource.deploymentName && <KVRow label={t.deploymentName} value={span.resource.deploymentName} />}
                {span.resource.namespaceName && <KVRow label={t.namespaceName} value={span.resource.namespaceName} />}
                {span.resource.clusterName && <KVRow label={t.clusterName} value={span.resource.clusterName} />}
                {span.resource.serviceVersion && <KVRow label={t.serviceVersion} value={span.resource.serviceVersion} />}
                {span.resource.instanceId && <KVRow label="Instance ID" value={span.resource.instanceId} />}
              </div>
            </section>
          )}

          {/* HTTP */}
          {span.http && (
            <section>
              <SectionHeader title={t.httpAttributes} />
              <div className="border border-[var(--border-color)] rounded-lg overflow-hidden">
                <KVTableRow label={t.httpMethod} value={span.http.method} />
                {span.http.route && <KVTableRow label={t.httpRoute} value={span.http.route} border />}
                {span.http.url && <KVTableRow label={t.httpUrl} value={span.http.url} border />}
                {span.http.statusCode !== undefined && (
                  <KVTableRow label={t.httpStatusCode} border>
                    <span className={`px-1.5 py-0.5 rounded text-xs font-mono font-semibold ${
                      span.http.statusCode < 300 ? 'bg-emerald-500/10 text-emerald-400' :
                      span.http.statusCode < 400 ? 'bg-blue-500/10 text-blue-400' :
                      span.http.statusCode < 500 ? 'bg-amber-500/10 text-amber-400' :
                      'bg-red-500/10 text-red-400'
                    }`}>
                      {span.http.statusCode}
                    </span>
                  </KVTableRow>
                )}
                {span.http.server && <KVTableRow label="Server" value={`${span.http.server}:${span.http.serverPort}`} border />}
              </div>
            </section>
          )}

          {/* DB */}
          {span.db && (
            <section>
              <SectionHeader title={t.dbAttributes} />
              <div className="border border-[var(--border-color)] rounded-lg overflow-hidden">
                <KVTableRow label={t.dbSystem} value={span.db.system} />
                {span.db.name && <KVTableRow label={t.dbName} value={span.db.name} border />}
                {span.db.operation && <KVTableRow label={t.dbOperation} value={span.db.operation} border />}
                {span.db.table && <KVTableRow label={t.dbTable} value={span.db.table} border />}
                {span.db.statement && <KVTableRow label={t.dbStatement} value={span.db.statement} border mono />}
              </div>
            </section>
          )}

          {/* Events */}
          {(() => {
            const displayEvents = span.events?.filter(e => !(span.error && e.name === 'exception')) ?? [];
            if (displayEvents.length === 0) return null;
            return (
              <section>
                <SectionHeader title={`Events (${displayEvents.length})`} />
                <div className="space-y-2">
                  {displayEvents.map((ev, i) => (
                    <div key={i} className="border border-[var(--border-color)] rounded-lg p-3">
                      <div className="flex items-center justify-between mb-1">
                        <span className="text-xs font-medium text-default">{ev.name}</span>
                        <span className="text-[10px] text-muted">{new Date(ev.timestamp).toLocaleTimeString()}</span>
                      </div>
                      {ev.attributes && Object.entries(ev.attributes).map(([k, v]) => (
                        <div key={k} className="text-xs text-muted mt-1">
                          <span className="text-muted/70">{k}:</span>{" "}
                          <span className="font-mono text-default break-all">{v}</span>
                        </div>
                      ))}
                    </div>
                  ))}
                </div>
              </section>
            );
          })()}

          {/* IDs */}
          <section>
            <SectionHeader title={t.spanIds} />
            <div className="space-y-1.5">
              <IdRow label="Span ID" value={span.spanId} />
              {span.parentSpanId && <IdRow label={t.parentSpan} value={span.parentSpanId} />}
              <IdRow label="Trace ID" value={trace.traceId} />
            </div>
          </section>

          {/* Correlated Logs */}
          <section>
            <SectionHeader title={`${t.correlatedLogs} — ${span.serviceName}`} />
            <div className="border border-[var(--border-color)] rounded-lg overflow-hidden">
              <SpanLogs t={t} traceId={trace.traceId} serviceName={span.serviceName} compact />
            </div>
          </section>
        </div>
      </div>
    </>
  );
}

// ============================================================
// Drawer sub-components
// ============================================================

function SectionHeader({ title }: { title: string }) {
  return <h4 className="text-[11px] font-semibold text-muted uppercase tracking-wider mb-2">{title}</h4>;
}

function MetricCard({ label, children, fullWidth }: { label: string; children: React.ReactNode; fullWidth?: boolean }) {
  return (
    <div className={`px-3 py-2 rounded-lg bg-[var(--hover-bg)] ${fullWidth ? "col-span-2" : ""}`}>
      <div className="text-[10px] text-muted mb-0.5">{label}</div>
      <div className="flex items-center">{children}</div>
    </div>
  );
}

function IdRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-[var(--hover-bg)] text-xs">
      <span className="text-muted flex-shrink-0 min-w-[80px]">{label}</span>
      <code className="text-default font-mono text-[11px] truncate">{value}</code>
    </div>
  );
}

function KVRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-[var(--hover-bg)] text-xs">
      <span className="text-muted flex-shrink-0 min-w-[100px]">{label}</span>
      <span className="text-default font-mono text-[11px] truncate">{value}</span>
    </div>
  );
}

function KVTableRow({ label, value, border, mono, children }: { label: string; value?: string; border?: boolean; mono?: boolean; children?: React.ReactNode }) {
  return (
    <div className={`flex gap-3 px-3 py-2 text-xs ${border ? "border-t border-[var(--border-color)]" : ""}`}>
      <span className="text-muted flex-shrink-0 min-w-[100px]">{label}</span>
      {children ?? <span className={`text-default break-all ${mono ? "font-mono text-[11px]" : ""}`}>{value}</span>}
    </div>
  );
}

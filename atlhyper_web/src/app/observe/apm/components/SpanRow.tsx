"use client";

import { ChevronRight, ChevronDown } from "lucide-react";
import type { Span } from "@/types/model/apm";
import { formatDurationMs } from "@/lib/format";
import { countDescendants, type SpanNode } from "./waterfall-utils";
import { isErrorSpan } from "@/lib/otel";

interface SpanRowProps {
  node: SpanNode;
  serviceColorMap: Map<string, string>;
  traceStartMs: number;
  traceDurationMs: number;
  isSelected: boolean;
  isCollapsed: boolean;
  onSelect: (span: Span) => void;
  onToggleCollapse: (spanId: string) => void;
}

export function SpanRow({
  node,
  serviceColorMap,
  traceStartMs,
  traceDurationMs,
  isSelected,
  isCollapsed,
  onSelect,
  onToggleCollapse,
}: SpanRowProps) {
  const { span, depth } = node;
  const color = serviceColorMap.get(span.serviceName) ?? "#94a3b8";
  const spanStartMs = new Date(span.timestamp).getTime();
  const offset = traceDurationMs > 0
    ? ((spanStartMs - traceStartMs) / traceDurationMs) * 100
    : 0;
  const width = traceDurationMs > 0
    ? (span.durationMs / traceDurationMs) * 100
    : 100;
  const childCount = countDescendants(node);
  const hasChildren = node.children.length > 0;
  const isError = isErrorSpan(span);

  const barIsWide = width > 15;
  const barH = 24;

  return (
    <div
      onClick={() => onSelect(span)}
      className={`flex items-center cursor-pointer border-b border-[var(--border-color)]/20 transition-colors ${
        isSelected ? "bg-primary/5" : "hover:bg-[var(--hover-bg)]"
      }`}
      style={{ height: 34 }}
    >
      <div
        className="w-[80px] flex-shrink-0 flex items-center gap-1 text-xs px-2"
        style={{ paddingLeft: `${depth * 12 + 8}px` }}
      >
        {hasChildren ? (
          <button
            onClick={(e) => { e.stopPropagation(); onToggleCollapse(span.spanId); }}
            className="flex items-center gap-0.5 p-0.5 rounded hover:bg-[var(--hover-bg)]"
          >
            {isCollapsed ? <ChevronRight className="w-3 h-3 text-muted" /> : <ChevronDown className="w-3 h-3 text-muted" />}
            <span className="text-[10px] text-muted">{childCount}</span>
          </button>
        ) : (
          <span className="w-[18px]" />
        )}
      </div>

      <div className="flex-1 relative overflow-hidden" style={{ height: barH }}>
        <div
          className="absolute top-0"
          style={{
            left: `${offset}%`,
            width: `${Math.max(width, 0.3)}%`,
            height: barH,
            borderRadius: 4,
            borderLeft: `3px solid ${isError ? "#ef4444" : color}`,
            background: isError ? "rgba(239,68,68,0.15)" : `${color}30`,
          }}
        >
          {barIsWide && (
            <div className="absolute inset-0 flex items-center gap-1 px-2 overflow-hidden">
              <span className="text-[11px] font-medium truncate" style={{ color: isError ? "#ef4444" : color }}>
                {span.spanName}
              </span>
              <span className="text-[10px] flex-shrink-0" style={{ color: `${isError ? "#ef4444" : color}99` }}>
                {formatDurationMs(span.durationMs)}
              </span>
            </div>
          )}
        </div>
        {!barIsWide && (() => {
          const nearRightEdge = offset + width > 65;
          return nearRightEdge ? (
            <div
              className="absolute flex items-center justify-end gap-1.5 whitespace-nowrap"
              style={{ right: `${100 - offset + 0.5}%`, top: 0, height: barH }}
            >
              <span className="text-[10px] text-muted flex-shrink-0">{formatDurationMs(span.durationMs)}</span>
              <span className="text-[11px] text-default truncate">{span.spanName}</span>
            </div>
          ) : (
            <div
              className="absolute flex items-center gap-1.5 whitespace-nowrap"
              style={{ left: `${offset + Math.max(width, 0.3) + 0.5}%`, top: 0, height: barH }}
            >
              <span className="text-[11px] text-default truncate">{span.spanName}</span>
              <span className="text-[10px] text-muted flex-shrink-0">{formatDurationMs(span.durationMs)}</span>
            </div>
          );
        })()}
      </div>
    </div>
  );
}

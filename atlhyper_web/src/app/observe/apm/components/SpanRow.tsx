"use client";

import { ChevronRight, ChevronDown, AlertCircle } from "lucide-react";
import type { Span } from "@/types/model/apm";
import { formatDurationMs } from "@/lib/format";
import { countDescendants, type SpanNode } from "./waterfall-utils";
import { isErrorSpan } from "@/lib/otel";

// ============================================================
// SpanRow — 瀑布图单行
//
// 视觉通道分工（硬约束，见 apm-panel-redesign.md 功能一）：
//   颜色     = 服务身份（error 不覆盖 —— 全 error 的 trace 也要能分服务）
//   红描边+⚠ = 错误
//   缩进     = 调用层级；跨服务边界另加服务色标签（不靠颜色对比脑补）
//   长度/偏移 = 时间；bar 内深色段 = self time，浅色段 = 等待子调用
//   透明度   = focus（非 focus 服务淡化但可见 —— 高亮而非裁剪）
// ============================================================

interface SpanRowProps {
  node: SpanNode;
  serviceColorMap: Map<string, string>;
  traceStartMs: number;
  traceDurationMs: number;
  isSelected: boolean;
  isCollapsed: boolean;
  /** null = 无 focus（全部全亮）；否则不在集合中的行淡化 */
  focusedIds: Set<string> | null;
  onSelect: (span: Span) => void;
  onToggleCollapse: (spanId: string) => void;
}

/** 服务名缩写：取首字母段，最多 4 字符（gateway → GATE, user → USER） */
function serviceAbbr(name: string): string {
  const last = name.split("-").pop() ?? name;
  return last.slice(0, 4).toUpperCase();
}

export function SpanRow({
  node,
  serviceColorMap,
  traceStartMs,
  traceDurationMs,
  isSelected,
  isCollapsed,
  focusedIds,
  onSelect,
  onToggleCollapse,
}: SpanRowProps) {
  const { span, depth, isBoundary, selfDurationMs } = node;
  const color = serviceColorMap.get(span.serviceName) ?? "#94a3b8";
  const spanStartMs = new Date(span.timestamp).getTime();
  const offset = traceDurationMs > 0
    ? ((spanStartMs - traceStartMs) / traceDurationMs) * 100
    : 0;
  const width = traceDurationMs > 0
    ? (span.durationMs / traceDurationMs) * 100
    : 100;
  const selfPct = span.durationMs > 0
    ? (selfDurationMs / span.durationMs) * 100
    : 100;
  const tracePct = traceDurationMs > 0
    ? Math.round((span.durationMs / traceDurationMs) * 100)
    : 100;
  const childCount = countDescendants(node);
  const hasChildren = node.children.length > 0;
  const isError = isErrorSpan(span);
  const isDimmed = focusedIds !== null && !focusedIds.has(span.spanId);

  const barH = 20;

  return (
    <div
      onClick={() => onSelect(span)}
      className={`flex items-center cursor-pointer border-b border-[var(--border-color)]/20 transition-all ${
        isSelected ? "bg-primary/5" : "hover:bg-[var(--hover-bg)]"
      } ${isDimmed ? "opacity-40" : ""}`}
      style={{ height: 34 }}
    >
      {/* 名字列：缩进 + 折叠 + 边界标签 + span 名 */}
      <div
        className="w-[240px] flex-shrink-0 flex items-center gap-1 text-xs pr-1 overflow-hidden"
        style={{ paddingLeft: `${depth * 14 + 8}px` }}
      >
        {hasChildren ? (
          <button
            onClick={(e) => { e.stopPropagation(); onToggleCollapse(span.spanId); }}
            className="flex items-center gap-0.5 p-0.5 rounded hover:bg-[var(--hover-bg)] flex-shrink-0"
          >
            {isCollapsed ? <ChevronRight className="w-3 h-3 text-muted" /> : <ChevronDown className="w-3 h-3 text-muted" />}
            <span className="text-[10px] text-muted">{childCount}</span>
          </button>
        ) : (
          <span className="w-[18px] flex-shrink-0" />
        )}
        {isBoundary && (
          <span
            className="flex-shrink-0 text-[9px] leading-none font-bold px-1 py-0.5 rounded"
            style={{ backgroundColor: color, color: "var(--card-bg)" }}
            title={span.serviceName}
          >
            {serviceAbbr(span.serviceName)}
          </span>
        )}
        <span
          className={`truncate text-[11px] ${isError ? "text-red-400 font-medium" : "text-default"}`}
          title={span.spanName}
        >
          {span.spanName}
        </span>
      </div>

      {/* 时间轨道：服务色 bar，内嵌 self-time 深色段，error 描边 + 角标 */}
      <div className="flex-1 relative overflow-visible" style={{ height: barH + 8 }}>
        <div
          className="absolute"
          style={{
            top: 4,
            left: `${offset}%`,
            width: `${Math.max(width, 0.5)}%`,
            height: barH,
            borderRadius: 3,
            borderLeft: `3px solid ${color}`,
            background: `${color}26`,
            outline: isError ? "1.5px solid #ef4444" : undefined,
            outlineOffset: isError ? "0px" : undefined,
          }}
        >
          {/* self-time 深色段：从左起，剩余浅色 = 等待子调用 */}
          <div
            className="absolute left-0 top-0 h-full"
            style={{
              width: `${Math.min(selfPct, 100)}%`,
              background: `${color}55`,
              borderRadius: "3px 0 0 3px",
            }}
          />
          {isError && (
            <span
              className="absolute flex items-center justify-center"
              style={{ right: -8, top: -7 }}
            >
              <AlertCircle className="w-3.5 h-3.5 text-red-500 bg-[var(--card-bg)] rounded-full" />
            </span>
          )}
        </div>
      </div>

      {/* 耗时列：绝对值 + 占 trace 总时长百分比 */}
      <div className="w-[110px] flex-shrink-0 text-right pr-3 font-mono tabular-nums">
        <span className={`text-[11px] ${isError ? "text-red-400" : "text-default"}`}>
          {formatDurationMs(span.durationMs)}
        </span>
        <span className="text-[10px] text-muted ml-1">{tracePct}%</span>
      </div>
    </div>
  );
}

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
  /** 各祖先层级是否还有后续兄弟 —— 决定树状竖线画到哪里（长度 = depth） */
  ancestorHasNext: boolean[];
  /** 本行在同级中是否为最后一个 —— 决定画 └ 还是 ├ */
  isLastChild: boolean;
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
  ancestorHasNext,
  isLastChild,
  onSelect,
  onToggleCollapse,
}: SpanRowProps) {
  const { span, depth, isBoundary, selfDurationMs, selfIntervals, spanInterval } = node;
  const color = serviceColorMap.get(span.serviceName) ?? "#94a3b8";
  // spanInterval / selfIntervals 已是相对 trace 起点的毫秒偏移（buildSpanTree 计算）
  const pct = (ms: number) => (traceDurationMs > 0 ? (ms / traceDurationMs) * 100 : 0);
  const offset = pct(spanInterval.startMs);
  const width = Math.max(pct(span.durationMs), 0.4);
  const tracePct = traceDurationMs > 0
    ? Math.round((span.durationMs / traceDurationMs) * 100)
    : 100;
  // self 段在整行时间轴上的绝对位置（不是 bar 内的相对位置）——
  // 被子调用占走的区间留空，那段时间在子行里着色
  const selfSegments = selfIntervals.map((iv) => ({
    left: pct(iv.startMs),
    width: Math.max(pct(iv.endMs - iv.startMs), 0.25),
  }));
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
      {/* 名字列：树状连接线 + 折叠 + 边界标签 + span 名 */}
      <div className="w-[240px] flex-shrink-0 flex items-center text-xs pr-1 overflow-hidden pl-2">
        {/* 树状连接线：竖线表示祖先层级仍有后续兄弟，└/├ 表示本行归属 */}
        {ancestorHasNext.map((hasNext, i) => (
          <span key={i} className="flex-shrink-0 self-stretch relative" style={{ width: 14 }}>
            {hasNext && (
              <span
                className="absolute top-0 bottom-0 border-l"
                style={{ left: 6, borderColor: "var(--border-color)" }}
              />
            )}
          </span>
        ))}
        {depth > 0 && (
          <span className="flex-shrink-0 self-stretch relative" style={{ width: 14 }}>
            <span
              className="absolute border-l"
              style={{
                left: 6, top: 0,
                height: isLastChild ? "50%" : "100%",
                borderColor: "var(--border-color)",
              }}
            />
            <span
              className="absolute border-t"
              style={{ left: 6, top: "50%", width: 8, borderColor: "var(--border-color)" }}
            />
          </span>
        )}
        <span className="flex items-center gap-1 min-w-0">
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
        </span>
      </div>

      {/* 时间轨道 —— 包裹形态：
          · 外层是【容器】（细描边 + 极淡填充），表示这个 span 的总跨度
          · 实色段只画在 selfIntervals 的【真实时间位置】：这才是它自己在干活
          · 被子调用占走的区间留空 —— 那段时间在下一行（子 span）着色
          每一毫秒只在它真正归属的层级上出现一次，嵌套关系由此可读 */}
      <div className="flex-1 relative overflow-visible" style={{ height: barH + 8 }}>
        {/* 容器：总跨度 */}
        <div
          className="absolute"
          style={{
            top: 4,
            left: `${offset}%`,
            width: `${width}%`,
            height: barH,
            borderRadius: 3,
            border: `1px solid ${color}66`,
            borderLeft: `3px solid ${color}`,
            background: `${color}14`,
            outline: isError ? "1.5px solid #ef4444" : undefined,
          }}
        />
        {/* self 段：按真实时间位置填实色 */}
        {selfSegments.map((seg, i) => (
          <div
            key={i}
            className="absolute"
            style={{
              top: 4,
              left: `${seg.left}%`,
              width: `${seg.width}%`,
              height: barH,
              background: `${color}80`,
              borderRadius: 2,
            }}
          />
        ))}
        {isError && (
          <span
            className="absolute flex items-center justify-center"
            style={{ left: `calc(${offset + width}% - 4px)`, top: -3 }}
          >
            <AlertCircle className="w-3.5 h-3.5 text-red-500 bg-[var(--card-bg)] rounded-full" />
          </span>
        )}
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

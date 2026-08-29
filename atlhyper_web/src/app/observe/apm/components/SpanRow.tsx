"use client";

import { ChevronRight, ChevronDown, AlertCircle } from "lucide-react";
import type { Span } from "@/types/model/apm";
import { formatDurationMs } from "@/lib/format";
import { countDescendants, type SpanNode } from "./waterfall-utils";
import { isErrorSpan } from "@/lib/otel";

// ============================================================
// SpanRow — 瀑布图单行（Elastic APM 布局）
//
// 布局：标签跟随 bar，不用左侧固定列
//   ┌ [折叠] [═════════ bar ═════════]
//   └        操作名  耗时 占比
//
// 为什么标签必须跟着 bar 走（2026-08-29 实测教训）：
//   曾用「左侧固定名字列 + 右侧轨道」，结果在 gateway→user 这类
//   「上游几乎不干活、全程等下游」的 trace 上（389/387/386ms），
//   三条 bar 长度与起点都几乎重合，右侧轨道看起来就是三条平行线，
//   层级信息全压在左侧几根细树线上，视觉重量完全不够。
//   标签跟随 bar 后，「bar + 标签」作为一个视觉单元整体在时间轴上递进，
//   即使 bar 长度接近，阶梯感仍在 —— 这是 ES 布局的关键，不是排版偏好。
//
// 视觉通道分工（硬约束）：
//   颜色     = 服务身份（error 不覆盖也不描边 —— 全 error 的 trace 也要能分服务）
//   红端帽+⚠ = 错误（只标在 bar 右端，不侵占 bar 本身）
//   缩进+树线 = 调用层级；跨服务边界另加服务色标签
//   长度/偏移 = 时间；bar 实心 = 本层总耗时，深色叠加段 = self time 的真实位置，
//              浅色即「在等下游」。⚠️ 不要把父 bar 挖空 —— 那会丢掉本层总跨度
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

/** 服务名缩写：取末段，最多 4 字符（geass-gateway → GATE, geass-user → USER） */
function serviceAbbr(name: string): string {
  const last = name.split("-").pop() ?? name;
  return last.slice(0, 4).toUpperCase();
}

/** 每层缩进宽度（树线列宽） */
const INDENT = 16;

export function SpanRow({
  node,
  serviceColorMap,
  traceDurationMs,
  isSelected,
  isCollapsed,
  focusedIds,
  ancestorHasNext,
  isLastChild,
  onSelect,
  onToggleCollapse,
}: SpanRowProps) {
  const { span, depth, isBoundary, selfIntervals, spanInterval } = node;
  const color = serviceColorMap.get(span.serviceName) ?? "#94a3b8";

  // spanInterval / selfIntervals 已是相对 trace 起点的毫秒偏移（buildSpanTree 计算）
  const pct = (ms: number) => (traceDurationMs > 0 ? (ms / traceDurationMs) * 100 : 0);
  const offset = pct(spanInterval.startMs);
  const width = Math.max(pct(span.durationMs), 0.4);
  const tracePct = traceDurationMs > 0
    ? Math.round((span.durationMs / traceDurationMs) * 100)
    : 100;
  // self 段按真实时间位置绘制（不是 bar 内的相对位置）
  const selfSegments = selfIntervals.map((iv) => ({
    left: pct(iv.startMs),
    width: Math.max(pct(iv.endMs - iv.startMs), 0.25),
  }));

  const childCount = countDescendants(node);
  const hasChildren = node.children.length > 0;
  const isError = isErrorSpan(span);
  const isDimmed = focusedIds !== null && !focusedIds.has(span.spanId);

  const barH = 14;
  // 标签跟随 bar 起点；接近右端时改为右对齐，避免溢出被截断
  const labelNearRight = offset > 62;

  return (
    <div
      onClick={() => onSelect(span)}
      className={`flex cursor-pointer border-b border-[var(--border-color)]/20 transition-all ${
        isSelected ? "bg-primary/5" : "hover:bg-[var(--hover-bg)]"
      } ${isDimmed ? "opacity-40" : ""}`}
      style={{ minHeight: 46 }}
    >
      {/* 缩进 + 树状连接线 + 折叠控件（宽度随层级增长，不占固定大列） */}
      <div className="flex-shrink-0 flex items-start pt-1.5 pl-2">
        {ancestorHasNext.map((hasNext, i) => (
          <span key={i} className="flex-shrink-0 relative" style={{ width: INDENT, height: 30 }}>
            {hasNext && (
              <span
                className="absolute top-0 bottom-0 border-l"
                style={{ left: 7, borderColor: "var(--border-color)" }}
              />
            )}
          </span>
        ))}
        {depth > 0 && (
          <span className="flex-shrink-0 relative" style={{ width: INDENT, height: 30 }}>
            <span
              className="absolute border-l"
              style={{
                left: 7, top: 0,
                height: isLastChild ? 11 : "100%",
                borderColor: "var(--border-color)",
              }}
            />
            <span
              className="absolute border-t"
              style={{ left: 7, top: 11, width: 9, borderColor: "var(--border-color)" }}
            />
          </span>
        )}
        {hasChildren ? (
          <button
            onClick={(e) => { e.stopPropagation(); onToggleCollapse(span.spanId); }}
            className="flex items-center gap-0.5 p-0.5 rounded hover:bg-[var(--hover-bg)] flex-shrink-0"
            style={{ marginTop: 2 }}
          >
            {isCollapsed ? <ChevronRight className="w-3 h-3 text-muted" /> : <ChevronDown className="w-3 h-3 text-muted" />}
            <span className="text-[10px] text-muted">{childCount}</span>
          </button>
        ) : (
          <span className="w-[22px] flex-shrink-0" />
        )}
      </div>

      {/* 时间轨道 + 跟随 bar 的标签 —— 二者共用同一坐标系，整体在时间轴上递进 */}
      <div className="flex-1 relative pr-4" style={{ minWidth: 0 }}>
        {/* bar 层 */}
        <div className="relative" style={{ height: barH + 8 }}>
          {/* 实心 bar：本层总耗时 */}
          <div
            className="absolute"
            style={{
              top: 6,
              left: `${offset}%`,
              width: `${width}%`,
              height: barH,
              borderRadius: 3,
              background: `${color}59`,
            }}
          />
          {/* self time 段：叠加在实心 bar 上，标出真正在干活的时间位置 */}
          {selfSegments.map((seg, i) => (
            <div
              key={i}
              className="absolute"
              style={{
                top: 6,
                left: `${seg.left}%`,
                width: `${seg.width}%`,
                height: barH,
                background: color,
                borderRadius: 2,
              }}
            />
          ))}
          {/* 错误：右端红色端帽 + 角标 */}
          {isError && (
            <>
              <div
                className="absolute"
                style={{
                  top: 6,
                  left: `calc(${offset + width}% - 2px)`,
                  width: 2,
                  height: barH,
                  background: "#ef4444",
                  borderRadius: "0 2px 2px 0",
                }}
              />
              <span
                className="absolute flex items-center justify-center"
                style={{ left: `calc(${offset + width}% - 5px)`, top: 0 }}
              >
                <AlertCircle className="w-3 h-3 text-red-500 bg-[var(--card-bg)] rounded-full" />
              </span>
            </>
          )}
        </div>

        {/* 标签层：起点对齐 bar，这是层级阶梯感的来源 */}
        <div className="relative" style={{ height: 18 }}>
          <div
            className="absolute flex items-center gap-1.5 whitespace-nowrap"
            style={
              labelNearRight
                ? { right: `${Math.max(100 - offset - width, 0)}%`, top: 0 }
                : { left: `${offset}%`, top: 0 }
            }
          >
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
              className={`text-[11px] ${isError ? "text-red-400 font-medium" : "text-default"}`}
              title={span.spanName}
            >
              {span.spanName}
            </span>
            <span className="text-[10px] font-mono tabular-nums text-muted">
              {formatDurationMs(span.durationMs)}
              <span className="ml-1 opacity-70">{tracePct}%</span>
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}

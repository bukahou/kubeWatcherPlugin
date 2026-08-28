import type { Span, TraceDetail } from "@/types/model/apm";

// 服务色板 —— 颜色是服务身份的唯一编码（按 trace 内首次出现顺序分配）。
// 错误、类型、时间各有独立视觉通道（描边/图标、缩进、长度），
// 禁止任何状态覆盖服务色 —— 2026-08-28 全 error 的 trace 曾因红色覆盖
// 服务色而完全看不出服务边界。
export const SERVICE_COLORS = [
  "#60a5fa", "#34d399", "#fbbf24", "#a78bfa",
  "#f87171", "#22d3ee", "#fb923c", "#818cf8",
];

export interface SpanNode {
  span: Span;
  children: SpanNode[];
  depth: number;
  /** self time = 自身耗时 − 直接子 span 耗时之和（钳 0：异步子调用可能超过父时长） */
  selfDurationMs: number;
  /** 服务边界：与父 span 属于不同服务（根节点为 false，边界靠标签而非颜色脑补） */
  isBoundary: boolean;
}

export function buildSpanTree(spans: Span[]): SpanNode[] {
  const spanMap = new Map<string, SpanNode>();
  const roots: SpanNode[] = [];

  for (const span of spans) {
    spanMap.set(span.spanId, {
      span, children: [], depth: 0, selfDurationMs: span.durationMs, isBoundary: false,
    });
  }

  for (const span of spans) {
    const node = spanMap.get(span.spanId)!;
    if (span.parentSpanId && spanMap.has(span.parentSpanId)) {
      const parent = spanMap.get(span.parentSpanId)!;
      parent.children.push(node);
      node.isBoundary = span.serviceName !== parent.span.serviceName;
    } else {
      roots.push(node);
    }
  }

  function finalize(node: SpanNode, depth: number) {
    node.depth = depth;
    const childSum = node.children.reduce((s, c) => s + c.span.durationMs, 0);
    node.selfDurationMs = Math.max(node.span.durationMs - childSum, 0);
    node.children.forEach((c) => finalize(c, depth + 1));
  }
  roots.forEach((r) => finalize(r, 0));
  return roots;
}

export function flattenTree(nodes: SpanNode[], collapsed: Set<string>): SpanNode[] {
  const result: SpanNode[] = [];
  function walk(node: SpanNode) {
    result.push(node);
    if (!collapsed.has(node.span.spanId)) {
      node.children.forEach(walk);
    }
  }
  nodes.forEach(walk);
  return result;
}

export function countDescendants(node: SpanNode): number {
  let count = node.children.length;
  for (const child of node.children) count += countDescendants(child);
  return count;
}

// ──────────────────────────────────────────────────────────────
// focus：高亮而非裁剪
// ──────────────────────────────────────────────────────────────
//
// 历史（2026-08-28）：前身 filterTraceForService 会把非 focus 服务的
// 祖先 span 从树里删掉 —— 从叶子服务（恰恰是排查者最常进入的报错服务）
// 看 trace 时只剩 1 行，上游是谁、错误从哪来完全不可见。
// 现在完整树永远保留，focus 只决定哪些行全亮、哪些行降透明度。

/**
 * 计算 focus 服务的 span 集合：该服务的入口 span（父 span 属于其他服务
 * 或无父）及其**所有后代**（含跨回其他服务的下游）。
 * 返回空集合表示无匹配（调用方应视为不做任何淡化）。
 */
export function focusSpanIds(trace: TraceDetail, focusService: string): Set<string> {
  const { spans } = trace;
  const spanMap = new Map<string, Span>();
  const childrenMap = new Map<string, string[]>();
  for (const span of spans) {
    spanMap.set(span.spanId, span);
    if (span.parentSpanId) {
      const list = childrenMap.get(span.parentSpanId) ?? [];
      list.push(span.spanId);
      childrenMap.set(span.parentSpanId, list);
    }
  }

  const entryIds: string[] = [];
  for (const span of spans) {
    if (span.serviceName !== focusService) continue;
    const parent = span.parentSpanId ? spanMap.get(span.parentSpanId) : undefined;
    if (!parent || parent.serviceName !== focusService) {
      entryIds.push(span.spanId);
    }
  }
  if (entryIds.length === 0) return new Set();

  const included = new Set<string>();
  const collect = (id: string) => {
    included.add(id);
    for (const childId of childrenMap.get(id) ?? []) collect(childId);
  };
  entryIds.forEach(collect);
  return included;
}

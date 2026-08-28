import type { Span, TraceDetail } from "@/types/model/apm";

// 服务色板 —— 颜色是服务身份的唯一编码（按 trace 内首次出现顺序分配）。
// 错误、类型、时间各有独立视觉通道（描边/图标、缩进、长度），
// 禁止任何状态覆盖服务色 —— 2026-08-28 全 error 的 trace 曾因红色覆盖
// 服务色而完全看不出服务边界。
export const SERVICE_COLORS = [
  "#60a5fa", "#34d399", "#fbbf24", "#a78bfa",
  "#f87171", "#22d3ee", "#fb923c", "#818cf8",
];

/** 时间轴上的一段区间（相对 trace 起点的毫秒偏移） */
export interface TimeInterval {
  startMs: number;
  endMs: number;
}

export interface SpanNode {
  span: Span;
  children: SpanNode[];
  depth: number;
  /** self time 总量 = selfIntervals 各段之和（重叠子调用只扣一次） */
  selfDurationMs: number;
  /**
   * self time 在时间轴上的**真实区间**（父区间减去子调用占用后的剩余）。
   *
   * 这是「包裹形态」的核心：父 span 的 bar 画成容器，只在这些区间填实色，
   * 被子调用占走的部分留空 —— 那段时间会在子 span 的行里着色。
   * **每一毫秒只在它真正归属的层级上出现一次**，嵌套关系由此可读。
   *
   * 网关那种「发起下游前 1ms + 收到响应后 1ms」的分布，
   * 用单个标量（selfDurationMs=2ms）画在 bar 左端是示意；
   * 用区间画才是事实。
   */
  selfIntervals: TimeInterval[];
  /** span 自身在时间轴上的区间（相对 trace 起点） */
  spanInterval: TimeInterval;
  /** 服务边界：与父 span 属于不同服务（根节点为 false，边界靠标签而非颜色脑补） */
  isBoundary: boolean;
}

/**
 * 区间减法：从 base 中挖掉 holes 覆盖的部分。
 *
 * holes 先按起点排序并合并重叠 —— 并行子调用重叠时若逐个扣减会重复扣，
 * 得出偏小甚至为负的 self time。
 */
function subtractIntervals(base: TimeInterval, holes: TimeInterval[]): TimeInterval[] {
  // 裁剪到 base 范围内（异步子调用或时钟偏移可能越界），丢弃空区间
  const clipped = holes
    .map((h) => ({
      startMs: Math.max(h.startMs, base.startMs),
      endMs: Math.min(h.endMs, base.endMs),
    }))
    .filter((h) => h.endMs > h.startMs)
    .sort((a, b) => a.startMs - b.startMs);

  // 合并重叠/相邻区间
  const merged: TimeInterval[] = [];
  for (const h of clipped) {
    const last = merged[merged.length - 1];
    if (last && h.startMs <= last.endMs) {
      last.endMs = Math.max(last.endMs, h.endMs);
    } else {
      merged.push({ ...h });
    }
  }

  // 取补集
  const result: TimeInterval[] = [];
  let cursor = base.startMs;
  for (const h of merged) {
    if (h.startMs > cursor) result.push({ startMs: cursor, endMs: h.startMs });
    cursor = Math.max(cursor, h.endMs);
  }
  if (cursor < base.endMs) result.push({ startMs: cursor, endMs: base.endMs });
  return result;
}

export function buildSpanTree(spans: Span[]): SpanNode[] {
  const spanMap = new Map<string, SpanNode>();
  const roots: SpanNode[] = [];

  // 时间轴原点取全部 span 的最早时刻，区间均为相对该点的偏移
  const startTimes = spans.map((s) => new Date(s.timestamp).getTime());
  const traceStart = startTimes.length > 0 ? Math.min(...startTimes) : 0;

  for (const span of spans) {
    const offset = new Date(span.timestamp).getTime() - traceStart;
    spanMap.set(span.spanId, {
      span, children: [], depth: 0,
      selfDurationMs: span.durationMs,
      spanInterval: { startMs: offset, endMs: offset + span.durationMs },
      selfIntervals: [],
      isBoundary: false,
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
    node.selfIntervals = subtractIntervals(
      node.spanInterval,
      node.children.map((c) => c.spanInterval),
    );
    // 标量与区间必须同源 —— 旧实现直接减子 span 耗时之和，
    // 并行重叠时会重复扣减得出偏小的值
    node.selfDurationMs = node.selfIntervals.reduce(
      (sum, i) => sum + (i.endMs - i.startMs), 0,
    );
    node.children.forEach((c) => finalize(c, depth + 1));
  }
  roots.forEach((r) => finalize(r, 0));
  return roots;
}

/** 扁平化后的一行：节点 + 渲染树状连接线所需的祖先信息 */
export interface FlatRow {
  node: SpanNode;
  /** 各祖先层级是否还有后续兄弟（长度 = depth）—— 决定竖线画到哪一层 */
  ancestorHasNext: boolean[];
  /** 本行在同级中是否为最后一个 —— 决定画 └ 还是 ├ */
  isLastChild: boolean;
}

export function flattenTree(nodes: SpanNode[], collapsed: Set<string>): FlatRow[] {
  const result: FlatRow[] = [];
  function walk(node: SpanNode, ancestorHasNext: boolean[], isLastChild: boolean) {
    result.push({ node, ancestorHasNext, isLastChild });
    if (!collapsed.has(node.span.spanId)) {
      node.children.forEach((child, i) =>
        walk(
          child,
          [...ancestorHasNext, !isLastChild],
          i === node.children.length - 1,
        ),
      );
    }
  }
  nodes.forEach((root, i) => walk(root, [], i === nodes.length - 1));
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

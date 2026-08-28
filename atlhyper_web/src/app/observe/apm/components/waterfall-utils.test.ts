import { describe, it, expect } from "vitest";
import { buildSpanTree, flattenTree, focusSpanIds } from "./waterfall-utils";
import type { Span, TraceDetail } from "@/types/model/apm";

// 最小 span 构造器 —— 只填测试关心的字段
function span(p: Partial<Span> & Pick<Span, "spanId" | "serviceName">): Span {
  return {
    timestamp: "2026-08-28T11:26:23.000Z",
    traceId: "t1",
    parentSpanId: "",
    spanName: p.spanId,
    spanKind: "Server",
    duration: 0,
    durationMs: 100,
    statusCode: "Unset",
    statusMessage: "",
    resource: {},
    events: [],
    ...p,
  };
}

// 真实案例的形状：gateway 入口 → gateway client → user server
// （2026-08-28 register 500 的三段式 trace）
const registerSpans: Span[] = [
  span({ spanId: "gw-root", serviceName: "geass-gateway", spanName: "POST /api/auth/register", durationMs: 373 }),
  span({ spanId: "gw-client", parentSpanId: "gw-root", serviceName: "geass-gateway", spanKind: "Client", spanName: "HTTP POST", durationMs: 372 }),
  span({ spanId: "user-server", parentSpanId: "gw-client", serviceName: "geass-user", spanName: "POST /user.v1.AuthService/Register", durationMs: 367 }),
];

const registerTrace: TraceDetail = {
  traceId: "t1", durationMs: 373, serviceCount: 2, spanCount: 3, spans: registerSpans,
};

describe("buildSpanTree", () => {
  const tree = buildSpanTree(registerSpans);

  it("层级与深度正确", () => {
    expect(tree).toHaveLength(1);
    expect(tree[0].span.spanId).toBe("gw-root");
    expect(tree[0].children[0].span.spanId).toBe("gw-client");
    expect(tree[0].children[0].children[0].depth).toBe(2);
  });

  it("self time = 自身耗时减去直接子 span 之和", () => {
    expect(tree[0].selfDurationMs).toBeCloseTo(1); // 373 - 372
    expect(tree[0].children[0].selfDurationMs).toBeCloseTo(5); // 372 - 367
    expect(tree[0].children[0].children[0].selfDurationMs).toBeCloseTo(367); // 叶子
  });

  // 2026-08-29 期望值随实现更正：旧算法「父时长 − 子时长之和」在并行重叠时
  // 把重叠部分扣两次（100−80−80=−60 → 钳 0），得出「父自身零耗时」的假象。
  // 新算法先合并子区间再取补集：两个 [0,80] 合并后仍是 [0,80]，
  // 父在 [80,100] 确实是自己在干活 —— 20ms 是事实，0 是旧算法的产物。
  it("并行子调用重叠时只扣一次，不得出虚假的零 self time", () => {
    const parallel = [
      span({ spanId: "p", serviceName: "a", durationMs: 100 }),
      span({ spanId: "c1", parentSpanId: "p", serviceName: "a", durationMs: 80 }),
      span({ spanId: "c2", parentSpanId: "p", serviceName: "a", durationMs: 80 }),
    ];
    const t = buildSpanTree(parallel);
    expect(t[0].selfDurationMs).toBe(20);
    expect(t[0].selfIntervals).toEqual([{ startMs: 80, endMs: 100 }]);
  });

  it("服务边界只标在服务切换点", () => {
    expect(tree[0].isBoundary).toBe(false); // 根
    expect(tree[0].children[0].isBoundary).toBe(false); // gateway → gateway
    expect(tree[0].children[0].children[0].isBoundary).toBe(true); // gateway → user
  });
});

describe("focusSpanIds（高亮而非裁剪）", () => {
  it("focus 叶子服务：只有叶子在集合中 —— 但树本身不被裁剪", () => {
    const ids = focusSpanIds(registerTrace, "geass-user");
    expect(ids).toEqual(new Set(["user-server"]));
    // 完整树守护：focus 不改变 spans 本身
    expect(buildSpanTree(registerSpans)).toHaveLength(1);
    expect(flattenTree(buildSpanTree(registerSpans), new Set())).toHaveLength(3);
  });

  it("focus 根服务：入口 span 的所有后代都在集合中（含下游服务）", () => {
    const ids = focusSpanIds(registerTrace, "geass-gateway");
    expect(ids).toEqual(new Set(["gw-root", "gw-client", "user-server"]));
  });

  it("focus 不存在的服务：返回空集合（调用方视为不淡化）", () => {
    expect(focusSpanIds(registerTrace, "no-such-svc").size).toBe(0);
  });

  it("服务在 trace 中多次进出：每个入口的后代都收齐", () => {
    // a → b → a（回调）：focus b 应含 b 的入口及其后代（包括回到 a 的部分）
    const spans = [
      span({ spanId: "a1", serviceName: "a", durationMs: 100 }),
      span({ spanId: "b1", parentSpanId: "a1", serviceName: "b", durationMs: 80 }),
      span({ spanId: "a2", parentSpanId: "b1", serviceName: "a", durationMs: 20 }),
    ];
    const t: TraceDetail = { traceId: "t2", durationMs: 100, serviceCount: 2, spanCount: 3, spans };
    expect(focusSpanIds(t, "b")).toEqual(new Set(["b1", "a2"]));
  });
});

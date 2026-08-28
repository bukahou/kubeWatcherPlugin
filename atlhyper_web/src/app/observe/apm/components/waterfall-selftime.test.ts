import { describe, it, expect } from "vitest";
import { buildSpanTree } from "./waterfall-utils";
import type { Span } from "@/types/model/apm";

// ──────────────────────────────────────────────────────────────
// self-time 区间（包裹形态的核心）
// ──────────────────────────────────────────────────────────────
//
// 现状：self-time 只是一个数字（373-372=1ms），画成 bar 左端固定一截 ——
// 那是示意不是事实。网关的 self time 真实分布是「调用下游之前一小段 +
// 下游返回之后一小段」，中间那段时间它在等待，不是它在干活。
//
// 改造后：每个 span 算出 selfIntervals（自身耗时在时间轴上的真实区间），
// 被子调用占据的区间留空 —— 那段时间会在子 span 的行里着色。
// 每一毫秒只在它真正归属的层级上出现一次，嵌套感由此而来。

function span(p: Partial<Span> & Pick<Span, "spanId" | "serviceName">): Span {
  return {
    timestamp: "2026-08-28T11:26:23.000Z",
    traceId: "t1", parentSpanId: "", spanName: p.spanId, spanKind: "Server",
    duration: 0, durationMs: 100, statusCode: "Unset", statusMessage: "",
    resource: {}, events: [],
    ...p,
  };
}

/** 基准时刻 + 偏移毫秒 → ISO 串 */
const T0 = Date.parse("2026-08-28T11:26:23.000Z");
const at = (offsetMs: number) => new Date(T0 + offsetMs).toISOString();

describe("selfIntervals — self time 在时间轴上的真实位置", () => {
  it("单个叶子 span：整段都是自己的", () => {
    const [root] = buildSpanTree([span({ spanId: "leaf", serviceName: "a", durationMs: 100 })]);
    expect(root.selfIntervals).toEqual([{ startMs: 0, endMs: 100 }]);
  });

  it("真实案例：网关的 self time 分成调用前 + 返回后两段", () => {
    // gw-root 0→373；gw-client 1→373（几乎立刻发起，等到最后）
    // 真实形态：网关自己只在 0-1ms 和 373-373ms 干活，中间在等
    const spans = [
      span({ spanId: "gw", serviceName: "gateway", timestamp: at(0), durationMs: 373 }),
      span({ spanId: "call", parentSpanId: "gw", serviceName: "gateway", timestamp: at(1), durationMs: 371 }),
    ];
    const [root] = buildSpanTree(spans);
    // 子调用占据 [1, 372]，父自身区间应为 [0,1] 与 [372,373]
    expect(root.selfIntervals).toEqual([
      { startMs: 0, endMs: 1 },
      { startMs: 372, endMs: 373 },
    ]);
    // 总量仍等于标量 self time（2ms），两种表示必须一致
    const total = root.selfIntervals.reduce((s, i) => s + (i.endMs - i.startMs), 0);
    expect(total).toBeCloseTo(root.selfDurationMs);
  });

  it("串行子调用：父的空隙出现在两个子调用之间", () => {
    // p 0→100；c1 10→30；c2 60→90 → 父自身区间 [0,10] [30,60] [90,100]
    const spans = [
      span({ spanId: "p", serviceName: "a", timestamp: at(0), durationMs: 100 }),
      span({ spanId: "c1", parentSpanId: "p", serviceName: "a", timestamp: at(10), durationMs: 20 }),
      span({ spanId: "c2", parentSpanId: "p", serviceName: "a", timestamp: at(60), durationMs: 30 }),
    ];
    const [root] = buildSpanTree(spans);
    expect(root.selfIntervals).toEqual([
      { startMs: 0, endMs: 10 },
      { startMs: 30, endMs: 60 },
      { startMs: 90, endMs: 100 },
    ]);
  });

  it("并行子调用重叠：占用区间合并，不重复扣减", () => {
    // p 0→100；c1 10→60；c2 30→80（重叠）→ 占用 [10,80]，父自身 [0,10] [80,100]
    const spans = [
      span({ spanId: "p", serviceName: "a", timestamp: at(0), durationMs: 100 }),
      span({ spanId: "c1", parentSpanId: "p", serviceName: "a", timestamp: at(10), durationMs: 50 }),
      span({ spanId: "c2", parentSpanId: "p", serviceName: "a", timestamp: at(30), durationMs: 50 }),
    ];
    const [root] = buildSpanTree(spans);
    expect(root.selfIntervals).toEqual([
      { startMs: 0, endMs: 10 },
      { startMs: 80, endMs: 100 },
    ]);
    // 标量 self time 按合并后计算：[0,10]=10ms + [80,100]=20ms = 30ms
    // （而非旧算法 100-50-50=0 —— 重叠部分被扣了两次）
    expect(root.selfDurationMs).toBeCloseTo(30);
  });

  it("子调用超出父范围（异步/时钟偏移）：裁剪到父区间内，不产生负数或越界", () => {
    const spans = [
      span({ spanId: "p", serviceName: "a", timestamp: at(0), durationMs: 100 }),
      span({ spanId: "c", parentSpanId: "p", serviceName: "a", timestamp: at(80), durationMs: 200 }),
    ];
    const [root] = buildSpanTree(spans);
    expect(root.selfIntervals).toEqual([{ startMs: 0, endMs: 80 }]);
    expect(root.selfDurationMs).toBeCloseTo(80);
  });

  it("子调用早于父开始（时钟偏移）：不产生负区间", () => {
    const spans = [
      span({ spanId: "p", serviceName: "a", timestamp: at(10), durationMs: 100 }),
      span({ spanId: "c", parentSpanId: "p", serviceName: "a", timestamp: at(0), durationMs: 30 }),
    ];
    const [root] = buildSpanTree(spans);
    // 父 [10,110]，子裁剪后占 [10,30] → 自身 [30,110]
    expect(root.selfIntervals).toEqual([{ startMs: 30, endMs: 110 }]);
    root.selfIntervals.forEach((i) => expect(i.endMs).toBeGreaterThan(i.startMs));
  });

  it("子调用铺满父区间：自身区间为空数组，不是 [0,0]", () => {
    const spans = [
      span({ spanId: "p", serviceName: "a", timestamp: at(0), durationMs: 100 }),
      span({ spanId: "c", parentSpanId: "p", serviceName: "a", timestamp: at(0), durationMs: 100 }),
    ];
    const [root] = buildSpanTree(spans);
    expect(root.selfIntervals).toEqual([]);
    expect(root.selfDurationMs).toBe(0);
  });
});

describe("serviceSelfTotals — 图例上的各服务耗时汇总", () => {
  it("按服务累加 self time，回答「每层花了多久」", () => {
    // gateway 自身 2ms，user 自身 367ms —— 这正是用户要的「网关内 vs user 内」
    const spans = [
      span({ spanId: "gw", serviceName: "gateway", timestamp: at(0), durationMs: 373 }),
      span({ spanId: "call", parentSpanId: "gw", serviceName: "gateway", timestamp: at(1), durationMs: 371 }),
      span({ spanId: "usr", parentSpanId: "call", serviceName: "user", timestamp: at(2), durationMs: 367 }),
    ];
    const [root] = buildSpanTree(spans);
    // gateway: root 自身 [0,1]+[372,373]=2ms，call 自身 371-367=4ms → 共 6ms
    // user: 367ms
    const totals = new Map<string, number>();
    const walk = (n: typeof root) => {
      totals.set(n.span.serviceName, (totals.get(n.span.serviceName) ?? 0) + n.selfDurationMs);
      n.children.forEach(walk);
    };
    walk(root);
    expect(totals.get("gateway")).toBeCloseTo(6);
    expect(totals.get("user")).toBeCloseTo(367);
  });
});

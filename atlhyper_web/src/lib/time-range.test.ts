import { describe, it, expect } from "vitest";
import {
  toSLOWindow,
  formatRangeParam,
  parseRangeParam,
  SLO_WINDOWS,
} from "./time-range";
import type { TimeRangeSelection } from "@/types/time-range";

describe("toSLOWindow", () => {
  // SLO 只有五个预聚合窗口（Agent 的 sloWindowConfigs），任意时间范围必须贴合到其中之一。
  // 向上取而不是向下取：向下取会让统计窗口比用户要求的短，可用率会失真；
  // 向上取只是范围更宽，语义安全。
  const cases: [TimeRangeSelection, string][] = [
    [{ mode: "preset", preset: "1h" }, "1h"],
    [{ mode: "preset", preset: "24h" }, "24h"],
    [{ mode: "preset", preset: "7d" }, "7d"],
    // 比最小窗口还短 → 用最小窗口
    [{ mode: "preset", preset: "15min" }, "1h"],
    [{ mode: "custom", value: 30, unit: "m" }, "1h"],
    // 落在两个窗口之间 → 向上取
    [{ mode: "custom", value: 3, unit: "h" }, "6h"],
    [{ mode: "custom", value: 2, unit: "d" }, "3d"],
    // 超过最大窗口 → 钳到最大（ClickHouse 只有 7 天数据）
    [{ mode: "preset", preset: "30d" }, "7d"],
    [{ mode: "custom", value: 90, unit: "d" }, "7d"],
  ];
  for (const [sel, want] of cases) {
    it(`${JSON.stringify(sel)} → ${want}`, () => {
      expect(toSLOWindow(sel).window).toBe(want);
    });
  }

  it("恰好等于某个窗口时不算降级", () => {
    // 预设里没有 6h，用同样恰好落在窗口上的 24h
    expect(toSLOWindow({ mode: "preset", preset: "24h" }).degraded).toBe(false);
    expect(toSLOWindow({ mode: "custom", value: 6, unit: "h" }).degraded).toBe(false);
  });

  it("需要贴合时标记为降级 —— 静默换掉窗口正是上一版那个 bug 的成因", () => {
    const r = toSLOWindow({ mode: "custom", value: 30, unit: "m" });
    expect(r.degraded).toBe(true);
    expect(r.window).toBe("1h");
  });

  it("绝对时间范围按跨度贴合", () => {
    const end = 1_724_500_000_000;
    const r = toSLOWindow({ mode: "absolute", start: end - 2 * 3_600_000, end });
    expect(r.window).toBe("6h"); // 2 小时 → 向上取 6h
    expect(r.degraded).toBe(true);
  });

  it("五个窗口与 Agent 的 sloWindowConfigs 一一对应", () => {
    expect(SLO_WINDOWS).toEqual(["1h", "6h", "24h", "3d", "7d"]);
  });
});

describe("URL 参数编解码", () => {
  // 时间轴要能通过 URL 分享，三种模式都得能往返
  const roundTrip: TimeRangeSelection[] = [
    { mode: "preset", preset: "1h" },
    { mode: "preset", preset: "7d" },
    { mode: "custom", value: 45, unit: "m" },
    { mode: "custom", value: 3, unit: "d" },
    { mode: "absolute", start: 1_724_400_000_000, end: 1_724_500_000_000 },
  ];
  for (const sel of roundTrip) {
    it(`往返: ${JSON.stringify(sel)}`, () => {
      const params = new URLSearchParams(formatRangeParam(sel));
      expect(parseRangeParam(params)).toEqual(sel);
    });
  }

  it("没有参数时返回 null，交由调用方决定默认值", () => {
    expect(parseRangeParam(new URLSearchParams())).toBeNull();
  });

  it("非法参数返回 null 而不是抛异常 —— URL 是用户可编辑的", () => {
    expect(parseRangeParam(new URLSearchParams("range=abc"))).toBeNull();
    expect(parseRangeParam(new URLSearchParams("range=10x"))).toBeNull();
    expect(parseRangeParam(new URLSearchParams("from=1&to=notanumber"))).toBeNull();
    // to 早于 from
    expect(parseRangeParam(new URLSearchParams("from=200&to=100"))).toBeNull();
  });

  it("preset 用可读 key，custom 用值+单位 —— URL 要人能看懂", () => {
    expect(formatRangeParam({ mode: "preset", preset: "24h" })).toBe("range=24h");
    expect(formatRangeParam({ mode: "custom", value: 45, unit: "m" })).toBe("range=45m");
  });
});

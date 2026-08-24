# 观测模块统一时间轴与信号联动

> 已完成并归档。原任务追踪见 `docs/tasks/active/tracker.md` 的历史版本。
> 设计文档：`docs/design/archive/observe-unified-timerange.md`
> 完成时间：2026-08-24

## 观测模块统一时间轴与信号联动 — 🔄 进行中

> 原设计文档: [observe-unified-timerange.md](../../design/active/observe-unified-timerange.md)
>
> 四页各写各的：三套时间机制、两套刷新机制；空数据分不清「没流量」还是「采集挂了」；
> 跨信号跳转只有 Logs→APM 一条单向链路。

- Phase 1: 全局时间轴 — ✅ 代码完成（da26567）
  - timeRangeStore + useObserveTimeRange（能力声明 full/sloWindows/trendOnly）
  - SLO 贴合到五个预聚合窗口并显式提示；Metrics 范围只作用趋势图
  - URL 同步（router.replace），持久化 URL > localStorage > 默认 1h
  - 引入 vitest（项目此前无前端测试框架），21 例
- Phase 2: 数据新鲜度 — ✅ 代码完成（bed117d）
  - model_v3.SignalFreshness；Agent 三张表各查 max(时间列)，挂 Dashboard 门面
  - Master 判定 live/idle/stale/absent —— 靠 metrics（拉取式）与 traces/logs（请求触发）
    的非对称区分「没流量」与「采集挂了」
  - 四页头部统一挂 SignalFreshnessBadge
- Phase 3: 信号联动 — ✅ 代码完成（本次提交）
  - SLO → APM / SLO → Logs(ERROR) / APM trace → Logs
  - 顺带统一 trace 参数名（Logs 页此前读 traceId，与另两处不一致）
- 待部署验证：agent v0.6.1 / controller v0.5.0 / web v0.6.0 构建中

## 上线结果

线上版本：agent v0.6.5 / controller v0.5.0 / web v0.6.1

实测验证（headless Chromium 抓渲染后 DOM）：
- 四页共享时间轴，切换页面保持，URL 可分享
- 信号新鲜度徽章正常：`metrics live · 刚刚` / `traces idle · 27 分钟前`
  （凌晨无人访问 geass 的真实场景，页面能自己说清是「没流量」而非「采集挂了」）
- Metrics 页显示「时间范围仅作用于趋势图，卡片与硬件矩阵始终显示当前值」

## 过程中修的问题（不在原计划内）

1. SLO 前端仍传 `1d`，后端窗口集已改 → 静默降级成 5 分钟数据却标着「1 天」（本轮自己引入）
2. `getOTelSnapshot` 的 early return 让新增字段静默失效，且「回退旧快照」对上层不可见
3. 新鲜度查询继承的 ctx 已被前面十几个查询耗尽 → 永远 deadline exceeded
   （手工执行只要 2 秒；解法与 collectSLOWindows 一致：独立 context）

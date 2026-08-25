# SLO 面板重构 — 任务归档

> 原设计文档: [slo-panel-redesign.md](../../design/archive/slo-panel-redesign.md)
> 完成：2026-08-25
> 线上：agent v0.6.5 / controller v0.5.2 / web v0.6.2

## 背景

SLO 契约改造（`ingress_*` 归一化）完成后面板恢复了数据流，
但 2026-08-23 实测发现**页面上每一个数字都是错的**，且缺少主流 SLO 实践的核心要素。

## Phase 1: 多实例聚合修复 + 真实域名 — ✅

- cilium-envoy 是 DaemonSet，多实例各自维护独立累积计数器。
  所有 counter/histogram 聚合加 `ResourceAttributes['service.instance.id']` 分区，
  各实例分别算 delta 再汇总（5 处查询全部重写为三层嵌套）
- 修复前 atlhyper-web 5 分钟 2xx 显示 238,679，实际 10 —— **失真 24000 倍**
- P95 由恒为 0 恢复为真实值（histogram delta 跨实例后差值为负被吃掉）
- 域名分组改用 `IngressSLO.DisplayName`（Agent 侧读 HTTPRoute 已解析好），
  不再显示 `geass-v3/geass-gateway` 这种内部标识

## Phase 2: 燃烧率 + 事件计数预算 + 目标模型 — ✅

- 新增 `atlhyper_master_v2/slo/burnrate.go`：`CalculateBurnRate` /
  `CalculateErrorBudget` / `burnRateStatus` / `EstimateExhaustHours`
- 多窗口燃烧率（Google SRE Workbook）：1h>14.4× / 6h>6× / 24h>3× / 3d>1×
- 错误预算改事件计数口径：「允许错 7 个，已经错了 1 个」而非「剩余 82.3%」——
  低流量下百分比极具误导性
- `slo_targets` 表去掉 `time_range`、新增 `window_days`（默认 7）：
  一个域名 = 一个固定窗口 + 一组目标，对齐 SLO 定义

## Phase 3: 详情区收敛 — ✅

- 删除 `DomainCard` / `DomainSummaryRow` / `CompareTab` / `OverviewTab`（552 行）
- 新增 `DomainDetail` / `BurnRateTable` / `GoodBadCount`
- 清单表内联展开（`<Fragment>` + `colSpan`），一个域名只渲染一次

**与原方案的出入**：原计划的「状态码构成 + 详情页跳 APM」**已撤回**。
状态码是算可用率的同一份数据换个形状，不产生新信息；跳转清单表每行已有。
当前 1 域名 = 1 服务的规模下，接口级下钻省的只是「去 APM 选一下服务」。
真正需要结构化下钻的时机是 AIOps 要自动推理「为什么超支」的时候。

## Phase 4: 长窗口存储 — ✅ 决策为 A（维持现状）

ClickHouse 与 Master SQLite 均为 emptyDir，Pod 重启数据归零。
三个选项（A 维持现状 / B 给 SQLite 持久卷 / C 加 rollup 表），
**用户决策：A**。

代码已是 A 的形态，零改动：

| 项 | 现状 |
|---|---|
| `defaultSLOWindowDays` | 7（对齐 ClickHouse TTL） |
| `slo_targets.window_days` | `DEFAULT 7` |
| `SLO_WINDOWS`（前端） | `1h/6h/24h/3d/7d`，无 30d |

## 修复后照出来的真实问题

面板修好后 `geass-api.bukahou.com` 显示 24h 内 123 个 5xx、可用率 86.1%、
燃烧率 13.9×，而修复前该行显示「可用率 100%、P95 0ms」。

**2026-08-25 查证：不是业务 bug。**按小时拆分 5xx 后，
109 + 14 = 123 全部落在 UTC 08-23 01:00–02:00（JST 10:00–11:17），
且只有 geass 两个服务有错 —— 与
[2026-08-23 ARP MAC 错配故障](~/AtlHyper/GitHub/config/clusters/incidents/2026-08-23-arp-mac-mismatch.md)
的时间线和影响面完全吻合（该档案记载「仅 geass 相关域名 503，其他全程正常」）。

**这反过来验证了 SLO 面板的价值**：修复后的数字能与独立记录的故障档案对上，
说明 counter 聚合口径是对的。

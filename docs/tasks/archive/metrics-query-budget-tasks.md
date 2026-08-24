# 指标采集查询预算优化

> 已完成并归档。
> 故障档案：config 仓 `clusters/incidents/2026-08-24-atlhyper-metrics-blank-page.md`
> 完成时间：2026-08-24

## 指标采集查询预算 — 🔄 收尾

> 故障档案: config 仓 `clusters/incidents/2026-08-24-atlhyper-metrics-blank-page.md`
>
> 每节点 27 个查询 × 7 节点串行共享 30 秒 ctx，后半段节点被 context 取消，数据静默缺失。

- 采集不全时打日志（detectMissingParts）— ✅ 完成（e160ad8）
- 标量 gauge 六合一，每节点 27 → 18 — ✅ 完成，agent v0.6.5 已上线
  - 实测 7 节点数据全部完整（此前 raspi5-one / raspi5-zero 长期缺 disks/networks/sensors）
  - 待观察：ClickHouse Broken pipe 计数是否同步下降
- 进一步压缩（fillCPU 4 个 / fillNetworks 2 个 / fillDisks 2 个）— 待议，收益递减
- 跨节点批量查询（一个查询取全部节点的标量）— 待议，需重构 buildNodeMetrics 结构

## 实测效果（agent v0.6.5）

| 指标 | 优化前 | 优化后 |
|---|---|---|
| ClickHouse 平均查询耗时 | 440 ms | ~200 ms |
| `Broken pipe`（查询被 ctx 取消） | 290 次 / 30 分钟 | 18 次 / 45 分钟（↓96%） |
| 节点数据完整性 | raspi5-one / raspi5-zero 长期缺 disks+networks+sensors | 7/7 完整 |

剩余两项（继续压缩 fillCPU/fillNetworks/fillDisks、跨节点批量查询）在当前集群规模下
收益递减，**失败率回升再动**，届时优先「跨节点批量」。

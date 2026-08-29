# 底层能力盘点 —— 有基础但未实现/未展示的功能

> 状态：active
> 盘点日期：2026-08-29
> 起因：用户观察「底层已有不少功能只差继续实现或前端展示」。
> 本次 APM 重构已两次证实该判断（LogAttributes 端到端透传却从不渲染、
> self-time 早已算出却没画上去），故做一次系统扫描。

## 方法

四路交叉扫描，逐条实测而非读代码推断：

1. Agent 侧采集/查询能力清单（`repository/interfaces.go`，81 个方法）
2. 快照模型字段（`model_v3/cluster/snapshot.go`）
3. Master 端点注册表（`gateway/routes.go`，**104 个端点**）
4. 前端实际调用（`atlhyper_web/src/` 全量 grep，**84 个端点**）

差集 → 逐个 curl 实测确认「是没数据还是没接线」。

## 结论：5 个真空缺口，后端全部完整且有真实数据

| # | 端点 | 后端状态 | 实测数据 | 前端 |
|---|---|---|---|---|
| **G1** | `/api/v2/aiops/baseline` | handler + service + `aiops/baseline/` 引擎齐全 | ✅ **有真实 EMA 数据** | ❌ 零引用 |
| **G2** | `/api/v2/aiops/incidents/patterns` | handler + `GetIncidentPatterns` | ⚠️ 返回 null（无历史事件） | ❌ 零引用 |
| **G3** | `/api/v2/aiops/graph/trace` | handler 完整（含 direction/depth 参数） | ⚠️ 空图（依赖图未构建足够数据） | ❌ 零引用 |
| **G4** | `/api/v2/events/by-resource` | `EventHandler.ListByResource` | ⚠️ 空（参数正确但无匹配事件） | ❌ 零引用 |
| **G5** | `/api/v2/observe/logs/summary` | `ObserveHandler.LogsSummary` | ⚠️ 空（`latestAt` 为零值） | ❌ 零引用 |

> 另有 `/api/deploy/*`（7 个）、`/api/github/*`（3 个）、`/api/v2/ai/reports`
> 前端**有**引用（`grep` 命中 1–2 个文件），非缺口，排除。`/health` 无需 UI。

---

## G1：AIOps 基线 —— 唯一「数据已在、只差展示」的完整缺口 ⭐

**这是本次盘点最有价值的发现。**

实测（`entity=kube-system/pod/cilium-envoy-flqc6`）：

```json
{"entityKey":"kube-system/pod/cilium-envoy-flqc6","states":[
  {"metricName":"not_ready_containers",   "ema":0, "variance":0, "count":53, "consecutiveZero":53},
  {"metricName":"max_container_restarts", "ema":0, "variance":0, "count":53},
  {"metricName":"restart_count",          "ema":0, "variance":0, "count":53},
  {"metricName":"is_running",             "ema":1, "variance":0, "count":53}
]}
```

- 每个实体 **4 个指标**的 EMA 基线与方差，`count=53` 表示已累积 53 个采样周期
- 风险实体侧实测 **20 个实体**在被跟踪（当前全部 `healthy`）
- **这些数字一直在算、一直在更新，但界面上没有任何地方能看到**

**价值**：基线是「这个实体平时长什么样」的量化描述 —— 它回答的是
「现在这个值算不算异常」，而现有 UI 只能回答「现在这个值是多少」。
CLAUDE.md 定义的 L3（推理引擎）能力，底层已就位、展示层缺席。

**接线成本**：低。端点已可用，参数为 `?cluster=<id>&entity=<key>`
（注意：**不是** `cluster_id`，与其余端点不一致 —— 见下方 R1）。

---

## G2–G5：接线成本同样低，但当前无数据可展示

四者共性：**后端完整、参数正确、返回结构正常，只是当前环境下内容为空**。

- **G2 事件模式**：需要历史 incident 积累。集群近期健康（20 实体全 healthy），
  故为 null —— 属正常，非缺陷
- **G3 依赖图追踪**：`graph/trace` 支持 upstream/downstream + depth 遍历，
  但依赖图需 APM 拓扑边喂养；实测返回空图
- **G4 按资源查事件**：参数为 `cluster_id/kind/namespace/name`（已读码确认），
  实测 clickhouse-0 无事件返回空
- **G5 日志摘要**：`totalEntries=0`、`latestAt` 为 1970 零值 —— ClickHouse
  8-29 凌晨重启后日志重新积累，且该端点可能与现用的 `logs/query` 口径不同

**判断**：这四个不宜「为接线而接线」。若接了 UI 却长期显示空白，
反而制造「功能坏了」的错觉 —— 应在有数据的场景下验证后再接。

---

## 盘点中发现的横向问题

### R1：端点参数命名不一致 ⚠️

| 端点 | 集群参数名 |
|---|---|
| 绝大多数（`/observe/*`、`/events`、资源类） | `cluster_id` |
| **`/aiops/*`**（baseline、risk/entities、graph/trace） | **`cluster`** |

本次盘点中我按 `cluster_id` 调 `aiops/baseline` 得到
`{"error":"missing cluster parameter"}`，一度误判为「端点无数据」。

**这不只是美观问题**：前端接线时若沿用惯例会直接失败，而错误信息
（"missing cluster parameter"）不会提示正确参数名是什么。

**建议**：aiops 系列兼容接受 `cluster_id`（保留 `cluster` 向后兼容），
或至少在错误信息中写明期望的参数名。

### R2：能力清单与展示层缺少对账机制

本次是**人工**做的四路交叉扫描。104 端点 vs 84 调用的差集，
在此之前没有任何机制会提示。同类问题（LogAttributes 透传却不渲染、
self-time 算了不画）已发生三次。

**建议**（成本递增，任选）：
- 轻量：把本文档的对账命令固化为脚本，随版本发布跑一次
- 中等：CI 中比对 `routes.go` 注册表与前端 grep 结果，差集超阈值告警
- 重量：端点注册时打标（`experimental` / `ui-wired` / `internal`），
  未打标即视为待接线

---

## 建议的执行顺序

1. **G1 基线展示**（唯一有真实数据的缺口，价值最高、成本最低）
2. **R1 参数统一**（顺手做，避免后续接线反复踩）
3. G3 依赖图追踪 —— 待 APM 拓扑数据充分后重新评估
4. G2 / G4 / G5 —— 等各自有数据的场景出现再接，避免展示空白

**明确不做**：为凑「功能完整」而接入长期空白的面板。
空面板比没有面板更糟 —— 它让人以为功能坏了。

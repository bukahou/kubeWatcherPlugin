# APM 面板重构设计 —— 对齐 Elastic APM 的排查体验

> 状态：active
> 创建：2026-08-28
> 参照系：Elastic APM (Kibana)，调研来源见附录 C

## 背景

2026-08-28 排查 geass 注册接口 500 时，APM 面板暴露出一条完整的「排查者视角失效链」：

| # | 现象 | 根因 | 定性 |
|---|------|------|------|
| 1 | 从 geass-user 进入 trace，瀑布图只有 1 个 span，无层级 | 前端 `filterTraceForService` 裁剪：只保留 focus 服务入口 span 及**后代**，祖先被丢弃 | 设计缺陷 |
| 2 | Span 状态 Error，关联日志只有一句「ERROR 请求完成」 | ① 日志按当前服务过滤，gateway 那条含 `exception.message` 的日志被挡住 ② `LogAttributes` 完全没渲染 | 设计缺陷 × 2 |
| 3 | 列表显示「1 Span 数 \| 1 服务数」，实际 3 span 2 服务 | `ListTraces` 的 WHERE 过滤作用在 `GROUP BY TraceId` **之前**，聚合的是残缺集合 | **Bug** |
| 4 | 全 error 的 trace 里看不出服务边界 | `SpanRow` 在 `isError` 时用红色**覆盖**服务色——颜色同时编码两个维度，冲突时服务身份消失 | 设计缺陷 |

四个问题叠加的结果：**站在报错服务的视角（恰恰是排查者最常进入的视角），既看不到调用它的上游，也看不到上游手里的错误详情**。真正的根因 `Data truncated for column 'auth_type'` 一直躺在 ClickHouse 里，UI 一个字都没显示。

### 与 Elastic APM 的对照（差距清单）

| ES APM 能力 | AtlHyper 现状 |
|---|---|
| Waterfall 颜色按服务分配，错误用标记而非换色 | 有 `SERVICE_COLORS` 但 error 覆盖服务色 |
| View full trace / focus 切换，上下文不丢 | focus 即裁剪，祖先不可见 |
| 日志按 `trace.id` 关联，跨服务全量 | 按当前服务过滤，跨服务日志被挡 |
| Span flyout 展示完整错误（exception + stack trace） | 只展示 `StatusMessage`（gRPC/HTTP 场景常为空） |
| 延迟分布直方图可框选 → 该区间的 trace 样本 | 直方图只读 |
| Correlations：自动找出慢/错请求中显著超标的属性 | 无 |
| Dependencies：DB/外部服务作为被调方的独立视图 | 仅服务详情内有 DBStats/HTTPStats |
| Transaction 抽象：入口请求是一等公民 | 每次查询用 `argMinIf` 临时推断根，且有 bug |

## 目标

1. **瀑布图一眼能读**：颜色只编码服务身份，错误、类型、时间各用独立视觉通道
2. **错误证据链不断**：从任何 span 进入，都能拿到整条 trace 的错误详情与跨服务日志
3. **列表数字与事实一致**：spanCount / serviceCount / rootService 反映完整 trace
4. **补齐两个高价值分析能力**：延迟分布下钻、Correlations 相关性分析

## 设计原则（先于功能）

| 原则 | 落实方式 |
|---|---|
| **标准 > 私有** | **不引入** Elastic 的 Transaction/Span 双文档模型。保持 OTel 单一 span 模型，把「入口 span」定义为**查询层约定**（固化 SQL 常量 + 单测），语义上等价于 ES 的 Transaction，但不偏离 OTel 存储 |
| **大后端小前端** | Correlations 打分、日志归并、错误证据链组装、self-time 计算全部在 Agent/Master 完成；前端只渲染 |
| **一个视觉通道只编码一个维度** | 颜色=服务，红色描边+图标=错误，缩进=层级，长度/偏移=时间，bar 内深浅分段=self/等待 |
| **契约层** | 入口 span 判定规则做成共享常量，配 `TestEntrySpanRule` 防止各查询各自为政 |

## 核心架构（数据流不变，只改查询与展示）

```
otel_traces (ClickHouse)
   │  Agent repository/ch/query/trace.go   ← 功能三修 ListTraces；功能五新增 correlation 查询
   ▼
model_v3/apm                               ← 功能二增 SpanError 证据链字段；功能五增 Correlation 类型
   │  Command/MQ（现有通道，无协议变更）
   ▼
Master gateway/handler/observe             ← 功能四增 max_duration 参数；功能五新 endpoint
   ▼
Web /observe/apm                           ← 功能一/二重构 waterfall 与详情；功能四/五新交互
```

---

## 功能一：Waterfall 服务视觉重构

### 用户故事

打开任何 trace，不看文字就能分辨「哪段是网关、哪段是下游、错误发生在哪个服务」；
从子服务进入时上游依然可见，只是视觉上突出当前服务。

### 视觉规则（全部为前端实现，数据已具备）

**1. 颜色只编码服务。**
`SERVICE_COLORS`（8 色）按 trace 内服务**首次出现顺序**分配（现状保留）。
`SpanRow` 删除 `isError ? "#ef4444" : color` 的覆盖逻辑——**error 不再改变 bar 颜色**。

**2. 错误改用独立通道。**
error span：bar 右端红色 `AlertCircle` 图标 + `1.5px` 红色描边 + span 名文字红色。
服务色保留在 bar 填充与左侧粗边条上。

**3. focus 从「裁剪」改为「高亮」。**
`filterTraceForService` 改名 `focusTraceOnService`，语义变更：
- 返回**完整** span 树，附加 `focusedSpanIds: Set<string>`（focus 服务入口 span 及后代）
- 非 focus 的 span 行整体 `opacity: 0.45`，但服务色条保持不透明（上下文可见，身份不丢）
- 顶部提供「查看完整链路」切换（对应 ES 的 View full trace），点击清除 focus

**4. 服务图例升级为控件。**
现有图例（色点+名称）增加：span 数徽标、单击 toggle 该服务高亮（对应 ES 的 legend toggle）、
当前 focus 服务加边框标识。

**5. 跨服务边界显式标注。**
子 span 的 `serviceName ≠` 父 span 时，span 名左侧渲染服务色小标签（服务名缩写），
对应 ES 在服务切换点打服务标记的做法——层级靠缩进，边界靠标签，不再要求用户对比颜色脑补。

**6. self time 上 bar。**
`SpanDrawer` 已计算 selfDuration（span 总耗时 − 子 span 耗时）。将其搬到 bar 内：
深色段 = self time，浅色段 = 等待子调用。gateway 372ms 里 367ms 在等 user 的场景一眼可读。

**7. 每行右侧固定列：耗时 + 占 trace 总时长百分比**（ES 同款）。

### 前端实现

```
atlhyper_web/src/app/observe/apm/components/
├── waterfall-utils.ts    [修改] focusTraceOnService 替代 filterTraceForService
│                                 SpanNode 增加 isFocused / isServiceBoundary / selfDurationMs
├── SpanRow.tsx           [修改] 视觉规则 1/2/5/6/7
├── TraceWaterfall.tsx    [修改] 图例控件化、focus 切换、完整链路按钮
└── trace-utils.ts        [删除] filterTraceForService（逻辑并入 waterfall-utils）
atlhyper_web/src/app/observe/apm/page.tsx  [修改] focusedTrace 改用新语义
```

树构建、self-time、边界标记均为纯函数 → **vitest 单测**（含：叶子服务 focus 时全树仍在、
selfDuration 对多子并行调用不为负、边界标签只出现在服务切换点）。

---

## 功能二：错误证据链

### 用户故事

点开一个 Error span，直接看到「哪一层、什么异常、什么消息」；
切到日志 tab，看到**整条 trace 全部服务**的日志，含结构化字段，红色的错误日志能展开出根因。

### 问题本质

错误详情在三个地方，现状只读了最弱的一个：

| 来源 | 现状 | 本次 register 案例 |
|---|---|---|
| Span `Events` 里的 exception 事件 | ✅ 已解析（`SpanError`） | 空（connect-go 未记录 event） |
| Span `StatusMessage` | ✅ 已展示 | 空 |
| **同 trace 的 ERROR 日志的 `LogAttributes`** | ❌ 完全未用 | **`exception.message` = 'Data truncated for column auth_type'** |

### API 变更

**1. Trace 详情响应：span 增加日志回填的错误证据（Agent 侧组装）**

`GetTraceDetail` 在查完 spans 后**追加一次日志查询**（`WHERE TraceId = ? AND SeverityText = 'ERROR'`），
按 SpanId 归属（日志带 SpanId 则精确匹配；不带则归入时间窗最近的同服务 span），组装：

```go
// model_v3/apm/trace.go
type SpanError struct {
    Type       string `json:"type"`
    Message    string `json:"message"`
    Stacktrace string `json:"stacktrace,omitempty"`
    Source     string `json:"source"` // "span_event" | "status_message" | "trace_log" ← 新增
}
```

填充优先级：`span_event` > `status_message` > `trace_log`。前端在错误卡片上标注来源
（"来自网关日志" 之类），让排查者知道证据是从哪一层拿到的。

**2. 日志 tab：默认全量，响应携带结构化字段**

现状 `SpanLogs` 传 `services: [serviceName]`——**删除此默认**，改为：

```
GET (Command) query_logs { trace_id }           ← 默认：整条 trace 全部服务
可选 chip 过滤：服务多选（前端对已加载数据过滤，不重新请求）
```

日志响应确认透传 `LogAttributes`（`repository/ch/query/log.go` 需核对字段是否已在 SELECT 中；
缺则补）。响应示例：

```json
{
  "timestamp": "2026-08-28T11:26:23.982Z",
  "severity": "ERROR",
  "service": "geass-gateway",
  "body": "BFF 下游调用失败",
  "attributes": {
    "code": "internal",
    "exception.message": "internal: Error 1265 (01000): Data truncated for column 'auth_type' at row 1",
    "exception.type": "*connect.Error",
    "path": "/api/auth/register"
  }
}
```

### 前端实现

```
SpanDrawer.tsx   [修改] 错误卡片：Type/Message/Source 标注；Message 可复制
SpanLogs.tsx     [修改] 去掉 service 强制过滤；每行前加服务色点（复用 waterfall 的 serviceColorMap）；
                        ERROR 行可展开渲染 attributes 键值表（exception.* 置顶、红色）
```

### 数据对照

| 前端字段 | Go struct | ClickHouse 来源 |
|---|---|---|
| `span.error.source` | `SpanError.Source` | 组装时标注 |
| `log.attributes` | `log.Entry` 的 Attributes map | `otel_logs.LogAttributes` |

---

## 功能三：Trace 列表聚合修正 + 入口 span 语义固化

### Bug 修正（ListTraces）

现状（过滤先于聚合，聚合残缺集合）：

```sql
FROM otel_traces
WHERE ServiceName = ? AND SpanName = ?     -- 先剔除其他服务的 span
GROUP BY TraceId                            -- 再聚合 → spanCount/serviceCount/rootSvc 全错
```

实测（trace `90453b...`）：当前实现得 `1 span / 1 服务`，正确为 `3 span / 2 服务`。

修法（两段式，Jaeger/Tempo 同款）：

```sql
FROM otel_traces
WHERE TraceId IN (
    SELECT DISTINCT TraceId FROM otel_traces
    WHERE <时间窗> AND ServiceName = ? AND SpanName = ? AND ...   -- 条件只用于定位
)
AND <时间窗>                                                       -- 外层保留时间窗利用主键索引
GROUP BY TraceId                                                   -- 聚合完整 trace
```

> 性能注：内层子查询与外层共用时间窗过滤，ClickHouse 对 `Timestamp` 主键的裁剪仍然生效；
> 7 天 TTL 下 trace 总量在万级，无需物化优化。

### 入口 span 语义固化（对齐 ES 的 Transaction，不改存储模型）

现状 `rootSvc/rootOp` 的推断逻辑内联在 `ListTraces` 的 SQL 里。抽成共享常量：

```go
// model_v3/apm/enum.go
// SQLEntrySpanCondition 入口 span（≈ Elastic APM 的 Transaction）判定：
// SpanKind = Server 的最早 span；退化场景（无 Server span）取无父 span。
// 所有需要「trace 的根是谁」的查询必须引用本常量，禁止各自内联。
```

配套 `TestEntrySpanRule`：三段式 trace（gateway server → gateway client → user server）
断言根 = gateway；单 span trace 断言根 = 自身；缺 Server span 断言退化路径。

### 验收

```
同一 trace 在列表页与详情页的 spanCount / serviceCount / rootService 完全一致；
从任何服务的事务表进入，面包屑显示的根服务 = 详情页第一行的服务。
```

---

## 功能四：延迟分布下钻

### 用户故事

延迟分布直方图上框选 300ms–500ms 区间，下方 trace 样本立即换成该区间内的请求
（ES 同款交互：click-and-drag 选 bucket → 加载样本）。

### API 变更

`ListTraces` 已有 `minDurationMs`，**新增 `maxDurationMs`**：

```
Command params: { service, operation, min_duration_ms, max_duration_ms, limit, ... }
SQL: 定位子查询中 Duration BETWEEN ? AND ?（作用于入口 span）
```

### 前端实现

```
LatencyDistribution.tsx  [修改] SVG 直方图加 brush 层（pointerdown/move/up 三事件，无库依赖）；
                                选区高亮 primary 色半透明；清除按钮
page.tsx                 [修改] 选区状态 → 重新请求 operationTraces（带 min/max）
```

直方图本身的数据源不变（现有每 bucket trace 计数）。

---

## 功能五：Correlations 相关性分析

### 用户故事

服务详情页选「延迟相关性」或「失败相关性」，系统自动回答：
**「慢/错的请求，和正常请求相比，什么属性显著超标？」**
例：`92% 的失败请求 user_agent 含 iPhone（正常请求中仅 8%）` → 直指 iOS 端问题。

### 原理（对齐 ES 的 significant terms，用 ClickHouse 实现）

- **前景集**：failure 模式 = `StatusCode = Error` 的入口 span；latency 模式 = 耗时 > 该操作 P95 的入口 span
- **背景集**：同服务同操作的全部入口 span
- 对候选字段白名单逐一统计：每个值在前景集占比 `fgRatio` vs 背景集占比 `bgRatio`
- **打分**：`score = fgRatio × log(fgRatio / bgRatio)`（覆盖率 × 提升度，避免低频值靠高 lift 刷分）
- Master 按 score 分级：`high`（score > 0.5 且 fgRatio > 0.3）/ `medium` / `low`，只返回 Top 10

候选字段白名单（v1 固定，后续可配）：

```
SpanAttributes:    url.path, http.request.method, http.response.status_code,
                   client.address, user_agent.original
ResourceAttributes: k8s.pod.name, k8s.node.name, service.version, service.instance.id
```

> `user_agent.original` 归一化为浏览器/OS 家族后再统计（原始 UA 基数过高，前端库不引入，
> Agent 侧用简单前缀规则：iPhone / Android / Windows / Mac / bot / other）。

### API

```
GET /api/v2/observe/traces/correlations?cluster_id=&service=&operation=&mode=latency|failure&time_range=

响应：
{
  "mode": "failure",
  "foregroundCount": 2, "backgroundCount": 156,
  "correlations": [
    {
      "field": "user_agent.family", "value": "iPhone",
      "fgRatio": 1.0, "bgRatio": 0.08, "score": 2.53,
      "impact": "high",
      "fgCount": 2, "bgCount": 12
    }
  ]
}
```

### 分层实现

```
Agent  repository/ch/query/correlation.go   [新增] 前景/背景两段聚合（一次查询，条件聚合 countIf）
model_v3/apm/correlation.go                 [新增] CorrelationResult / CorrelationItem
Master service/query + gateway handler       [新增] action + endpoint + convert（impact 分级在此）
Web    components/CorrelationsPanel.tsx      [新增] 表格：字段值 | 前景占比条 | 背景占比条 | impact 徽标
       ServiceOverview.tsx                   [修改] 挂载入口（错误率卡片旁 "分析相关性" 按钮）
```

### 边界条件（测试必须覆盖）

- 前景集为空（无错误/无慢请求）→ 返回空列表 + `foregroundCount: 0`，前端显示「无可分析样本」
- 前景集 < 5 时结果附 `lowSample: true`，前端标注「样本过少，仅供参考」（本次 register 案例即 2 条）
- 字段值缺失（attribute 不存在）按 `"(none)"` 统计——缺失本身可能就是相关项

---

## 功能六（future，不进本期）：Dependencies 独立视图

已有 `DBStats` / `HTTPStats` 打底。后续把「被调方」（`db.system` 目标、外部 HTTP host）
聚合为独立依赖表：延迟 / 吞吐 / 错误率 / Impact 排序，对应 ES 的 Dependencies 页。
本次 `auth_type` 案例在该视图下会直接表现为「TiDB 依赖错误率飙升」。
→ 归档至 `docs/design/future/apm-dependencies-design.md` 占位，待瀑布图与关联链路稳定后启动。

---

## 实施阶段

| Phase | 内容 | 独立可验证的闭环 |
|---|---|---|
| **1** | 功能一（视觉重构）+ 功能三（聚合修正） | 从 geass-user 进入 register trace：完整 3 span、双服务双色、error 标记、列表数字正确 |
| **2** | 功能二（错误证据链） | 同一 trace 的 Error span 详情直接显示 `Data truncated...`；日志 tab 三条全出、gateway 色点区分 |
| **3** | 功能四（延迟分布下钻） | 框选直方图区间，样本列表随之刷新 |
| **4** | 功能五（Correlations） | 对 register 操作跑 failure 模式，`user_agent.family=iPhone` 出现在结果中 |

Phase 1/2 有依赖（2 复用 1 的 serviceColorMap 导出）；3、4 相互独立，可并行。
每 Phase：TDD（查询与纯函数先写测试）→ `go build` + `next build` → 部署 → 用 register trace 实测。

## 文件变更清单

```
model_v3/apm/
├── trace.go                        [修改] SpanError.Source
├── correlation.go                  [新增] Correlation 类型
└── enum.go                         [修改] 入口 span SQL 常量 + 注释

atlhyper_agent_v2/repository/ch/query/
├── trace.go                        [修改] ListTraces 两段式；GetTraceDetail 追加错误日志组装
├── trace_test.go                   [修改] 聚合修正用例（过滤后 spanCount 仍为全量）
├── correlation.go                  [新增] Correlations 查询
├── correlation_test.go             [新增]
└── log.go                          [核对/修改] LogAttributes 透传

atlhyper_master_v2/
├── service/interfaces.go + query/  [修改] Correlations 方法
├── gateway/handler/observe/observe_apm.go  [修改] correlations endpoint；logs 参数
└── model/convert/                  [修改] impact 分级

atlhyper_web/src/app/observe/apm/
├── components/waterfall-utils.ts   [修改] focusTraceOnService / selfDuration / 边界标记 + vitest
├── components/SpanRow.tsx          [修改] 视觉规则全套
├── components/TraceWaterfall.tsx   [修改] 图例控件 / focus 切换
├── components/SpanDrawer.tsx       [修改] 错误证据卡片
├── components/SpanLogs.tsx         [修改] 全量日志 + 服务色点 + attributes 展开
├── components/LatencyDistribution.tsx  [修改] brush
├── components/CorrelationsPanel.tsx    [新增]
├── components/trace-utils.ts       [删除] 并入 waterfall-utils
└── page.tsx                        [修改] focus 语义 / brush 状态 / correlations 挂载

atlhyper_web/src/
├── api/observe-apm.ts + datasource/apm.ts  [修改] correlations / max_duration
├── types/model/apm.ts              [修改] 对应类型
└── i18n/（types + zh + ja）         [修改] 新增文案
```

## 附录 A：明确不做的事

- **不引入 Elastic 双文档模型 / ECS 字段改写**——OTel 语义约定即是我们的 ECS，L1 契约层已承担统一字段职责
- **不做 ES 的 ML 异常检测**（拓扑节点健康色沿用现有阈值规则）——那是 AIOps 模块（L3/L4）的领域，APM 面板不重复建设
- **不做 Investigate 菜单大而全跳转**——信号联动已有统一时间轴与 signal-link 机制，复用之

## 附录 B：UI 设计稿

交互稿（HTML，AtlHyper 暗色主题实测配色）随本设计发布为 Artifact：
瀑布图新视觉 / Span 错误证据卡 / 跨服务日志 / Correlations 面板 / 延迟分布 brush。

## 附录 C：调研来源

- [Trace sample timeline | Elastic Docs](https://www.elastic.co/docs/solutions/observability/apm/trace-sample-timeline)——瀑布图配色（颜色代表服务，按出现顺序）、span flyout、View full trace、Investigate
- [Transactions UI | Elastic Docs](https://www.elastic.co/docs/solutions/observability/apm/transactions-ui)——五图布局、Impact 排序、延迟分布 brush → 500 样本、4xx 不计入服务端失败率的口径
- [Find transaction latency and failure correlations | Elastic Docs](https://www.elastic.co/docs/solutions/observability/apm/find-transaction-latency-failure-correlations)——Correlations 两种模式与分级展示
- [APM correlations root cause | Elastic Blog](https://www.elastic.co/blog/apm-correlations-elastic-observability-root-cause-transactions)——significant terms 原理（"uncommonly common"）
- [Service map | Elastic Docs](https://www.elastic.co/docs/solutions/observability/apm/service-map)——节点/边形态、异常分级色、flyout
- [Dependencies | Elastic Docs](https://www.elastic.co/docs/solutions/observability/apm/dependencies)——依赖视图指标与 Operations 下钻

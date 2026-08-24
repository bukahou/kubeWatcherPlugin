# 观测模块统一时间轴与信号联动

> 状态：active
> 创建：2026-08-24
> 关联：`docs/design/active/slo-panel-redesign.md`（SLO 面板重构）

## 背景

观测模块的四个信号页各写各的，同一个大模块里出现三套时间机制、两套刷新机制、两种目录结构：

| 页面 | 时间范围 | 自动刷新 | 组件位置 |
|------|---------|---------|---------|
| Metrics | **无选择器**（固定实时 + 展开后 24h 趋势） | 手写 `setInterval` 10s | `app/observe/metrics/components/` |
| APM | `TimeRangeSelection`（预设 + 自定义 + 绝对） | `useAutoRefresh` | `app/observe/apm/components/` |
| Logs | `TimeRangeSelection` | `useAutoRefresh` | `app/observe/logs/components/` |
| SLO | 自己的一套档位 | 手写 `setInterval` 30s | **`components/slo/`**（唯一例外） |

**后果**：从 APM 看到 14:30 有异常，切到 Logs 要重新选时间，切到 Metrics 根本没法选。
而排查故障时「同一时刻四个信号都发生了什么」正是核心操作 —— 时间上下文在页面间丢失，
等于每切一次页面就重新开始一次调查。

第二个问题：**页面区分不了「没有流量」和「采集挂了」**。
2026-08-24 实测 traces/logs 停了 78 分钟，查了 ClickHouse + ingress 计数才确认
是凌晨没人访问 geass（近 80 分钟仅 1 个请求），不是采集断了。
但页面上两者长得一模一样 —— 都是空白。`OTelGuard` 只检查集群是否部署 OTel，不看数据新鲜度。

第三个问题：**跨信号关联是单向的**。Logs → APM 有跳转（`LogDetail` 的 `/observe/apm?trace=`），
APM → Logs 没有，SLO → 任何信号都没有。SLO 面板刚发现 `geass-api` 有 123 个 5xx，
但点不进去看是哪些请求 —— 这正是 L1「统一语义层」要消除的数据孤岛。

## 目标

1. 四个信号页共享同一个时间轴，切换页面不丢时间上下文，URL 可分享
2. 空数据时说清楚原因：是没有流量，还是采集异常
3. 从 SLO / APM 能带着上下文跳到其他信号

## 核心架构

```
                    ┌──────────────────────────┐
                    │  timeRangeStore (zustand) │  ← 全局时间轴，URL 同步
                    │  TimeRangeSelection       │
                    └────────────┬─────────────┘
                                 │
        ┌────────────┬───────────┼───────────┬────────────┐
        ▼            ▼           ▼           ▼            ▼
    Metrics        APM         Logs         SLO      （未来信号）
   仅作用趋势图   全范围      全范围     贴合预聚合窗口
        │            │           │           │
        └────────────┴───────────┴───────────┴──── 每页声明 capability
                                 │
                    ┌────────────▼─────────────┐
                    │ useObserveTimeRange()     │  ← 读全局值 + 按 capability 降级
                    └───────────────────────────┘
```

### 关键约束：不是所有信号都支持任意时间范围

| 信号 | 能力 | 原因 |
|------|------|------|
| APM / Logs | 任意范围（预设 / 自定义 / 绝对） | ClickHouse 按需查询 |
| SLO | **仅 5 个固定窗口**（1h/6h/24h/3d/7d） | Agent 预聚合，窗口是 `sloWindowConfigs` 定死的 |
| Metrics | 卡片是当前快照；趋势图支持范围 | 硬件矩阵「最近 6 小时的温度」没有意义 |

因此统一时间轴不能是「一个值广播给所有人」，而是
**全局值 + 每页声明能力 + 自动降级**，并且降级要**显式告诉用户**
（选了 30 分钟、SLO 实际用的是 1 小时窗口，必须写出来，不能默默换掉）。

这也是上一版 SLO 出过的错：前端传 `1d`、后端窗口集里没有，静默 fallback 到 5 分钟数据，
标签却写着「1 天」。降级必须可见。

---

## 功能一：全局时间轴（Phase 1）

### 用户故事

在 APM 选了「最近 6 小时」，切到 Logs 仍是 6 小时；刷新页面还在；复制 URL 发给别人，
对方看到的是同一段时间。

### 状态设计

`store/timeRangeStore.ts`（zustand，与 `clusterStore` 同模式）：

```ts
interface TimeRangeStore {
  selection: TimeRangeSelection;
  setSelection: (s: TimeRangeSelection) => void;
  // 从 URL 初始化；无参数时用 localStorage，再无则默认 1h
  hydrateFromUrl: (params: URLSearchParams) => void;
}
```

**持久化三层优先级**：URL 参数 > localStorage > 默认 `preset: 1h`。
URL 优先是为了让分享链接生效；localStorage 是为了刷新不丢。

**URL 参数格式**（与现有 `TimeRangeSelection` 三种模式一一对应）：

```
?range=1h                    preset
?range=30m                   custom（值+单位，正则 ^\d+[mhd]$）
?from=1724500000&to=...      absolute（epoch 秒）
```

### 能力声明与降级

`hooks/useObserveTimeRange.ts`：

```ts
type RangeCapability = "full" | "sloWindows" | "trendOnly";

function useObserveTimeRange(capability: RangeCapability): {
  selection: TimeRangeSelection;   // 全局原值（给选择器回显）
  effective: TimeRangeSelection;   // 降级后的实际值（给数据请求）
  degraded: boolean;               // 是否发生降级
  note?: string;                   // 降级说明（i18n key），如「SLO 使用 1h 窗口」
}
```

`sloWindows` 的降级规则：向上取最近的可用窗口（30m → 1h，2d → 3d，10d → 7d），
因为**向下取会让统计窗口比用户要求的短，可用率会失真**；向上取只是范围更宽，语义安全。

### 数据对照表

| 前端 | 后端参数 | 备注 |
|------|---------|------|
| `toSince(effective)` | `since=6h` | APM / Logs 沿用现有 |
| `toAbsoluteParams(effective)` | `start_time` / `end_time` | 绝对模式 |
| `toSLOWindow(effective)` | `time_range=6h` | **新增**，映射到五个预聚合窗口之一 |

### 改动

- `store/timeRangeStore.ts` [新增]
- `hooks/useObserveTimeRange.ts` [新增]
- `lib/time-range.ts` [修改] 加 `toSLOWindow` / `parseRangeParam` / `formatRangeParam`
- 四个页面 [修改] 去掉各自的 `useState<TimeRange>`，改用 hook
- `components/common/TimeRangePicker.tsx` [修改] 接受 `disabledPresets` 与降级提示

### 验收

四页之间来回切换，时间保持；SLO 页选 30 分钟时显示「实际使用 1h 窗口」；
复制 URL 到新标签页打开，时间一致。

---

## 功能二：数据新鲜度（Phase 2）

### 用户故事

打开 APM 看到空白时，页面直接告诉我「最近 78 分钟没有请求」还是「采集异常」，
不用去查 ClickHouse。

### 判定归后端（大后端小前端）

「多久算陈旧」「是无流量还是采集异常」都是判定，必须在 Master 算。
前端只渲染文案和颜色。

区分两者的依据：**其他信号是否还在流动**。
- traces/logs 停了，但 metrics 还在 → 采集通路是好的，就是没有请求 → 「无流量」
- metrics 也停了 → 采集链路有问题 → 「采集异常」

metrics 是 Collector 主动拉取，只要节点活着就一定有数据；traces/logs 由请求触发。
这个非对称正好可以用来区分两种情况 —— 单看某一个信号是分不出来的。

### API

`GET /api/v2/observe/freshness?cluster_id=requiem`

```json
{
  "message": "获取成功",
  "data": {
    "signals": [
      { "signal": "metrics", "lastDataAt": "2026-08-24T07:08:23Z", "lagSeconds": 12,  "status": "live" },
      { "signal": "traces",  "lastDataAt": "2026-08-24T05:50:28Z", "lagSeconds": 4680, "status": "idle" },
      { "signal": "logs",    "lastDataAt": "2026-08-24T05:50:28Z", "lagSeconds": 4680, "status": "idle" },
      { "signal": "slo",     "lastDataAt": "2026-08-24T07:08:23Z", "lagSeconds": 12,  "status": "live" }
    ],
    "collectorHealthy": true
  }
}
```

`status` 取值：

| 值 | 含义 | 判定 |
|----|------|------|
| `live` | 数据新鲜 | lag < 该信号阈值 |
| `idle` | 没有流量（不是故障） | lag 超阈值，但 metrics 仍 live |
| `stale` | 采集异常 | lag 超阈值，且 metrics 也超阈值 |
| `absent` | 从未有过数据 | 无任何记录 |

阈值：metrics 5 分钟（拉取式，超了就是真有问题）；traces / logs / slo 15 分钟
（请求触发式，安静一会儿很正常）。

### 数据来源

Agent 上报快照时带上各信号的最新数据时间戳 —— Master 不连 ClickHouse，
只能由 Agent 提供。`model_v3/cluster.OTelSnapshot` 新增：

```go
// SignalFreshness 各信号最近一条数据的时间。
// 用来在页面上区分「没有流量」和「采集挂了」—— 两者在页面上都是空白，
// 但一个不用管，一个要救火。
type SignalFreshness struct {
    MetricsAt time.Time `json:"metricsAt"`
    TracesAt  time.Time `json:"tracesAt"`
    LogsAt    time.Time `json:"logsAt"`
}
```

### 改动

- `model_v3/cluster/snapshot.go` [修改] `OTelSnapshot.Freshness *SignalFreshness`
- Agent `repository/ch/query/freshness.go` [新增] 三张表各查一次 `max(时间列)`
- Agent `service/snapshot/otel_collector.go` [修改] 填充
- Master `model/observe.go` [新增] `FreshnessResponse`
- Master `service/query/observe_freshness.go` [新增] 判定逻辑 + 测试
- Master handler + 路由 [修改]
- Web `components/observe/SignalFreshnessBadge.tsx` [新增]，四页头部统一挂载
- i18n

### 验收

凌晨无人访问时，APM 页显示「无流量 · 最后数据 78 分钟前」而不是空白；
把 collector 停掉，显示「采集异常」。

---

## 功能三：信号联动（Phase 3）

### 用户故事

SLO 面板显示 `geass-api.bukahou.com` 有 123 个 5xx —— 点一下直接看到是哪些请求错了。

### 跳转矩阵

| 从 | 到 | 携带上下文 | 现状 |
|----|----|-----------|------|
| Logs 详情 | APM trace | `trace` | ✅ 已有 |
| APM trace | Logs | `trace` + 时间 | ❌ 补 |
| SLO 域名 | APM 服务 | `service` + 时间 | ❌ 补 |
| SLO 域名 | Logs（仅错误） | `service` + `severity=ERROR` + 时间 | ❌ 补 |

时间上下文由功能一的全局时间轴天然带过去，跳转只需带业务维度。

### 服务名映射问题

SLO 的 serviceKey 是 `{namespace}/{service}`（如 `geass-v3/geass-gateway`），
APM 的 ServiceName 来自 OTel 资源属性（如 `geass-gateway`），Logs 同 APM。
**三者命名不一致**，跳转前要映射。

映射规则：取 serviceKey 的 `/` 后半段。这是当前部署下的事实（K8s Service 名
与 `OTEL_SERVICE_NAME` 一致），但不保证永远成立 —— 因此：
- 映射函数集中一处（`lib/signal-link.ts`），带测试
- 跳转后目标页若查无此服务，显示「未找到服务 X 的数据」而不是空白

> 更彻底的做法是让 Agent 在快照里给出 serviceKey ↔ OTel ServiceName 的映射表，
> 但那需要 Agent 侧关联 K8s Service 与 OTel 资源属性，成本高于收益，先不做。
> 前端映射失败时有明确提示，不会静默出错。

### 改动

- `lib/signal-link.ts` [新增] 跳转 URL 构造 + serviceKey → ServiceName 映射 + 测试
- `components/slo/SLOListTable.tsx` [修改] 域名行加两个跳转入口
- `app/observe/apm/components/TraceWaterfall.tsx` [修改] 加「查看日志」
- 目标页 [修改] 读 URL 参数预填过滤条件，查无数据时给明确提示

---

## 实施阶段

| Phase | 内容 | 闭环验证 |
|-------|------|---------|
| **1** | 全局时间轴（store + hook + 四页接入 + URL 同步） | 切页面时间保持；SLO 降级有提示；URL 可分享 |
| **2** | 数据新鲜度（Agent 采集 → Master 判定 → 四页徽章） | 空数据时能看出是无流量还是采集异常 |
| **3** | 信号联动（SLO → APM/Logs，APM → Logs） | 从 SLO 的 123 个 5xx 三次点击定位到具体请求 |

Phase 1 纯前端；Phase 2 改 model_v3，agent + controller + web 同批部署；Phase 3 纯前端。

每个 Phase：TDD（测试先红）→ 实现 → `go build` / `next build` → 验收四项 → commit → tracker。

## 文件变更清单

```
atlhyper_web/src/
├── store/timeRangeStore.ts                        [新增] 全局时间轴
├── hooks/useObserveTimeRange.ts                   [新增] 能力声明 + 降级
├── lib/time-range.ts                              [修改] toSLOWindow / URL 参数编解码
├── lib/time-range.test.ts                         [新增] 降级规则与编解码
├── lib/signal-link.ts                             [新增] 跨信号跳转 URL
├── lib/signal-link.test.ts                        [新增]
├── components/common/TimeRangePicker.tsx          [修改] 禁用项 + 降级提示
├── components/observe/SignalFreshnessBadge.tsx    [新增]
├── app/observe/{metrics,apm,logs,slo}/page.tsx    [修改] 接入全局时间轴
├── components/slo/SLOListTable.tsx                [修改] 跳转入口
└── types/i18n.ts + i18n/locales/{zh,ja}.ts        [修改]

model_v3/cluster/snapshot.go                       [修改] SignalFreshness

atlhyper_agent_v2/
├── repository/ch/query/freshness.go               [新增]
├── repository/interfaces.go                       [修改]
└── service/snapshot/otel_collector.go             [修改]

atlhyper_master_v2/
├── model/observe.go                               [新增] FreshnessResponse
├── service/query/observe_freshness.go             [新增] 判定
├── service/query/observe_freshness_test.go        [新增]
├── service/interfaces.go                          [修改] QueryOTel 加方法
├── gateway/handler/observe/observe_freshness.go   [新增]
└── gateway/routes.go                              [修改]
```

## 附录：约束与风险

- **SLO 组件目录**：`components/slo/` 与其他三页的 `app/observe/*/components/` 不一致。
  本轮不动 —— 与时间轴无关，混在一起会让 diff 难读。单独一次纯移动提交更清晰
- **Metrics 的时间语义**：卡片与硬件矩阵永远是当前快照，时间轴只作用于趋势图。
  UI 上必须写明，否则用户会以为选了 6 小时看到的是 6 小时前的温度
- **降级必须可见**：SLO 选 30 分钟实际用 1h 窗口，页面要写出来。
  静默降级正是上一版「标签写 1 天、数据是 5 分钟」那个 bug 的成因
- **服务名映射**：SLO 的 `{ns}/{svc}` 与 OTel ServiceName 靠约定一致，
  不保证永远成立；映射集中一处 + 失败有提示，不静默出错

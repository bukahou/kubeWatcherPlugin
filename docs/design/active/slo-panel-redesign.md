# SLO 面板重构设计

> 状态：active
> 创建：2026-08-23
> 前置：`docs/design/archive/slo-ingress-contract-design.md`（ingress_* 契约层）

## 背景

SLO 契约改造（`ingress_*` 归一化）完成后，面板恢复了数据流，但 2026-08-23 实测发现
**页面上的每一个数字都是错的**，且面板本身缺少主流 SLO 实践的核心要素。

### 实测现状

| 项 | 页面显示 | 实际 | 偏差原因 |
|----|---------|------|---------|
| atlhyper-web 5 分钟 2xx 请求 | 238,679 | **10** | counter delta 跨 envoy 实例污染 |
| geass-gateway 1 天请求总数 | 44,942,602 | — | 同上 |
| 各服务 P95 延迟 | 0 ms | — | histogram delta 同样跨实例，差值为负被吃掉 |
| 域名 | `geass-v3/geass-gateway` | `geass-api.bukahou.com` | 路由映射表空 + fallback 用了 ServiceKey |
| SLO 目标 | 95% / 300ms（默认值） | 未配置 | `slo_targets` 表 0 行 |

### 与主流的差距

对照 [Google SRE Workbook](https://sre.google/workbook/alerting-on-slos/)、Grafana SLO、Datadog SLO：

| 主流要素 | 现状 |
|---------|------|
| 燃烧率（Burn Rate）—— 归一化后跨服务可比，面板核心列 | ❌ 完全没有 |
| 多窗口燃烧率（1h/6h/24h/3d） | ❌ 没有 |
| 固定滚动窗口 + 单一目标 | ❌ 目标按 1d/7d/30d 各存一套 |
| 首屏一行一个 SLO，一眼扫完 | ❌ 首屏 6 个聚合卡，要展开才见关键信息 |
| Good vs Bad 事件计数（分子分母都露出来） | ⚠️ 有字段但数值错误 |

## 目标

1. 面板上的数字与 ClickHouse 原始数据一致（可验证）
2. 补齐燃烧率与多窗口判定，让「谁在烧预算、还能烧多久」一眼可见
3. SLO 目标模型对齐定义：一个域名 = 一个固定窗口 + 一组目标

## 核心架构

```
cilium-envoy (2 实例: 192.168.0.10:9964 / .12:9964)
   │ Prometheus scrape 15s
   ▼
Collector transform/ingress_normalize → ingress_request_total{namespace,service,status_class}
   │                                     ingress_request_duration_bucket (19 bounds)
   ▼
ClickHouse otel_metrics_sum / otel_metrics_histogram   ← TTL 7 天，emptyDir
   │
   ▼
Agent repository/ch/query/slo.go
   │  ⚠️ 关键：counter 必须按 service.instance.id 分别算 delta 再求和
   ▼
OTelSnapshot.SLOIngress + SLOWindows{1d,7d,30d}
   │
   ▼
Master
   ├── /slo/domains/v2      域名分组 + 目标 + 燃烧率
   ├── /slo/domains/detail  单域名详情
   └── slo/calculator.go    错误预算 + 燃烧率 + 状态判定
   │
   ▼
Web /observe/slo   SLO 清单表（含燃烧率）→ 展开详情
```

### 三条环境约束（设计必须服从）

| 约束 | 影响 |
|------|------|
| **两个 envoy 实例** | 所有 counter/histogram 聚合必须先按实例分区，否则数据失真数万倍 |
| **ClickHouse TTL 7 天** | 30 天窗口没有数据。窗口档位应对齐真实能力 |
| **ClickHouse + SQLite 均为 emptyDir** | Pod 重启数据归零。任何"长期 SLO"在当前存储下都是幻觉，见「存储决策」 |

---

## 功能一：counter 多实例聚合修复（Phase 1）

### 用户故事

打开 SLO 面板，看到的请求数、错误率、P95 与 ClickHouse 里实际发生的一致；
可以用一条 SQL 手工核对。

### 问题

`buildIngressCountQuery` 的窗口函数：

```sql
lagInFrame(Value, 1, Value) OVER (
    PARTITION BY Attributes['namespace'], Attributes['service'], Attributes['status_class']
    ORDER BY TimeUnix)
```

缺少实例维度。两个 envoy 各有独立的累积计数器（实测 12327 与 4556），
按时间交错排序后，每次切换实例都产生 ±7771 的假 delta。

同样问题存在于 `buildIngressLatencyQuery`（`argMax/argMin(BucketCounts)` 跨实例取快照）、
`buildIngressHistoryQuery`、`buildIngressSummaryQuery`。

### 修法

所有 ingress 查询的分区键加 `ResourceAttributes['service.instance.id']`：

```sql
-- 计数：各实例分别算 delta，最后按 {ns,svc,class} 求和
SELECT ns, svc, class, sum(delta) AS delta FROM (
    SELECT ns, svc, class,
           sum(if(Value >= prevValue, Value - prevValue, Value)) AS delta
    FROM (
        SELECT ..., lagInFrame(Value, 1, Value) OVER (
            PARTITION BY instance, ns, svc, class ORDER BY TimeUnix) AS prevValue
        FROM otel_metrics_sum WHERE MetricName = 'ingress_request_total' ...
    ) GROUP BY instance, ns, svc, class
) GROUP BY ns, svc, class
```

直方图：各实例分别取窗口内最新/最旧 `BucketCounts` 做差，再逐桶相加。

### 验收

```
ClickHouse 手工查询（按实例分区）与 API 返回的 totalRequests 差异 < 1%
P95 不再恒为 0
```

---

## 功能二：真实域名分组（Phase 1）

### 用户故事

面板上显示 `geass-api.bukahou.com`，不是 `geass-v3/geass-gateway`。

### 问题

两处都断了：

1. `RouteUpdater.syncFromIngresses` 只读 `snapshot.Ingresses`，
   但集群已迁到 Cilium Gateway API —— **Ingress 0 个、HTTPRoute 5 个**，表永远为空
2. 它生成的 ServiceKey 是 Traefik 时代的 `{ns}-{svc}-{port}@kubernetes`，
   跟 Agent 现在用的 `{ns}/{svc}` 对不上，即便有 Ingress 也匹配不了

而 Agent 侧其实已经解决了：`RouteRepository.GetServiceHostMap` 读 HTTPRoute，
`IngressSLO.DisplayName` 里已经是真实域名（实测 `/observe/slo/ingress` 返回正确域名）。

### 修法

1. **短路径**：`buildDomainSLOV2Fallback` 用 `ing.DisplayName` 作为域名（为空才退回 ServiceKey）
2. **正路径**：`RouteUpdater` 改为消费 Agent 已解析好的映射，而不是自己解析 K8s 资源
   —— Agent 已经读过 HTTPRoute，Master 再读一遍 Ingress 是重复且过时的实现。
   ServiceKey 统一为 `{ns}/{svc}`。

> 为什么不让 Master 直接读 HTTPRoute：Master 不连 K8s API，快照是唯一入口。
> Agent 侧的 RouteRepository 已经是这层抽象，Master 复用即可。

---

## 功能三：燃烧率与 SLO 目标模型（Phase 2）

### 用户故事

首屏一行一个域名，看到「预算还剩多少、正在以几倍速度烧、按这个速度还能撑多久」。

### 燃烧率定义

```
burnRate = 实际错误率 / (1 - 目标可用率)
```

归一化含义：`1×` = 正好在窗口结束时用完预算；`14.4×` = 1 小时烧掉 2% 月度预算。

Google SRE 阈值（用于上色，不做告警）：

| 窗口 | 阈值 | 语义 |
|------|------|------|
| 1h | > 14.4× | 危急 |
| 6h | > 6× | 警告 |
| 24h | > 3× | 关注 |
| 3d | > 1× | 观察 |

### API

`GET /api/v2/slo/domains/v2?cluster_id=requiem&time_range=1d`

响应在现有结构上新增（每个 domain 与 service 同级）：

```json
{
  "domain": "geass-api.bukahou.com",
  "summary": { "availability": 99.97, "p95Latency": 47, "errorRate": 0.03,
               "requestsPerSec": 0.4, "totalRequests": 1420,
               "goodRequests": 1419, "badRequests": 1 },
  "target": { "availability": 99.5, "p95Latency": 300, "windowDays": 7 },
  "errorBudget": {
    "remainingPct": 82.3,
    "consumedEvents": 1,
    "allowedEvents": 7,
    "burnRates": [
      { "window": "1h",  "rate": 0.4, "threshold": 14.4, "status": "good" },
      { "window": "6h",  "rate": 0.2, "threshold": 6,    "status": "good" },
      { "window": "24h", "rate": 0.9, "threshold": 3,    "status": "good" },
      { "window": "3d",  "rate": 1.2, "threshold": 1,    "status": "warn" }
    ],
    "exhaustEta": "2026-09-04T10:00:00Z"
  },
  "status": "healthy"
}
```

`exhaustEta` 为 null 表示当前燃烧率下窗口内不会耗尽。

### 目标模型变更

现状 `slo_targets` 按 `(cluster_id, host, time_range)` 存三套目标，不符合 SLO 定义。
改为一个域名一条记录：

| 字段 | 变更 |
|------|------|
| `time_range` | **删除** |
| `window_days` | **新增**，默认 7（对齐 ClickHouse TTL，见存储决策） |
| `availability_target` | 保留 |
| `p95_latency_target` | 保留 |

> SQLite 按项目规范直接改 `migrations.go` 的 CREATE TABLE，不写增量迁移。
> 该表当前 0 行，无数据迁移成本。

前端的 1d/7d/30d 切换语义随之改变：**只影响图表的查看范围，不改变 SLO 目标窗口**。

### 错误预算算法

现状 `CalculateErrorBudgetRemaining(actualAvail, targetAvail)` 用可用率差值比，
在窗口一致时与事件计数等价，但分子分母不可见。改为事件计数口径：

```
allowedEvents  = totalRequests × (1 - targetAvailability)
consumedEvents = badRequests
remainingPct   = (allowedEvents - consumedEvents) / allowedEvents × 100
```

好处：面板能直接显示「允许错 7 个，已经错了 1 个」，比「预算剩 82.3%」直观得多。

---

## 功能四：面板重构（Phase 3）

### 首屏：SLO 清单表（替代 6 个聚合卡）

```
域名                    目标     当前SLI   预算余量        1h     6h     P95     状态
geass-api.bukahou.com  99.5%   99.97%   ████████░ 82%  0.4×   0.2×    47ms    ✓
bukahou.com            99.0%   99.70%   ███░░░░░░ 31%  3.1×   1.8×   112ms    ⚠
argocd.bukahou.com     99.0%   99.88%   ██████░░░ 65%  0.9×   0.5×   325ms    ✓
```

- 燃烧率按阈值上色（>14.4 红 / >6 橙 / >1 黄），归一化后可跨服务比较
- 排序：状态 → 燃烧率降序（最该看的排最前）
- 保留一行汇总（总 RPS、健康/告警数），但不再占据整屏

### 展开详情：现有四 tab 保留，Overview 补两块

1. **多窗口燃烧率表**：1h / 6h / 24h / 3d 四档 × 实际值 × 阈值 × 判定
2. **Good vs Bad 计数**：`1419 好 / 1 坏`，以及「允许错 7 个」

### 前端职责边界

燃烧率、阈值判定、状态、排序、ETA **全部在 Master 算好**，前端只渲染颜色与进度条。

---

## 存储决策（需用户拍板，不阻塞 Phase 1–3）

ClickHouse 与 Master SQLite **均为 emptyDir**，Pod 重启数据归零。这意味着：

- 30 天窗口的 SLO 在当前存储下无法成立
- SLO 目标配置（`slo_targets`）也会随 controller 重启丢失

三个选项：

| 方案 | 做法 | 代价 |
|------|------|------|
| **A. 对齐现实**（Phase 1–3 默认） | 窗口档位 1h/6h/24h/7d，默认 `window_days=7`，删掉 30d | 零成本，无月度 SLO |
| **B. 只给 SQLite 持久卷** | Master `/app/data` 换 PVC/hostPath（几十 MB），SLO 目标与长期聚合存这里 | 需要一块小持久卷；ClickHouse 仍 emptyDir |
| **C. ClickHouse rollup 表** | 建 `slo_ingress_1m`（分钟粒度，TTL 90 天） | 约 500 万行/90 天，但 **emptyDir 下重启即失效**，需与 B 配合才有意义 |

**Phase 1–3 按 A 实施**，不依赖任何存储变更。B/C 待决策。

---

## 实施阶段

| Phase | 内容 | 闭环验证 |
|-------|------|---------|
| **1** | counter/histogram 多实例聚合 + 真实域名 | 页面数字与手工 SQL 一致、域名正确、P95 非 0 |
| **2** | 燃烧率 + 事件计数口径预算 + 目标模型 | API 返回 burnRates，首屏清单表带燃烧率列 |
| **3** | 面板重构（清单表 + 详情两块） | 首屏一眼看出谁在烧预算 |
| **4** | 存储（依赖决策 B/C） | 30 天窗口成立、目标配置不丢 |

每个 Phase：TDD（测试先红）→ 实现 → `go build` / `next build` → 四项验收 → commit → tracker。

## 文件变更清单

```
atlhyper_agent_v2/repository/ch/query/
├── slo.go                                    [修改] 4 个 build*Query 加实例分区；直方图逐实例差分
└── slo_test.go                               [修改] 两实例交错序列的 delta 断言

atlhyper_master_v2/
├── slo/calculator.go                         [修改] 燃烧率、事件计数口径预算、ETA
├── slo/calculator_test.go                    [新增] 燃烧率与预算算法
├── slo/route_updater.go                      [修改] 消费 Agent 的域名映射，ServiceKey 统一 {ns}/{svc}
├── database/types.go                         [修改] SLOTarget 去 TimeRange，加 WindowDays
├── database/sqlite/migrations.go             [修改] slo_targets 表定义
├── database/sqlite/slo.go                    [修改] 目标读写 SQL
├── model/slo.go                              [修改] ErrorBudget / BurnRate 响应结构
├── gateway/handler/slo/slo_domains.go        [修改] fallback 用 DisplayName；组装燃烧率
└── gateway/handler/slo/slo_domains_test.go   [新增]

atlhyper_web/src/
├── types/slo.ts                              [修改] ErrorBudget / BurnRate 类型
├── app/observe/slo/page.tsx                  [修改] 首屏换 SLO 清单表
├── components/slo/SLOListTable.tsx           [新增] 清单表（燃烧率列）
├── components/slo/BurnRateTable.tsx          [新增] 多窗口燃烧率
├── components/slo/OverviewTab.tsx            [修改] 补 Good/Bad 计数与燃烧率表
├── components/slo/SLOTargetModal.tsx         [修改] 去掉 timeRange 维度
├── mock/slo/                                 [修改] 对齐新字段
└── types/i18n.ts + i18n/locales/{zh,ja}.ts   [修改]
```

## 附录：约束与风险

- **histogram 数据只有 2026-08-23 00:56 起**（契约改造后才开始），7 天窗口在一周后才完整
- **envoy 实例数会变**：Cilium Gateway API 的 envoy 是 DaemonSet，节点增减会改变实例数。
  按实例分区的实现天然适配，不需要硬编码实例列表
- **4xx 不计入错误**：`isErrorStatusClass` 只认 `5`。实测 atlhyper-web 的 4xx 量很大
  （未登录时前端调用鉴权接口），计入会让可用率失真 —— 保持现状是对的

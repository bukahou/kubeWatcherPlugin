# SLO Ingress 契约设计

> 状态: 执行中
> 创建时间: 2026-08-23
> 范围: SLO 模块（仅 Ingress，Mesh 移除）

## 背景

SLO 模块两路数据源全部失效，实测 0 行：

| 子模块 | 查询的指标 | 状态 |
|--------|-----------|------|
| Ingress SLO | `traefik_service_requests_total` | ❌ Traefik 已从集群移除 |
| Mesh SLO | `mesh_request_total` | ❌ Istio 已移除，契约层空转 |

根因不是"换了实现"，而是 **Ingress 这一路从未做过抽象**：

```sql
-- repository/ch/query/slo.go 现状
SELECT Attributes['service'] AS svc,
       Attributes['code']    AS code,     -- ← Traefik 专属 label 名
FROM otel_metrics_sum
WHERE MetricName = 'traefik_service_requests_total'  -- ← Traefik 专属指标名
```

指标名与 label 名双重耦合。对比之下 Mesh 那一路 2026-04 做过泛用化
（`istio_requests_total` → `mesh_request_total`），Istio 装了又拆都没伤到 Agent 代码
——**同一项目内正反例并存，本设计照正例做**。

## 设计原则

**`slo.go` 中不得出现任何 ingress 实现的名字。**

```
各实现指标 ──[Collector transform: 唯一适配层]──→ ingress_* 契约 ──→ Agent SQL
 实现专属、会变        换实现只改这里              永不变          永不改
```

判定标准：假设明天换成 Nginx Ingress，需改动的文件应**只有 `collector.yaml`**。
若清单中出现 `slo.go`，则抽象失败。

## 契约定义

```
ingress_request_total{namespace, service, port, status_class}
ingress_request_duration_seconds_bucket{namespace, service, port, le}
ingress_request_duration_seconds_count{namespace, service, port}
ingress_request_duration_seconds_sum{namespace, service, port}
ingress_request_timeout_total{namespace, service, port}
```

### label 语义

| label | 含义 | 取值约定 |
|-------|------|---------|
| `namespace` | 后端服务所在 K8s namespace | K8s 标准命名 |
| `service` | 后端 Service 名 | K8s 标准命名 |
| `port` | 后端 Service 端口 | 数字字符串 |
| `status_class` | HTTP 状态码类别 | `"1"` / `"2"` / `"3"` / `"4"` / `"5"` |
| `le` | 延迟桶上界 | **秒**，浮点字符串 |

三条约定的理由（均来自本次踩坑）：

1. **`status_class` 用类别而非精确码** —— 各实现粒度不同（Envoy 只给 class，
   Traefik 给精确码）。契约取最小公分母，保证任何实现都能满足。
   实现若能提供精确码，可附加**可选** label `status_code`。
2. **`le` 单位固定为秒** —— Envoy 用毫秒、Traefik 用秒。不统一的话 Agent
   就得知道底下是谁，抽象即失效。换算由 transform 负责。
3. **用 `namespace`/`service` 而非 `serviceKey`** —— K8s 原生概念，任何 ingress
   实现都能映射；`serviceKey` 是 AtlHyper 内部拼接产物，属于上层职责。

## 实现适配表

换 ingress 实现时，在此表加一行 + 改 `collector.yaml` 的 transform，其余不动。

| 实现 | 源指标 | namespace/service/port ← | status_class ← | le 换算 |
|------|--------|--------------------------|----------------|---------|
| **Cilium Gateway**（当前） | `envoy_cluster_upstream_rq_xx`<br>`envoy_cluster_upstream_rq_time_bucket` | 正则拆 `envoy_cluster_name`：<br>`ingress/cilium-gateway-<gw>/<ns>_<svc>_<port>` | `envoy_response_code_class` | 毫秒→秒 |
| Traefik（历史） | `traefik_service_requests_total`<br>`traefik_service_request_duration_seconds_bucket` | 拆 `service` | `code` 取首字符 | 已是秒 |
| Nginx Ingress | `nginx_ingress_controller_requests` | `namespace` + `service` | `status` 取首字符 | 已是秒 |
| Envoy Gateway / Istio Gateway | `envoy_*`（同 Cilium） | cluster 命名规则不同，正则需调整 | 同左 | 毫秒→秒 |

## 范围决策：移除 Mesh

SLO 只保留 Ingress。理由：

| | 覆盖 | 语义 | 现状 |
|---|------|------|------|
| Mesh SLO | 全集群 workload | 仅 L7 基础字段 | 数据源已失效 |
| **APM 拓扑** | 埋点服务 | span name、业务属性、精确到接口 | ✅ 已上线，8 条真实调用边 |

职责划分：**SLO = 外部视角（入口层，与应用解耦）/ APM = 内部视角（应用链路）/
AIOps = 跨信号关联**。服务间调用关系属于应用内部，归 APM；SLO 不应重复承载。

Mesh 相关一律**删除而非保留空壳** —— 无数据的抽象层比没有更糟，会让人误以为
这条路还活着（本次调研中即因此浪费排查时间）。

## 数据源

**cilium-envoy (9964)** 单一数据源。实测三要素齐全：

| SLO 要素 | Envoy 指标 |
|---------|-----------|
| 请求数 + 状态码 | `envoy_cluster_upstream_rq_xx{envoy_response_code_class, envoy_cluster_name}` |
| 延迟分布 | `envoy_cluster_upstream_rq_time_bucket{le, envoy_cluster_name}` |
| 超时 | `envoy_cluster_upstream_rq_timeout` |

`envoy_cluster_name` 格式规整，可直接解析：

```
ingress/cilium-gateway-public-ingress/geass-v3_geass-gateway_8080
└─ 固定前缀 ──────────────────────┘ └─ns──┘ └──service──┘ └port┘
```

**不采 Hubble** —— Envoy 只处理经 Gateway 的南北向流量，pod↔pod 走 eBPF 直接
转发不经 Envoy；东西向数据由 APM 覆盖，无需 Hubble。

### 采集范围说明

cilium-envoy 是 DaemonSet，但只有持有 Gateway VIP 的节点承载实际流量
（实测 desk-two，listener 请求 115253）。其余节点 Envoy 有配置无请求，
采到零值，不影响聚合。

## 改动清单

```
config 仓 clusters/requiem/apps/atlhyper/
└── collector.yaml                          [修改] ✅ 已完成
    ├── scrape: istio-sidecars → cilium-envoy
    ├── transform: istio_to_mesh → ingress_normalize
    └── metrics pipeline 引用更新

atlhyper/
├── atlhyper_agent_v2/repository/ch/query/
│   ├── slo.go                              [修改] traefik_* → ingress_*；删 mesh 方法
│   └── contract.go                         [修改] 新增 ingress_request_total 契约自检
├── atlhyper_agent_v2/repository/ch/
│   └── summary.go                          [修改] SLO 摘要同步改契约名
├── model_v3/slo/slo.go                     [修改] 删 ServiceSLO / ServiceEdge
├── atlhyper_master_v2/gateway/
│   ├── routes.go                           [修改] 删 /slo/mesh/*、/observe/slo/{services,edges}
│   └── handler/                            [修改] 删对应 handler
└── atlhyper_web/src/app/observe/slo/       [修改] 删服务网格部分
```

`IngressSLO` 模型**不动** —— 字段（`serviceKey`/`rps`/`successRate`/`p99Ms`/
`statusCodes`/`latencyBuckets`）全部是平台无关语义，本次改造验证了原设计正确。

## displayName 的来源

`IngressSLO.displayName` 需要真实域名（如 `geass-api.bukahou.com`），
但**任何 ingress 实现的指标都不带域名维度**（实测 Envoy 无 vhost 统计、
Hubble 无 host label）。

解法：**从 HTTPRoute 反查**。Agent 已采集 K8s 快照，HTTPRoute 是 Gateway API
标准资源，`spec.hostnames` + `spec.rules[].backendRefs` 可建立
`namespace/service → 域名` 映射。

这本身也是抽象：换 ingress 实现只要仍用 Gateway API，映射逻辑不变；
且比从指标标签取更可靠（一个服务挂多域名时能全部列出）。

## 已知局限

1. **分位数为近似值** —— Envoy 桶边界固定不可配
   （0.5/1/5/10/25/50/100/250/500/1000/2500ms…），P99 精度受桶宽限制。
   Traefik 时代同样如此，不算退化。
2. **仅覆盖走 Gateway 的服务** —— 当前 5 个（akasha / argocd / atlhyper-web /
   geass-v2 / geass-v3）。这是正确的，SLO 本就只看对外入口。
3. **`method` 维度缺失** —— Envoy 的 `rq_xx` 不带 method；契约中标为可选。

## 契约漂移防护

将 `ingress_request_total` 纳入 Agent 现有的枚举契约自检
（`repository/ch/query/contract.go`），检查 `status_class` 取值 ⊆ {1,2,3,4,5}。

同一机制已在防 APM 枚举漂移（2026-08 Collector 升级致 SpanKind 格式变化，
静默潜伏 27 天）。指标契约同样属于"不报错、只是查不到"的静默失败类型，
必须有主动探测。

## 实施阶段

- Phase 1: Collector 加 scrape + transform — ✅ 完成
- Phase 2: 验证 ClickHouse 中 `ingress_*` 数据与 label 齐全
- Phase 3: 改 Agent `slo.go`（TDD，先写断言）+ 契约自检扩展
- Phase 4: HTTPRoute 域名映射
- Phase 5: 删除 Mesh 相关代码（Agent / model / Master / 前端）
- Phase 6: 端到端验收 —— SLO 页显示 5 个后端服务

## 验收

1. SLO 页 Ingress 列出 5 个后端服务，含 RPS / 成功率 / P50/P90/P95/P99
2. `displayName` 显示真实域名而非 workload 名
3. Agent 契约自检输出 `[契约自检] 通过 table=otel_metrics_sum column=status_class`
4. **抽象验证（纸面演练）**：列出"换成 Nginx Ingress"需改动的文件，
   清单中不得出现 `slo.go`

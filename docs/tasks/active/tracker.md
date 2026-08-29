# 任务追踪

> **本文件是任务状态的唯一权威源。**
> 只保留「待办」和「进行中」的任务。完成后归档到 `docs/tasks/archive/`。
>
> 状态标记：`✅` 完成 / `🔄` 进行中 / 无标记 = 待办

> **线上版本**（2026-08-29）：agent v0.7.0 / controller v0.6.0 / web v0.7.0
>
> 近期完成并已归档：观测统一时间轴与信号联动、指标采集查询预算优化、
> SLO 面板重构、**APM 面板重构（对齐 ES APM，Phase 1–4 + 部署验收）** —— 见 `docs/tasks/archive/`

---

## 压测暴露的观测缺陷（2026-08-29 geass-v3 压测，阶段 1–4）

> **待优化，均未修复**（除 P0 已扩容）。用户 2026-08-29 指示「先记录，后续需要优化」。
> 证据在此固化：`system.query_log` 与容器日志都有 TTL，原始数据会滚掉。
>
> **一句话**：写入路径扛住了 6000 条/秒；**垮的是读取路径与健壮性**。
> 「性能」只是触发条件，最危险的一条（① 静默撒谎）与性能完全无关。

### 压测实测基线（阶段 2 峰值）

| 项 | 实测值 |
|---|---|
| 摄取速率 | traces ~2900/s + logs ~3100/s ≈ **6000 条/秒** |
| ClickHouse CPU | **1994–2002m / 2000m 全程顶死**，一次未松 |
| ClickHouse 内存 | 峰值 2180Mi / 3Gi（73%） |
| 查询耗时 | avg 1.7–2.2s，p95 2.5–4.5s，**max 60.2s** |
| Agent 超时 | 15–32 次/分钟 |
| 并发合并 | 峰值 28 个 |
| 落库总量 | traces 362 万 + logs 265 万行 |

### ① 查询失败时静默渲染零值 —— ✅ 已修复（2026-08-29，commit 39763c8）
>
> 三层联动落地：model_v3 加 `Unavailable []string` + Section* 契约常量；
> Agent 8 个 fill* 返回 error、失败 section 记账、节点间并行(限4)、
> 部分失败不再丢节点；Master 对 Unavailable section 置 nil 格子；
> 前端 NodeCard 渲染 `--` 占位。TDD 5 例先红后绿。
> 正常路径线上回归通过（7 节点真实数值）；失败路径由单测覆盖
> （线上验证需人为制造 ClickHouse 超时，不值得为此伤害生产）。

**现象**：面板 7 节点中 5 个显示 `CPU 0.0% / 内存 0.0% / 磁盘 0.0%` +「未识别硬件」，
用户据此判断「树莓派宕机了」。**实际七节点数据全部正常到达 ClickHouse。**

**机制**：Agent 逐节点**串行**采集共享一个 10s context
（`AGENT_CLICKHOUSE_TIMEOUT` 默认值，未被 env 覆盖）。靠后的节点被 context 取消，
错误被静默吞掉 —— `metrics.go` 各 `fill*` 函数遇错直接 `return`，留下零值：

```go
rows, err := r.client.Query(ctx, query, ip)
if err != nil {
    return          // ← 静默返回，nm 保持零值
}
```

前端把零值当真实测量渲染成 `0.0%`，而非「查询失败 / 数据不可用」。

**为什么最危险**：给它 100 个核也不解决 —— 任何一次查询因任何原因失败，
面板照样显示看起来合法的 `0.0%`。观测平台在自己失效时假装正常，
比不报警更糟。这是**诚实性缺陷**，不是性能缺陷。

**修复方向**：并行采集替代串行；错误必须上抛而非静默吞；
模型区分「零值」与「无数据」（指针 / Optional / 显式 `available` 字段）；
前端对「无数据」渲染为灰色占位而非 0.0%。

### ② otel_metrics_gauge 排序键使时间过滤无法剪枝 —— 真架构根因

```
sorting_key: ServiceName, MetricName, Attributes, toUnixTimestamp64Nano(TimeUnix)
```

| 查询条件 | 能否走索引 |
|---|---|
| `MetricName = 'node_cpu_seconds_total'` | 部分（前缀 ServiceName 不定） |
| `ResourceAttributes['net.host.name'] = ?` | ❌ Map 列不在索引内，逐行取值 |
| `TimeUnix >= now() - INTERVAL 2 MINUTE` | ❌ **时间在键末位**，前三项未锁定即无法剪枝 |

**实测（空闲状态，非高负载）**：单次查询扫 **54,898 行 / 读 19.85 MiB / 耗时 1,426 ms**。
查最近 2 分钟却读该指标全部历史 → **查询成本随保留期线性增长，与当前流量无关**。

**含义**：扩到 6 核只是推迟问题。数据攒到 3 天 TTL 满负荷时会再次顶上来。

**归属注意**：该表由 **OTel collector 的 `create_schema` 自动建**，非 AtlHyper 建表。
改排序键须覆盖建表语句（collector 侧配置或预建表后关闭 create_schema）。

**修复方向**：时间前移进排序键 / node 标识提升为独立列而非 Map / 预聚合物化视图。

### ③ collector sending_queue 过小 —— ✅ 已修复（2026-08-29，config 仓）
>
> queue_size 50→1000（峰值速率下缓冲 1.6s→33s）+ num_consumers 20 +
> memory_limiter 2200MiB + 资源 3核/3Gi（实测 CPU 峰值 879m/1000m=88%）。

```
error: "sending queue is full", rejected_items: 512   × 261 次
```

**实测丢弃 133,632 条 trace**（仅 traces，logs/metrics 未丢），
约占同期落库量 362 万的 **3.6%**。丢弃贯穿 15:14–15:23 整个阶段 2。

当前配置：`sending_queue.queue_size: 50`、`batch.send_batch_size: 512`
→ 缓冲仅 50×512 = 25,600 条，在 2900 spans/s 下**不足 9 秒**。

collector 自身资源并非瓶颈：峰值仅 431m / 261Mi（限 1 核 / 1Gi）。

**修复方向**：加大 `queue_size`；考虑 `num_consumers` 提升并发；
高负载场景引入 tail_sampling（保留全部错误 + 慢 trace + 基线采样）。

**对压测结论的影响**：AtlHyper 侧观测到的 trace QPS **系统性低估约 3.6%**，
用其数据反推 geass 吞吐时需修正。

### ④ 负载停止后观测质量反而更差 —— 合并积压

阶段 2 结束瞬间（摄取归零）实测：

| 时刻 | 摄取 | CH CPU | 查询 avg | 查询 p95 |
|---|---|---|---|---|
| 负载中 15:23 | traces 164,646/分 | 1997m | 1867ms | 2616ms |
| **停止后 15:25** | **0** | **仍 2000m** | **4623ms** | **17,793ms** |

流量停了 CPU 照样打满 —— 摄取期堆积的合并任务（峰值 28 个）集中还债。

**运维含义**：**压测刚结束就查看结果，看到的是最差的观测质量**。
需等合并积压消化完再取数。这条对任何"事件后立即排查"的场景都成立。

### ⑤ 日志时间戳慢 1 小时 —— ✅ 已修复（2026-08-29）
>
> 根因是三个 Dockerfile 都烤了 `TZ=Asia/Shanghai`(UTC+8)，不只 Agent ——
> controller 写 SQLite 的审计/事件时间戳也一直偏 1 小时。已改 Asia/Tokyo，
> 三组件容器时间与宿主机 JST 完全对齐。

```
容器真实时间: 2026-08-29T14:42:07.388+09:00
Agent 自己打:            time=13:42:07
```

差值恒为 1 小时（形似 TZ 被设为 UTC+8 而非 JST）。
**排查时已实际踩坑一次**：按 Agent 日志时间误判「超时一小时前已停止」，
实为正在发生，靠容器时间戳才纠正。故障复盘按日志时间对因果会整体错位一小时。

### 阶段 3–4 追加证据

**② 的定性需下调优先级（原判断偏重）**：扩容到 6 核后，同一条 Agent 查询
从 **1,426 ms → 11 ms**，而表数据量还大了 10 倍（200 万 → 2,372 万行）。
说明 ClickHouse 有**分片级 min/max 剪枝**兜底，排序键不是本次急性死因；
真正的杀手是 **CPU 饥饿**。排序键仍是真问题（决定成本随保留期增长的斜率），
但属慢性病，优先级应低于 ① 与 ③。

**扩容避免了一次崩溃（不只是提速）**：阶段 3 ClickHouse 内存峰值 **5,827 Mi**，
远超旧配置 3Gi 上限。不扩容则会 OOM —— 观测栈在压测最关键一档直接消失。

**阶段 3 承载实测**：峰值摄取约 **25,000 条/秒**（traces 940,739/分 + logs 562,339/分），
CPU 触顶 5999m/6000m，但 **Agent 超时 0、collector 丢弃 0**，全程零重启。

**③ 得到强化证据（阶段 4）**：写入专项的 trace 量远低于阶段 3，
却仍出现 **46 次队列满丢弃**（约 23,552 条）。说明丢弃不是被总量压垮，
而是**突发尖峰打满了仅 9 秒缓冲的队列** —— 加大 `queue_size` 的必要性成立。

### ⑥ 观测层能测出业务侧测不到的东西（能力验证，非缺陷）

阶段 4 热行锁竞争，两侧测量差异巨大：

| 层级 | 场景 A（同一行） | 场景 B（不同行） | 倍数 |
|---|---|---|---|
| k6 端到端 | 28.3 ms | 13.7 ms | 2.1× |
| **AtlHyper 的 media span** | **26.9 ms** | **2.2 ms** | **12.2×** |

k6 的测量混入约 26ms 固定开销（客户端网络 + gateway 处理 + 服务间调用），
把竞争倍数稀释了 6 倍。**在发生竞争的那一层看，代价是 12.2 倍而非 2.1 倍。**

场景由 span 按分钟切分独立识别，QPS 与 k6 侧吻合（A 3,378 vs 3,318；B 3,949 vs 4,126）。

排队模型验证：3,378 QPS 单行串行 → 每次独占 0.296ms，并发约 100 →
预期等待 29.6ms，实测 26.9ms。属线性排队而非病态锁争用。

**这条证明了 APM 的价值**：同一现象，端到端测量会低估 6 倍，
只有分层 span 能定位到真正发生竞争的那一跳。

### 未解释的账目（不猜，留待后续）

阶段 4 场景 A 的额外延迟总量 = 202,687 次 × 24.7ms = **5,006,369 ms**，
而 MySQL `Innodb_row_lock_time` 在整个阶段 4 的增量仅 **47,330 ms** ——
**登记在册的行锁等待只能解释 0.95%**。

排队模型完美吻合、锁计数器却对不上，两者必有一个没说全。
可能是 InnoDB status 计数器不统计短事务快速排队，也可能竞争在别的层
（连接池 / latch）。**无证据，标记未解释。** 查清需开 performance_schema
等待事件采集（要改 MySQL 配置）。

> ⚠️ 计数器陷阱（已踩）：`Innodb_row_lock_waits` / `_time` / `_time_avg`
> 都是**自 MySQL 启动以来的累计值**。阶段 4 开始前实测 `lock_waits=323,928`，
> 直接引用当时读数会把 99.99% 的历史累积当成本阶段结果。**必须取增量。**

### P0 已处理：ClickHouse 资源扩容（2026-08-29 15:27）

`limits: 2 核 / 3Gi` → **6 核 / 8Gi**，`requests: 250m/1Gi` → `500m/2Gi`。
PVC 已挂载，重启数据零丢失（362 万 traces 完整保留）。
理由：2 核在 6000 条/秒下 100% 顶死，而集群有 35 核空闲 —— 保守配置是观测失效的直接原因。

> ⚠️ 这只解决第一层（配置）。② 是架构问题，扩容只是推迟；① 与性能完全无关，扩容不影响。

---

## 待处理对线 — geass-v3 电影详情 404（证据已固化，等用户择时发起）

> **状态：仅登记，未发起。** 对线协议只有用户能触发；用户 2026-08-29 指示
> 「先记下，后续需要对线」，因 geass-v3 侧正在准备压测，不打断其主线。
>
> ⚠️ ClickHouse TTL 3 天且压测会改变现场，故证据在此固化，不依赖线上数据留存。

**观测事实**（2026-08-29 14:11–14:31 JST，压测前的空闲状态）：

| 项 | 值 |
|---|---|
| 触发端点 | `POST /api/movie/detail`（geass-gateway） |
| 下游调用 | `POST http://geass-media.geass-v3.svc:9002/media.movie.v1.MovieService/GetByID` |
| 下游响应 | **404**（`error.type=404`，路由不存在） |
| 调用比 | **1:1，无重试**（593 次调用 / 593 trace / 593 父 span） |
| **用户侧结果** | **593 × 404 vs 305 × 200 —— 失败率 66%** |
| 速率 | 约 30 个失败请求/分钟，形态为脉冲（按分钟 86 / 223 / 76 / 207） |

**复现命令**（ClickHouse，atlhyper ns）：

```sql
SELECT SpanAttributes['http.response.status_code'] AS code, count()
FROM atlhyper.otel_traces
WHERE Timestamp >= now()-INTERVAL 20 MINUTE AND SpanName='POST /api/movie/detail'
GROUP BY code;
```

**已知边界（不得越过此线下结论）**：

- 观测层只能证明「gateway 调了这个路由、media 返回 404」，
  **不能**证明是 gateway 调错了地址、还是 media 少实现了这个 RPC ——
  两种可能都与证据相容，归因需 geass-v3 侧答辩
- 无法判断问题起自何时：ClickHouse 数据卷于 2026-08-29 13:43 迁移 PVC 时清空，
  现有数据不含更早历史
- 305 个成功请求说明存在另一条不经过该 RPC 的路径，原因未知

**对压测的影响**：空闲态就有 4.37% 的 gateway span 是这个错误，
压测时会等比放大并污染错误率基线 —— 建议压测前先澄清。

---

## 底层能力盘点 — ✅ 盘点完成，待决策接线范围

> 结果文档: [capability-inventory.md](../../design/active/capability-inventory.md)
>
> 四路交叉扫描（Agent 能力 81 方法 / 快照字段 / Master 端点 104 / 前端调用 84），
> 差集逐个 curl 实测。结论：**5 个真空缺口，后端全部完整**。

- **G1 `/aiops/baseline`（EMA 基线）— 唯一有真实数据的缺口** ⭐ 待接线
  实测返回每实体 4 指标的 EMA + 方差（count=53 采样周期），20 个实体在跟踪。
  数据一直在算、一直在更新，界面上无处可看。属 CLAUDE.md 的 L3 能力，
  底层就位、展示层缺席
- G2 事件模式 / G3 依赖图追踪 / G4 按资源查事件 / G5 日志摘要 — 后端完整
  但当前无数据，**不宜为接线而接线**（空面板比没面板更糟）
- **R1 参数命名不一致**：aiops 系列用 `cluster`，其余用 `cluster_id`。
  盘点时按惯例调用直接失败且错误信息不提示正确参数名 — 待修
- R2 缺少能力/展示对账机制：同类问题已发生三次，建议固化对账脚本

---

## QueryService 拆分重构 — 待办

> 原设计文档: [master-v2-query-service-split-design.md](../../design/active/master-v2-query-service-split-design.md)

- Phase 1: AdminQueryService 拆分 — 待办
  - admin.go: 新增 AdminQueryService struct + 15 个方法 receiver 变更
  - impl.go: 删除 10 个 admin repo 字段
  - factory.go + master.go: 构造注入更新
  - 验证: `go build ./...` + `go test ./atlhyper_master_v2/service/query/ -v` 全 PASS
- Phase 2: AIOpsQueryService 拆分 — 待办
  - aiops.go: 新增 AIOpsQueryService struct + 13 个方法 receiver 变更
  - impl.go: 删除 aiopsEngine, aiopsAI, aiReportRepo 字段
  - factory.go + master.go: 构造注入更新
  - 验证: `go build ./...` + `go test ./atlhyper_master_v2/service/query/ -v` 全 PASS
- Phase 3: SLOQueryService 拆分 — 待办
  - slo.go: 新增 SLOQueryService struct + 6 个方法 receiver 变更
  - impl.go: 删除 sloRepo 字段
  - slo_test.go: mock + 构造更新（23 个测试迁移）
  - factory.go + master.go: 构造注入更新
  - 验证: `go build ./...` + `go test ./atlhyper_master_v2/service/query/ -v` 全 PASS
- Phase 4: OTelQueryService 拆分 — 待办
  - otel.go: 新增 OTelQueryService struct + 2 个方法 receiver 变更
  - overview_test.go: OTel 测试构造对象从 QueryService 改为 OTelQueryService（原位更新，不搬迁文件）
  - factory.go + master.go: 构造注入更新
  - 验证: `go build ./...` + `go test ./atlhyper_master_v2/service/query/ -v` 全 PASS
- Phase 5: K8sQueryService 拆分 — 待办
  - k8s.go: 新增 K8sQueryService struct + 19 个方法 receiver 变更
  - k8s_test.go: mock + 构造更新（26 个测试迁移）
  - factory.go + master.go: 构造注入更新
  - 验证: `go build ./...` + `go test ./atlhyper_master_v2/service/query/ -v` 全 PASS
- Phase 6: OverviewQueryService 拆分 + 收尾 — 待办
  - overview.go: 新增 OverviewQueryService struct + 11 个方法 receiver 变更
  - overview_test.go: Overview 测试 mock + 构造更新（31 个测试迁移）
  - impl_test.go: 删除旧构造测试
  - impl.go: 删除 QueryService、QueryServiceDeps、NewQueryService
  - master.go: EventTrigger 引用从 q 改为 overviewQ
  - factory.go: 最终形态
  - 验证: `go build ./...` + `go test ./atlhyper_master_v2/service/query/ -v` 全 PASS

---

## 待决策事项（不阻塞开发，等用户拍板）

- **公开仓库安全加固** — 2026-08-24 全仓扫描结果
  - `.env.development` / `.env.production` 已被 git 跟踪，当前只含 `NEXT_PUBLIC_*`
    （Next.js 里本就打包进客户端，无风险）。**风险在将来**：`.gitignore` 只挡了 `.env`
    与 `.env.local`，若有人往 `.env.production` 加后端密钥会直接进公开仓且无拦截。
    选项：加醒目警示注释 / 装 husky 做 pre-commit 扫描 / 改用 `.env.example`
  - `atlhyper_agent_v2/testdata/*.txt` 含**旧集群**内网 IP 2171 处
    （192.168.0.7/.33/.46/.130/.153/.182，与现集群 .10~.23 不同），
    内容是 node-exporter 指标 dump，无凭证。`docs/design/archive/` 数篇同样含旧 IP。
    选项：脱敏当前文件（不动历史）/ 不处理（RFC1918 不可路由，风险低）
  - 已确认干净：无私钥证书、无云厂商密钥、无邮箱、Webhook 仅脱敏示例、
    部署脚本无凭证、无 CI 配置

---

## 构建提速：消灭 QEMU 模拟 — 待办（低优先级）

> 现状：buildx 多架构构建中 arm64 一半走 QEMU 用户态模拟 —— 单线程
> 吃满一个核（5.3GHz boost，Tctl 贴 95°C 温度墙），Go 编译慢 5-10 倍。
> 2026-08-29 凌晨构建实测：agent 一个镜像 10+ 分钟，多数耗在 QEMU。

- Dockerfile.agent / Dockerfile.controller：改 `FROM --platform=$BUILDPLATFORM golang`
  + `GOARCH=$TARGETARCH` 交叉编译，编译全程原生，QEMU 只剩最终层 COPY
- Dockerfile.web：builder 阶段固定 `--platform=$BUILDPLATFORM`（next build
  产物是架构无关的 JS，只跑一次），运行阶段按 TARGETPLATFORM 双架构打包
  - ⚠️ 前置检查：`npm ls` 确认无 native addon（sharp 等 .node 二进制
    是架构相关的，会破坏产物同构假设）
- 验证：两架构镜像 `docker manifest inspect` 齐全 + 树莓派节点实际拉起

---

## 代码整洁（低优先级）

- **SLO 组件目录不一致**：`components/slo/` 与其他三个观测页的
  `app/observe/*/components/` 不同。纯移动，与功能无关，建议单独一次提交
- **三处架构违规**（存量）：
  - `ObserveHandler` 持完整 `service.Query` 而非最小接口
  - `gateway/handler/admin/deploy.go` 直接持 `database.DeployConfigRepository` 等三个
  - `gateway/handler/settings/github.go` 直接持 `database.GitHubInstallationRepository` 等两个

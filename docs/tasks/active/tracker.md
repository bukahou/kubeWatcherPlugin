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

## 压测暴露的观测缺陷（2026-08-29 geass-v3 压测期间发现）

> 三个问题串在同一条链路上：③ 是根因，①② 让它以最糟的方式暴露。
> **均未修复** —— 修改需改码 + 重启 Agent，会在压测数据里打洞，待压测结束后处理。
> 证据在此固化：`system.query_log` 有 TTL，原始数据会滚掉。

### ① 节点面板在高负载下静默显示零值 ⚠️ 最危险

**现象**：压测期间「硬件健康」面板 7 个节点里 5 个显示 `CPU 0.0% / 内存 0.0% / 磁盘 0.0%`
+「未识别硬件」，用户误判为「树莓派宕机了」。**实际七个节点数据全部正常到达 ClickHouse**
（`node_load1` 每节点 20 个点、`hwmon_chip_names` 每节点 40–100 点、最新 3 秒前）。

**机制**：Agent 逐节点**串行**采集，共享一个 10s context（`AGENT_CLICKHOUSE_TIMEOUT`
默认值，未被 env 覆盖）。排在前面的节点赶在超时前完成，靠后的被 context 取消。
错误被**静默吞掉** —— `metrics.go` 的各 `fill*` 函数遇错直接 `return`，留下零值：

```go
rows, err := r.client.Query(ctx, query, ip)
if err != nil {
    return          // ← 静默返回，nm 保持零值
}
```

前端拿到零值后当作真实测量渲染成 `0.0%`，而不是「查询失败」。

**为什么这是最危险的一条**：观测系统在负载最高、最需要它的时候开始撒谎，
且撒的是"看起来合法"的谎（0.0% 而非错误提示）。

**判据（复现）**：
```bash
kubectl -n atlhyper logs deploy/atlhyper-agent --since=10m --timestamps | grep "deadline exceeded"
```

### ② Agent 日志时间戳慢 1 小时

```
容器真实时间: 2026-08-29T14:42:07.388+09:00
Agent 自己打:            time=13:42:07
```

差值恒为 1 小时（形似 TZ 被设为 UTC+8 而非 JST）。**排查时已实际踩坑一次**：
按 Agent 日志时间判断"超时一小时前就停了"，实为正在发生，靠容器时间戳才纠正。
故障复盘按日志时间对因果会整体错位一小时。

### ③ otel_metrics_gauge 排序键使时间过滤无法剪枝 —— 真根因

```
sorting_key: ServiceName, MetricName, Attributes, toUnixTimestamp64Nano(TimeUnix)
```

| 查询条件 | 能否走索引 |
|---|---|
| `MetricName = 'node_cpu_seconds_total'` | 部分（前缀 ServiceName 不定） |
| `ResourceAttributes['net.host.name'] = ?` | ❌ Map 列不在索引内，逐行取值 |
| `TimeUnix >= now() - INTERVAL 2 MINUTE` | ❌ **时间在键末位**，前三项未锁定即无法剪枝 |

**实测单次查询**（表总大小仅几 MiB）：扫 **54,898 行 / 读 19.85 MiB / 耗时 1,426ms**。
查最近 2 分钟却读该指标全部历史 → **查询成本随保留期线性增长**。

**归属注意**：`otel_metrics_gauge` 由 **OTel collector 的 `create_schema` 自动建表**，
不是 AtlHyper 建的。要改排序键须覆盖建表语句（collector 侧配置或预建表）。

**耗时随压测的变化**（`system.query_log`，时间已转 JST）：

| 时刻 | 平均 | p95 | 最大 | 失败查询 |
|---|---|---|---|---|
| 14:20 压测前 | 1284ms | 1986 | 1995 | 0 |
| 14:35 | 1677 | 3100 | 4798 | 7 |
| 14:40（用户截图时刻） | 1602 | 2873 | **6992** | 8 |
| 14:45 峰值 | 1742 | **3303** | 5698 | **12** |
| 15:00 阶段切换（流量停） | 1365 | 2104 | 2688 | 0 |
| 15:05 | 1247 | 1770 | 2201 | 0 |

**面板"自行恢复"不是修好了，是压力撤了** —— 阶段 2 重建 Pod 期间 trace 归零，
ClickHouse 空出来，查询回落到 1.2s，七节点全部赶在 deadline 内完成。

### 修复方向（待压测后决策，不在此定案）

| 层 | 方向 |
|---|---|
| ③ 根因 | 排序键前移时间 / 把 node 标识提升为独立列而非 Map / 预聚合 |
| ① 机制 | 并行采集替代串行；错误必须上抛而非静默吞；前端区分"零值"与"无数据" |
| ① 参数 | 调大 `AGENT_CLICKHOUSE_TIMEOUT`（治标） |
| ② | 修正 Agent 时区 |

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

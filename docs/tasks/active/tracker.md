# 任务追踪

> **本文件是任务状态的唯一权威源。**
> 只保留「待办」和「进行中」的任务。完成后归档到 `docs/tasks/archive/`。
>
> 状态标记：`✅` 完成 / `🔄` 进行中 / 无标记 = 待办

> **线上版本**（2026-08-25）：agent v0.6.5 / controller v0.5.2 / web v0.6.2
>
> 近期完成并已归档：节点指标硬件健康+USE 改版、观测统一时间轴与信号联动、
> 指标采集查询预算优化、**SLO 面板重构（Phase 1–4 全完成）** —— 见 `docs/tasks/archive/`

---

## APM 面板重构（对齐 Elastic APM）— 待用户确认设计

> 原设计文档: [apm-panel-redesign.md](../../design/active/apm-panel-redesign.md)
> UI 设计稿: https://claude.ai/code/artifact/af7edbfb-7d18-46fd-a31e-5c8352e4f5e5
>
> 起因：2026-08-28 排查 geass 注册 500，暴露四个问题——focus 裁剪丢上游、
> 日志按服务过滤挡住根因、ListTraces 聚合 bug（过滤先于 GROUP BY）、
> error 红色覆盖服务色。根因 `Data truncated for column 'auth_type'`
> 一直在 ClickHouse 里，UI 一个字没显示。

- Phase 1: Waterfall 视觉重构 + ListTraces 两段式聚合修正 — ✅ 开发完成（ef59e08）
- Phase 2: 错误证据链 — ✅ 开发完成（7369095）
- Phase 3: 延迟分布 brush 下钻 — ✅ 开发完成（b509e3c）
  实现偏差：纯前端（ECharts brush + 已加载样本即时过滤），
  设计文档中的 max_duration_ms 后端参数不再需要 —— 直方图数据本就来自
  已加载的 200 条样本
- Phase 4: Correlations — ✅ 开发完成（bb20d9a）
  实现偏差：打分与 impact 分级在 Agent 而非 Master —— APM 域 Master 是
  纯透传 + 缓存（无 convert 层），Agent 直出前端形态是该域既定架构
- **部署 + 真实 trace 验收 — 待办**（需构建 agent / controller / web 三镜像；
  验收用 register 500 的 trace：3 span 双色、错误卡显示 Data truncated、
  correlations 报出 iPhone）
- Future: Dependencies 独立视图（不进本期）

---

## 底层能力盘点 —— 有基础但未实现/未展示的功能 — 待分析

> 用户假设（2026-08-28）：底层已有不少功能只差继续实现或前端展示。
> 本次 APM 重构已证实两例：LogAttributes 端到端透传但 UI 从未渲染、
> SpanDrawer 已算 self-time 但没画到瀑布图上。
>
> 任务：系统扫描 Agent 查询层 / model_v3 / Master 端点 / 前端 datasource，
> 列出「数据已采集但无查询」「有查询但无端点」「有端点但无 UI」「有 UI 但
> 只用了部分字段」四类清单，按接线成本 × 用户价值排序后给方案。

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

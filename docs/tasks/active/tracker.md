# 任务追踪

> **本文件是任务状态的唯一权威源。**
> 只保留「待办」和「进行中」的任务。完成后归档到 `docs/tasks/archive/`。
>
> 状态标记：`✅` 完成 / `🔄` 进行中 / 无标记 = 待办

---

## 节点指标：硬件健康 + USE 改版 — 🔄 进行中

> 原设计文档: [node-metrics-hardware-use-design.md](../../design/active/node-metrics-hardware-use-design.md)
>
> 数据源只有 node-exporter；硬件健康置顶；每信号独立判定；无传感器显示「无数据」；阈值判定全部在 Master。

- Phase 0: 采集对齐 + 契约自检 + 删死代码 — ✅ 完成（fc1608b，agent v0.5.4 已上线）
  - Collector keep regex 16 → 55，由 `metrics_catalog.go` 的 `NodeExporterKeepRegex()` 生成 ✅（config 0343a08，ClickHouse 实测 55/55 到齐）
  - 指标名对齐：未抽常量（内联 SQL 1100+ 行，抽常量伤可读性），改为 `metrics_catalog_test.go` 扫源码守护 ✅
  - contract.go 新增 `VerifyMetricsCollected`「查而不采」检查（启动 + 10 分钟周期）✅
  - 删 GPUCard / ProcessTable + i18n 词条 ✅
  - agent v0.5.4 自检日志 `清单=55 缺失=0`；API 实测 PSI / TCP / 系统 / VMStat 四卡 7 节点全部有数据 ✅
- Phase 1a: 硬件模型 + 采集 + /metrics/hardware + 矩阵 + 速览 tile — 待办
- Phase 1b: 温度卡全传感器化 — 待办
- Phase 2a: 磁盘 USE → DiskCard — 待办
- Phase 2b: 网络/内存/CPU/系统 USE + 详情折叠 + 页面重排 — 待办
- Phase 3: 节点对比表 — 待办
- Phase 4: drivetemp / systemd / SMART — 待决策

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

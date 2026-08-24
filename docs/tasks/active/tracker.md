# 任务追踪

> **本文件是任务状态的唯一权威源。**
> 只保留「待办」和「进行中」的任务。完成后归档到 `docs/tasks/archive/`。
>
> 状态标记：`✅` 完成 / `🔄` 进行中 / 无标记 = 待办

---

## 观测模块统一时间轴与信号联动 — 🔄 进行中

> 原设计文档: [observe-unified-timerange.md](../../design/active/observe-unified-timerange.md)
>
> 四页各写各的：三套时间机制、两套刷新机制；空数据分不清「没流量」还是「采集挂了」；
> 跨信号跳转只有 Logs→APM 一条单向链路。

- Phase 1: 全局时间轴 — ✅ 代码完成（da26567）
  - timeRangeStore + useObserveTimeRange（能力声明 full/sloWindows/trendOnly）
  - SLO 贴合到五个预聚合窗口并显式提示；Metrics 范围只作用趋势图
  - URL 同步（router.replace），持久化 URL > localStorage > 默认 1h
  - 引入 vitest（项目此前无前端测试框架），21 例
- Phase 2: 数据新鲜度 — ✅ 代码完成（bed117d）
  - model_v3.SignalFreshness；Agent 三张表各查 max(时间列)，挂 Dashboard 门面
  - Master 判定 live/idle/stale/absent —— 靠 metrics（拉取式）与 traces/logs（请求触发）
    的非对称区分「没流量」与「采集挂了」
  - 四页头部统一挂 SignalFreshnessBadge
- Phase 3: 信号联动 — ✅ 代码完成（本次提交）
  - SLO → APM / SLO → Logs(ERROR) / APM trace → Logs
  - 顺带统一 trace 参数名（Logs 页此前读 traceId，与另两处不一致）
- 待部署验证：agent v0.6.1 / controller v0.5.0 / web v0.6.0 构建中

---

## SLO 面板重构 — 🔄 进行中

> 原设计文档: [slo-panel-redesign.md](../../design/active/slo-panel-redesign.md)
>
> 实测发现面板上每个数字都是错的（counter delta 跨 envoy 实例污染，偏差 24000 倍）；
> 域名显示成 serviceKey；缺燃烧率。存储决策（emptyDir）待用户拍板，Phase 1–3 不依赖它。

- Phase 1: 多实例聚合修复 + 真实域名 — ✅ 完成（上线验证）
  - 五处查询（count/latency/history/summary/timeSeries）按 service.instance.id 分区
  - V1/V2 handler 域名改用 DisplayName
  - 验收：手工 SQL 与 API 逐服务对齐 —— atlhyper-web 3256、geass-gateway 895(123 个 5xx)、
    argocd 85、geass-web 59(1 个 5xx)、akasha 18，全部一致（修复前 geass-gateway 显示 4494 万）
- Phase 2: 燃烧率 + 事件计数口径预算 + 目标模型 — ✅ 完成（上线验证）
  - 四窗口燃烧率（Google SRE 阈值），Agent 窗口集改 1h/6h/24h/3d/7d（删 30d：TTL 只有 7 天）
  - 预算改「允许错 N 个 / 已经错 M 个」，目标改一域名一套（去 time_range，加 window_days）
  - 首屏换 SLO 清单表；目标弹窗改配窗口天数
  - 线上：agent v0.6.0 / controller v0.4.9 / web v0.5.8
- Phase 3: 详情页补多窗口燃烧率表 + Good/Bad 计数 — 待办
  （清单表已含 1h/6h 两列与 已错/允许；详情页 OverviewTab 尚未补四窗口全表）
- Phase 4: 存储（依赖 emptyDir 决策 B/C）— 待决策

---

## 节点指标：硬件健康 + USE 改版 — 🔄 收尾（Phase 0–3 已上线验证；Phase 4 待决策，UI 意见待用户反馈后合并一轮）

> 原设计文档: [node-metrics-hardware-use-design.md](../../design/active/node-metrics-hardware-use-design.md)
>
> 数据源只有 node-exporter；硬件健康置顶；每信号独立判定；无传感器显示「无数据」；阈值判定全部在 Master。

- Phase 0: 采集对齐 + 契约自检 + 删死代码 — ✅ 完成（fc1608b，agent v0.5.4 已上线）
  - Collector keep regex 16 → 55，由 `metrics_catalog.go` 的 `NodeExporterKeepRegex()` 生成 ✅（config 0343a08，ClickHouse 实测 55/55 到齐）
  - 指标名对齐：未抽常量（内联 SQL 1100+ 行，抽常量伤可读性），改为 `metrics_catalog_test.go` 扫源码守护 ✅
  - contract.go 新增 `VerifyMetricsCollected`「查而不采」检查（启动 + 10 分钟周期）✅
  - 删 GPUCard / ProcessTable + i18n 词条 ✅
  - agent v0.5.4 自检日志 `清单=55 缺失=0`；API 实测 PSI / TCP / 系统 / VMStat 四卡 7 节点全部有数据 ✅
- Phase 1a: 硬件模型 + 采集 + /metrics/hardware + 矩阵 + 速览 tile — ✅ 完成（4838e1d / e7c4609）
  - model_v3: NodeHardware（欠压/风扇/散热）+ FreqMaxHz + await/queue + ChipName + 画像/分类纯函数
  - Agent: fillHardware、全传感器温度、频率、await；清单 55 → 65
  - Master: 画像阈值表 + GET /observe/metrics/hardware，判定全在后端
  - Web: HardwareMatrix + HardwareSummaryTiles
  - 集群实测 7 节点全部出数；盘温无自报阈值时改用磁盘档（e7c4609）
- Phase 1b: 温度卡全传感器化 — ✅ 完成（a3c4530）
  - HardwareRow.Sensors 逐传感器判定（与矩阵共用同一套阈值来源）
  - TemperatureCard 按 CPU/磁盘/其他分组 + 供电/风扇行，无传感器显示「无数据」
- Phase 2a: 磁盘 USE → DiskCard — ✅ 完成（3875883）
  - inode 使用率 + 只读标记；DiskCard 按 利用率/饱和度/错误 三列重排
- Phase 2b: 网络/内存/CPU/系统 USE — ✅ 完成（3875883）
  - OOMKillTotal（累计值）、ProcsRunning/Blocked、ArpEntries、校时；CPU 每核负载
  - swap 段改为常驻显示（「关着」与「查不到」是两回事）
  - 清单 65 → 74，config f0a6ee2 已部署
- Phase 3: 节点对比表 — ✅ 完成（1dcc186）
  - GET /observe/metrics/compare（硬件 5 列 + 资源 6 列），硬件列复用矩阵判定不重算
  - NodeCompareTable：节点列 sticky、硬件/资源分隔线、异常格加粗
- 部署后实测修复 — ✅ 完成（93ef979 / 上一条 commit）
  - 磁盘条目分裂（filesystem 与 diskstats 设备名不一致）→ normalizeBlockDevice + IO 回填到分区行
  - 磁盘 IO 只取整盘（/proc/diskstats 同时上报 mmcblk0 与 mmcblk0p1/p2，分区行重复）
  - 盘用量取根分区（原取所有分区最大值 → 三台 raspi5 都显示 512MB boot 分区的 37.24%）
  - 网络错误列只算 err 不算 drop（虚拟网卡丢组播是常态，原来 7 台全黄）
  - PSI 阈值 warn 50 / crit 80（K8s 节点常态 20–60%，原 crit 50 把 desk-one 标红）
  - CPU 频率查询窗口 2 → 5 分钟 + 失败记 Warn（偶发某节点显示无数据，每次换一台）
  - 复验：频率列 7 节点全有数据；raspi4/raspi5 各 2 行零孤儿；desk 仅剩 dm-0 一行
  - 线上版本：agent v0.5.8 / controller v0.4.7 / web v0.5.7
- 已知限制（不修）：LVM 根分区（/dev/mapper/x）与 diskstats 的 dm-N 无法从名字关联，
  三台 desk 的根分区行拿不到 IO，dm-0 独立成行。宁可缺 IO 也不张冠李戴。
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

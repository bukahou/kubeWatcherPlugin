# 节点指标：硬件健康 + USE 改版

> 已完成并归档。原任务追踪见 `docs/tasks/active/tracker.md` 的历史版本。
> 设计文档：`docs/design/archive/node-metrics-hardware-use-design.md`
> 完成时间：2026-08-24

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

## Phase 4 决策结果（2026-08-24）

| 项 | 决策 | 说明 |
|----|------|------|
| desk 加载 `drivetemp` | ✅ 已做 | 两行命令的事，三台 desk 的 SATA SSD 温度从「无数据」变成 43–46°C。IaC 落在 config 仓 `clusters/local_init/k0s/host-modules/` |
| `systemd` collector | ❌ 不做 | 集群里唯一相关的 unit 只有 k0scontroller，其状态 K8s Node condition 已覆盖；而读 systemd 要给 DaemonSet 挂 `/run/systemd/private`，为一个 unit 提权不划算 |
| SMART | ❌ 不做 | 用户决定。（顺带发现 desk 上 smartmontools 本来就在跑，将来要做门槛很低） |

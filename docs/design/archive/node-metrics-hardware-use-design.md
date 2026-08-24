# 节点指标：硬件健康 + USE 改版设计

> 状态: 执行中
> 创建时间: 2026-08-23
> 设计稿（布局参考，样式以现有组件为准）: https://claude.ai/code/artifact/3ed6a288-08b6-4f5b-b2f4-5884573ea0c7

## 背景 + 目标

### 现状（2026-08-23 实测）

```
node-exporter v1.8.2 暴露   274 个指标族（7 节点全覆盖，hostNetwork）
Collector keep               16 个   ← 瓶颈
Agent 查询引用               55 个
```

**39 个指标 Agent 在查、Collector 没采**，导致 PSI / TCP / VMStat / System 四张卡全空，
Disk / Network / Memory / Temperature 半空。与已清理的 `kube_*`（采而不查）是同一种病的镜像。

GPUCard / ProcessTable 无后端模型、无页面引用，是死代码（集群节点无 GPU，
node-exporter 不提供逐进程数据）。

### 集群特性决定的优先级

集群是**消费级硬件 7/24 运行**（HP 迷你机 + 树莓派，WD Blue SA510 无 PLP、Pi4 用 MicroSD），
过热 / 欠压 / 磁盘过劳会造成不可逆损坏。两次真实故障（config 仓 `incidents/`）里，
8/22 的根因是 SSD fsync 7.8s——`node_disk_write_time_seconds_total` 一直在 node-exporter 里，
只是没被采集。

因此：**硬件健康置顶，四大资源（CPU / 内存 / 磁盘 / 温度）突出，其余收进详情。**

### 目标

1. 采集与查询两端对齐，并用契约自检防止再次漂移
2. 硬件健康成为一等公民：温度（全传感器）/ 欠压 / 风扇 / 降频 / 磁盘压力
3. 四大资源按 USE（利用率 / 饱和度 / 错误）补齐
4. 新增节点对比表，回答「哪台不一样」
5. 节点没有的传感器显示「无数据」，不隐藏

## 核心架构

```
node-exporter (:9100, 7 节点)
   │ Prometheus scrape 15s，keep 由 Agent 查询清单驱动
   ▼
Collector → ClickHouse otel_metrics_{gauge,sum}
   │
   ▼
Agent repository/ch/query/metrics.go   ← 指标名 Go 常量集中；SQL 只认 node_* 事实标准名
   │ Service → Repository → SDK，不跳层
   ▼
OTelSnapshot.MetricsNodes []metrics.NodeMetrics   ← model_v3 即 API 契约（camelCase）
   │
   ▼
Master
   ├── /observe/metrics/nodes        直接返回 MetricsNodes（现状，不变）
   ├── /observe/metrics/summary      扩展硬件 tile（service/query 计算）
   ├── /observe/metrics/hardware     [新] 硬件健康矩阵，含 status（service/query 判定）
   └── /observe/metrics/compare      [新] 节点对比表，含 status（service/query 判定）
   │
   ▼
Web /observe/metrics   布局：速览 → 硬件矩阵 → 四大资源 → 节点对比 → 详情（折叠）
```

### 四条原则

| 原则 | 含义 |
|------|------|
| **数据源唯一** | node-exporter，不引 dcgm / process-exporter / SMART exporter。SMART 等扩展走 node-exporter 自身的 textfile collector |
| **不抽契约层** | `node_*` 是 Prometheus 生态事实标准，十年未变，没有第二个实现。套契约层是为一致性牺牲可读性。只把指标名做成 Go 常量集中定义 |
| **独立判定** | 每个信号只看自己的阈值：温度超了标温度，CPU 高了标 CPU。跨信号关联（如"空转却高温"）属于 AIOps，不在本页 |
| **大后端小前端** | 阈值、status、排序、"最高温节点"全部在 Master `service/query` 算好；前端只渲染 chip 颜色 |

### 硬件画像（thresholds 的选择依据）

节点类型由 Agent 上报的 `Kernel`（uname machine）+ 传感器 chip 集合识别，Master 据此选阈值表。
**不硬编码节点名。**

| 画像 | 识别 | CPU 温 max/crit | 备注 |
|------|------|-----------------|------|
| x86 desk | `x86_64` + `coretemp` | 传感器自带；缺则 85 / 100（i7-7700T Tj=100） | SATA SSD 温度需 `drivetemp` 模块 |
| raspi5 | `aarch64` + `nvme` 或 `pwm-fan` | 80 / 85（BCM2712 85°C 降频） | NVMe 自带 max/crit；有 `rpi_volt` 欠压位 |
| raspi4 | `aarch64` + 无 nvme | 80 / 85（BCM2711） | 无盘温；有欠压位 |

阈值优先级：**传感器自带 `_max`/`_crit` > 画像表 > 通用保守值（温度 70/85）**。

---

## 功能一：采集对齐与契约自检（Phase 0）

### 用户故事

打开指标页，PSI / TCP / VMStat / 系统资源 四张卡有数据；Collector 与 Agent 两端若再漂移，
Agent 日志 10 分钟内报 ERROR。

### 改动

| 层 | 内容 |
|----|------|
| Collector（config 仓） | `node-exporter` job 的 keep regex 从 16 个改为 Agent 引用的全部指标（以 Agent 常量清单为准，见功能二/三的新增） |
| Agent `repository/ch/query/metrics.go` | 指标名抽成包级常量块 `metricNode*`（现状散在 SQL 字符串里） |
| Agent `repository/ch/query/contract.go` | 新增 **查而不采** 检查：对 Agent 引用的每个 MetricName，近 15 分钟在 ClickHouse 必须有行；缺失 → ERROR `[契约自检] 指标未采集` |
| Web | 删除 `GPUCard.tsx`、`ProcessTable.tsx` 及 i18n 词条 |

### 验收

- 四张空卡出数据
- Agent 日志出现 `[契约自检] 通过 table=otel_metrics column=MetricName 缺失=0`
- 故意从 keep 去掉一个指标，10 分钟内看到 ERROR（手动验证一次）

---

## 功能二：硬件健康（Phase 1）

### 用户故事

速览区一眼看到：最高温在哪台、NVMe 温度、有没有欠压、有没有热降频、磁盘 await 最高；
矩阵里 7 节点 × 8 列逐格 chip；缺传感器的格子显示「无数据」。

### API

`GET /api/v2/observe/metrics/hardware?cluster_id=requiem`

```json
{
  "message": "获取成功",
  "data": {
    "rows": [
      {
        "nodeName": "raspi5-one", "profile": "raspi5", "profileLabel": "Pi 5 · NVMe KIOXIA",
        "cpuTemp":   {"value": 60.0, "max": 80, "crit": 85, "status": "good"},
        "diskTemp":  {"value": 49.9, "max": 82.9, "crit": 84.9, "status": "good", "label": "nvme0"},
        "otherTemp": {"value": 61.3, "max": 85, "crit": 85, "status": "good", "label": "RP1"},
        "undervolt": {"alarm": false, "status": "good"},
        "fan":       {"rpm": 5422, "state": 2, "maxState": 4, "status": "good"},
        "cpuFreq":   {"currentGHz": 1.5, "maxGHz": 2.4, "ratioPct": 62, "status": "good"},
        "diskAwait": {"valueMs": 1.1, "device": "nvme0n1", "status": "good"},
        "overall": "good"
      },
      {
        "nodeName": "desk-zero", "profile": "desk", "profileLabel": "HP 800 G3 · i7-7700T",
        "cpuTemp":   {"value": 50, "max": 85, "crit": 100, "status": "good"},
        "diskTemp":  null,
        "otherTemp": null,
        "undervolt": null,
        "fan":       null,
        "cpuFreq":   {"currentGHz": 0.8, "maxGHz": 3.8, "ratioPct": 21, "status": "good"},
        "diskAwait": {"valueMs": 1.2, "device": "sda", "status": "good"},
        "overall": "good"
      }
    ],
    "summary": {
      "maxTemp":      {"value": 65, "nodeName": "desk-two", "sensor": "coretemp core1", "status": "good"},
      "maxDiskTemp":  {"value": 71, "nodeName": "raspi5-two", "status": "warn"},
      "undervoltNodes": 1,
      "throttledNodes": 0,
      "maxDiskAwait": {"valueMs": 124, "nodeName": "desk-one", "device": "sda", "status": "warn"}
    }
  }
}
```

约定：**字段为 `null` = 该节点无此传感器**，前端渲染「无数据」chip。`status` 取值 `good | warn | crit`。

### 判定规则（独立，不关联）

| 格 | warn | crit |
|----|------|------|
| cpuTemp / diskTemp / otherTemp | ≥ max | ≥ crit |
| undervolt | — | `alarm == true` |
| fan | 温度 ≥ max 且 rpm == 0（风扇停转）| — |
| cpuFreq | ratio < 60% **且** 节点 CPU 使用率 > 50%（排除空闲降频）| — |
| diskAwait | ≥ 50 ms | ≥ 200 ms |
| overall | 任一 warn | 任一 crit |

> cpuFreq 这条是"单指标内部"的判定（频率低本身无意义，要结合该节点自己的负载才构成"热降频"），
> 不属于跨节点/跨信号关联。

### 数据对照表

| 前端 TS（`api/node-metrics.ts`） | Go `model_v3/metrics` | ClickHouse MetricName |
|---|---|---|
| `NodeTemperature.sensors[]` (已有) | `TempSensor{Chip,Sensor,CurrentC,MaxC,CritC}` (已有) | `node_hwmon_temp_celsius` / `_max_celsius` / `_crit_celsius`、`node_thermal_zone_temp` |
| `NodeHardware.undervoltAlarm?: boolean` | `NodeHardware.UndervoltAlarm *bool` | `node_hwmon_in_lcrit_alarm_volts{chip=~"rpi.*hwmon"}` |
| `NodeHardware.fans[]` | `FanSensor{Chip,Sensor,RPM}` | `node_hwmon_fan_rpm` |
| `NodeHardware.cooling[]` | `CoolingDevice{Name,Type,CurState,MaxState}` | `node_cooling_device_cur_state` / `_max_state` |
| `NodeCPU.freqHz[]` (已有) + `NodeCPU.freqMaxHz` | `NodeCPU.FreqMaxHz float64` | `node_cpu_scaling_frequency_hertz` / `_max_hertz` |
| `NodeDisk.awaitReadMs / awaitWriteMs / queueDepth` | `NodeDisk.AwaitReadMs, AwaitWriteMs, QueueDepth float64` | `node_disk_{read,write}_time_seconds_total` ÷ `_{reads,writes}_completed_total`、`node_disk_io_time_weighted_seconds_total` |
| `NodeMetrics.hardwareProfile` | `NodeMetrics.HardwareProfile string` | 由 Agent 按 `node_uname_info{machine}` + hwmon chip 集合推导 |

`NodeHardware` 为新结构体，挂在 `NodeMetrics.Hardware`。温度沿用现有 `NodeTemperature`（已含全传感器 + 阈值），
只需 Agent 查询把 `thermal_zone` 和所有 hwmon chip 都填进 `Sensors`（现状只取 CPU）。

### 后端

- Agent `metrics.go`：新增 `fillHardware`（欠压 / 风扇 / 散热）、`fillDisks` 补 await 与 queue、`fillCPU` 补 freqMax、`fillTemperature` 改全量传感器、`detectHardwareProfile`
- Master `service/query/metrics_hardware.go` [新]：`GetHardwareHealth(ctx, clusterID)` — 读 `OTelSnapshot.MetricsNodes`，按画像选阈值，产出上面的 JSON；阈值表 `metrics_thresholds.go` [新]
- Master `service/interfaces.go`：`QueryOTel` 加 `GetHardwareHealth`
- Master `gateway/handler/observe/observe_metrics.go`：加 `MetricsHardware` handler；`routes.go` 注册
- `model/` 加响应结构 `HardwareHealthResponse` 等（Master API 响应模型，与 model_v3 分开）

### 前端

- `api/observe-metrics.ts`：`getMetricsHardware`
- `app/observe/metrics/components/HardwareMatrix.tsx` [新]、`HardwareSummaryTiles.tsx` [新]
- `TemperatureCard.tsx`：改为全传感器列表 + 阈值刻度 + 电压/风扇行
- i18n：`nm.hardware.*`

---

## 功能三：四大资源 USE（Phase 2）

### 用户故事

CPU / 内存 / 磁盘 / 温度 四张大卡首屏；每张卡底部三格：利用率 / 饱和度 / 错误。
网络 / PSI / TCP / VMStat / 系统 收进「详情」折叠区。

### 数据对照表（新增字段）

| 资源 | 前端 TS | Go | ClickHouse |
|---|---|---|---|
| 内存 E | `NodeVMStat.oomKillTotal` | `NodeVMStat.OOMKillTotal int64` | `node_vmstat_oom_kill` |
| 磁盘 E | `NodeDisk.readOnly`, `inodeUsagePct` | `NodeDisk.ReadOnly bool`, `InodeUsagePct float64` | `node_filesystem_readonly`, `node_filesystem_files` / `_files_free` |
| 网络 S | `NodeNetwork.saturationPct` | `NodeNetwork.SaturationPct float64` | (rx+tx bytes/s) ÷ `node_network_speed_bytes` |
| 网络 E | `NodeNetwork.arpEntries` | `NodeNetwork.ArpEntries int64` | `node_arp_entries` |
| CPU S | `NodeSystem.procsBlocked / procsRunning`, `contextSwitchesPerSec` | 同名 | `node_procs_blocked`, `node_procs_running`, `node_context_switches_total` |
| 系统 | `NodeSystem.timeSync{offsetMs, synced}` | `NodeSystem.TimeOffsetMs float64, TimeSynced bool` | `node_timex_offset_seconds`, `node_timex_sync_status` |

PSI / TCP / Softnet 字段已存在，只缺采集（功能一解决）。

### 后端 / 前端

- Agent：各 `fill*` 补字段
- Master：`/observe/metrics/summary` 无改动（四大卡直接读 `MetricsNodes`）
- 前端：`CPUCard / MemoryCard / DiskCard` 加 USE 三格；`page.tsx` 重排：速览 → 硬件矩阵 → 四大 → 对比 → `<details>` 详情

---

## 功能四：节点对比表（Phase 3）

### API

`GET /api/v2/observe/metrics/compare?cluster_id=requiem`

```json
{ "data": { "columns": ["temp","freqRatio","undervolt","diskAwait","diskUtil","cpu","psiCpu","mem","psiMem","netErr","netDrop"],
  "rows": [ { "nodeName": "desk-one", "cells": { "temp": {"value": 52, "status": "good"}, "diskAwait": {"value": 124, "status": "warn"}, ... }, "overall": "warn" } ] } }
```

列顺序硬件优先，由后端固定；每格带 status；前端只渲染。阈值复用功能二的表。

### 后端 / 前端

- Master `service/query/metrics_compare.go` [新] `GetNodeComparison`；`QueryOTel` 加方法；handler + 路由
- 前端 `NodeCompareTable.tsx` [新]

---

## Phase 4（需决策，不在本轮）

| 项 | 做法 | 收益 |
|----|------|------|
| desk 加载 `drivetemp` | config 仓 `local_init` 加 `modules-load.d`，各 desk 执行一次 | 三块 SATA SSD 温度可见 |
| `systemd` collector | node-exporter DaemonSet 加参数 | k0s / containerd unit 状态 |
| SMART | node-exporter textfile collector + cron `smartctl` | SSD 写寿命；SD 卡无 SMART |

---

## 实施阶段

每个 Phase 是一个闭环（用户在 UI 看得到完整变化），按资源切而非按层切。

- Phase 0: 采集对齐 + 契约自检 + 删死代码
- Phase 1a: 硬件模型 + Agent 采集 + `/metrics/hardware` + 矩阵 + 速览 tile
- Phase 1b: 温度卡全传感器化（含电压 / 风扇行）
- Phase 2a: 磁盘 USE（await / 队列 / inode / 只读）→ DiskCard
- Phase 2b: 网络 / 内存 / CPU / 系统 USE → 对应卡 + 详情折叠 + 页面重排
- Phase 3: 节点对比表
- Phase 4: 决策后另开

每个 Phase：TDD（测试先红）→ 实现 → `go build` / `next build` → Phase 验收清单四项 → commit → tracker 更新。
model_v3 变更的 Phase，agent 与 controller 同批构建部署。

## 文件变更清单

```
config 仓 clusters/requiem/apps/atlhyper/
└── collector.yaml                                   [修改] node-exporter keep regex

model_v3/metrics/
└── node_metrics.go                                  [修改] NodeHardware / 各结构体新字段 / HardwareProfile

atlhyper_agent_v2/repository/ch/query/
├── metrics.go                                       [修改] 指标常量块、fillHardware、各 fill* 补字段
├── metrics_test.go                                  [新增] SQL 断言 + 画像识别 + await 计算
└── contract.go                                      [修改] 查而不采检查

atlhyper_master_v2/
├── service/interfaces.go                            [修改] QueryOTel 加 GetHardwareHealth / GetNodeComparison
├── service/query/metrics_thresholds.go              [新增] 画像阈值表
├── service/query/metrics_hardware.go                [新增] 硬件健康判定
├── service/query/metrics_compare.go                 [新增] 节点对比
├── service/query/metrics_hardware_test.go           [新增]
├── service/query/metrics_compare_test.go            [新增]
├── model/metrics.go                                 [新增] HardwareHealthResponse / NodeComparisonResponse
├── gateway/handler/observe/observe_metrics.go       [修改] 两个 handler
└── gateway/routes.go                                [修改] 两条路由

atlhyper_web/src/
├── api/node-metrics.ts                              [修改] 类型镜像 model_v3
├── api/observe-metrics.ts                           [修改] getMetricsHardware / getMetricsCompare
├── app/observe/metrics/page.tsx                     [修改] 布局重排
├── app/observe/metrics/components/
│   ├── HardwareSummaryTiles.tsx                     [新增]
│   ├── HardwareMatrix.tsx                           [新增]
│   ├── NodeCompareTable.tsx                         [新增]
│   ├── DetailsSection.tsx                           [新增] 折叠区
│   ├── TemperatureCard.tsx                          [修改] 全传感器
│   ├── CPUCard / MemoryCard / DiskCard.tsx          [修改] USE 三格
│   ├── GPUCard.tsx                                  [删除]
│   └── ProcessTable.tsx                             [删除]
├── types/i18n.ts + i18n/locales/{zh,ja}.ts          [修改]
└── mock/                                            [修改] 对齐新字段
```

## 附录：约束与风险

- **硬件画像识别可能误判**：新加一种硬件时阈值退回通用保守值（70/85），UI 显示画像为 `unknown`，不报错
- **数据量**：node_* 从 16 族增至约 75 族，预估 ~3000 万行 / 7 天，列存 30 MB 级
- **`drivetemp` 未加载前** desk 的 SSD 温度列恒为「无数据」，这是正确行为，不是 bug
- **Master 与 Agent 版本**：Phase 1a 改 model_v3 后，旧 Agent 上报的快照缺 `hardware` 字段，Master 判定函数必须对 nil 容错（显示无数据）

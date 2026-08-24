# Agent Metrics / Summary 5min 窗口 Counter Reset 修复设计

> 清理节点指标和集群概览中 7 处裸 `argMax-argMin` rate 计算的 counter reset 隐患。方案：复用已有 `gaugeCounterDelta` 单次 reset 处理 + 抽 DRY 常量 `counterRateExpr`。

---

## 1. 背景

上一个任务（Ingress SLO counter reset 修复）完成后发现项目中还有 7 处类似隐患。本任务清理完，结束本类技术债。

与 SLO 修复的关键差异：**窗口长度不同**，所以方案也不同。

| 任务 | 窗口 | 方案 | Reset 风险 |
|------|------|------|-----------|
| 上次 SLO Ingress 修复 | 1d/7d/30d | Prometheus window function（完整） | 多次 reset 是常态 |
| **本次 Node Metrics** | **5min** | **`gaugeCounterDelta`（单次 reset 安全）** | 多次 reset 概率 ≈ 0 |

## 2. 受影响位置

| # | 函数 / SQL 位置 | 指标 | 分组 |
|---|----------------|------|------|
| 1 | `metrics.go` `fillCPU` | `node_cpu_seconds_total` | (cpu, mode) |
| 2 | `metrics.go` `fillDisks` ioQuery | `node_disk_*_total` × 5 | (device, metric) |
| 3 | `metrics.go` `fillNetworks` rateQuery | `node_network_*_total` × 8 | (iface, metric) |
| 4 | `metrics.go` `fillPSI` | `node_pressure_*_total` × 5 | metric |
| 5 | `metrics.go` `fillVMStat` rateQuery 模板 | vmstat × 4 + softnet × 2 | metric |
| 6 | `summary.go` `GetSLOSummary` ingressQuery | `traefik_service_requests_total` | svc |
| 7 | `summary.go` `GetMetricsSummary` cpuQuery | `node_cpu_seconds_total` | (ip, cpu, mode) |

所有位置都是：
```sql
(argMax(Value, TimeUnix) - argMin(Value, TimeUnix)) /
(toUnixTimestamp(argMax(TimeUnix, TimeUnix)) - toUnixTimestamp(argMin(TimeUnix, TimeUnix))) AS rate
```

## 3. 修复方案

### 3.1 新增 `counterRateExpr` 常量（DRY 优化）

在 `slo.go`（`gaugeCounterDelta` 定义处的下方）添加：

```go
// counterRateExpr 按 counter-reset-safe delta 除以时间跨度，得到 per-second rate。
// 使用场景：5min 等短窗口，reset 概率极低，单次 reset 处理足够。
// 对于长窗口（1d/7d/30d），应改用 lagInFrame 的 Prometheus 完整算法（参见 queryIngressSLO）。
const counterRateExpr = gaugeCounterDelta + ` /
    (toUnixTimestamp(argMax(TimeUnix, TimeUnix)) - toUnixTimestamp(argMin(TimeUnix, TimeUnix)))`
```

### 3.2 7 处 SQL 字符串改为字符串拼接

**改动前（裸字面量）：**
```go
query := `
    SELECT Attributes['mode'] AS mode,
           (argMax(Value, TimeUnix) - argMin(Value, TimeUnix)) /
           (toUnixTimestamp(argMax(TimeUnix, TimeUnix)) - toUnixTimestamp(argMin(TimeUnix, TimeUnix))) AS rate
    FROM ...
`
```

**改动后（拼接常量）：**
```go
query := `
    SELECT Attributes['mode'] AS mode, ` + counterRateExpr + ` AS rate
    FROM ...
`
```

无 reset 时结果与原实现完全等价（ClickHouse 实测验证）。单次 reset 时正确补偿（算法与 Linkerd mesh 一致）。

## 4. 不做的事

- ❌ **不用 window function 方案**：5min 窗口多次 reset 概率近零，大炮打蚊子
- ❌ **不重命名 `gaugeCounterDelta`**：避免扰动 Linkerd 现有代码，注释说明通用性即可
- ❌ **不修 `fillVMStat` 的 gauge 表语义问题**：vmstat 是 counter 却被 OTel 存到 gauge 表，这是 receiver 层面的分类问题，独立任务

## 5. 文件变更清单

```
atlhyper_agent_v2/
└── repository/ch/
    ├── query/slo.go [修改]
    │   └── 新增 counterRateExpr 常量（gaugeCounterDelta 下方）
    ├── query/metrics.go [修改]
    │   ├── fillCPU                      (5 处之 1)
    │   ├── fillDisks ioQuery            (5 处之 2)
    │   ├── fillNetworks rateQuery       (5 处之 3)
    │   ├── fillPSI                      (5 处之 4)
    │   └── fillVMStat rateQuery 模板    (5 处之 5)
    └── summary.go [修改]
        ├── GetSLOSummary ingressQuery   (2 处之 1)
        └── GetMetricsSummary cpuQuery   (2 处之 2)
```

## 6. 验证计划

### 6.1 ClickHouse 实测（已完成）
`node_cpu_seconds_total` 的新 SQL 输出与原 SQL 在无 reset 时完全一致（idle 0.9917... 等），证明算法兼容。

### 6.2 编译 + 单元测试
```bash
go build ./...
go test ./atlhyper_agent_v2/repository/ch/...
```

### 6.3 本地端到端
本地运行 Agent → Master → Web：
- 节点监控页面（/cluster/node/<name>）：CPU / Disk / Network / PSI / VMStat 数据正常
- 概览页面（/overview）：集群 CPU、Ingress RPS 数据正常

## 7. 风险与回滚

### 风险
- SQL 字符串拼接对占位符（如 fillVMStat 的 `%s`）敏感。`gaugeCounterDelta` / `counterRateExpr` 内部不含 `%`，无冲突。
- ClickHouse 新增一个 `max(Value)` 聚合函数，性能影响可忽略。

### 回滚
Git revert 单个 commit 即可。SQL 修改不涉及 schema 或数据迁移。

## 8. 与上一任务的延续关系

- 上一任务 commit 712f902 用方案 B（Prometheus window function）修 Ingress 长窗口
- 本任务用方案 A（`gaugeCounterDelta`）修节点 metrics 短窗口
- 两种方案的选择依据在 tracker 和 commit message 中有明确记录
- 本任务完成后，项目中所有 counter rate 计算都具备 reset 安全性

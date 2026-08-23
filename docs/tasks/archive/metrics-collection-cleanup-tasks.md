# 指标采集瘦身 — 已完成

> 记录时间：2026-08-23
> 完成时间：2026-08-23
> 来源：SLO 改造调研过程中的附带发现，单独立项。

## 背景

`otel_metrics_gauge` 表当前 **1.08 亿行**，其中绝大部分来自 kube-state-metrics
采集的 65 个 `kube_*` 指标（`kube_pod_restart_policy`、`kube_deployment_spec_replicas`、
`kube_pod_status_container_ready_time` 等）。

**这 65 个指标 Agent 一次都没有查询过。**

实测对比（`repository/ch/query/metrics.go` 实际引用的 MetricName）：

```
node_boot_time_seconds        node_cpu_seconds_total
node_filesystem_avail_bytes   node_filesystem_size_bytes
node_hwmon_temp_celsius       node_hwmon_temp_crit_celsius
node_hwmon_temp_max_celsius   node_network_mtu_bytes
node_network_speed_bytes      node_network_up
node_uname_info               node_load1
```

全部是 `node_*`（node-exporter）。K8s 对象状态（Pod/Deployment/Node 的副本数、
状态、重启次数等）由 Agent 直接从 API Server 拉快照获得，**不经过 kube-state-metrics**。

因此 kube-state-metrics 这一路是纯写入、零读取。

## 需要评估的问题

1. **是否完全移除 kube-state-metrics scrape**
   - 移除后存储压力大幅下降（这是当前最大的单项收益）
   - 风险：未来若要做「K8s 对象的时序趋势」（如某 Deployment 副本数的历史变化），
     快照方式给不了历史，仍需 kube-state-metrics
2. **若保留，是否收窄 keep 规则**
   - 现规则：`kube_(pod|deployment|node|namespace)_.*` —— 过于宽泛
   - 可只保留确有分析价值的少数几个
3. **ClickHouse 侧的历史数据处理**
   - `ttl: 168h` 会自然过期，是否需要提前 TRUNCATE/DROP PARTITION

## 关联

- 同类问题：Collector 里 `istio-sidecars` scrape job 在 Istio 拆除后仍在空转
  （0 target），一并清理
- SLO 改造会新增 cilium-envoy scrape，届时可一并 review 全部 scrape job 的必要性


---

## 决策与执行（2026-08-23）

### 65 个指标的语义分类

| 类别 | 代表指标 | 快照能否替代 | 结论 |
|------|---------|-------------|------|
| 静态元数据 | kube_pod_info / _owner / _tolerations / _restart_policy | ✅ | 纯浪费（tolerations 单项 7.9 万行，值终生不变） |
| 实时状态 | kube_pod_status_phase / _ready / kube_node_status_condition | ✅ 且更细（10s vs 30s） | 冗余 |
| 容量配额 | kube_node_status_allocatable / capacity | ✅ 当前值 | 趋势分析才需要历史 |
| **副本/滚动更新** | kube_deployment_status_replicas_* / observed_generation | ❌ | **唯一不可替代** |
| 异常重启 | kube_pod_container_status_last_terminated_* | ✅ 且 AIOps 已落成 Incident | 补充性 |

### 决策：方案 A（完全移除）

理由：唯一不可替代的第 4 类，服务的是「滚动更新复盘」「容量趋势」这类功能，
当前不在规划内；且 TTL 仅 168h，本就没有长期历史可言。将来要做时再打开，
届时把 keep regex 收窄到真正需要的少数几个即可。

### 执行

- config 仓 `851c5fa`：Collector 删除 kube-state-metrics scrape job，
  注释里保留「何时该加回来」的说明与建议的收窄 regex
- ClickHouse：`ALTER TABLE otel_metrics_gauge DELETE WHERE MetricName LIKE 'kube_%'`
  清理存量 9594 万行

### 一处认知修正

原记录暗示这是「最大的存储收益」——**这个判断是错的**。ClickHouse 列存压缩后
1.036 亿行仅占 87 MB（约 0.9 字节/行），删除只省约 80 MB，且 data 卷是 emptyDir。

真实收益是三项较小的成本：查询扫描量（不带 MetricName 过滤时少扫 13 倍行，
契约自检的 i/o timeout 即源于此）、写入与网络开销、以及认知负担
（1 亿行会误导后来者对系统规模的判断）。

**性质是「清理无人使用的管道」，不是性能优化。**

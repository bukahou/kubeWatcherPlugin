# 指标采集瘦身 — 待规划

> 记录时间：2026-08-23
> 来源：SLO 改造调研过程中的附带发现，**不属于 SLO 改造范围**，单独立项。

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

# 平台组件健康 — 待规划

> 记录时间：2026-08-23
> 来源：两次真实故障（config 仓 `clusters/incidents/` 2026-08-22、2026-08-23）
> 状态：待详细安排，本文件只记录方案骨架与已核实的事实

## 背景

两次故障根因完全不同，但 AtlHyper 当时看到的全是绿的：

| | 2026-08-22 | 2026-08-23 |
|---|---|---|
| 根因 | 消费级 SSD fsync 7.8s → etcd 换主 → Cilium watch 中断 | 三台 desk 的 ARP 表把 raspi5-one 的 IP 记成 CPLB VIP 的 MAC |
| 平台看到的 | 7 节点 Ready、Pod 全 Running、CPU/内存/带宽正常 | 同上 |
| 共同症状 | **跨节点 Pod 流量不通** | **跨节点 Pod 流量不通** |
| 发现方式 | 用户报告，约 40 分钟后 | 用户报告，2 小时 21 分后 |

**结论：AtlHyper 监控一切，唯独不监控让自己能工作的那层。** 且不能按根因逐个补
（下次会是第三种根因），要先抓症状类，再补根因信号。

## 方案骨架（三层）

### 第 1 层：症状检测 —— 跨节点 Pod 连通性

| 方案 | 来源 | 成本 | 依赖 |
|---|---|---|---|
| A. 采 Cilium 连通矩阵 | `cilium_unreachable_nodes` / `cilium_unreachable_health_endpoints`（cilium-health 每分钟已在算，即 `Cluster health 5/7` 那个数） | Collector 加 scrape | **需开 9962**（`cilium-config` 加 `prometheus-serve-addr: :9962`）并重启 DaemonSet —— 等 Cilium 根因排查完 |
| B. AtlHyper 自建探针 | 轻量 DaemonSet 互 ping 对端 Pod，上报可达矩阵 | 新组件 ~200 行 | 与 CNI 无关，换 Calico 照用 |

契约名：`cluster_unreachable_nodes`。A 从 Cilium 映射，B 自产。Agent 只认契约名。
建议 A 先上、B 作后备。

### 第 2 层：根因信号

| 信号 | 来源 | 成本 | 对应故障 |
|---|---|---|---|
| 磁盘 fsync 延迟 | node-exporter `node_disk_write_time_seconds_total` ÷ `writes_completed_total` | **极低**，随指标对齐（M1）一并解决 | 8/22 |
| etcd 健康 | k0s etcd `/metrics`（2379，需客户端证书）| 需调查 k0s 的暴露方式 | 8/22 |
| ARP 一致性 | 无被动指标，只能主动探测（incidents 报告里的比对脚本） | 需宿主机级执行载体：hostNetwork DaemonSet 或 cron | 8/23 |
| Cilium agent 自身 | 9962 的 `cilium_errors_warnings_total`、BPF map 压力 | 同第 1 层 A | 两次 |

### 第 3 层：告警

`aiops/baseline.ExtractDeterministicAnomalies` + Notifier（Slack/Email）现成，
契约指标异常 → Incident → 通知。不用新建，只需把新信号接进去。

## 已核实的事实（省下次调查）

- Cilium agent 9962（Prometheus）**未开**：`cilium-config` 缺 `prometheus-serve-addr`
- Cilium operator 9963 已监听；Hubble 9965 已开（`hubble_drop_total` 等 10 类，单节点 191 series）
- `cilium-health status` 的连通矩阵是现成的，只是没被采集
- 三台 CP 的 SSD 均为 WD Blue SA510（消费级，无 PLP），etcd fsync 抖动是硬件特性
- ARP 故障的错误 MAC `98:03:8e:cf:0e:33` 归属未查明（incidents 报告的待议项）

## 建议顺序

```
磁盘 fsync（随 M1 指标对齐，立即可做）
  → Cilium 稳定后开 9962 → 第 1 层 A + Cilium 自身指标 → 接 AIOps 落 Incident
  → ARP 探测（先定执行载体，属架构选择）
  → etcd（先调查暴露方式）
```

## 待决策

1. 第 1 层选 A 还是 B，或 A 先 B 后
2. ARP 探测的执行载体（hostNetwork DaemonSet / 节点 cron / 集成进 Agent）
3. 平台健康数据在 UI 落在哪：metrics 页加 tab / 新建 `/observe/infra` / 只喂 AIOps 不单独展示

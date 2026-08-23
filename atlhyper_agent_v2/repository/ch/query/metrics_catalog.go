// metrics_catalog.go 是 Agent 依赖的 node-exporter 指标全集 —— 采集与查询两端的唯一对齐点。
//
// 两个消费方：
//   - Collector 的 keep regex 由 NodeExporterKeepRegex() 生成（config 仓 collector.yaml），决定"采什么"
//   - 契约自检 VerifyMetricsCollected 按本清单核对 ClickHouse 里"采到了没有"
//
// 为什么需要它：
//
//	2026-08 实测 Agent 查询引用 55 个 node_* 指标，而 Collector 只 keep 了 16 个 ——
//	39 个"查而不采"，PSI / TCP / VMStat / 系统资源 四张卡片长期空白且无任何报错。
//	与此前 kube_* 的"采而不查"（65 个指标 9600 万行无人读）是同一种病的镜像。
//	根因都是两端各自演进、没有单一对齐点。
//
// 维护规则（由 metrics_catalog_test.go 强制）：
//   - metrics.go / summary.go 的 SQL 每引用一个新 node_* 指标，必须同步加进本清单
//   - 清单里的每一项必须在 SQL 中被实际使用（不允许"预留"）
//   - 改完清单后重新生成 keep regex 更新 collector.yaml
//
// 为什么不给 node_* 做契约层（像 ingress_* 那样）：
//
//	node-exporter 是 Prometheus 生态的事实标准，十年未变且没有第二个实现；
//	ingress 做契约是因为实现会换（Traefik → Cilium），这里不存在该风险。
//	套一层 machine_cpu_* 之类的别名只会为一致性牺牲可读性。
package query

import "strings"

// NodeExporterMetrics 按 node-exporter collector 分组。顺序仅为阅读方便，无语义。
var NodeExporterMetrics = []string{
	// ── cpu / loadavg ──
	"node_cpu_seconds_total",
	"node_load1",
	"node_load5",
	"node_load15",

	// ── meminfo ──
	"node_memory_MemTotal_bytes",
	"node_memory_MemAvailable_bytes",
	"node_memory_MemFree_bytes",
	"node_memory_Cached_bytes",
	"node_memory_Buffers_bytes",
	"node_memory_SwapTotal_bytes",
	"node_memory_SwapFree_bytes",

	// ── diskstats / filesystem ──
	"node_disk_read_bytes_total",
	"node_disk_written_bytes_total",
	"node_disk_reads_completed_total",
	"node_disk_writes_completed_total",
	"node_disk_io_time_seconds_total",
	"node_disk_read_time_seconds_total",        // ÷ reads_completed = 平均读延迟
	"node_disk_write_time_seconds_total",       // ÷ writes_completed = 平均写延迟
	"node_disk_io_time_weighted_seconds_total", // 速率 = 平均队列深度
	"node_filesystem_size_bytes",
	"node_filesystem_avail_bytes",
	"node_filesystem_files",      // inode 总数
	"node_filesystem_files_free", // inode 剩余
	"node_filesystem_readonly",   // 1 = 内核已把该文件系统改判只读

	// ── netdev / netclass ──
	"node_network_up",
	"node_network_speed_bytes",
	"node_network_mtu_bytes",
	"node_network_receive_bytes_total",
	"node_network_transmit_bytes_total",
	"node_network_receive_packets_total",
	"node_network_transmit_packets_total",
	"node_network_receive_errs_total",
	"node_network_transmit_errs_total",
	"node_network_receive_drop_total",
	"node_network_transmit_drop_total",

	// ── hwmon（温度 / 风扇 / 电压）+ thermal cooling device ──
	"node_hwmon_chip_names", // chip 路径 → 可读名（coretemp / nvme / rp1_adc / rpi_volt / pwmfan）
	"node_hwmon_temp_celsius",
	"node_hwmon_temp_max_celsius",
	"node_hwmon_temp_crit_celsius",
	"node_hwmon_fan_rpm",
	"node_hwmon_in_lcrit_alarm_volts", // 树莓派欠压告警位
	"node_cooling_device_cur_state",
	"node_cooling_device_max_state",

	// ── cpufreq（热降频判定）──
	"node_cpu_scaling_frequency_hertz",
	"node_cpu_scaling_frequency_max_hertz",

	// ── pressure (PSI) ──
	"node_pressure_cpu_waiting_seconds_total",
	"node_pressure_memory_waiting_seconds_total",
	"node_pressure_memory_stalled_seconds_total",
	"node_pressure_io_waiting_seconds_total",
	"node_pressure_io_stalled_seconds_total",

	// ── netstat / sockstat / softnet ──
	"node_netstat_Tcp_CurrEstab",
	"node_sockstat_sockets_used",
	"node_sockstat_TCP_alloc",
	"node_sockstat_TCP_inuse",
	"node_sockstat_TCP_tw",
	"node_softnet_dropped_total",
	"node_softnet_times_squeezed_total",

	// ── conntrack / filefd / entropy ──
	"node_nf_conntrack_entries",
	"node_nf_conntrack_entries_limit",
	"node_filefd_allocated",
	"node_filefd_maximum",
	"node_entropy_available_bits",
	"node_arp_entries",

	// ── 运行队列 / 校时 ──
	"node_procs_running",
	"node_procs_blocked",
	"node_timex_offset_seconds",
	"node_timex_sync_status",

	// ── vmstat ──
	"node_vmstat_pgfault",
	"node_vmstat_pgmajfault",
	"node_vmstat_pswpin",
	"node_vmstat_pswpout",
	"node_vmstat_oom_kill", // 内存 USE 的「错误」项

	// ── uname / boot ──
	"node_uname_info",
	"node_boot_time_seconds",
}

// NodeExporterKeepRegex 生成 Collector metric_relabel_configs 用的 keep 正则（不含首尾锚点，
// Prometheus relabel 会自动做全匹配）。直接复制输出到 collector.yaml 的 node-exporter job。
func NodeExporterKeepRegex() string {
	return strings.Join(NodeExporterMetrics, "|")
}

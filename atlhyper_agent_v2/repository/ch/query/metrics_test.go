package query

import (
	"math"
	"testing"

	"AtlHyper/model_v3/metrics"
)

// awaitMs: 平均 IO 延迟 = 累计耗时速率 ÷ 完成次数速率，换算毫秒。
// 无 IO（ops 速率为 0）时不能除零，返回 0。
func TestAwaitMs(t *testing.T) {
	cases := []struct {
		name     string
		timeRate float64 // 秒/秒
		opsRate  float64 // 次/秒
		want     float64
	}{
		{"1.1ms await", 0.0011 * 100, 100, 1.1},
		{"124ms await", 0.124 * 8, 8, 124},
		{"no io", 0, 0, 0},
		{"time but no ops (counter glitch)", 0.5, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := awaitMs(c.timeRate, c.opsRate); math.Abs(got-c.want) > 0.01 {
				t.Errorf("awaitMs(%v, %v) = %v, want %v", c.timeRate, c.opsRate, got, c.want)
			}
		})
	}
}

// sanitizeTempLimit: hwmon 偶尔上报无意义的阈值（NVMe temp2 max = 65261.85），
// 超出物理合理范围的阈值视为「无阈值」（0），交给 Master 按画像兜底。
func TestSanitizeTempLimit(t *testing.T) {
	cases := []struct{ in, want float64 }{
		{82.85, 82.85},
		{100, 100},
		{65261.85, 0},
		{-273.15, 0},
		{0, 0},
	}
	for _, c := range cases {
		if got := sanitizeTempLimit(c.in); got != c.want {
			t.Errorf("sanitizeTempLimit(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

// 新增的硬件指标必须同时进入清单（否则 Collector 不采、查询静默为空）。
func TestNodeExporterMetrics_ContainsHardwareMetrics(t *testing.T) {
	required := []string{
		"node_hwmon_chip_names",
		"node_hwmon_fan_rpm",
		"node_hwmon_in_lcrit_alarm_volts",
		"node_cooling_device_cur_state",
		"node_cooling_device_max_state",
		"node_cpu_scaling_frequency_hertz",
		"node_cpu_scaling_frequency_max_hertz",
		"node_disk_read_time_seconds_total",
		"node_disk_write_time_seconds_total",
		"node_disk_io_time_weighted_seconds_total",
	}
	have := make(map[string]struct{}, len(NodeExporterMetrics))
	for _, m := range NodeExporterMetrics {
		have[m] = struct{}{}
	}
	for _, m := range required {
		if _, ok := have[m]; !ok {
			t.Errorf("清单缺少硬件指标 %s", m)
		}
	}
}

// 磁盘 USE 的三类信号各自依赖不同指标，缺任何一个对应格子就会空白。
func TestNodeExporterMetrics_ContainsDiskUSEMetrics(t *testing.T) {
	required := map[string]string{
		"node_filesystem_size_bytes":               "利用率（容量）",
		"node_filesystem_avail_bytes":              "利用率（容量）",
		"node_filesystem_files":                    "利用率（inode）",
		"node_filesystem_files_free":               "利用率（inode）",
		"node_filesystem_readonly":                 "错误（只读）",
		"node_disk_io_time_seconds_total":          "饱和度（繁忙比例）",
		"node_disk_io_time_weighted_seconds_total": "饱和度（队列深度）",
	}
	have := make(map[string]struct{}, len(NodeExporterMetrics))
	for _, m := range NodeExporterMetrics {
		have[m] = struct{}{}
	}
	for m, why := range required {
		if _, ok := have[m]; !ok {
			t.Errorf("清单缺少 %s（%s）", m, why)
		}
	}
}

// 内存 / 系统 USE 的信号来源
func TestNodeExporterMetrics_ContainsSystemUSEMetrics(t *testing.T) {
	required := map[string]string{
		"node_vmstat_oom_kill":      "内存错误（OOM 杀进程）",
		"node_procs_running":        "CPU 饱和度（运行队列）",
		"node_procs_blocked":        "IO 饱和度（D 状态进程）",
		"node_arp_entries":          "网络（邻居表）",
		"node_timex_offset_seconds": "校时偏移",
		"node_timex_sync_status":    "校时状态",
	}
	have := make(map[string]struct{}, len(NodeExporterMetrics))
	for _, m := range NodeExporterMetrics {
		have[m] = struct{}{}
	}
	for m, why := range required {
		if _, ok := have[m]; !ok {
			t.Errorf("清单缺少 %s（%s）", m, why)
		}
	}
}

// normalizeBlockDevice: 文件系统与 diskstats 用的是两套设备名 ——
// 前者 /dev/nvme0n1p2（带前缀的分区），后者 nvme0n1（裸块设备）。
// 不归一化就会让同一块盘裂成「有容量无 IO」和「有 IO 无容量」两行。
func TestNormalizeBlockDevice(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/dev/nvme0n1p2", "nvme0n1"},
		{"/dev/nvme0n1p1", "nvme0n1"},
		{"nvme0n1", "nvme0n1"},
		{"/dev/sda2", "sda"},
		{"/dev/sda", "sda"},
		{"sda1", "sda"},
		{"/dev/mmcblk0p2", "mmcblk0"},
		{"mmcblk0", "mmcblk0"},
		// LVM：/dev/mapper/xxx 在 diskstats 里叫 dm-N，无法从名字推导，保持原样
		{"/dev/mapper/ubuntu--vg-ubuntu--lv", "mapper/ubuntu--vg-ubuntu--lv"},
		{"dm-0", "dm-0"},
		{"", ""},
	}
	for _, c := range cases {
		if got := normalizeBlockDevice(c.in); got != c.want {
			t.Errorf("normalizeBlockDevice(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// /proc/diskstats 同时上报整盘和分区（mmcblk0、mmcblk0p1、mmcblk0p2 各一行）。
// IO 只取整盘 —— 分区级记录的数据已包含在整盘里，重复计入会让孤儿行冒出来。
func TestIsWholeBlockDevice(t *testing.T) {
	whole := []string{"sda", "nvme0n1", "mmcblk0", "dm-0", "mapper/ubuntu--vg-ubuntu--lv"}
	part := []string{"sda1", "sda2", "nvme0n1p2", "mmcblk0p1", "mmcblk0p2"}
	for _, d := range whole {
		if !isWholeBlockDevice(d) {
			t.Errorf("%s 应视为整盘", d)
		}
	}
	for _, d := range part {
		if isWholeBlockDevice(d) {
			t.Errorf("%s 是分区，不该视为整盘", d)
		}
	}
}

// nil 切片会序列化成 JSON null，而前端类型声明的是数组 —— n.disks.find() 直接崩，整页白屏。
// 2026-08-24 线上事故：两个节点查询超时导致 disks/networks/sensors 全为 nil，
// 其余五个节点正常，页面却整个挂掉。单个查询失败只该让那一格空着。
func TestEnsureNodeMetricsSlices(t *testing.T) {
	nm := &metrics.NodeMetrics{} // 所有 fill* 都失败的极端情况
	ensureNodeMetricsSlices(nm)

	if nm.Disks == nil {
		t.Error("Disks 仍为 nil —— 会序列化成 null")
	}
	if nm.Networks == nil {
		t.Error("Networks 仍为 nil")
	}
	if nm.Temperature.Sensors == nil {
		t.Error("Temperature.Sensors 仍为 nil")
	}
	if nm.CPU.FreqHz == nil {
		t.Error("CPU.FreqHz 仍为 nil")
	}

	// 已有数据的不能被清掉
	withData := &metrics.NodeMetrics{
		Disks:    []metrics.NodeDisk{{Device: "sda"}},
		Networks: []metrics.NodeNetwork{{Interface: "eth0"}},
	}
	ensureNodeMetricsSlices(withData)
	if len(withData.Disks) != 1 || len(withData.Networks) != 1 {
		t.Error("已有数据被覆盖了")
	}

	// Hardware 为 nil 时不应 panic（旧版本上报或无传感器）
	ensureNodeMetricsSlices(&metrics.NodeMetrics{Hardware: nil})

	hw := &metrics.NodeMetrics{Hardware: &metrics.NodeHardware{}}
	ensureNodeMetricsSlices(hw)
	if hw.Hardware.Fans == nil || hw.Hardware.Cooling == nil {
		t.Error("Hardware 内的切片仍为 nil")
	}
}

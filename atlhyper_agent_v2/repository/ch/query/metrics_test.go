package query

import (
	"math"
	"testing"
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

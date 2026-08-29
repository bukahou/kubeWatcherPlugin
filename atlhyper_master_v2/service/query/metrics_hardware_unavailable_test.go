package query

import (
	"testing"

	"AtlHyper/model_v3/metrics"
)

// ──────────────────────────────────────────────────────────────
// Unavailable section → 硬件矩阵格子必须为 nil（前端渲染灰色占位）
// ──────────────────────────────────────────────────────────────
//
// 契约（types/hardware.ts）本来就允许 cpuUsage: HardwareUsageCell | null，
// 前端遇 null 渲染 NoData 灰色占位。缺的一直是：模型无法告知 convert 层
// 「这部分采集失败了」，于是零值被做成了绿色的 0.0%。

func nmWithUnavailable(secs ...string) *metrics.NodeMetrics {
	return &metrics.NodeMetrics{
		NodeName:    "n1",
		CPU:         metrics.NodeCPU{UsagePct: 0, Cores: 0},
		Memory:      metrics.NodeMemory{UsagePct: 0},
		Unavailable: secs,
	}
}

func TestBuildHardwareRow_UnavailableSectionsAreNil(t *testing.T) {
	row := buildHardwareRow(nmWithUnavailable(
		metrics.SectionCPU, metrics.SectionMemory,
		metrics.SectionDisks, metrics.SectionTemperature,
	))
	if row.CPUUsage != nil {
		t.Errorf("cpu 采集失败，CPUUsage 应为 nil，得到 %+v —— 会渲染成绿色 0.0%%", row.CPUUsage)
	}
	if row.MemUsage != nil {
		t.Errorf("memory 采集失败，MemUsage 应为 nil，得到 %+v", row.MemUsage)
	}
	if row.DiskUsage != nil {
		t.Errorf("disks 采集失败，DiskUsage 应为 nil，得到 %+v", row.DiskUsage)
	}
	if row.CPUFreq != nil {
		t.Errorf("cpu 采集失败，CPUFreq 应为 nil，得到 %+v", row.CPUFreq)
	}
	if row.DiskAwait != nil {
		t.Errorf("disks 采集失败，DiskAwait 应为 nil，得到 %+v", row.DiskAwait)
	}
}

// 零值但未声明 Unavailable = 真实读数，必须照常渲染（0% 是合法测量）
func TestBuildHardwareRow_ZeroWithoutUnavailable_IsRealReading(t *testing.T) {
	row := buildHardwareRow(nmWithUnavailable( /* 无 */ ))
	if row.CPUUsage == nil {
		t.Error("未声明 Unavailable 的零值是真实读数，CPUUsage 不应为 nil")
	}
	if row.CPUUsage != nil && row.CPUUsage.Value != 0 {
		t.Errorf("Value = %v, want 0", row.CPUUsage.Value)
	}
}

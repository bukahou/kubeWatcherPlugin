// node_hardware.go 节点硬件健康模型（Phase 1a）
//
// 数据源只有 node-exporter 的 hwmon / cooling_device / cpufreq collector。
// 这里放的纯函数（传感器分类、硬件画像识别）被 Agent（填充）与 Master（判定）共用，
// 避免两端各维护一份 chip 名字表而漂移。
package metrics

import "strings"

// NodeHardware 风扇 / 散热 / 电压。切片为空表示该节点没有对应传感器。
type NodeHardware struct {
	// UndervoltAlarm 欠压告警位（树莓派 rpi_volt 的 in0_lcrit_alarm）。nil = 无此传感器。
	UndervoltAlarm *bool           `json:"undervoltAlarm,omitempty"`
	Fans           []FanSensor     `json:"fans"`
	Cooling        []CoolingDevice `json:"cooling"`
}

// FanSensor hwmon 风扇转速（node_hwmon_fan_rpm）
type FanSensor struct {
	Chip   string  `json:"chip"`
	Sensor string  `json:"sensor"`
	RPM    float64 `json:"rpm"`
}

// CoolingDevice 内核 thermal cooling device（node_cooling_device_{cur,max}_state）。
// Type 取值来自内核：pwm-fan / Processor / intel_powerclamp / TCC Offset ...
type CoolingDevice struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	CurState int    `json:"curState"`
	MaxState int    `json:"maxState"`
}

// HardwareProfile 硬件画像 —— Master 据此选阈值表。不硬编码节点名。
type HardwareProfile string

const (
	ProfileDesk    HardwareProfile = "desk"    // x86_64 小主机
	ProfileRaspi5  HardwareProfile = "raspi5"  // aarch64 + NVMe / pwm-fan
	ProfileRaspi4  HardwareProfile = "raspi4"  // aarch64，无 NVMe、无风扇
	ProfileUnknown HardwareProfile = "unknown" // 识别失败，Master 退回通用保守阈值
)

// DetectHardwareProfile 由 uname machine + hwmon chip 可读名集合推导画像。
// chipNames 来自 node_hwmon_chip_names 的 chip_name 标签。
func DetectHardwareProfile(machine string, chipNames []string) HardwareProfile {
	switch machine {
	case "x86_64":
		return ProfileDesk
	case "aarch64":
		for _, c := range chipNames {
			if c == "nvme" || c == "pwmfan" {
				return ProfileRaspi5
			}
		}
		return ProfileRaspi4
	}
	return ProfileUnknown
}

// TempSensorClass 温度传感器分类 —— 矩阵的三列温度格
type TempSensorClass string

const (
	TempClassCPU   TempSensorClass = "cpu"   // coretemp / k10temp / SoC thermal zone
	TempClassDisk  TempSensorClass = "disk"  // nvme / drivetemp
	TempClassOther TempSensorClass = "other" // RP1 / 主板 / 其他
)

// Class 按 chip 可读名（优先）或 chip 路径判定传感器属于哪一列。
func (s TempSensor) Class() TempSensorClass {
	name := strings.ToLower(s.ChipName)
	chip := strings.ToLower(s.Chip)
	switch {
	case name == "coretemp" || name == "k10temp" || name == "cpu_thermal" || name == "cpu-thermal",
		strings.Contains(chip, "coretemp"), strings.Contains(chip, "thermal_zone"):
		return TempClassCPU
	case name == "nvme" || name == "drivetemp", strings.HasPrefix(chip, "nvme"):
		return TempClassDisk
	}
	return TempClassOther
}

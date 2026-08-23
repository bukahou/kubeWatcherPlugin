// metrics_thresholds.go 硬件画像阈值表
//
// 阈值优先级（高 → 低）：
//  1. 传感器自带的 max / crit（node_hwmon_temp_{max,crit}_celsius）—— 厂商给的最准
//  2. 本文件的画像表 —— 按 SoC / 平台的降频点
//  3. 通用保守值 —— 画像识别失败时兜底，宁可早报警
//
// 判定全部在 Master：Agent 只上报原始读数，前端只渲染颜色。
package query

import "AtlHyper/model_v3/metrics"

// tempThreshold 一类传感器的告警 / 危险温度
type tempThreshold struct {
	max  float64 // warn 线
	crit float64 // crit 线
}

// profileThresholds 一种硬件画像的各类阈值
type profileThresholds struct {
	label string
	cpu   tempThreshold
	disk  tempThreshold
	other tempThreshold
}

// 通用保守值：画像未识别或没有画像条目时使用
var genericThresholds = profileThresholds{
	label: "未识别硬件",
	cpu:   tempThreshold{max: 70, crit: 85},
	disk:  tempThreshold{max: 70, crit: 80},
	other: tempThreshold{max: 85, crit: 95},
}

// hardwareProfiles 家庭集群在用的三种硬件。
// 消费级硬件 7/24 运行，阈值取「开始降频」而非「烧毁」。
var hardwareProfiles = map[metrics.HardwareProfile]profileThresholds{
	metrics.ProfileDesk: {
		label: "x86 小主机",
		// Intel Tj 通常 100°C，但小机箱持续 85°C 已经不健康
		cpu:   tempThreshold{max: 85, crit: 100},
		disk:  tempThreshold{max: 60, crit: 70}, // SATA SSD 正常工作在 30–50°C
		other: tempThreshold{max: 85, crit: 95},
	},
	metrics.ProfileRaspi5: {
		label: "Raspberry Pi 5",
		// BCM2712 在 85°C 硬降频，80°C 起软降频
		cpu: tempThreshold{max: 80, crit: 85},
		// NVMe 多数会自报 max/crit（优先用），这里只兜底没有自报的盘：
		// 规格上 82°C 起限速，但 Pi 5 的小空间里持续 70°C 已经该关注了
		disk:  tempThreshold{max: 70, crit: 80},
		other: tempThreshold{max: 85, crit: 95},
	},
	metrics.ProfileRaspi4: {
		label: "Raspberry Pi 4",
		// BCM2711 同样 80 / 85
		cpu: tempThreshold{max: 80, crit: 85},
		// Pi 4 只有 SD 卡，没有温度传感器；留着是为了画像表结构一致
		disk:  tempThreshold{max: 70, crit: 80},
		other: tempThreshold{max: 85, crit: 95},
	},
}

// thresholdsFor 取画像对应的阈值表，未知画像退回通用保守值
func thresholdsFor(p metrics.HardwareProfile) profileThresholds {
	if t, ok := hardwareProfiles[p]; ok {
		return t
	}
	return genericThresholds
}

// 非温度类的固定阈值（与硬件画像无关）
const (
	diskAwaitWarnMs = 50.0  // 机械盘正常 < 20ms，SSD < 5ms
	diskAwaitCritMs = 200.0 // 到这个量级 Pod 已经开始超时
	// 热降频：频率低于标称的这个比例，且节点确实在干活，才算降频而非省电空转
	throttleRatioPct = 60.0
	throttleBusyPct  = 50.0
)

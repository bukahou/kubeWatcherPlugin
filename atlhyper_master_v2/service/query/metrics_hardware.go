// metrics_hardware.go 硬件健康判定（GET /api/v2/observe/metrics/hardware）
//
// 每格只看自己的信号：温度超了标温度，磁盘慢了标磁盘。
// 跨信号推理（"空转却高温说明散热坏了"）属于 AIOps，不在这里。
//
// 无传感器 → 该格 nil（前端「无数据」），绝不用零值冒充正常：
// 一个恒显示 0°C 的绿色格子比空白格危险得多。
package query

import (
	"context"
	"math"
	"strings"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/metrics"
)

// GetHardwareHealth 产出硬件健康矩阵 + 速览。
// 快照缺失返回空结果而非错误 —— 集群刚接入时页面显示空表，不该报错。
func (q *QueryService) GetHardwareHealth(ctx context.Context, clusterID string) (*model.HardwareHealthResponse, error) {
	resp := &model.HardwareHealthResponse{Rows: []model.HardwareRow{}}

	otel, err := q.GetOTelSnapshot(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	if otel == nil {
		return resp, nil
	}

	for i := range otel.MetricsNodes {
		resp.Rows = append(resp.Rows, buildHardwareRow(&otel.MetricsNodes[i]))
	}
	resp.Summary = buildHardwareSummary(otel.MetricsNodes, resp.Rows)
	return resp, nil
}

// buildHardwareRow 判定单节点的每一格
func buildHardwareRow(nm *metrics.NodeMetrics) model.HardwareRow {
	profile := nm.HardwareProfile
	if profile == "" {
		profile = metrics.ProfileUnknown
	}
	th := thresholdsFor(profile)

	row := model.HardwareRow{
		NodeName:     nm.NodeName,
		Profile:      string(profile),
		ProfileLabel: th.label,
		CPUTemp:      tempCell(nm.Temperature.Sensors, metrics.TempClassCPU, th.cpu),
		DiskTemp:     tempCell(nm.Temperature.Sensors, metrics.TempClassDisk, th.other),
		OtherTemp:    tempCell(nm.Temperature.Sensors, metrics.TempClassOther, th.other),
		Undervolt:    undervoltCell(nm.Hardware),
		CPUFreq:      freqCell(nm.CPU),
		DiskAwait:    awaitCell(nm.Disks),
	}
	row.Fan = fanCell(nm.Hardware, row.CPUTemp)
	row.Overall = worstStatus(
		statusOf(row.CPUTemp), statusOf(row.DiskTemp), statusOf(row.OtherTemp),
		statusOf(row.Undervolt), statusOf(row.Fan), statusOf(row.CPUFreq), statusOf(row.DiskAwait),
	)
	return row
}

// tempCell 取某一类传感器里最热的一个成格。该类没有传感器 → nil。
func tempCell(sensors []metrics.TempSensor, class metrics.TempSensorClass, fallback tempThreshold) *model.HardwareTempCell {
	var hottest *metrics.TempSensor
	for i := range sensors {
		if sensors[i].Class() != class {
			continue
		}
		if hottest == nil || sensors[i].CurrentC > hottest.CurrentC {
			hottest = &sensors[i]
		}
	}
	if hottest == nil {
		return nil
	}

	// 传感器自带阈值优先；只有 crit 没有 max 时，把 crit 的 90% 当作提前量
	max, crit := hottest.MaxC, hottest.CritC
	switch {
	case max > 0 && crit > 0:
	case max > 0:
		crit = fallback.crit
	case crit > 0:
		max = hwRound(crit*0.9, 2)
	default:
		max, crit = fallback.max, fallback.crit
	}

	cell := &model.HardwareTempCell{
		Value: hottest.CurrentC,
		Max:   max,
		Crit:  crit,
		Label: sensorLabel(*hottest, class),
	}
	switch {
	case cell.Value >= crit:
		cell.Status = model.HardwareCrit
	case cell.Value >= max:
		cell.Status = model.HardwareWarn
	default:
		cell.Status = model.HardwareGood
	}
	return cell
}

// sensorLabel 给传感器起一个人看得懂的名字。
// 磁盘要能区分是哪块盘（nvme_nvme0 → nvme0），其他类用 hwmon 的 chip 可读名。
func sensorLabel(s metrics.TempSensor, class metrics.TempSensorClass) string {
	parts := strings.Split(s.Chip, "_")
	if class == metrics.TempClassDisk {
		for _, p := range parts {
			if strings.HasPrefix(p, "nvme") && len(p) > len("nvme") {
				return p
			}
		}
	}
	if s.ChipName != "" {
		return s.ChipName
	}
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return s.Chip
}

// undervoltCell 欠压是硬件级危险（供电不足会静默损坏 SD 卡 / SSD），直接 crit
func undervoltCell(hw *metrics.NodeHardware) *model.HardwareUndervoltCell {
	if hw == nil || hw.UndervoltAlarm == nil {
		return nil
	}
	cell := &model.HardwareUndervoltCell{Alarm: *hw.UndervoltAlarm, Status: model.HardwareGood}
	if cell.Alarm {
		cell.Status = model.HardwareCrit
	}
	return cell
}

// fanCell 只统计真正的风扇（hwmon 转速传感器 / pwm-fan 类 cooling device）。
// x86 的 Processor / intel_powerclamp cooling device 是频率限制器，不是风扇，不进这格。
func fanCell(hw *metrics.NodeHardware, cpuTemp *model.HardwareTempCell) *model.HardwareFanCell {
	if hw == nil {
		return nil
	}
	cell := &model.HardwareFanCell{Status: model.HardwareGood}
	found := false

	if len(hw.Fans) > 0 {
		rpm := hw.Fans[0].RPM
		cell.RPM = &rpm
		found = true
	}
	for _, c := range hw.Cooling {
		if !isFanCooling(c.Type) {
			continue
		}
		cell.State, cell.MaxState = c.CurState, c.MaxState
		found = true
		break
	}
	if !found {
		return nil
	}

	// 风扇停转本身不是故障（低温时就该停）；温度已经超线还不转才是
	if cell.RPM != nil && *cell.RPM == 0 && cpuTemp != nil && cpuTemp.Status != model.HardwareGood {
		cell.Status = model.HardwareWarn
	}
	return cell
}

func isFanCooling(t string) bool {
	lower := strings.ToLower(t)
	return strings.Contains(lower, "fan")
}

// freqCell 热降频判定：频率明显低于标称 且 节点确实在忙。
// 只看频率会把省电空转误报成降频，所以要带上本节点自己的 CPU 使用率
// —— 这是单一信号内部的限定条件，不是跨信号关联。
func freqCell(cpu metrics.NodeCPU) *model.HardwareFreqCell {
	if len(cpu.FreqHz) == 0 || cpu.FreqMaxHz <= 0 {
		return nil
	}
	var sum float64
	for _, f := range cpu.FreqHz {
		sum += f
	}
	current := sum / float64(len(cpu.FreqHz))

	cell := &model.HardwareFreqCell{
		CurrentGHz: hwRound(current/1e9, 2),
		MaxGHz:     hwRound(cpu.FreqMaxHz/1e9, 2),
		RatioPct:   hwRound(current/cpu.FreqMaxHz*100, 1),
		Status:     model.HardwareGood,
	}
	if cell.RatioPct < throttleRatioPct && cpu.UsagePct > throttleBusyPct {
		cell.Status = model.HardwareWarn
	}
	return cell
}

// awaitCell 取该节点最慢的块设备（读写取较大者）
func awaitCell(disks []metrics.NodeDisk) *model.HardwareAwaitCell {
	var worst *model.HardwareAwaitCell
	for _, d := range disks {
		v := math.Max(d.AwaitReadMs, d.AwaitWriteMs)
		if v <= 0 {
			continue
		}
		if worst == nil || v > worst.ValueMs {
			worst = &model.HardwareAwaitCell{ValueMs: v, Device: d.Device}
		}
	}
	if worst == nil {
		return nil
	}
	switch {
	case worst.ValueMs >= diskAwaitCritMs:
		worst.Status = model.HardwareCrit
	case worst.ValueMs >= diskAwaitWarnMs:
		worst.Status = model.HardwareWarn
	default:
		worst.Status = model.HardwareGood
	}
	return worst
}

// buildHardwareSummary 速览 tile：全集群最热的点、欠压 / 降频节点数、最慢的盘
func buildHardwareSummary(nodes []metrics.NodeMetrics, rows []model.HardwareRow) model.HardwareSummary {
	var s model.HardwareSummary
	for i := range rows {
		row := rows[i]
		if row.CPUTemp != nil {
			s.MaxTemp = pickMaxTemp(s.MaxTemp, row.CPUTemp, row.NodeName)
		}
		if row.OtherTemp != nil {
			s.MaxTemp = pickMaxTemp(s.MaxTemp, row.OtherTemp, row.NodeName)
		}
		if row.DiskTemp != nil {
			s.MaxTemp = pickMaxTemp(s.MaxTemp, row.DiskTemp, row.NodeName)
			s.MaxDiskTemp = pickMaxTemp(s.MaxDiskTemp, row.DiskTemp, row.NodeName)
		}
		if row.Undervolt != nil && row.Undervolt.Alarm {
			s.UndervoltNodes++
		}
		if row.CPUFreq != nil && row.CPUFreq.Status != model.HardwareGood {
			s.ThrottledNodes++
		}
		if row.DiskAwait != nil {
			if s.MaxDiskAwait == nil || row.DiskAwait.ValueMs > s.MaxDiskAwait.ValueMs {
				s.MaxDiskAwait = &model.HardwareMaxAwait{
					ValueMs:  row.DiskAwait.ValueMs,
					NodeName: row.NodeName,
					Device:   row.DiskAwait.Device,
					Status:   row.DiskAwait.Status,
				}
			}
		}
	}
	return s
}

func pickMaxTemp(cur *model.HardwareMaxTemp, cell *model.HardwareTempCell, node string) *model.HardwareMaxTemp {
	if cur != nil && cur.Value >= cell.Value {
		return cur
	}
	return &model.HardwareMaxTemp{
		Value:    cell.Value,
		NodeName: node,
		Sensor:   cell.Label,
		Status:   cell.Status,
	}
}

// statusOf 取任意格的状态；nil（无数据）视为 good —— 缺传感器不是故障
func statusOf(cell interface{}) model.HardwareStatus {
	switch c := cell.(type) {
	case *model.HardwareTempCell:
		if c != nil {
			return c.Status
		}
	case *model.HardwareUndervoltCell:
		if c != nil {
			return c.Status
		}
	case *model.HardwareFanCell:
		if c != nil {
			return c.Status
		}
	case *model.HardwareFreqCell:
		if c != nil {
			return c.Status
		}
	case *model.HardwareAwaitCell:
		if c != nil {
			return c.Status
		}
	}
	return model.HardwareGood
}

func worstStatus(list ...model.HardwareStatus) model.HardwareStatus {
	worst := model.HardwareGood
	for _, s := range list {
		if s == model.HardwareCrit {
			return model.HardwareCrit
		}
		if s == model.HardwareWarn {
			worst = model.HardwareWarn
		}
	}
	return worst
}

// hwRound 保留小数位（Master 侧只此一处需要，不引第三方库）
func hwRound(v float64, decimals int) float64 {
	p := math.Pow(10, float64(decimals))
	return math.Round(v*p) / p
}

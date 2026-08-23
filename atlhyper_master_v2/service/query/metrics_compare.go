// metrics_compare.go 节点对比表（GET /api/v2/observe/metrics/compare）
//
// 硬件矩阵回答「这台机器的传感器怎么样」，对比表回答「七台里哪台不一样」。
// 因此这里不重新判定硬件 —— 直接复用 buildHardwareRow 的结论，
// 只补上四大资源那几列。同一个读数在两个页面显示不同颜色是最难排查的 UI bug。
package query

import (
	"context"
	"fmt"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/metrics"
)

// compareColumns 列顺序：硬件在前（会烧板子的先看），资源在后。
// 前端按这个数组渲染表头，不做重排。
var compareColumns = []string{
	"cpuTemp", "diskTemp", "undervolt", "freqRatio", "diskAwait",
	"diskUsage", "cpu", "psiCpu", "mem", "psiMem", "netErr",
}

// 资源列的告警线。硬件列的阈值来自画像表，这里只管资源。
const (
	resourceWarnPct = 80.0
	resourceCritPct = 90.0
	psiWarnPct      = 20.0 // PSI：20% 的时间有任务在等资源已经很不健康
	psiCritPct      = 50.0
)

// GetNodeComparison 产出节点横向对比表
func (q *QueryService) GetNodeComparison(ctx context.Context, clusterID string) (*model.NodeComparisonResponse, error) {
	resp := &model.NodeComparisonResponse{Columns: compareColumns, Rows: []model.CompareRow{}}

	otel, err := q.GetOTelSnapshot(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	if otel == nil {
		return resp, nil
	}

	for i := range otel.MetricsNodes {
		nm := &otel.MetricsNodes[i]
		hwRow := buildHardwareRow(nm) // 复用硬件判定，不重算
		resp.Rows = append(resp.Rows, model.CompareRow{
			NodeName: nm.NodeName,
			Profile:  hwRow.Profile,
			Cells:    compareCells(nm, hwRow),
			Overall:  worstStatus(hwRow.Overall, resourceOverall(nm)),
		})
	}
	return resp, nil
}

// compareCells 每列都必须有键 —— 缺信号用 value=nil，省略键会让表格错位
func compareCells(nm *metrics.NodeMetrics, hw model.HardwareRow) map[string]model.CompareCell {
	cells := make(map[string]model.CompareCell, len(compareColumns))

	cells["cpuTemp"] = tempCompareCell(hw.CPUTemp, "°C")
	cells["diskTemp"] = tempCompareCell(hw.DiskTemp, "°C")

	// 欠压用 0/1 而不是文字：文案属于前端 i18n，后端只给语义
	if hw.Undervolt != nil {
		alarm := 0.0
		if hw.Undervolt.Alarm {
			alarm = 1
		}
		cells["undervolt"] = model.CompareCell{Value: &alarm, Status: hw.Undervolt.Status}
	} else {
		cells["undervolt"] = model.CompareCell{Status: model.HardwareGood}
	}

	if hw.CPUFreq != nil {
		v := hw.CPUFreq.RatioPct
		cells["freqRatio"] = model.CompareCell{Value: &v, Text: fmt.Sprintf("%.0f%%", v), Status: hw.CPUFreq.Status}
	} else {
		cells["freqRatio"] = model.CompareCell{Status: model.HardwareGood}
	}

	if hw.DiskAwait != nil {
		v := hw.DiskAwait.ValueMs
		cells["diskAwait"] = model.CompareCell{Value: &v, Text: fmt.Sprintf("%.1f ms", v), Status: hw.DiskAwait.Status}
	} else {
		cells["diskAwait"] = model.CompareCell{Status: model.HardwareGood}
	}

	// 资源列：取该节点最差的那一项（对比表是找异常，不是看明细）
	cells["diskUsage"] = pctCell(worstDiskUsagePct(nm.Disks), resourceWarnPct, resourceCritPct)
	cells["cpu"] = pctCell(nm.CPU.UsagePct, resourceWarnPct, resourceCritPct)
	cells["psiCpu"] = pctCell(nm.PSI.CPUSomePct, psiWarnPct, psiCritPct)
	cells["mem"] = pctCell(nm.Memory.UsagePct, resourceWarnPct, resourceCritPct)
	cells["psiMem"] = pctCell(nm.PSI.MemSomePct, psiWarnPct, psiCritPct)
	cells["netErr"] = netErrCell(nm.Networks)

	return cells
}

func tempCompareCell(cell *model.HardwareTempCell, unit string) model.CompareCell {
	if cell == nil {
		return model.CompareCell{Status: model.HardwareGood}
	}
	v := cell.Value
	return model.CompareCell{Value: &v, Text: fmt.Sprintf("%.1f%s", v, unit), Status: cell.Status}
}

func pctCell(v, warn, crit float64) model.CompareCell {
	value := hwRound(v, 1)
	cell := model.CompareCell{Value: &value, Text: fmt.Sprintf("%.1f%%", value), Status: model.HardwareGood}
	switch {
	case value >= crit:
		cell.Status = model.HardwareCrit
	case value >= warn:
		cell.Status = model.HardwareWarn
	}
	return cell
}

// netErrCell 网络错误 + 丢包速率之和。任何非零都值得看一眼，持续增长才是问题
func netErrCell(nets []metrics.NodeNetwork) model.CompareCell {
	var total float64
	for _, n := range nets {
		total += n.RxErrPerSec + n.TxErrPerSec + n.RxDropPerSec + n.TxDropPerSec
	}
	total = hwRound(total, 2)
	cell := model.CompareCell{Value: &total, Text: fmt.Sprintf("%.2f/s", total), Status: model.HardwareGood}
	if total > 0 {
		cell.Status = model.HardwareWarn
	}
	return cell
}

func worstDiskUsagePct(disks []metrics.NodeDisk) float64 {
	var worst float64
	for _, d := range disks {
		if d.UsagePct > worst {
			worst = d.UsagePct
		}
		// inode 用尽和容量用尽后果一样（写不进去），取两者更差的
		if d.InodeUsagePct > worst {
			worst = d.InodeUsagePct
		}
	}
	return worst
}

// resourceOverall 资源列里最差的状态，与硬件 overall 合并成整行结论
func resourceOverall(nm *metrics.NodeMetrics) model.HardwareStatus {
	cells := []model.CompareCell{
		pctCell(worstDiskUsagePct(nm.Disks), resourceWarnPct, resourceCritPct),
		pctCell(nm.CPU.UsagePct, resourceWarnPct, resourceCritPct),
		pctCell(nm.PSI.CPUSomePct, psiWarnPct, psiCritPct),
		pctCell(nm.Memory.UsagePct, resourceWarnPct, resourceCritPct),
		pctCell(nm.PSI.MemSomePct, psiWarnPct, psiCritPct),
	}
	statuses := make([]model.HardwareStatus, 0, len(cells))
	for _, c := range cells {
		statuses = append(statuses, c.Status)
	}
	return worstStatus(statuses...)
}

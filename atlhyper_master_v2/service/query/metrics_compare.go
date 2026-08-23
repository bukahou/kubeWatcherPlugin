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
	// PSI 在 K8s 节点上常态就有 20–60%（一堆 Pod 抢 CPU 是设计如此），
	// 实测同集群 6 台在 0.6–3%、1 台 53% —— 阈值要能把那 1 台挑出来，又不能把它判死
	psiWarnPct = 50.0
	psiCritPct = 80.0
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
	cells["diskUsage"] = pctCell(primaryDiskUsagePct(nm.Disks), resourceWarnPct, resourceCritPct)
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

// netErrCell 只统计真错误（err），不含 drop。
//
// 虚拟接口（lxc*/cilium_*）的 drop 常态非零 —— 组播与广播被内核丢掉是正常的，
// 实测 7 台全部 0.50/s。把 drop 算进来的结果是整列全黄，等于没有信号。
// drop 仍然在节点卡的 NetworkCard 里按接口列出。
func netErrCell(nets []metrics.NodeNetwork) model.CompareCell {
	var total float64
	for _, n := range nets {
		total += n.RxErrPerSec + n.TxErrPerSec
	}
	total = hwRound(total, 2)
	cell := model.CompareCell{Value: &total, Text: fmt.Sprintf("%.2f/s", total), Status: model.HardwareGood}
	if total > 0 {
		cell.Status = model.HardwareWarn
	}
	return cell
}

// primaryDiskUsagePct 「这台机器的盘满没满」看根分区。
//
// 取所有分区里最满的那个是错的：/boot/firmware 只有 512MB，用掉 37% 毫无意义，
// 却会盖住真正的根分区（实测三台 raspi5 都显示 37.24%，全是那个 boot 分区）。
// 没有根分区时退回容量最大的那块盘。
func primaryDiskUsagePct(disks []metrics.NodeDisk) float64 {
	var primary *metrics.NodeDisk
	for i := range disks {
		d := &disks[i]
		if d.MountPoint == "/" {
			primary = d
			break
		}
		if d.MountPoint != "" && (primary == nil || d.TotalBytes > primary.TotalBytes) {
			primary = d
		}
	}
	if primary == nil {
		return 0
	}
	// inode 用尽和容量用尽后果一样（写不进去），取更差的
	if primary.InodeUsagePct > primary.UsagePct {
		return primary.InodeUsagePct
	}
	return primary.UsagePct
}

// resourceOverall 资源列里最差的状态，与硬件 overall 合并成整行结论
func resourceOverall(nm *metrics.NodeMetrics) model.HardwareStatus {
	cells := []model.CompareCell{
		pctCell(primaryDiskUsagePct(nm.Disks), resourceWarnPct, resourceCritPct),
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

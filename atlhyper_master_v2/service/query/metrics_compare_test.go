package query

import (
	"context"
	"testing"

	"AtlHyper/atlhyper_master_v2/model"
)

func TestGetNodeComparison_ColumnsAndOrder(t *testing.T) {
	resp, err := hardwareService(t).GetNodeComparison(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	// 列顺序由后端固定，硬件在前 —— 前端不重排，改顺序要改这里
	want := []string{"cpuTemp", "diskTemp", "undervolt", "freqRatio", "diskAwait", "diskUsage", "cpu", "psiCpu", "mem", "psiMem", "netErr"}
	if len(resp.Columns) != len(want) {
		t.Fatalf("列数 = %d，期望 %d：%v", len(resp.Columns), len(want), resp.Columns)
	}
	for i, c := range want {
		if resp.Columns[i] != c {
			t.Errorf("第 %d 列 = %q，期望 %q", i, resp.Columns[i], c)
		}
	}
	if len(resp.Rows) != 4 {
		t.Fatalf("行数 = %d，期望 4", len(resp.Rows))
	}
	// 每行都要有全部列的键，缺信号用 value=nil 而不是省略键 —— 否则表格会错位
	for _, r := range resp.Rows {
		for _, c := range resp.Columns {
			if _, ok := r.Cells[c]; !ok {
				t.Errorf("%s 缺少列 %q", r.NodeName, c)
			}
		}
	}
}

func TestGetNodeComparison_ReusesHardwareVerdicts(t *testing.T) {
	resp, _ := hardwareService(t).GetNodeComparison(context.Background(), "c1")
	byNode := map[string]model.CompareRow{}
	for _, r := range resp.Rows {
		byNode[r.NodeName] = r
	}

	// 与硬件矩阵同源：desk-zero 的 await 124ms 是 warn，raspi4 欠压是 crit
	if c := byNode["desk-zero"].Cells["diskAwait"]; c.Value == nil || *c.Value != 124 || c.Status != model.HardwareWarn {
		t.Errorf("desk-zero diskAwait = %+v", c)
	}
	// 欠压格用 0/1 表达，不带自然语言文案（文案是前端 i18n 的职责）
	if c := byNode["raspi4-zero"].Cells["undervolt"]; c.Status != model.HardwareCrit || c.Value == nil || *c.Value != 1 || c.Text != "" {
		t.Errorf("raspi4 undervolt = %+v", c)
	}
	// 没有盘温传感器的 desk：value 为 nil，状态不算故障
	if c := byNode["desk-zero"].Cells["diskTemp"]; c.Value != nil || c.Status != model.HardwareGood {
		t.Errorf("desk-zero diskTemp = %+v，期望 value=nil", c)
	}
	// overall 与硬件矩阵一致
	if byNode["raspi4-zero"].Overall != model.HardwareCrit || byNode["raspi5-one"].Overall != model.HardwareGood {
		t.Errorf("overall 不一致: raspi4=%q raspi5=%q", byNode["raspi4-zero"].Overall, byNode["raspi5-one"].Overall)
	}
}

func TestGetNodeComparison_ResourceCells(t *testing.T) {
	resp, _ := hardwareService(t).GetNodeComparison(context.Background(), "c1")
	byNode := map[string]model.CompareRow{}
	for _, r := range resp.Rows {
		byNode[r.NodeName] = r
	}
	// desk-zero CPU 80% → warn（>= 80）
	if c := byNode["desk-zero"].Cells["cpu"]; c.Value == nil || *c.Value != 80 || c.Status != model.HardwareWarn {
		t.Errorf("desk-zero cpu = %+v", c)
	}
	// raspi5-one CPU 10% → good
	if c := byNode["raspi5-one"].Cells["cpu"]; c.Status != model.HardwareGood {
		t.Errorf("raspi5-one cpu = %+v", c)
	}
	// 夹具里没有内存数据 → 0%，不应误判为故障
	if c := byNode["mystery"].Cells["mem"]; c.Status != model.HardwareGood {
		t.Errorf("mystery mem = %+v", c)
	}
}

func TestGetNodeComparison_NoSnapshot(t *testing.T) {
	svc := &QueryService{store: &mockStoreForOverview{}}
	resp, err := svc.GetNodeComparison(context.Background(), "nope")
	if err != nil {
		t.Fatalf("缺快照不应报错: %v", err)
	}
	if resp == nil || len(resp.Rows) != 0 || len(resp.Columns) == 0 {
		t.Errorf("期望空行 + 固定列，得到 %+v", resp)
	}
}

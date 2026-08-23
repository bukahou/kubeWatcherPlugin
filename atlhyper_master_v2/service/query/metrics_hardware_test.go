package query

import (
	"context"
	"testing"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/metrics"
)

func boolPtr(b bool) *bool { return &b }

// 夹具：四种典型节点
func hardwareFixtureSnapshot() *cluster.ClusterSnapshot {
	return &cluster.ClusterSnapshot{OTel: &cluster.OTelSnapshot{MetricsNodes: []metrics.NodeMetrics{
		{ // raspi5：全传感器齐全，一切正常
			NodeName: "raspi5-one", HardwareProfile: metrics.ProfileRaspi5,
			CPU: metrics.NodeCPU{UsagePct: 10, FreqHz: []float64{1.5e9, 1.5e9, 1.5e9, 1.5e9}, FreqMaxHz: 2.4e9},
			Temperature: metrics.NodeTemperature{Sensors: []metrics.TempSensor{
				{Chip: "thermal_thermal_zone0", ChipName: "cpu_thermal", Sensor: "temp1", CurrentC: 58.95},
				{Chip: "nvme_nvme0", ChipName: "nvme", Sensor: "temp1", CurrentC: 49.85, MaxC: 82.85, CritC: 84.85},
				{Chip: "nvme_nvme0", ChipName: "nvme", Sensor: "temp2", CurrentC: 49.85},
				{Chip: "1000120000_pcie_1f000c8000_adc", ChipName: "rp1_adc", Sensor: "temp1", CurrentC: 61.3},
			}},
			Hardware: &metrics.NodeHardware{
				UndervoltAlarm: boolPtr(false),
				Fans:           []metrics.FanSensor{{Chip: "platform_cooling_fan", Sensor: "fan1", RPM: 5423}},
				Cooling:        []metrics.CoolingDevice{{Name: "0", Type: "pwm-fan", CurState: 2, MaxState: 4}},
			},
			Disks: []metrics.NodeDisk{{Device: "nvme0n1", AwaitReadMs: 1.1, AwaitWriteMs: 0.8}},
		},
		{ // desk：旧版 Agent 没有 Hardware；coretemp 只有 crit 没有 max；负载高且降频 → 热降频 warn；await 124ms warn
			NodeName: "desk-zero", HardwareProfile: metrics.ProfileDesk,
			CPU: metrics.NodeCPU{UsagePct: 80, FreqHz: []float64{0.9e9, 0.9e9}, FreqMaxHz: 3.8e9},
			Temperature: metrics.NodeTemperature{Sensors: []metrics.TempSensor{
				{Chip: "platform_coretemp_0", ChipName: "coretemp", Sensor: "temp1", CurrentC: 62, CritC: 80},
				{Chip: "platform_coretemp_0", ChipName: "coretemp", Sensor: "temp2", CurrentC: 52, CritC: 80},
			}},
			Disks: []metrics.NodeDisk{{Device: "sda", AwaitReadMs: 124, AwaitWriteMs: 3}},
		},
		{ // raspi4：欠压告警 → crit
			NodeName: "raspi4-zero", HardwareProfile: metrics.ProfileRaspi4,
			CPU: metrics.NodeCPU{UsagePct: 5, FreqHz: []float64{1.5e9}, FreqMaxHz: 1.5e9},
			Temperature: metrics.NodeTemperature{Sensors: []metrics.TempSensor{
				{Chip: "thermal_thermal_zone0", ChipName: "cpu_thermal", Sensor: "temp1", CurrentC: 81},
			}},
			Hardware: &metrics.NodeHardware{UndervoltAlarm: boolPtr(true)},
		},
		{ // 画像未知且什么都没有：全部无数据，不报错
			NodeName: "mystery",
		},
	}}}
}

func hardwareService(t *testing.T) *QueryService {
	t.Helper()
	store := &mockStoreForOverview{snapshots: map[string]*cluster.ClusterSnapshot{"c1": hardwareFixtureSnapshot()}}
	return &QueryService{store: store}
}

func findRow(t *testing.T, resp *model.HardwareHealthResponse, node string) model.HardwareRow {
	t.Helper()
	for _, r := range resp.Rows {
		if r.NodeName == node {
			return r
		}
	}
	t.Fatalf("row %s not found", node)
	return model.HardwareRow{}
}

func TestGetHardwareHealth_Raspi5AllGood(t *testing.T) {
	resp, err := hardwareService(t).GetHardwareHealth(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	r := findRow(t, resp, "raspi5-one")
	if r.Profile != "raspi5" {
		t.Errorf("profile = %q", r.Profile)
	}
	// CPU 温：传感器无阈值 → 画像表 80/85
	if r.CPUTemp == nil || r.CPUTemp.Value != 58.95 || r.CPUTemp.Max != 80 || r.CPUTemp.Crit != 85 || r.CPUTemp.Status != model.HardwareGood {
		t.Errorf("cpuTemp = %+v", r.CPUTemp)
	}
	// 磁盘温：取 nvme 自带阈值，label 为 chip 短名，忽略无阈值的 temp2
	if r.DiskTemp == nil || r.DiskTemp.Value != 49.85 || r.DiskTemp.Max != 82.85 || r.DiskTemp.Crit != 84.85 || r.DiskTemp.Label != "nvme0" {
		t.Errorf("diskTemp = %+v", r.DiskTemp)
	}
	if r.OtherTemp == nil || r.OtherTemp.Value != 61.3 || r.OtherTemp.Label != "rp1_adc" || r.OtherTemp.Status != model.HardwareGood {
		t.Errorf("otherTemp = %+v", r.OtherTemp)
	}
	if r.Undervolt == nil || r.Undervolt.Alarm || r.Undervolt.Status != model.HardwareGood {
		t.Errorf("undervolt = %+v", r.Undervolt)
	}
	if r.Fan == nil || r.Fan.RPM == nil || *r.Fan.RPM != 5423 || r.Fan.State != 2 || r.Fan.MaxState != 4 || r.Fan.Status != model.HardwareGood {
		t.Errorf("fan = %+v", r.Fan)
	}
	// 1.5/2.4 = 62.5%，空闲降频不算热降频
	if r.CPUFreq == nil || r.CPUFreq.CurrentGHz != 1.5 || r.CPUFreq.MaxGHz != 2.4 || r.CPUFreq.RatioPct != 62.5 || r.CPUFreq.Status != model.HardwareGood {
		t.Errorf("cpuFreq = %+v", r.CPUFreq)
	}
	if r.DiskAwait == nil || r.DiskAwait.ValueMs != 1.1 || r.DiskAwait.Device != "nvme0n1" || r.DiskAwait.Status != model.HardwareGood {
		t.Errorf("diskAwait = %+v", r.DiskAwait)
	}
	if r.Overall != model.HardwareGood {
		t.Errorf("overall = %q", r.Overall)
	}
}

func TestGetHardwareHealth_DeskOldAgentAndThrottle(t *testing.T) {
	resp, _ := hardwareService(t).GetHardwareHealth(context.Background(), "c1")
	r := findRow(t, resp, "desk-zero")
	// 多个 coretemp 取最高；只有 crit=80 没有 max → max 退回 crit*0.9=72
	if r.CPUTemp == nil || r.CPUTemp.Value != 62 || r.CPUTemp.Max != 72 || r.CPUTemp.Crit != 80 || r.CPUTemp.Status != model.HardwareGood {
		t.Errorf("cpuTemp = %+v", r.CPUTemp)
	}
	// 没有 nvme / drivetemp → 无数据；旧版 Agent 没有 Hardware → 欠压 / 风扇无数据
	if r.DiskTemp != nil || r.OtherTemp != nil || r.Undervolt != nil || r.Fan != nil {
		t.Errorf("expected nil cells for missing sensors: disk=%v other=%v uv=%v fan=%v", r.DiskTemp, r.OtherTemp, r.Undervolt, r.Fan)
	}
	// 0.9/3.8 = 23.7% 且 CPU 80% → 热降频 warn
	if r.CPUFreq == nil || r.CPUFreq.Status != model.HardwareWarn || r.CPUFreq.RatioPct != 23.7 {
		t.Errorf("cpuFreq = %+v", r.CPUFreq)
	}
	if r.DiskAwait == nil || r.DiskAwait.ValueMs != 124 || r.DiskAwait.Status != model.HardwareWarn {
		t.Errorf("diskAwait = %+v", r.DiskAwait)
	}
	if r.Overall != model.HardwareWarn {
		t.Errorf("overall = %q", r.Overall)
	}
}

func TestGetHardwareHealth_UndervoltIsCrit(t *testing.T) {
	resp, _ := hardwareService(t).GetHardwareHealth(context.Background(), "c1")
	r := findRow(t, resp, "raspi4-zero")
	if r.Undervolt == nil || !r.Undervolt.Alarm || r.Undervolt.Status != model.HardwareCrit {
		t.Errorf("undervolt = %+v", r.Undervolt)
	}
	// 81 ≥ raspi4 画像 max 80 → warn（但 < crit 85）
	if r.CPUTemp == nil || r.CPUTemp.Status != model.HardwareWarn {
		t.Errorf("cpuTemp = %+v", r.CPUTemp)
	}
	if r.Fan != nil {
		t.Errorf("raspi4 无风扇，fan 应为 nil，得到 %+v", r.Fan)
	}
	if r.Overall != model.HardwareCrit {
		t.Errorf("overall = %q", r.Overall)
	}
}

func TestGetHardwareHealth_UnknownNodeAllNull(t *testing.T) {
	resp, _ := hardwareService(t).GetHardwareHealth(context.Background(), "c1")
	r := findRow(t, resp, "mystery")
	if r.Profile != "unknown" {
		t.Errorf("profile = %q", r.Profile)
	}
	if r.CPUTemp != nil || r.DiskTemp != nil || r.OtherTemp != nil || r.Undervolt != nil || r.Fan != nil || r.CPUFreq != nil || r.DiskAwait != nil {
		t.Errorf("all cells should be nil: %+v", r)
	}
	if r.Overall != model.HardwareGood {
		t.Errorf("无数据不是故障，overall 应为 good，得到 %q", r.Overall)
	}
}

func TestGetHardwareHealth_Summary(t *testing.T) {
	resp, _ := hardwareService(t).GetHardwareHealth(context.Background(), "c1")
	s := resp.Summary
	// 全集群最高温：raspi4-zero cpu_thermal 81（warn）
	if s.MaxTemp == nil || s.MaxTemp.Value != 81 || s.MaxTemp.NodeName != "raspi4-zero" || s.MaxTemp.Status != model.HardwareWarn {
		t.Errorf("maxTemp = %+v", s.MaxTemp)
	}
	if s.MaxDiskTemp == nil || s.MaxDiskTemp.Value != 49.85 || s.MaxDiskTemp.NodeName != "raspi5-one" {
		t.Errorf("maxDiskTemp = %+v", s.MaxDiskTemp)
	}
	if s.UndervoltNodes != 1 || s.ThrottledNodes != 1 {
		t.Errorf("undervolt=%d throttled=%d", s.UndervoltNodes, s.ThrottledNodes)
	}
	if s.MaxDiskAwait == nil || s.MaxDiskAwait.ValueMs != 124 || s.MaxDiskAwait.NodeName != "desk-zero" || s.MaxDiskAwait.Device != "sda" || s.MaxDiskAwait.Status != model.HardwareWarn {
		t.Errorf("maxDiskAwait = %+v", s.MaxDiskAwait)
	}
}

func TestGetHardwareHealth_NoSnapshot(t *testing.T) {
	svc := &QueryService{store: &mockStoreForOverview{snapshots: map[string]*cluster.ClusterSnapshot{}}}
	resp, err := svc.GetHardwareHealth(context.Background(), "nope")
	if err != nil {
		t.Fatalf("缺快照不应报错: %v", err)
	}
	if resp == nil || len(resp.Rows) != 0 {
		t.Errorf("expected empty rows, got %+v", resp)
	}
}

// 盘温没有自报阈值时，必须退回「磁盘」那一档，而不是宽松的「其他传感器」档
// —— 实测 raspi5-two 的 SK hynix BC501 不上报 max/crit，用 85/95 判定等于永不报警。
func TestTempCell_DiskFallbackUsesDiskThreshold(t *testing.T) {
	sensors := []metrics.TempSensor{
		{Chip: "nvme_nvme0", ChipName: "nvme", Sensor: "temp1", CurrentC: 72},
	}
	th := thresholdsFor(metrics.ProfileRaspi5)

	cell := tempCell(sensors, metrics.TempClassDisk, th.disk)
	if cell == nil {
		t.Fatal("expected disk cell")
	}
	if cell.Max != 70 || cell.Crit != 80 {
		t.Errorf("盘温阈值 = %v/%v，期望 70/80", cell.Max, cell.Crit)
	}
	if cell.Status != model.HardwareWarn {
		t.Errorf("72°C 超过 70°C 应为 warn，得到 %q", cell.Status)
	}
	// 同样温度按「其他传感器」档判定会被漏掉 —— 这正是本用例要防的
	if other := tempCell(sensors, metrics.TempClassDisk, th.other); other.Status != model.HardwareGood {
		t.Errorf("对照组：other 档下 72°C 应为 good，得到 %q", other.Status)
	}
}

// 温度卡要展示全部传感器（不只是每类最热的那个），且每个都带后端判定。
func TestGetHardwareHealth_SensorsListed(t *testing.T) {
	resp, _ := hardwareService(t).GetHardwareHealth(context.Background(), "c1")

	r := findRow(t, resp, "raspi5-one")
	if len(r.Sensors) != 4 {
		t.Fatalf("raspi5 应列出全部 4 个传感器，得到 %d", len(r.Sensors))
	}
	byLabel := map[string]model.HardwareSensorCell{}
	for _, s := range r.Sensors {
		byLabel[s.Label+"/"+s.Sensor] = s
	}
	// 自报阈值的 NVMe temp1
	if c, ok := byLabel["nvme0/temp1"]; !ok || c.Class != "disk" || c.Max != 82.85 || c.Status != model.HardwareGood {
		t.Errorf("nvme0/temp1 = %+v", c)
	}
	// 无自报阈值的 NVMe temp2 → 退回磁盘档 70/80
	if c, ok := byLabel["nvme0/temp2"]; !ok || c.Max != 70 || c.Crit != 80 {
		t.Errorf("nvme0/temp2 应退回磁盘档 70/80，得到 %+v", c)
	}
	// SoC 温度走 CPU 档
	if c, ok := byLabel["cpu_thermal/temp1"]; !ok || c.Class != "cpu" || c.Max != 80 || c.Crit != 85 {
		t.Errorf("cpu_thermal/temp1 = %+v", c)
	}
	// RP1 走其他档
	if c, ok := byLabel["rp1_adc/temp1"]; !ok || c.Class != "other" || c.Max != 85 {
		t.Errorf("rp1_adc/temp1 = %+v", c)
	}

	// 没有传感器的节点必须是空数组而非 nil —— 前端 .map 不该拿到 null
	m := findRow(t, resp, "mystery")
	if m.Sensors == nil || len(m.Sensors) != 0 {
		t.Errorf("mystery.Sensors = %v，期望空数组", m.Sensors)
	}
}

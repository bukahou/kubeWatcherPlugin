package concentrator

import (
	"testing"
	"time"

	"AtlHyper/model_v3/apm"
	"AtlHyper/model_v3/metrics"
	"AtlHyper/model_v3/slo"
)

func TestConcentrator_IngestAndFlush(t *testing.T) {
	c := NewConcentrator()
	now := time.Now().Truncate(time.Minute)

	// 模拟 3 分钟的数据（从过去到现在）
	for i := 0; i < 3; i++ {
		ts := now.Add(-time.Duration(2-i) * time.Minute)
		nodes := []metrics.NodeMetrics{
			{
				NodeName:    "worker-1",
				CPU:         metrics.NodeCPU{UsagePct: float64(10 + i*10), UserPct: 5, SystemPct: 3, IOWaitPct: 1, Load1: 1.0, Load5: 0.8, Load15: 0.6},
				Memory:      metrics.NodeMemory{UsagePct: float64(50 + i), SwapUsagePct: 2.0},
				Disks:       []metrics.NodeDisk{{MountPoint: "/", UsagePct: 30.0, ReadBytesPerSec: 100, WriteBytesPerSec: 200, IOUtilPct: 5.0}},
				Networks:    []metrics.NodeNetwork{{Interface: "eth0", Up: true, RxBytesPerSec: 1000, TxBytesPerSec: 500, RxPktPerSec: 50, TxPktPerSec: 30}},
				Temperature: metrics.NodeTemperature{CPUTempC: 45.0},
				PSI:         metrics.NodePSI{CPUSomePct: 1.0, MemSomePct: 0.5, IOSomePct: 0.3},
				TCP:         metrics.NodeTCP{CurrEstab: 100, SocketsUsed: 200},
			},
		}
		sloIngress := []slo.IngressSLO{
			{ServiceKey: "api-gw", RPS: float64(100 + i*10), SuccessRate: 99.5, P50Ms: 5.0, P99Ms: 15.0, ErrorRate: 0.5},
		}
		apmServices := []apm.APMService{
			{Name: "gateway", Namespace: "default", RPS: 200, SuccessRate: 99.0, AvgDurationMs: 10, P99Ms: 50, ErrorCount: 2},
		}
		c.Ingest(nodes, sloIngress, apmServices, ts)
	}

	// Flush 节点时序
	nodeSeries := c.FlushNodeSeries()
	if len(nodeSeries) != 1 {
		t.Fatalf("expected 1 node series, got %d", len(nodeSeries))
	}
	if nodeSeries[0].NodeName != "worker-1" {
		t.Errorf("expected node name 'worker-1', got '%s'", nodeSeries[0].NodeName)
	}
	if len(nodeSeries[0].Points) != 3 {
		t.Fatalf("expected 3 node points, got %d", len(nodeSeries[0].Points))
	}
	// 验证第一个点
	if nodeSeries[0].Points[0].CPUPct != 10.0 {
		t.Errorf("expected CPUPct 10.0, got %f", nodeSeries[0].Points[0].CPUPct)
	}
	// 验证最后一个点（25 字段全覆盖）
	lastPt := nodeSeries[0].Points[2]
	if lastPt.CPUPct != 30.0 {
		t.Errorf("expected CPUPct 30.0, got %f", lastPt.CPUPct)
	}
	if lastPt.UserPct != 5.0 {
		t.Errorf("expected UserPct 5.0, got %f", lastPt.UserPct)
	}
	if lastPt.SystemPct != 3.0 {
		t.Errorf("expected SystemPct 3.0, got %f", lastPt.SystemPct)
	}
	if lastPt.Load5 != 0.8 {
		t.Errorf("expected Load5 0.8, got %f", lastPt.Load5)
	}
	if lastPt.DiskPct != 30.0 {
		t.Errorf("expected DiskPct 30.0, got %f", lastPt.DiskPct)
	}
	if lastPt.DiskReadBps != 100.0 {
		t.Errorf("expected DiskReadBps 100.0, got %f", lastPt.DiskReadBps)
	}
	if lastPt.DiskWriteBps != 200.0 {
		t.Errorf("expected DiskWriteBps 200.0, got %f", lastPt.DiskWriteBps)
	}
	if lastPt.DiskIOUtilPct != 5.0 {
		t.Errorf("expected DiskIOUtilPct 5.0, got %f", lastPt.DiskIOUtilPct)
	}
	if lastPt.NetRxBps != 1000.0 {
		t.Errorf("expected NetRxBps 1000.0, got %f", lastPt.NetRxBps)
	}
	if lastPt.NetTxBps != 500.0 {
		t.Errorf("expected NetTxBps 500.0, got %f", lastPt.NetTxBps)
	}
	if lastPt.NetRxPktSec != 50.0 {
		t.Errorf("expected NetRxPktSec 50.0, got %f", lastPt.NetRxPktSec)
	}
	if lastPt.CPUTempC != 45.0 {
		t.Errorf("expected CPUTempC 45.0, got %f", lastPt.CPUTempC)
	}
	if lastPt.CPUSomePct != 1.0 {
		t.Errorf("expected CPUSomePct 1.0, got %f", lastPt.CPUSomePct)
	}
	if lastPt.TCPEstab != 100 {
		t.Errorf("expected TCPEstab 100, got %d", lastPt.TCPEstab)
	}
	if lastPt.SocketsUsed != 200 {
		t.Errorf("expected SocketsUsed 200, got %d", lastPt.SocketsUsed)
	}
	if lastPt.SwapUsagePct != 2.0 {
		t.Errorf("expected SwapUsagePct 2.0, got %f", lastPt.SwapUsagePct)
	}

	// Flush SLO 时序（6 字段）
	sloSeries := c.FlushSLOSeries()
	if len(sloSeries) != 1 {
		t.Fatalf("expected 1 SLO series, got %d", len(sloSeries))
	}
	if sloSeries[0].ServiceName != "api-gw" {
		t.Errorf("expected service name 'api-gw', got '%s'", sloSeries[0].ServiceName)
	}
	if len(sloSeries[0].Points) != 3 {
		t.Fatalf("expected 3 SLO points, got %d", len(sloSeries[0].Points))
	}
	sloPt := sloSeries[0].Points[0]
	if sloPt.RPS != 100.0 {
		t.Errorf("expected RPS 100.0, got %f", sloPt.RPS)
	}
	if sloPt.P50Ms != 5.0 {
		t.Errorf("expected P50Ms 5.0, got %f", sloPt.P50Ms)
	}
	if sloPt.ErrorRate != 0.5 {
		t.Errorf("expected ErrorRate 0.5, got %f", sloPt.ErrorRate)
	}

	// Flush APM 时序
	apmSeries := c.FlushAPMSeries()
	if len(apmSeries) != 1 {
		t.Fatalf("expected 1 APM series, got %d", len(apmSeries))
	}
	if apmSeries[0].ServiceName != "gateway" {
		t.Errorf("expected service name 'gateway', got '%s'", apmSeries[0].ServiceName)
	}
	if apmSeries[0].Namespace != "default" {
		t.Errorf("expected namespace 'default', got '%s'", apmSeries[0].Namespace)
	}
	if len(apmSeries[0].Points) != 3 {
		t.Fatalf("expected 3 APM points, got %d", len(apmSeries[0].Points))
	}
	apmPt := apmSeries[0].Points[0]
	if apmPt.RPS != 200.0 {
		t.Errorf("expected RPS 200.0, got %f", apmPt.RPS)
	}
	if apmPt.SuccessRate != 99.0 {
		t.Errorf("expected SuccessRate 99.0, got %f", apmPt.SuccessRate)
	}
	if apmPt.AvgMs != 10.0 {
		t.Errorf("expected AvgMs 10.0, got %f", apmPt.AvgMs)
	}
	if apmPt.P99Ms != 50.0 {
		t.Errorf("expected P99Ms 50.0, got %f", apmPt.P99Ms)
	}
	if apmPt.ErrorCount != 2 {
		t.Errorf("expected ErrorCount 2, got %d", apmPt.ErrorCount)
	}
}

func TestConcentrator_GAUGESemantic(t *testing.T) {
	// 同一分钟内多次写入，取最新值
	c := NewConcentrator()
	now := time.Now().Truncate(time.Minute)

	for i := 0; i < 6; i++ {
		ts := now.Add(time.Duration(i*10) * time.Second) // 同一分钟内 6 次
		nodes := []metrics.NodeMetrics{
			{
				NodeName: "node-1",
				CPU:      metrics.NodeCPU{UsagePct: float64(10 + i)},
				Memory:   metrics.NodeMemory{UsagePct: 50.0},
			},
		}
		c.Ingest(nodes, nil, nil, ts)
	}

	series := c.FlushNodeSeries()
	if len(series) != 1 {
		t.Fatalf("expected 1 series, got %d", len(series))
	}
	// 应该只有 1 个数据点（同一分钟），值为最后写入的值
	if len(series[0].Points) != 1 {
		t.Fatalf("expected 1 point (same minute), got %d", len(series[0].Points))
	}
	if series[0].Points[0].CPUPct != 15.0 { // 10 + 5
		t.Errorf("expected CPUPct 15.0 (last write), got %f", series[0].Points[0].CPUPct)
	}
}

func TestConcentrator_RingOverwrite(t *testing.T) {
	// 超过 60 分钟后旧数据被覆盖
	c := NewConcentrator()
	now := time.Now().Truncate(time.Minute)

	// 写入 70 分钟的数据（从过去到现在）
	for i := 0; i < 70; i++ {
		ts := now.Add(-time.Duration(69-i) * time.Minute)
		nodes := []metrics.NodeMetrics{
			{
				NodeName: "node-1",
				CPU:      metrics.NodeCPU{UsagePct: float64(i)},
				Memory:   metrics.NodeMemory{UsagePct: 50.0},
			},
		}
		c.Ingest(nodes, nil, nil, ts)
	}

	series := c.FlushNodeSeries()
	if len(series) != 1 {
		t.Fatalf("expected 1 series, got %d", len(series))
	}
	// 应该最多 60 个点（环形缓冲区容量）
	if len(series[0].Points) > ringCapacity {
		t.Errorf("expected at most %d points, got %d", ringCapacity, len(series[0].Points))
	}
	// 数据从 now-69 到 now 写入（70 个点），环容量 60
	// flush 窗口 [now-59, now] = 60 个点
	// 对应的是 i=10..69（CPUPct = 10..69）
	if len(series[0].Points) > 0 {
		firstPt := series[0].Points[0]
		if firstPt.CPUPct != 10.0 {
			t.Errorf("expected first CPUPct 10.0 (oldest surviving), got %f", firstPt.CPUPct)
		}
		lastPt := series[0].Points[len(series[0].Points)-1]
		if lastPt.CPUPct != 69.0 {
			t.Errorf("expected last CPUPct 69.0, got %f", lastPt.CPUPct)
		}
	}
}

func TestConcentrator_MultipleNodes(t *testing.T) {
	c := NewConcentrator()
	now := time.Now().Truncate(time.Minute)

	nodes := []metrics.NodeMetrics{
		{NodeName: "master-1", CPU: metrics.NodeCPU{UsagePct: 20}, Memory: metrics.NodeMemory{UsagePct: 40}},
		{NodeName: "worker-1", CPU: metrics.NodeCPU{UsagePct: 60}, Memory: metrics.NodeMemory{UsagePct: 70}},
		{NodeName: "worker-2", CPU: metrics.NodeCPU{UsagePct: 80}, Memory: metrics.NodeMemory{UsagePct: 90}},
	}
	c.Ingest(nodes, nil, nil, now)

	series := c.FlushNodeSeries()
	if len(series) != 3 {
		t.Fatalf("expected 3 node series, got %d", len(series))
	}
}

func TestConcentrator_EmptyFlush(t *testing.T) {
	c := NewConcentrator()
	nodeSeries := c.FlushNodeSeries()
	sloSeries := c.FlushSLOSeries()
	apmSeries := c.FlushAPMSeries()

	if len(nodeSeries) != 0 {
		t.Errorf("expected 0 node series, got %d", len(nodeSeries))
	}
	if len(sloSeries) != 0 {
		t.Errorf("expected 0 SLO series, got %d", len(sloSeries))
	}
	if len(apmSeries) != 0 {
		t.Errorf("expected 0 APM series, got %d", len(apmSeries))
	}
}

func TestConcentrator_APMMultipleServices(t *testing.T) {
	c := NewConcentrator()
	now := time.Now().Truncate(time.Minute)

	for i := 0; i < 3; i++ {
		ts := now.Add(-time.Duration(2-i) * time.Minute)
		services := []apm.APMService{
			{Name: "api", Namespace: "prod", RPS: 100, SuccessRate: 99, AvgDurationMs: 5, P99Ms: 20, ErrorCount: 1},
			{Name: "worker", Namespace: "prod", RPS: 50, SuccessRate: 98, AvgDurationMs: 15, P99Ms: 80, ErrorCount: 3},
		}
		c.Ingest(nil, nil, services, ts)
	}

	series := c.FlushAPMSeries()
	if len(series) != 2 {
		t.Fatalf("expected 2 APM series, got %d", len(series))
	}

	for _, s := range series {
		if len(s.Points) != 3 {
			t.Errorf("expected 3 points for %s, got %d", s.ServiceName, len(s.Points))
		}
		if s.Namespace != "prod" {
			t.Errorf("expected namespace 'prod' for %s, got '%s'", s.ServiceName, s.Namespace)
		}
	}
}

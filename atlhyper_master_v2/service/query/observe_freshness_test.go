package query

import (
	"context"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/cluster"
)

func freshnessSvc(t *testing.T, f *cluster.SignalFreshness) *QueryService {
	t.Helper()
	return &QueryService{store: &mockStoreForOverview{
		snapshots: map[string]*cluster.ClusterSnapshot{
			"c1": {OTel: &cluster.OTelSnapshot{Freshness: f}},
		},
	}}
}

func findSignal(t *testing.T, resp *model.FreshnessResponse, name string) model.SignalFreshnessItem {
	t.Helper()
	for _, s := range resp.Signals {
		if s.Signal == name {
			return s
		}
	}
	t.Fatalf("signal %s not found", name)
	return model.SignalFreshnessItem{}
}

// 全部新鲜 —— 正常运行时的样子
func TestGetSignalFreshness_AllLive(t *testing.T) {
	now := time.Now()
	svc := freshnessSvc(t, &cluster.SignalFreshness{
		MetricsAt: now.Add(-10 * time.Second),
		TracesAt:  now.Add(-30 * time.Second),
		LogsAt:    now.Add(-25 * time.Second),
	})
	resp, err := svc.GetSignalFreshness(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"metrics", "traces", "logs"} {
		if s := findSignal(t, resp, name); s.Status != model.FreshnessLive {
			t.Errorf("%s = %q，期望 live", name, s.Status)
		}
	}
	if !resp.CollectorHealthy {
		t.Error("metrics 新鲜时采集链路应视为健康")
	}
}

// 关键用例：traces/logs 停了但 metrics 还在 —— 是没人访问，不是故障。
// 2026-08-24 实测就是这个情况：traces 停 78 分钟，查了 ingress 计数才确认
// 近 80 分钟只有 1 个请求。页面必须能自己说清楚，不该让人去查 ClickHouse。
func TestGetSignalFreshness_IdleNotStale(t *testing.T) {
	now := time.Now()
	svc := freshnessSvc(t, &cluster.SignalFreshness{
		MetricsAt: now.Add(-12 * time.Second),
		TracesAt:  now.Add(-78 * time.Minute),
		LogsAt:    now.Add(-78 * time.Minute),
	})
	resp, _ := svc.GetSignalFreshness(context.Background(), "c1")

	if s := findSignal(t, resp, "metrics"); s.Status != model.FreshnessLive {
		t.Errorf("metrics = %q，期望 live", s.Status)
	}
	for _, name := range []string{"traces", "logs"} {
		s := findSignal(t, resp, name)
		if s.Status != model.FreshnessIdle {
			t.Errorf("%s = %q，期望 idle（没有流量，不是故障）", name, s.Status)
		}
		if s.LagSeconds < 4600 || s.LagSeconds > 4800 {
			t.Errorf("%s lag = %d，期望约 4680", name, s.LagSeconds)
		}
	}
	if !resp.CollectorHealthy {
		t.Error("metrics 仍在流动时采集链路是健康的")
	}
}

// metrics 也停了 —— 拉取式信号断了只能是采集链路出问题
func TestGetSignalFreshness_StaleWhenMetricsDown(t *testing.T) {
	now := time.Now()
	svc := freshnessSvc(t, &cluster.SignalFreshness{
		MetricsAt: now.Add(-20 * time.Minute),
		TracesAt:  now.Add(-30 * time.Minute),
		LogsAt:    now.Add(-30 * time.Minute),
	})
	resp, _ := svc.GetSignalFreshness(context.Background(), "c1")
	for _, name := range []string{"metrics", "traces", "logs"} {
		if s := findSignal(t, resp, name); s.Status != model.FreshnessStale {
			t.Errorf("%s = %q，期望 stale（采集异常）", name, s.Status)
		}
	}
	if resp.CollectorHealthy {
		t.Error("metrics 停了，采集链路不该报健康")
	}
}

// 零值时间 = 从来没有过数据，与「有过但停了」是两回事
func TestGetSignalFreshness_Absent(t *testing.T) {
	now := time.Now()
	svc := freshnessSvc(t, &cluster.SignalFreshness{
		MetricsAt: now.Add(-10 * time.Second),
		// traces / logs 为零值
	})
	resp, _ := svc.GetSignalFreshness(context.Background(), "c1")
	for _, name := range []string{"traces", "logs"} {
		s := findSignal(t, resp, name)
		if s.Status != model.FreshnessAbsent {
			t.Errorf("%s = %q，期望 absent", name, s.Status)
		}
		if s.LastDataAt != "" {
			t.Errorf("%s 不该有 lastDataAt: %q", name, s.LastDataAt)
		}
	}
}

// 旧版 Agent 不上报 Freshness —— 返回 absent 而不是报错或假装健康
func TestGetSignalFreshness_NilFromOldAgent(t *testing.T) {
	svc := freshnessSvc(t, nil)
	resp, err := svc.GetSignalFreshness(context.Background(), "c1")
	if err != nil {
		t.Fatalf("旧版 Agent 不该导致报错: %v", err)
	}
	if len(resp.Signals) != 3 {
		t.Fatalf("仍应返回三个信号，得到 %d", len(resp.Signals))
	}
	for _, s := range resp.Signals {
		if s.Status != model.FreshnessAbsent {
			t.Errorf("%s = %q，期望 absent", s.Signal, s.Status)
		}
	}
}

// 快照不存在（集群刚接入）
func TestGetSignalFreshness_NoSnapshot(t *testing.T) {
	svc := &QueryService{store: &mockStoreForOverview{}}
	resp, err := svc.GetSignalFreshness(context.Background(), "nope")
	if err != nil {
		t.Fatalf("缺快照不该报错: %v", err)
	}
	if len(resp.Signals) != 3 {
		t.Errorf("期望三个信号占位，得到 %d", len(resp.Signals))
	}
}

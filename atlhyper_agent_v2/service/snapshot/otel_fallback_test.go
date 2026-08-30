package snapshot

import (
	"testing"

	"AtlHyper/model_v3/apm"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/metrics"
)

// ──────────────────────────────────────────────────────────────
// 逐信号回退：某一路查询失败不得抹掉其余成功的数据
// ──────────────────────────────────────────────────────────────
//
// 2026-08-30 实证的缺陷链（用户报告「有时候没数据，有时候又出现」）：
//
//	9 路 dashboard 查询中一路失败 → 该字段留 nil 且不触发全局回退
//	→ 带 nil 的快照被写进 30s TTL 缓存
//	→ 之后 30 秒每轮都复制这个 nil
//	→ Master 见 MetricsNodes == nil 返 404「数据尚未就绪」
//	→ 面板空白 30 秒后突然恢复
//
// 同时 hasError 那条路径也有问题：注释写「全部失败时回退」，
// 代码却是任一概览失败就丢弃整份快照（含 8 路成功结果）。
//
// 两者是同一个原则问题 —— 部分失败不该丢弃全部，
// 与 2026-08-29 修的 buildNodeMetrics 缺陷同源。

func TestRestoreMissingFromPrev_FillsOnlyNilFields(t *testing.T) {
	prev := &cluster.OTelSnapshot{
		MetricsNodes:  []metrics.NodeMetrics{{NodeName: "old-node"}},
		APMServices:   []apm.APMService{{Name: "old-svc"}},
		MetricsSummary: &metrics.Summary{TotalNodes: 7},
	}
	// 本轮：MetricsNodes 查询失败(nil)，APMServices 成功
	fresh := &cluster.OTelSnapshot{
		MetricsNodes: nil,
		APMServices:  []apm.APMService{{Name: "new-svc"}},
	}

	restoreMissingFromPrev(fresh, prev)

	if len(fresh.MetricsNodes) != 1 || fresh.MetricsNodes[0].NodeName != "old-node" {
		t.Errorf("失败的信号应沿用上一份数据，得到 %+v —— nil 会让面板空白 30 秒", fresh.MetricsNodes)
	}
	if len(fresh.APMServices) != 1 || fresh.APMServices[0].Name != "new-svc" {
		t.Errorf("成功的信号必须保留本轮新值，不得被旧值覆盖，得到 %+v", fresh.APMServices)
	}
	if fresh.MetricsSummary == nil || fresh.MetricsSummary.TotalNodes != 7 {
		t.Error("指针型字段同样应回退")
	}
}

func TestRestoreMissingFromPrev_NilPrevIsSafe(t *testing.T) {
	fresh := &cluster.OTelSnapshot{}
	restoreMissingFromPrev(fresh, nil) // Agent 首轮：无上一份，不得 panic
	if fresh.MetricsNodes != nil {
		t.Error("无上一份快照时不应凭空造数据")
	}
}

// 空切片与 nil 语义不同：空切片是「查到了，结果为空」，必须保留
func TestRestoreMissingFromPrev_EmptySliceIsRealResult(t *testing.T) {
	prev := &cluster.OTelSnapshot{
		MetricsNodes: []metrics.NodeMetrics{{NodeName: "old-node"}},
	}
	fresh := &cluster.OTelSnapshot{
		MetricsNodes: []metrics.NodeMetrics{}, // 查询成功但集群确实没节点
	}
	restoreMissingFromPrev(fresh, prev)
	if len(fresh.MetricsNodes) != 0 {
		t.Errorf("空切片是真实结果，不该被旧值覆盖，得到 %+v", fresh.MetricsNodes)
	}
}

// 回退的数据不得被 Concentrator 当作新采样点摄入 —— 否则同一份读数
// 会在预聚合时序里重复出现，把「查询失败」伪装成「值保持不变」。
// restoreMissingFromPrev 须报告哪些信号是回退来的，供调用方跳过摄入。
func TestRestoreMissingFromPrev_ReportsRestoredSignals(t *testing.T) {
	prev := &cluster.OTelSnapshot{
		MetricsNodes: []metrics.NodeMetrics{{NodeName: "old"}},
		APMServices:  []apm.APMService{{Name: "old-svc"}},
	}
	fresh := &cluster.OTelSnapshot{
		MetricsNodes: nil,                                  // 失败 → 回退
		APMServices:  []apm.APMService{{Name: "new-svc"}},   // 成功
	}
	r := restoreMissingFromPrev(fresh, prev)

	if !r.MetricsNodes {
		t.Error("MetricsNodes 是回退来的，应在报告中标记为 true")
	}
	if r.APMServices {
		t.Error("APMServices 是本轮新取的，不应标记为回退")
	}
}

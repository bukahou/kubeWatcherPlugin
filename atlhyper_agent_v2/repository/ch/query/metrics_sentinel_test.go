package query

import (
	"testing"

	"AtlHyper/model_v3/metrics"
)

// ──────────────────────────────────────────────────────────────
// 指标页重构（2026-08-31）的后端修缮
// ──────────────────────────────────────────────────────────────

// fd 上限在现代内核是 2^63-1（无限制），此前直接透传导致前端渲染出
// 「Max: 9223372036854.8M」。语义判定属后端（大后端小前端），
// 前端不该拿 9.2e18 做魔法数比较。
func TestApplyScalarGauge_FilefdUnlimited(t *testing.T) {
	nm := &metrics.NodeMetrics{}
	applyScalarGauge(nm, "node_filefd_maximum", 9.223372036854776e18, 0)
	if !nm.System.FilefdUnlimited {
		t.Error("2^63-1 应判定为无限制")
	}
	if nm.System.FilefdMax != 0 {
		t.Errorf("无限制时 FilefdMax 应清零避免误用，得到 %d", nm.System.FilefdMax)
	}

	nm2 := &metrics.NodeMetrics{}
	applyScalarGauge(nm2, "node_filefd_maximum", 1048576, 0)
	if nm2.System.FilefdUnlimited || nm2.System.FilefdMax != 1048576 {
		t.Errorf("真实上限应原样保留: max=%d unlimited=%v", nm2.System.FilefdMax, nm2.System.FilefdUnlimited)
	}
}

// NodeIP 字段此前被填成 hostname（net.host.name 即主机名，反查失败时
// 直接拿它凑数），前端出现「archangel archangel」双名。
// 修法：从 K8s 节点信息反查真实 InternalIP，查不到留空（空可隐藏，错值会误导）。
func TestNodeIPByName(t *testing.T) {
	ipToName := map[string]string{
		"192.168.0.11": "desk-one",
		"192.168.0.40": "archangel",
	}
	byName := nodeIPByName(ipToName)
	if byName["desk-one"] != "192.168.0.11" {
		t.Errorf("desk-one → %q, want 192.168.0.11", byName["desk-one"])
	}
	if got, ok := byName["unknown-node"]; ok {
		t.Errorf("未知节点不应有条目，得到 %q", got)
	}
}

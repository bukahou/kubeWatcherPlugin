package query

import (
	"strings"
	"testing"

	"AtlHyper/model_v3/apm"
)

// ──────────────────────────────────────────────────────────────
// Correlations：打分纯函数
// ──────────────────────────────────────────────────────────────

func TestScoreCorrelations_RealCase(t *testing.T) {
	// 真实案例形状（register 500）：2 条失败全是 iPhone，背景 25 条里 2 条
	rows := []correlationRow{
		{Field: "user_agent.family", Value: "iOS", FgCount: 2, BgCount: 2},
		{Field: "user_agent.family", Value: "Windows", FgCount: 0, BgCount: 20},
		{Field: "http.request.method", Value: "POST", FgCount: 2, BgCount: 25}, // 全集值
	}
	items := scoreCorrelations(rows, 2, 25)

	// fgCount=0 的行被丢弃（与前景无关）
	for _, it := range items {
		if it.FgCount == 0 {
			t.Errorf("fgCount=0 的值不应出现在结果中: %+v", it)
		}
	}
	// iOS 排第一：fgRatio=1.0, bgRatio=0.08, lift=12.5
	if len(items) == 0 || items[0].Value != "iOS" {
		t.Fatalf("iOS 应排第一，得到 %+v", items)
	}
	if items[0].Lift < 12 || items[0].Lift > 13 {
		t.Errorf("lift = %v, want ≈12.5", items[0].Lift)
	}
	if items[0].Impact != apm.CorrelationImpactHigh {
		t.Errorf("iOS impact = %q, want high", items[0].Impact)
	}
	// 全集值（POST 前景背景占比都是 100%）lift=1 → score≈0 → low
	for _, it := range items {
		if it.Value == "POST" && it.Impact != apm.CorrelationImpactLow {
			t.Errorf("全集值应为 low impact: %+v", it)
		}
	}
}

func TestScoreCorrelations_Boundaries(t *testing.T) {
	// 前景为空 → 空结果
	if got := scoreCorrelations([]correlationRow{{Field: "f", Value: "v", FgCount: 0, BgCount: 5}}, 0, 5); len(got) != 0 {
		t.Errorf("前景为空应返回空，得到 %+v", got)
	}
	// 背景为 0（防御除零）
	if got := scoreCorrelations(nil, 0, 0); len(got) != 0 {
		t.Errorf("空输入应返回空")
	}
	// Top 10 截断
	rows := make([]correlationRow, 0, 15)
	for i := 0; i < 15; i++ {
		rows = append(rows, correlationRow{Field: "f", Value: strings.Repeat("v", i+1), FgCount: 3, BgCount: 4})
	}
	if got := scoreCorrelations(rows, 10, 100); len(got) > 10 {
		t.Errorf("应截断到 Top 10，得到 %d 条", len(got))
	}
}

// ──────────────────────────────────────────────────────────────
// SQL builder 守护
// ──────────────────────────────────────────────────────────────

func TestCorrelationQuery_Shape(t *testing.T) {
	q := buildCorrelationQuery("Timestamp >= now() - INTERVAL 3600 SECOND AND ServiceName = ?", apm.CorrelationModeFailure, 0)

	// 前景/背景都必须基于入口 span（≈ Transaction），否则内部 span 会稀释统计
	if !strings.Contains(q, "SpanKind = "+apm.SQLSpanKindServer) {
		t.Errorf("相关性统计必须限定入口 span\nSQL:\n%s", q)
	}
	// failure 模式的前景条件
	if !strings.Contains(q, "StatusCode = "+apm.SQLStatusCodeError) {
		t.Errorf("failure 模式缺少前景条件\nSQL:\n%s", q)
	}
	// 字段白名单必须包含案例验证过的关键维度
	for _, f := range []string{"user_agent.family", "url.path", "client.address", "k8s.pod.name", "service.version"} {
		if !strings.Contains(q, f) {
			t.Errorf("字段白名单缺少 %q\nSQL:\n%s", f, q)
		}
	}
	// 缺失值归一化为 (none)：缺失本身可能就是相关项
	if !strings.Contains(q, "(none)") {
		t.Errorf("缺失值应归一化为 (none)\nSQL:\n%s", q)
	}
}

func TestCorrelationQuery_LatencyMode(t *testing.T) {
	q := buildCorrelationQuery("Timestamp >= now() - INTERVAL 3600 SECOND", apm.CorrelationModeLatency, 250*1e6)
	// latency 模式前景 = 超过阈值（纳秒）
	if !strings.Contains(q, "Duration > 250000000") {
		t.Errorf("latency 模式缺少阈值前景条件\nSQL:\n%s", q)
	}
	if strings.Contains(q, "StatusCode = "+apm.SQLStatusCodeError) {
		t.Errorf("latency 模式不应含 failure 条件\nSQL:\n%s", q)
	}
}

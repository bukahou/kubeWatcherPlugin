package query

import (
	"strings"
	"testing"
)

// ──────────────────────────────────────────────────────────────
// ListTraces 两段式聚合守护
// ──────────────────────────────────────────────────────────────
//
// 历史（2026-08-28）：ListTraces 的过滤条件（service/operation 等）直接写在
// 聚合查询的 WHERE 里，作用在 GROUP BY TraceId 之前 —— 于是从 geass-user
// 进入时其他服务的 span 先被剔除，spanCount/serviceCount/rootService
// 全部基于残缺集合计算。实测同一条 trace：过滤后聚合得 1 span / 1 服务，
// 真实为 3 span / 2 服务，且 rootService 被错判为叶子服务。
//
// 正确形态（Jaeger / Tempo 同款）：条件只用于**定位 TraceId 集合**（内层
// 子查询），外层对完整 trace 聚合。本文件把这个结构固化为测试。

// TestListTracesQuery_TwoPhase 断言过滤条件只出现在 TraceId 定位子查询中。
func TestListTracesQuery_TwoPhase(t *testing.T) {
	q := buildListTracesQuery(
		"Timestamp >= now() - INTERVAL 300 SECOND AND ServiceName = ? AND SpanName = ?",
		"ts DESC", 50,
	)

	// 1. 必须是两段式：外层通过 TraceId IN (子查询) 关联
	if !strings.Contains(q, "TraceId IN (") {
		t.Fatalf("ListTraces 必须用 TraceId IN (子查询) 定位，禁止在聚合层直接过滤\nSQL:\n%s", q)
	}

	// 2. 过滤条件（ServiceName）只允许出现一次，且必须在 TraceId IN 子查询
	//    之后 —— 出现两次或出现在子查询之前，都意味着聚合集合被过滤，
	//    spanCount 会再次失真
	if n := strings.Count(q, "ServiceName = ?"); n != 1 {
		t.Errorf("过滤条件应恰好出现一次（在定位子查询中），实际 %d 次\nSQL:\n%s", n, q)
	}
	if strings.Index(q, "ServiceName = ?") < strings.Index(q, "TraceId IN (") {
		t.Errorf("过滤条件出现在聚合层（TraceId IN 之前）—— 聚合的将是残缺 trace\nSQL:\n%s", q)
	}

	// 3. 聚合列完整性：spanCount 与 serviceCount 必须基于完整 trace
	for _, col := range []string{"count() AS spanCount", "count(DISTINCT ServiceName) AS serviceCount"} {
		if !strings.Contains(q, col) {
			t.Errorf("缺少聚合列 %q\nSQL:\n%s", col, q)
		}
	}
}

// TestListTracesQuery_EntrySpanRule 断言根服务判定引用统一的入口 span 规则：
// 优先 SpanKind = Server 的最早 span，退化到无父 span。
// 该规则 ≈ Elastic APM 的 Transaction 语义，所有「trace 的根是谁」的
// 查询必须同源，禁止各自内联变体。
func TestListTracesQuery_EntrySpanRule(t *testing.T) {
	q := buildListTracesQuery("Timestamp >= now() - INTERVAL 300 SECOND", "ts DESC", 50)

	if !strings.Contains(q, "SpanKind = 'Server'") {
		t.Errorf("根服务判定必须优先 Server span（入口 span 规则）\nSQL:\n%s", q)
	}
	if !strings.Contains(q, "ParentSpanId = ''") {
		t.Errorf("根服务判定缺少无父 span 的退化路径\nSQL:\n%s", q)
	}
}

// TestListTracesQuery_LimitClamp 排序与 LIMIT 作用于聚合结果（外层）。
func TestListTracesQuery_LimitClamp(t *testing.T) {
	q := buildListTracesQuery("Timestamp >= now() - INTERVAL 60 SECOND", "durationMs DESC", 10)
	// LIMIT 必须在最外层（最后一次出现的 GROUP BY 之后）
	lastGroup := strings.LastIndex(q, "GROUP BY TraceId")
	limitPos := strings.LastIndex(q, "LIMIT 10")
	if limitPos < lastGroup {
		t.Errorf("LIMIT 必须作用于聚合后的结果\nSQL:\n%s", q)
	}
	if !strings.Contains(q, "ORDER BY durationMs DESC") {
		t.Errorf("排序参数未生效\nSQL:\n%s", q)
	}
}

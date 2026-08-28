package query

import (
	"testing"
	"time"

	"AtlHyper/model_v3/apm"
)

// ──────────────────────────────────────────────────────────────
// 错误证据链回填（attachLogErrors）
// ──────────────────────────────────────────────────────────────
//
// 背景（2026-08-28 register 500）：connect-go 的 Error span 既没有
// exception 事件也没有 StatusMessage，真正的根因
// 'Data truncated for column auth_type' 只存在于 **gateway 的 ERROR
// 日志属性** 里 —— 而排查者点开的往往是下游 user 服务的 span。
// attachLogErrors 把日志证据回填到 span 上，并标注来源，
// 让任何一个 Error span 都能给出「哪一层、什么异常、什么消息」。

func errSpan(id, service, msg string) apm.Span {
	return apm.Span{
		SpanId:        id,
		ServiceName:   service,
		StatusCode:    apm.StatusCodeError,
		StatusMessage: msg,
		Timestamp:     time.Date(2026, 8, 28, 11, 26, 23, 0, time.UTC),
	}
}

func exLog(spanId, service, exType, exMsg string) traceErrorLog {
	return traceErrorLog{
		SpanId: spanId, ServiceName: service,
		Attributes: map[string]string{
			"exception.type": exType, "exception.message": exMsg,
		},
	}
}

// 真实案例形状：user span 无任何自带错误信息，gateway 日志持有 exception
func TestAttachLogErrors_CrossServiceEvidence(t *testing.T) {
	spans := []apm.Span{
		errSpan("gw-root", "geass-gateway", ""),
		errSpan("user-1", "geass-user", ""),
	}
	logs := []traceErrorLog{
		exLog("", "geass-gateway", "*connect.Error", "internal: Error 1265: Data truncated for column 'auth_type'"),
		{SpanId: "", ServiceName: "geass-user", Attributes: map[string]string{"status": "500"}}, // 无 exception 的日志不算证据
	}

	attachLogErrors(spans, logs)

	// user span 的证据来自 gateway 日志（跨服务），并标注来源服务
	if spans[1].Error == nil {
		t.Fatal("user span 应从 gateway 日志获得错误证据")
	}
	if spans[1].Error.Source != apm.ErrorSourceTraceLog {
		t.Errorf("Source = %q, want trace_log", spans[1].Error.Source)
	}
	if spans[1].Error.SourceService != "geass-gateway" {
		t.Errorf("SourceService = %q, want geass-gateway", spans[1].Error.SourceService)
	}
	if spans[1].Error.Message == "" || spans[1].Error.Type != "*connect.Error" {
		t.Errorf("证据内容不完整: %+v", spans[1].Error)
	}
}

// 优先级：span 已有 Events exception → 不覆盖
func TestAttachLogErrors_SpanEventWins(t *testing.T) {
	spans := []apm.Span{errSpan("s1", "svc", "")}
	spans[0].Error = &apm.SpanError{Type: "MyError", Message: "from event"}
	attachLogErrors(spans, []traceErrorLog{exLog("", "svc", "Other", "from log")})

	if spans[0].Error.Message != "from event" {
		t.Errorf("span_event 证据不应被日志覆盖: %+v", spans[0].Error)
	}
	if spans[0].Error.Source != apm.ErrorSourceSpanEvent {
		t.Errorf("已有 Events 错误应标注 source=span_event，得到 %q", spans[0].Error.Source)
	}
}

// 优先级：StatusMessage 次之
func TestAttachLogErrors_StatusMessageSecond(t *testing.T) {
	spans := []apm.Span{errSpan("s1", "svc", "rpc error: deadline exceeded")}
	attachLogErrors(spans, []traceErrorLog{exLog("", "svc", "Other", "from log")})

	if spans[0].Error == nil || spans[0].Error.Source != apm.ErrorSourceStatusMessage {
		t.Fatalf("有 StatusMessage 时应优先用它: %+v", spans[0].Error)
	}
	if spans[0].Error.Message != "rpc error: deadline exceeded" {
		t.Errorf("Message = %q", spans[0].Error.Message)
	}
}

// SpanId 精确匹配优先于服务匹配
func TestAttachLogErrors_SpanIdBeatsService(t *testing.T) {
	spans := []apm.Span{errSpan("s1", "svc-a", "")}
	logs := []traceErrorLog{
		exLog("", "svc-a", "ServiceMatch", "同服务"),
		exLog("s1", "svc-b", "ExactMatch", "精确 SpanId"),
	}
	attachLogErrors(spans, logs)
	if spans[0].Error.Type != "ExactMatch" {
		t.Errorf("SpanId 精确匹配应优先，得到 %+v", spans[0].Error)
	}
}

// 同服务日志优先于跨服务日志
func TestAttachLogErrors_ServiceBeatsCross(t *testing.T) {
	spans := []apm.Span{errSpan("s1", "svc-a", "")}
	logs := []traceErrorLog{
		exLog("", "svc-b", "Cross", "跨服务"),
		exLog("", "svc-a", "Same", "同服务"),
	}
	attachLogErrors(spans, logs)
	if spans[0].Error.Type != "Same" {
		t.Errorf("同服务证据应优先，得到 %+v", spans[0].Error)
	}
}

// 非 Error span 一律不回填；无 exception 字段的日志不构成证据
func TestAttachLogErrors_Negative(t *testing.T) {
	ok := apm.Span{SpanId: "ok", ServiceName: "svc", StatusCode: apm.StatusCodeOk}
	spans := []apm.Span{ok, errSpan("e1", "svc", "")}
	attachLogErrors(spans, []traceErrorLog{
		{SpanId: "", ServiceName: "svc", Attributes: map[string]string{"status": "500"}},
	})
	if spans[0].Error != nil {
		t.Error("非 Error span 不应被回填")
	}
	if spans[1].Error != nil {
		t.Error("无 exception 字段的日志不构成证据")
	}
}

package apm

import "testing"

// TestSpan_IsError 验证错误判定使用 ClickHouse 实际写入的 StatusCode 取值。
//
// 回归背景: 曾硬编码为 "STATUS_CODE_ERROR"（Collector 旧版格式），
// 导致该方法在新版 Collector 下恒为 false，前端错误计数始终为 0。
func TestSpan_IsError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode string
		want       bool
	}{
		{"错误状态", StatusCodeError, true},
		{"未设置状态", StatusCodeUnset, false},
		{"成功状态", StatusCodeOk, false},
		{"空值", "", false},
		{"旧版格式不应被识别", "STATUS_CODE_ERROR", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Span{StatusCode: tt.statusCode}
			if got := s.IsError(); got != tt.want {
				t.Errorf("IsError() = %v, want %v (StatusCode=%q)", got, tt.want, tt.statusCode)
			}
		})
	}
}

// TestSpan_IsServer 验证入站 span 判定。
func TestSpan_IsServer(t *testing.T) {
	tests := []struct {
		name     string
		spanKind string
		want     bool
	}{
		{"服务端 span", SpanKindServer, true},
		{"客户端 span", SpanKindClient, false},
		{"内部 span", SpanKindInternal, false},
		{"空值", "", false},
		{"旧版格式不应被识别", "SPAN_KIND_SERVER", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Span{SpanKind: tt.spanKind}
			if got := s.IsServer(); got != tt.want {
				t.Errorf("IsServer() = %v, want %v (SpanKind=%q)", got, tt.want, tt.spanKind)
			}
		})
	}
}

// TestSpan_IsClient 验证出站 span 判定 —— 跨服务拓扑边推导依赖它。
func TestSpan_IsClient(t *testing.T) {
	tests := []struct {
		name     string
		spanKind string
		want     bool
	}{
		{"客户端 span", SpanKindClient, true},
		{"服务端 span", SpanKindServer, false},
		{"空值", "", false},
		{"旧版格式不应被识别", "SPAN_KIND_CLIENT", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Span{SpanKind: tt.spanKind}
			if got := s.IsClient(); got != tt.want {
				t.Errorf("IsClient() = %v, want %v (SpanKind=%q)", got, tt.want, tt.spanKind)
			}
		})
	}
}

// TestSpan_IsRoot 验证根 span 判定（未受本次变更影响，补作回归保护）。
func TestSpan_IsRoot(t *testing.T) {
	if !(&Span{ParentSpanId: ""}).IsRoot() {
		t.Error("空 ParentSpanId 应判定为根 span")
	}
	if (&Span{ParentSpanId: "abc123"}).IsRoot() {
		t.Error("非空 ParentSpanId 不应判定为根 span")
	}
}

// TestSQLLiterals 验证 SQL 字面量常量带正确的单引号包裹。
//
// 这些常量会被直接拼进 ClickHouse 查询语句，少一个引号就是语法错误，
// 多一层引号则匹配不到任何行 —— 两种情况都不会在编译期暴露。
func TestSQLLiterals(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"SpanKind Server", SQLSpanKindServer, "'Server'"},
		{"SpanKind Client", SQLSpanKindClient, "'Client'"},
		{"StatusCode Error", SQLStatusCodeError, "'Error'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("SQL 字面量 = %s, want %s", tt.got, tt.want)
			}
		})
	}
}

// TestExpectedSets 保证契约自检用的集合与常量定义同步。
func TestExpectedSets(t *testing.T) {
	for _, k := range []string{SpanKindServer, SpanKindClient, SpanKindInternal} {
		if !contains(ExpectedSpanKinds, k) {
			t.Errorf("ExpectedSpanKinds 缺少 %q", k)
		}
	}
	for _, c := range []string{StatusCodeUnset, StatusCodeOk, StatusCodeError} {
		if !contains(ExpectedStatusCodes, c) {
			t.Errorf("ExpectedStatusCodes 缺少 %q", c)
		}
	}
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

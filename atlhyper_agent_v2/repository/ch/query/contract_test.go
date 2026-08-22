package query

import (
	"reflect"
	"testing"
)

func TestDiff(t *testing.T) {
	tests := []struct {
		name     string
		actual   []string
		expected []string
		want     []string
	}{
		{"全部符合", []string{"Server", "Client"}, []string{"Server", "Client", "Internal"}, nil},
		{"出现旧版格式", []string{"SPAN_KIND_SERVER", "Client"}, []string{"Server", "Client"}, []string{"SPAN_KIND_SERVER"}},
		{"大小写视为不同", []string{"info"}, []string{"INFO"}, []string{"info"}},
		{"实际为空", nil, []string{"INFO"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := diff(tt.actual, tt.expected); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("diff() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestEnumContracts_Registered 保证三类契约都已登记（新增信号源时同步更新此处）。
func TestEnumContracts_Registered(t *testing.T) {
	want := map[string]bool{
		"otel_traces.SpanKind":   false,
		"otel_traces.StatusCode": false,
		"otel_logs.SeverityText": false,
	}
	for _, c := range enumContracts {
		key := c.table + "." + c.column
		if _, ok := want[key]; ok {
			want[key] = true
		}
		if len(c.expected) == 0 {
			t.Errorf("%s 的预期集合为空", key)
		}
	}
	for k, seen := range want {
		if !seen {
			t.Errorf("契约 %s 未登记", k)
		}
	}
}

package aiops

import (
	"math"
	"testing"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// ──────────────────────────────────────────────────────────────
// 基线响应构造
// ──────────────────────────────────────────────────────────────
//
// 冷启动阈值（aiops.ColdStartMinCount）决定基线是否可用于告警：
// count 未达阈值时引擎只学习不判异常。前端必须能区分「基线为 0」
// 和「还没学够、这个 0 不作数」—— 否则会把学习中的实体误读为已就绪。
//
// 阈值由后端下发而非前端硬编码：前端复制后端常量正是本项目
// 「契约层静默失效」踩过三次的坑（底层改了，前端不知道）。

func TestBuildBaselineResponse_ExposesColdStartThreshold(t *testing.T) {
	src := &aiops.EntityBaseline{
		EntityKey: "ns/pod/foo",
		States: []*aiops.BaselineState{
			{MetricName: "restart_count", EMA: 0, Count: 61},
		},
	}
	got := buildBaselineResponse(src)

	if got.ColdStartMinCount != int64(aiops.ColdStartMinCount) {
		t.Errorf("ColdStartMinCount = %d, want %d（必须与引擎常量同源）",
			got.ColdStartMinCount, aiops.ColdStartMinCount)
	}
	if len(got.States) != 1 {
		t.Fatalf("States 长度 = %d, want 1", len(got.States))
	}
	if got.States[0].Ready {
		t.Error("count=61 < 100 应为未就绪")
	}
}

func TestBuildBaselineResponse_ReadyAtThreshold(t *testing.T) {
	tests := []struct {
		count int64
		want  bool
	}{
		{0, false},
		{99, false},
		{100, true}, // 边界：达到阈值即就绪
		{500, true},
	}
	for _, tt := range tests {
		src := &aiops.EntityBaseline{
			States: []*aiops.BaselineState{{Count: tt.count}},
		}
		if got := buildBaselineResponse(src).States[0].Ready; got != tt.want {
			t.Errorf("count=%d → Ready=%v, want %v", tt.count, got, tt.want)
		}
	}
}

// 引擎未启用时返回 nil，不能 panic
func TestBuildBaselineResponse_NilSource(t *testing.T) {
	if got := buildBaselineResponse(nil); got != nil {
		t.Errorf("nil 输入应返回 nil, got %+v", got)
	}
}

// Go nil slice 序列化为 null，前端需兜底；此处保证返回空切片
func TestBuildBaselineResponse_NilStates(t *testing.T) {
	got := buildBaselineResponse(&aiops.EntityBaseline{EntityKey: "k"})
	if got.States == nil {
		t.Error("States 应为空切片而非 nil")
	}
}

// 标准差由后端算，前端不做 sqrt —— 派生字段属后端职责（大后端小前端）
func TestBuildBaselineResponse_StdDev(t *testing.T) {
	src := &aiops.EntityBaseline{
		States: []*aiops.BaselineState{
			{MetricName: "cpu", Variance: 6.25}, // sqrt = 2.5
			{MetricName: "zero", Variance: 0},
		},
	}
	got := buildBaselineResponse(src)
	if math.Abs(got.States[0].StdDev-2.5) > 1e-9 {
		t.Errorf("StdDev = %v, want 2.5", got.States[0].StdDev)
	}
	if got.States[1].StdDev != 0 {
		t.Errorf("方差为 0 时 StdDev 应为 0, got %v", got.States[1].StdDev)
	}
}

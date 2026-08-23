package query

import (
	"os"
	"strings"
	"testing"
)

// ──────────────────────────────────────────────────────────────
// 抽象契约守护 —— 本文件最重要的测试
// ──────────────────────────────────────────────────────────────

// vendorNames 是各 ingress / mesh 实现的专属标识。
//
// SLO 查询必须只认 ingress_* 契约，不得出现任何实现的名字 ——
// 否则换 ingress 实现时 Agent 代码就得跟着改，抽象即失效。
//
// 历史：Ingress 一路曾硬编码 traefik_service_requests_total 与
// Attributes['code']，Traefik 从集群移除后 SLO 页长期空白且无人察觉。
// Mesh 一路因为做了 mesh_request_total 契约，Istio 装了又拆都未伤及代码。
// 本测试把"照正例做"变成编译期之后的强制约束。
var vendorNames = []string{
	"traefik", "envoy", "istio", "linkerd", "nginx", "haproxy",
	"contour", "kong", "cilium", "hubble", "gateway-api",
}

// TestIngressSQL_NoVendorNames 断言 Ingress SLO 的 SQL 不含实现专属名。
func TestIngressSQL_NoVendorNames(t *testing.T) {
	queries := map[string]string{
		"count":   buildIngressCountQuery("AND TimeUnix > now() - INTERVAL 5 MINUTE"),
		"latency": buildIngressLatencyQuery("AND TimeUnix > now() - INTERVAL 5 MINUTE"),
		"history": buildIngressHistoryQuery("AND TimeUnix > now() - INTERVAL 1 DAY", 300),
		"summary": buildIngressSummaryQuery(),
	}
	for name, q := range queries {
		lower := strings.ToLower(q)
		for _, vendor := range vendorNames {
			if strings.Contains(lower, vendor) {
				t.Errorf("%s 查询含实现专属名 %q —— 抽象失效，换 ingress 实现时此处会断\nSQL:\n%s",
					name, vendor, q)
			}
		}
	}
}

// TestIngressSQL_UsesContractNames 断言使用的是契约定义的指标名与 label。
func TestIngressSQL_UsesContractNames(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{
			"计数查询",
			buildIngressCountQuery(""),
			[]string{metricIngressRequestTotal, "status_class", "namespace", "service"},
		},
		{
			"延迟查询",
			buildIngressLatencyQuery(""),
			[]string{metricIngressDurationBucket, "namespace", "service"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, want := range tt.want {
				if !strings.Contains(tt.query, want) {
					t.Errorf("查询缺少契约标识 %q\nSQL:\n%s", want, tt.query)
				}
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────
// serviceKey
// ──────────────────────────────────────────────────────────────

func TestIngressServiceKey(t *testing.T) {
	tests := []struct {
		namespace, service, want string
	}{
		{"geass-v3", "geass-gateway", "geass-v3/geass-gateway"},
		{"atlhyper", "atlhyper-web", "atlhyper/atlhyper-web"},
		{"", "orphan", "orphan"}, // namespace 缺失时退化为纯服务名
		{"ns", "", ""},           // service 缺失视为无效
	}
	for _, tt := range tests {
		if got := ingressServiceKey(tt.namespace, tt.service); got != tt.want {
			t.Errorf("ingressServiceKey(%q,%q) = %q, want %q", tt.namespace, tt.service, got, tt.want)
		}
	}
}

// ──────────────────────────────────────────────────────────────
// 单位：契约桶为毫秒，P*Ms 字段直接取用，不得二次换算
// ──────────────────────────────────────────────────────────────

// TestHistToLatency_BucketsAreMilliseconds 锁定单位约定。
//
// 回归背景：Traefik 的直方图桶单位是秒，旧实现在 histToLatency 里乘了 1000。
// ingress_* 契约的桶来自 Envoy，单位已是毫秒 —— 若沿用 *1000，P99 会放大一千倍
// （65ms 显示成 65000ms），而这不会导致任何报错，只是数字离谱。
func TestHistToLatency_BucketsAreMilliseconds(t *testing.T) {
	// 桶边界 [0.5, 1, 5, 10, 25, 50] 毫秒，全部请求落在 (10, 25] 这一桶
	hist := &svcHistogram{
		bounds: []float64{0.5, 1, 5, 10, 25, 50},
		counts: []uint64{0, 0, 0, 0, 100, 0, 0},
	}
	p50, _, _, p99, buckets := histToLatency(hist)

	if p50 < 10 || p50 > 25 {
		t.Errorf("P50 = %v，应落在 (10,25] 毫秒区间内；若约为 %v 则说明发生了秒→毫秒的重复换算", p50, p50/1000)
	}
	if p99 < 10 || p99 > 25 {
		t.Errorf("P99 = %v，应落在 (10,25] 毫秒区间内", p99)
	}
	if len(buckets) == 0 {
		t.Fatal("buckets 为空")
	}
	if buckets[0].LE != 0.5 {
		t.Errorf("首桶 LE = %v, want 0.5（契约单位毫秒，不再 ×1000）", buckets[0].LE)
	}
}

// ──────────────────────────────────────────────────────────────
// 错误判定：契约用 status_class，不是精确状态码
// ──────────────────────────────────────────────────────────────

func TestIsErrorStatusClass(t *testing.T) {
	tests := []struct {
		class string
		want  bool
	}{
		{"5", true},  // 服务端错误计入 SLO 违约
		{"4", false}, // 客户端错误不计（用户输入问题非服务质量问题）
		{"2", false},
		{"3", false},
		{"1", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isErrorStatusClass(tt.class); got != tt.want {
			t.Errorf("isErrorStatusClass(%q) = %v, want %v", tt.class, got, tt.want)
		}
	}
}

// ──────────────────────────────────────────────────────────────
// 平均延迟：由 histogram 的 Sum/Count 求得，而非从桶估算
// ──────────────────────────────────────────────────────────────

// TestHistAvgMs 验证均值取自 Sum/Count。
//
// 桶只能给出分位数的近似（受桶宽限制），但 Envoy 同时导出了 Sum 与 Count，
// 两者相除是【精确】均值。此前 avgMs 恒为 0 —— 数据一直在，只是没用上。
func TestHistAvgMs(t *testing.T) {
	tests := []struct {
		name  string
		sum   float64
		count uint64
		want  float64
	}{
		{"正常均值", 1665997.8, 32900, 50.64},
		{"零请求", 0, 0, 0}, // 不得除零
		{"单请求", 42, 1, 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &svcHistogram{sum: tt.sum, count: tt.count}
			if got := histAvgMs(h); got != tt.want {
				t.Errorf("histAvgMs(sum=%v,count=%v) = %v, want %v", tt.sum, tt.count, got, tt.want)
			}
		})
	}
}

// TestAddHistogramDelta_AccumulatesSumCount 验证多行聚合时 Sum/Count 也做 delta 累加。
//
// 同一服务在窗口内可能有多个系列（不同 status_class），各自是独立累积计数器，
// 必须逐系列取 delta 再相加 —— 直接用最新值会把历史总量算进来。
func TestAddHistogramDelta_AccumulatesSumCount(t *testing.T) {
	h := &svcHistogram{}
	// 系列 A：count 100→150 (delta 50)，sum 1000→1600 (delta 600)
	addHistogramDelta(h, []float64{1, 5}, []uint64{10, 20}, []uint64{5, 10}, 1600, 1000, 150, 100)
	// 系列 B：count 20→30 (delta 10)，sum 200→400 (delta 200)
	addHistogramDelta(h, []float64{1, 5}, []uint64{8, 12}, []uint64{4, 6}, 400, 200, 30, 20)

	if h.count != 60 {
		t.Errorf("count = %d, want 60 (50+10)", h.count)
	}
	if h.sum != 800 {
		t.Errorf("sum = %v, want 800 (600+200)", h.sum)
	}
	if got := histAvgMs(h); got != roundTo(800.0/60.0, 2) {
		t.Errorf("avgMs = %v, want %v", got, roundTo(800.0/60.0, 2))
	}
}

// TestAddHistogramDelta_CounterReset 验证 Sum/Count 的 counter reset 处理。
func TestAddHistogramDelta_CounterReset(t *testing.T) {
	h := &svcHistogram{}
	// latest < earliest 表示窗口内发生了 reset，delta 取 latest 本身
	addHistogramDelta(h, []float64{1}, []uint64{5}, []uint64{100}, 50, 900, 5, 90)
	if h.count != 5 {
		t.Errorf("reset 后 count = %d, want 5 (取 latest)", h.count)
	}
	if h.sum != 50 {
		t.Errorf("reset 后 sum = %v, want 50 (取 latest)", h.sum)
	}
}

// ──────────────────────────────────────────────────────────────
// 多实例聚合
// ──────────────────────────────────────────────────────────────
//
// cilium-envoy 是 DaemonSet，每个节点一个实例，各自维护独立的累积计数器。
// 2026-08-23 实测：两个实例（12327 与 4556）在同一分区里按时间交错排序，
// 每次切换实例都产生 ±7771 的假 delta —— 5 分钟真实 10 个请求被算成 238679，
// 偏差 24000 倍，同时让 histogram 的差值变负、P95 恒为 0。
//
// 因此所有跨时间做差的查询都必须先按实例分区。

func TestIngressSQL_PartitionsByInstance(t *testing.T) {
	cases := map[string]string{
		"count":   buildIngressCountQuery("AND 1=1"),
		"history": buildIngressHistoryQuery("AND 1=1", 60),
		"latency": buildIngressLatencyQuery("AND 1=1"),
	}
	for name, q := range cases {
		if !strings.Contains(q, labelInstance) {
			t.Errorf("%s 查询未按实例区分（缺 %s）—— 多个 envoy 实例的计数器会互相污染", name, labelInstance)
		}
	}
}

// 计数类查询必须在 PARTITION BY 里带上实例，光在 SELECT 里出现没有用。
func TestIngressSQL_InstanceInPartitionClause(t *testing.T) {
	for name, q := range map[string]string{
		"count":   buildIngressCountQuery("AND 1=1"),
		"history": buildIngressHistoryQuery("AND 1=1", 60),
	} {
		idx := strings.Index(q, "PARTITION BY")
		if idx < 0 {
			t.Fatalf("%s 查询没有 PARTITION BY", name)
		}
		end := strings.Index(q[idx:], "ORDER BY")
		if end < 0 {
			t.Fatalf("%s 的 PARTITION BY 后没有 ORDER BY", name)
		}
		if !strings.Contains(q[idx:idx+end], labelInstance) {
			t.Errorf("%s 的 PARTITION BY 子句里没有实例维度:\n%s", name, q[idx:idx+end])
		}
	}
}

// 直方图按实例分组后会返回多行（实例数 × 服务数），Go 侧必须累加而非覆盖。
// addHistogramDelta 已经是累加语义，这里锁住这个行为不被改坏。
func TestAddHistogramDelta_MultipleInstancesAccumulate(t *testing.T) {
	bounds := []float64{10, 50, 100}
	hist := &svcHistogram{}

	// 实例 A：窗口内 5 个请求
	addHistogramDelta(hist, bounds, []uint64{3, 5, 6, 6}, []uint64{1, 2, 3, 3}, 500, 100, 20, 15)
	// 实例 B：窗口内 4 个请求
	addHistogramDelta(hist, bounds, []uint64{10, 12, 14, 14}, []uint64{9, 11, 12, 13}, 800, 700, 44, 40)

	// 桶计数应为两个实例 delta 之和
	want := []uint64{2 + 1, 3 + 1, 3 + 2, 3 + 1}
	for i, w := range want {
		if hist.counts[i] != w {
			t.Errorf("桶 %d = %d，期望 %d（两实例 delta 之和）", i, hist.counts[i], w)
		}
	}
	if hist.count != (20-15)+(44-40) {
		t.Errorf("count = %d，期望 9", hist.count)
	}
	if hist.sum != (500-100)+(800-700) {
		t.Errorf("sum = %v，期望 500", hist.sum)
	}
}

// 时序查询同样跨时间做差，同样必须按实例分区 —— 否则趋势图上全是尖刺。
func TestGetSLOTimeSeriesSQL_PartitionsByInstance(t *testing.T) {
	// 用一个不会执行的 client 只为拿到 SQL 是不可能的（SQL 在函数内构造），
	// 因此这里直接扫源码，与 metrics_catalog_test 同样的守护思路。
	src, err := os.ReadFile("slo.go")
	if err != nil {
		t.Fatalf("读 slo.go 失败: %v", err)
	}
	body := string(src)
	start := strings.Index(body, "func (r *sloRepository) GetSLOTimeSeries")
	if start < 0 {
		t.Fatal("找不到 GetSLOTimeSeries")
	}
	end := strings.Index(body[start:], "\n}\n")
	fn := body[start : start+end]

	idx := strings.Index(fn, "PARTITION BY")
	if idx < 0 {
		t.Fatal("GetSLOTimeSeries 没有 PARTITION BY")
	}
	clause := fn[idx : idx+strings.Index(fn[idx:], "ORDER BY")]
	// 源码里是 %s 占位符，看不到常量名；但实例来自 ResourceAttributes，
	// 而 ns/svc/class 来自 Attributes —— 本文件里 ResourceAttributes 只用于实例。
	if !strings.Contains(clause, "ResourceAttributes[") {
		t.Errorf("GetSLOTimeSeries 的 PARTITION BY 缺实例维度:\n%s", clause)
	}
}

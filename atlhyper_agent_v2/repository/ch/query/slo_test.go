package query

import (
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

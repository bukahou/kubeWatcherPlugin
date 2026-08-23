package k8s

import (
	"reflect"
	"testing"

	"AtlHyper/model_v3/cluster"
)

// ──────────────────────────────────────────────────────────────
// 路由映射：serviceKey → 对外域名
// ──────────────────────────────────────────────────────────────
//
// SLO 的 displayName 需要真实域名（geass-api.bukahou.com），而任何 ingress
// 实现的【指标】都不带域名维度（实测 Envoy 无 vhost 统计、Hubble 无 host label）。
// 因此改从 K8s 路由资源反查。
//
// 抽象要点：映射源必须是 K8s 标准资源，不是某个 ingress 实现的私有 CRD。
// Gateway API 的 HTTPRoute 与 Ingress 都满足；Traefik 的 IngressRoute 之类不行。

func TestBuildRouteMap_HTTPRoute(t *testing.T) {
	routes := []RouteEntry{
		{Hostnames: []string{"geass-api.bukahou.com"}, Backends: []string{"geass-v3/geass-gateway"}},
		{Hostnames: []string{"argocd.bukahou.com"}, Backends: []string{"argocd/argocd-server"}},
	}
	got := BuildRouteMap(routes)
	want := map[string]string{
		"geass-v3/geass-gateway": "geass-api.bukahou.com",
		"argocd/argocd-server":   "argocd.bukahou.com",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BuildRouteMap() = %v, want %v", got, want)
	}
}

// TestBuildRouteMap_MultiHostname 一个服务挂多个域名时全部保留，逗号分隔。
//
// 这正是从路由资源反查优于从指标标签取的地方：指标只能带一个 host，
// 而路由资源能表达完整的多域名关系。
func TestBuildRouteMap_MultiHostname(t *testing.T) {
	routes := []RouteEntry{
		{Hostnames: []string{"a.example.com", "b.example.com"}, Backends: []string{"ns/svc"}},
	}
	if got := BuildRouteMap(routes)["ns/svc"]; got != "a.example.com, b.example.com" {
		t.Errorf("多域名 = %q, want %q", got, "a.example.com, b.example.com")
	}
}

// TestBuildRouteMap_MultiBackend 一条路由分流到多个后端时，每个后端都记录该域名。
func TestBuildRouteMap_MultiBackend(t *testing.T) {
	routes := []RouteEntry{
		{Hostnames: []string{"api.example.com"}, Backends: []string{"ns/svc-a", "ns/svc-b"}},
	}
	m := BuildRouteMap(routes)
	for _, k := range []string{"ns/svc-a", "ns/svc-b"} {
		if m[k] != "api.example.com" {
			t.Errorf("%s = %q, want api.example.com", k, m[k])
		}
	}
}

// TestBuildRouteMap_SameBackendMultiRoute 同一后端被多条路由引用时合并域名且去重。
func TestBuildRouteMap_SameBackendMultiRoute(t *testing.T) {
	routes := []RouteEntry{
		{Hostnames: []string{"a.example.com"}, Backends: []string{"ns/svc"}},
		{Hostnames: []string{"b.example.com"}, Backends: []string{"ns/svc"}},
		{Hostnames: []string{"a.example.com"}, Backends: []string{"ns/svc"}}, // 重复
	}
	if got := BuildRouteMap(routes)["ns/svc"]; got != "a.example.com, b.example.com" {
		t.Errorf("合并结果 = %q, want %q", got, "a.example.com, b.example.com")
	}
}

// TestBuildRouteMap_EdgeCases 边界：无域名的路由、无后端的路由都应被跳过而非产生空键。
func TestBuildRouteMap_EdgeCases(t *testing.T) {
	routes := []RouteEntry{
		{Hostnames: nil, Backends: []string{"ns/svc"}},                 // 无域名（通配路由）
		{Hostnames: []string{"x.example.com"}, Backends: nil},          // 无后端
		{Hostnames: []string{""}, Backends: []string{"ns/svc2"}},       // 空域名
		{Hostnames: []string{"y.example.com"}, Backends: []string{""}}, // 空后端
	}
	got := BuildRouteMap(routes)
	if len(got) != 0 {
		t.Errorf("边界输入应产出空映射，实际 = %v", got)
	}
}

func TestBuildRouteMap_Nil(t *testing.T) {
	if got := BuildRouteMap(nil); len(got) != 0 {
		t.Errorf("nil 输入 = %v, want 空映射", got)
	}
}

// TestIngressesToRouteEntries 验证原生 Ingress 的归约，含 nil Backend 保护。
//
// IngressPath.Backend 是指针，实际集群里可能为 nil（如仅用 defaultBackend 的规则）。
// 不做 nil 判断会在采集时 panic —— 编译期不会暴露。
func TestIngressesToRouteEntries(t *testing.T) {
	svc := func(name string) *cluster.IngressBackend {
		return &cluster.IngressBackend{Service: &cluster.IngressServiceBackend{Name: name}}
	}
	ings := []cluster.Ingress{
		{
			Summary: cluster.IngressSummary{Namespace: "shop"},
			Spec: cluster.IngressSpec{Rules: []cluster.IngressRule{
				{Host: "shop.example.com", Paths: []cluster.IngressPath{
					{Path: "/", Backend: svc("web")},
					{Path: "/api", Backend: svc("api")},
					{Path: "/dup", Backend: svc("web")}, // 重复后端应去重
					{Path: "/nil", Backend: nil},        // nil 必须跳过而非 panic
				}},
			}},
		},
	}
	got := BuildRouteMap(IngressesToRouteEntries(ings))
	want := map[string]string{
		"shop/web": "shop.example.com",
		"shop/api": "shop.example.com",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("= %v, want %v", got, want)
	}
}

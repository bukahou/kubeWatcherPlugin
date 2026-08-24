package slo

import (
	"testing"

	"AtlHyper/atlhyper_master_v2/model"

	slomodel "AtlHyper/model_v3/slo"
)

// 路由映射表为空时走 fallback。此时域名必须取 DisplayName —— Agent 侧的
// RouteRepository 已经从 HTTPRoute 解析出真实域名并填在这里，
// 用 ServiceKey 会让页面显示 "geass-v3/geass-gateway" 而不是 "geass-api.bukahou.com"。
func TestFallbackDomainName(t *testing.T) {
	cases := []struct {
		name string
		ing  slomodel.IngressSLO
		want string
	}{
		{
			"有真实域名时用域名",
			slomodel.IngressSLO{ServiceKey: "geass-v3/geass-gateway", DisplayName: "geass-api.bukahou.com"},
			"geass-api.bukahou.com",
		},
		{
			"没有域名时退回 ServiceKey",
			slomodel.IngressSLO{ServiceKey: "akasha/akasha", DisplayName: ""},
			"akasha/akasha",
		},
		{
			"DisplayName 等于 ServiceKey（未解析）时也是 ServiceKey",
			slomodel.IngressSLO{ServiceKey: "argocd/argocd-server", DisplayName: "argocd/argocd-server"},
			"argocd/argocd-server",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := fallbackDomainName(c.ing); got != c.want {
				t.Errorf("fallbackDomainName(%+v) = %q, want %q", c.ing, got, c.want)
			}
		})
	}
}

// 域名 → serviceKey 反查。
//
// 与 fallbackDomainName 是一对：那个把 serviceKey 变成展示用的域名，
// 这个把域名变回查询用的 serviceKey。少了它，Phase 1 改域名显示之后
// 历史图与延迟分布就查不到数据了（2026-08-24 实测：清单表数字全对，
// 但 SLO 趋势 / 错误预算消耗 / 延迟分布三处都是「暂无数据」）。
func TestServiceKeysForDomain(t *testing.T) {
	ingress := []slomodel.IngressSLO{
		{ServiceKey: "geass-v3/geass-gateway", DisplayName: "geass-api.bukahou.com"},
		{ServiceKey: "atlhyper/atlhyper-web", DisplayName: "bukahou.com"},
		{ServiceKey: "akasha/akasha", DisplayName: "akasha.bukahou.com"},
	}

	t.Run("按真实域名反查到 serviceKey", func(t *testing.T) {
		got := serviceKeysForDomain("geass-api.bukahou.com", ingress, nil)
		if len(got) != 1 || !got["geass-v3/geass-gateway"] {
			t.Errorf("= %v，期望 {geass-v3/geass-gateway}", got)
		}
	})

	t.Run("路由映射表有数据时优先用它", func(t *testing.T) {
		mappings := []*model.SLORouteMapping{
			{ServiceKey: "ns-a/svc-a"}, {ServiceKey: "ns-b/svc-b"},
		}
		got := serviceKeysForDomain("geass-api.bukahou.com", ingress, mappings)
		if len(got) != 2 || !got["ns-a/svc-a"] || !got["ns-b/svc-b"] {
			t.Errorf("= %v，期望映射表里的两个", got)
		}
	})

	t.Run("传的本来就是 serviceKey 时原样返回", func(t *testing.T) {
		// V1 端点与旧链接仍可能直接传 serviceKey
		got := serviceKeysForDomain("geass-v3/geass-gateway", ingress, nil)
		if len(got) != 1 || !got["geass-v3/geass-gateway"] {
			t.Errorf("= %v", got)
		}
	})

	t.Run("完全查不到时退回原值 —— 宁可查空也不要 panic", func(t *testing.T) {
		got := serviceKeysForDomain("unknown.example.com", ingress, nil)
		if len(got) != 1 || !got["unknown.example.com"] {
			t.Errorf("= %v", got)
		}
	})

	t.Run("一个域名挂多个后端时全部返回", func(t *testing.T) {
		multi := []slomodel.IngressSLO{
			{ServiceKey: "ns/svc-1", DisplayName: "shared.example.com"},
			{ServiceKey: "ns/svc-2", DisplayName: "shared.example.com"},
		}
		got := serviceKeysForDomain("shared.example.com", multi, nil)
		if len(got) != 2 {
			t.Errorf("= %v，期望两个后端都在", got)
		}
	})
}

// 反查的数据源必须跟随查询窗口。
//
// SLOIngress 是固定 5 分钟窗口，低流量服务随时会从里面消失 ——
// 2026-08-24 实测凌晨时段它只剩 1 个服务（我正在访问的面板本身），
// 另外四个域名的历史图与延迟分布因此全部查不到数据。
// 查 24h 窗口就该用 24h 窗口的服务列表。
func TestIngressForLookup_PrefersWindow(t *testing.T) {
	windowed := []slomodel.IngressSLO{
		{ServiceKey: "geass-v3/geass-gateway", DisplayName: "geass-api.bukahou.com"},
		{ServiceKey: "akasha/akasha", DisplayName: "akasha.bukahou.com"},
	}
	recent := []slomodel.IngressSLO{
		{ServiceKey: "atlhyper/atlhyper-web", DisplayName: "bukahou.com"},
	}

	t.Run("窗口有数据时用窗口的", func(t *testing.T) {
		got := ingressForLookup(windowed, recent)
		if len(got) != 2 {
			t.Fatalf("= %d 条，期望用窗口里的 2 条", len(got))
		}
		keys := serviceKeysForDomain("geass-api.bukahou.com", got, nil)
		if !keys["geass-v3/geass-gateway"] {
			t.Errorf("低流量服务应能反查到: %v", keys)
		}
	})

	t.Run("窗口为空时退回 5 分钟列表", func(t *testing.T) {
		got := ingressForLookup(nil, recent)
		if len(got) != 1 || got[0].ServiceKey != "atlhyper/atlhyper-web" {
			t.Errorf("= %v", got)
		}
	})

	t.Run("都为空时返回空而非 panic", func(t *testing.T) {
		if got := ingressForLookup(nil, nil); len(got) != 0 {
			t.Errorf("= %v", got)
		}
	})
}

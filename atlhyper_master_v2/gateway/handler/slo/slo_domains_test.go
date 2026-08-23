package slo

import (
	"testing"

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

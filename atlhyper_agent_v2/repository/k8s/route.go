package k8s

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/common/logger"
)

var routeLog = logger.Module("RouteRepo")

// routeRepository 路由映射仓库实现
type routeRepository struct {
	client      sdk.K8sClient
	ingressRepo repository.IngressRepository
}

// NewRouteRepository 创建路由映射仓库
func NewRouteRepository(client sdk.K8sClient, ingressRepo repository.IngressRepository) repository.RouteRepository {
	return &routeRepository{client: client, ingressRepo: ingressRepo}
}

// GetServiceHostMap 汇总两类标准路由资源，产出 serviceKey → 域名。
//
// 优先 Gateway API HTTPRoute（本集群使用），未装则退回原生 Ingress。
// 两者并存时合并 —— 迁移过渡期可能同时存在。
func (r *routeRepository) GetServiceHostMap(ctx context.Context) (map[string]string, error) {
	var entries []RouteEntry

	httpRoutes, err := FetchHTTPRoutes(ctx, r.client.RestConfig())
	if err != nil {
		// 路由映射失败只影响 displayName 的美观，不该让整个 SLO 采集失败
		routeLog.Warn("HTTPRoute 查询失败，displayName 将退回 serviceKey", "err", err)
	}
	entries = append(entries, httpRoutes...)

	if r.ingressRepo != nil {
		ingresses, ierr := r.ingressRepo.List(ctx, "", model.ListOptions{})
		if ierr != nil {
			routeLog.Warn("Ingress 查询失败", "err", ierr)
		} else {
			entries = append(entries, IngressesToRouteEntries(ingresses)...)
		}
	}

	return BuildRouteMap(entries), nil
}

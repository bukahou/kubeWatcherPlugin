// atlhyper_master_v2/service/query/slo.go
// SLO 查询实现 — OTelSnapshot 直读模式
//
// 服务网格拓扑和服务详情从 OTelSnapshot 直读，不再依赖 SQLite 时序表。
// Handler（Gateway 层）通过 service.Query 接口调用。
package query

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
	slomodel "AtlHyper/model_v3/slo"
)

// determineMeshStatus 根据错误率和延迟判断服务状态
func determineMeshStatus(errRatePct, p99Ms float64) string {
	if errRatePct > 5 {
		return "critical"
	}
	if errRatePct > 1 || p99Ms > 500 {
		return "warning"
	}
	return "healthy"
}

// totalFromStatusCodes 从状态码计算总请求数
func totalFromStatusCodes(codes []slomodel.StatusCodeCount) int64 {
	var total int64
	for _, sc := range codes {
		total += sc.Count
	}
	return total
}

// ==================== SLO DB 查询方法 ====================

// GetSLOTargets 查询 SLO 目标配置（database → model 转换）
func (q *QueryService) GetSLOTargets(ctx context.Context, clusterID string) ([]model.SLOTargetResponse, error) {
	targets, err := q.sloRepo.GetTargets(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	result := make([]model.SLOTargetResponse, len(targets))
	for i, t := range targets {
		result[i] = toModelTargetResponse(t)
	}
	return result, nil
}

// GetSLORouteMappingByServiceKey 按 ServiceKey 查询路由映射
func (q *QueryService) GetSLORouteMappingByServiceKey(ctx context.Context, clusterID, serviceKey string) (*model.SLORouteMapping, error) {
	m, err := q.sloRepo.GetRouteMappingByServiceKey(ctx, clusterID, serviceKey)
	if err != nil {
		return nil, err
	}
	return toModelRouteMapping(m), nil
}

// GetSLORouteMappingsByDomain 按域名查询路由映射列表
func (q *QueryService) GetSLORouteMappingsByDomain(ctx context.Context, clusterID, domain string) ([]*model.SLORouteMapping, error) {
	mappings, err := q.sloRepo.GetRouteMappingsByDomain(ctx, clusterID, domain)
	if err != nil {
		return nil, err
	}
	return toModelRouteMappings(mappings), nil
}

// GetSLOAllDomains 查询所有域名
func (q *QueryService) GetSLOAllDomains(ctx context.Context, clusterID string) ([]string, error) {
	return q.sloRepo.GetAllDomains(ctx, clusterID)
}

// ==================== database → model 转换函数 ====================

// toModelTargetResponse 将 database.SLOTarget 转换为 model.SLOTargetResponse
func toModelTargetResponse(src *database.SLOTarget) model.SLOTargetResponse {
	return model.SLOTargetResponse{
		ID:                 src.ID,
		ClusterID:          src.ClusterID,
		Host:               src.Host,
		WindowDays:         src.WindowDays,
		AvailabilityTarget: src.AvailabilityTarget,
		P95LatencyTarget:   src.P95LatencyTarget,
		CreatedAt:          src.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:          src.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// toModelRouteMapping 将 database.SLORouteMapping 转换为 model.SLORouteMapping
func toModelRouteMapping(src *database.SLORouteMapping) *model.SLORouteMapping {
	if src == nil {
		return nil
	}
	return &model.SLORouteMapping{
		Domain:      src.Domain,
		PathPrefix:  src.PathPrefix,
		IngressName: src.IngressName,
		Namespace:   src.Namespace,
		TLS:         src.TLS,
		ServiceKey:  src.ServiceKey,
		ServiceName: src.ServiceName,
		ServicePort: src.ServicePort,
	}
}

// toModelRouteMappings 批量转换
func toModelRouteMappings(src []*database.SLORouteMapping) []*model.SLORouteMapping {
	if src == nil {
		return []*model.SLORouteMapping{}
	}
	result := make([]*model.SLORouteMapping, len(src))
	for i := range src {
		result[i] = toModelRouteMapping(src[i])
	}
	return result
}

// getTimeStart 根据时间范围计算起始时间
func getTimeStart(now time.Time, timeRange string) time.Time {
	switch timeRange {
	case "1h":
		return now.Add(-time.Hour)
	case "6h":
		return now.Add(-6 * time.Hour)
	case "24h", "1d":
		return now.Add(-24 * time.Hour)
	case "7d":
		return now.Add(-7 * 24 * time.Hour)
	case "30d":
		return now.Add(-30 * 24 * time.Hour)
	default:
		return now.Add(-24 * time.Hour)
	}
}

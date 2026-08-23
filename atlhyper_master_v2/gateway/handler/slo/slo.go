// atlhyper_master_v2/gateway/handler/slo.go
// SLO API Handler — 共用结构体和辅助函数
//
// 各职责分布在:
//
//	slo_domains.go   — Domains / DomainsV2 / DomainDetail / DomainHistory
//	slo_targets.go   — Targets / StatusHistory
//	slo_latency.go   — LatencyDistribution
//	slo_mesh.go      — MeshTopology / ServiceDetail (独立 Handler)
package slo

import (
	"context"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/service"
	"AtlHyper/common/logger"
	model_v3 "AtlHyper/model_v3"
)

// SLO 状态常量（来源: model_v3.HealthStatus，避免裸字符串硬编码）
const (
	statusHealthy  = string(model_v3.HealthStatusHealthy)
	statusWarning  = string(model_v3.HealthStatusWarning)
	statusCritical = string(model_v3.HealthStatusCritical)
	statusUnknown  = string(model_v3.HealthStatusUnknown)
)

var sloLog = logger.Module("SLO-Handler")

// SLOHandler SLO API Handler
type SLOHandler struct {
	querySvc service.Query
	opsSvc   service.Ops
}

// NewSLOHandler 创建 SLOHandler
func NewSLOHandler(querySvc service.Query, opsSvc service.Ops) *SLOHandler {
	return &SLOHandler{
		querySvc: querySvc,
		opsSvc:   opsSvc,
	}
}

// defaultClusterID 获取默认集群 ID
func (h *SLOHandler) defaultClusterID(_ context.Context) string {
	agents, err := h.querySvc.ListClusters(context.Background())
	if err == nil && len(agents) > 0 {
		return agents[0].ClusterID
	}
	return "default"
}

// buildTargetMap 构建目标配置 map
func buildTargetMap(targets []model.SLOTargetResponse) map[string]map[string]model.SLOTargetResponse {
	result := make(map[string]map[string]model.SLOTargetResponse)
	for _, t := range targets {
		if result[t.Host] == nil {
			result[t.Host] = make(map[string]model.SLOTargetResponse)
		}
		result[t.Host][t.TimeRange] = t
	}
	return result
}

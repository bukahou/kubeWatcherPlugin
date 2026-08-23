// atlhyper_master_v2/service/operations/slo.go
// SLO 写入服务 — 接收 model 类型请求，转换为 database 类型后写入
package operations

import (
	"context"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
)

// defaultSLOWindowDays SLO 默认滚动窗口。取 7 天对齐 ClickHouse 的 TTL ——
// 再长的窗口没有数据支撑，配了也只会显示空。
const defaultSLOWindowDays = 7

// SLOService SLO 写入服务
type SLOService struct {
	sloRepo database.SLORepository
}

// NewSLOService 创建 SLOService
func NewSLOService(sloRepo database.SLORepository) *SLOService {
	return &SLOService{sloRepo: sloRepo}
}

// UpsertSLOTarget 创建或更新 SLO 目标（model → database 转换）
func (s *SLOService) UpsertSLOTarget(ctx context.Context, req *model.UpdateSLOTargetRequest) error {
	if req.WindowDays <= 0 {
		req.WindowDays = defaultSLOWindowDays
	}
	target := &database.SLOTarget{
		ClusterID:          req.ClusterID,
		Host:               req.Host,
		WindowDays:         req.WindowDays,
		AvailabilityTarget: req.AvailabilityTarget,
		P95LatencyTarget:   req.P95LatencyTarget,
	}
	return s.sloRepo.UpsertTarget(ctx, target)
}

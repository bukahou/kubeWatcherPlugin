// Package ch ClickHouse 仓库实现
package ch

import (
	"context"
	"time"

	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/model_v3/apm"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/log"
	"AtlHyper/model_v3/metrics"
	"AtlHyper/model_v3/slo"
)

// dashboardRepository Dashboard 数据采集（组合委托现有 repos）
type dashboardRepository struct {
	metrics   repository.MetricsQueryRepository
	trace     repository.TraceQueryRepository
	slo       repository.SLOQueryRepository
	log       repository.LogQueryRepository
	freshness repository.FreshnessQueryRepository
}

// GetSignalFreshness 委托给新鲜度仓库。
// 挂在 Dashboard 门面上而不是单独注入 —— 这个门面的职责就是「快照需要的 OTel 数据」，
// 新鲜度属于同一职责，另开一个构造参数只会让 20 多个参数的签名更难读。
func (r *dashboardRepository) GetSignalFreshness(ctx context.Context) (*cluster.SignalFreshness, error) {
	if r.freshness == nil {
		return nil, nil
	}
	return r.freshness.GetSignalFreshness(ctx)
}

// NewDashboardRepository 创建 Dashboard 仓库
func NewDashboardRepository(
	m repository.MetricsQueryRepository,
	t repository.TraceQueryRepository,
	s repository.SLOQueryRepository,
	l repository.LogQueryRepository,
	f repository.FreshnessQueryRepository,
) repository.OTelDashboardRepository {
	return &dashboardRepository{metrics: m, trace: t, slo: s, log: l, freshness: f}
}

func (r *dashboardRepository) GetMetricsSummary(ctx context.Context) (*metrics.Summary, error) {
	return r.metrics.GetMetricsSummary(ctx)
}

func (r *dashboardRepository) ListAllNodeMetrics(ctx context.Context) ([]metrics.NodeMetrics, error) {
	return r.metrics.ListAllNodeMetrics(ctx)
}

func (r *dashboardRepository) ListAPMServices(ctx context.Context) ([]apm.APMService, error) {
	return r.trace.ListServices(ctx, 15*time.Minute, "", "")
}

func (r *dashboardRepository) GetAPMTopology(ctx context.Context) (*apm.Topology, error) {
	return r.trace.GetTopology(ctx, 15*time.Minute, "", "")
}

func (r *dashboardRepository) GetSLOSummary(ctx context.Context) (*slo.SLOSummary, error) {
	return r.slo.GetSLOSummary(ctx)
}

func (r *dashboardRepository) ListIngressSLO(ctx context.Context, since time.Duration) ([]slo.IngressSLO, error) {
	return r.slo.ListIngressSLO(ctx, since)
}

func (r *dashboardRepository) ListIngressSLOPrevious(ctx context.Context, since time.Duration) ([]slo.IngressSLO, error) {
	return r.slo.ListIngressSLOPrevious(ctx, since)
}

func (r *dashboardRepository) GetIngressSLOHistory(ctx context.Context, since, bucket time.Duration) ([]slo.SLOHistoryPoint, error) {
	return r.slo.GetIngressSLOHistory(ctx, since, bucket)
}

func (r *dashboardRepository) ListAPMOperations(ctx context.Context) ([]apm.OperationStats, error) {
	if r.trace == nil {
		return nil, nil
	}
	return r.trace.ListOperations(ctx, 15*time.Minute, "", "")
}

func (r *dashboardRepository) ListRecentTraces(ctx context.Context, limit int) ([]apm.TraceSummary, error) {
	if r.trace == nil {
		return nil, nil
	}
	return r.trace.ListTraces(ctx, "", "", 0, limit, 15*time.Minute, "", "", "", "", "")
}

func (r *dashboardRepository) GetLogsSummary(ctx context.Context) (*log.Summary, error) {
	if r.log == nil {
		return nil, nil
	}
	return r.log.GetSummary(ctx)
}

func (r *dashboardRepository) ListRecentLogs(ctx context.Context, limit int) ([]log.Entry, error) {
	if r.log == nil {
		return nil, nil
	}
	return r.log.ListRecentEntries(ctx, limit)
}

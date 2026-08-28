package mock

import (
	"context"
	"time"

	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/model_v3/apm"
	"AtlHyper/model_v3/log"
	"AtlHyper/model_v3/metrics"
	"AtlHyper/model_v3/slo"
)

// OTelSummaryRepository mock
type OTelSummaryRepository struct {
	GetAPMSummaryFn     func(ctx context.Context) (totalServices, healthyServices int, totalRPS, avgSuccessRate, avgP99Ms float64, err error)
	GetSLOSummaryFn     func(ctx context.Context) (int, float64, error)
	GetMetricsSummaryFn func(ctx context.Context) (monitoredNodes int, avgCPUPct, avgMemPct, maxCPUPct, maxMemPct float64, err error)
}

func (m *OTelSummaryRepository) GetAPMSummary(ctx context.Context) (totalServices, healthyServices int, totalRPS, avgSuccessRate, avgP99Ms float64, err error) {
	if m.GetAPMSummaryFn != nil {
		return m.GetAPMSummaryFn(ctx)
	}
	return 0, 0, 0, 0, 0, nil
}

func (m *OTelSummaryRepository) GetSLOSummary(ctx context.Context) (ingressServices int, ingressAvgRPS float64, err error) {
	if m.GetSLOSummaryFn != nil {
		return m.GetSLOSummaryFn(ctx)
	}
	return 0, 0, nil
}

func (m *OTelSummaryRepository) GetMetricsSummary(ctx context.Context) (monitoredNodes int, avgCPUPct, avgMemPct, maxCPUPct, maxMemPct float64, err error) {
	if m.GetMetricsSummaryFn != nil {
		return m.GetMetricsSummaryFn(ctx)
	}
	return 0, 0, 0, 0, 0, nil
}

// TraceQueryRepository mock
type TraceQueryRepository struct {
	ListTracesFn           func(ctx context.Context, service, operation string, minDurationMs float64, limit int, since time.Duration, sort string, startTime, endTime string, statusCode, method string) ([]apm.TraceSummary, error)
	GetTraceDetailFn       func(ctx context.Context, traceID string) (*apm.TraceDetail, error)
	ListServicesFn         func(ctx context.Context, since time.Duration, startTime, endTime string) ([]apm.APMService, error)
	GetTopologyFn          func(ctx context.Context, since time.Duration, startTime, endTime string) (*apm.Topology, error)
	ListOperationsFn       func(ctx context.Context, since time.Duration, startTime, endTime string) ([]apm.OperationStats, error)
	GetHTTPStatsFn         func(ctx context.Context, service string, since time.Duration, startTime, endTime string) ([]apm.HTTPStats, error)
	GetDBStatsFn           func(ctx context.Context, service string, since time.Duration, startTime, endTime string) ([]apm.DBOperationStats, error)
	GetServiceTimeSeriesFn func(ctx context.Context, service string, since time.Duration) ([]apm.TimePoint, error)
	GetTraceCorrelationsFn func(ctx context.Context, service, operation, mode string, since time.Duration, startTime, endTime string) (*apm.CorrelationResult, error)
}

func (m *TraceQueryRepository) ListTraces(ctx context.Context, service, operation string, minDurationMs float64, limit int, since time.Duration, sort string, startTime, endTime string, statusCode, method string) ([]apm.TraceSummary, error) {
	if m.ListTracesFn != nil {
		return m.ListTracesFn(ctx, service, operation, minDurationMs, limit, since, sort, startTime, endTime, statusCode, method)
	}
	return []apm.TraceSummary{}, nil
}

func (m *TraceQueryRepository) GetTraceDetail(ctx context.Context, traceID string) (*apm.TraceDetail, error) {
	if m.GetTraceDetailFn != nil {
		return m.GetTraceDetailFn(ctx, traceID)
	}
	return nil, nil
}

func (m *TraceQueryRepository) ListServices(ctx context.Context, since time.Duration, startTime, endTime string) ([]apm.APMService, error) {
	if m.ListServicesFn != nil {
		return m.ListServicesFn(ctx, since, startTime, endTime)
	}
	return []apm.APMService{}, nil
}

func (m *TraceQueryRepository) GetTopology(ctx context.Context, since time.Duration, startTime, endTime string) (*apm.Topology, error) {
	if m.GetTopologyFn != nil {
		return m.GetTopologyFn(ctx, since, startTime, endTime)
	}
	return nil, nil
}

func (m *TraceQueryRepository) ListOperations(ctx context.Context, since time.Duration, startTime, endTime string) ([]apm.OperationStats, error) {
	if m.ListOperationsFn != nil {
		return m.ListOperationsFn(ctx, since, startTime, endTime)
	}
	return []apm.OperationStats{}, nil
}

func (m *TraceQueryRepository) GetHTTPStats(ctx context.Context, service string, since time.Duration, startTime, endTime string) ([]apm.HTTPStats, error) {
	if m.GetHTTPStatsFn != nil {
		return m.GetHTTPStatsFn(ctx, service, since, startTime, endTime)
	}
	return []apm.HTTPStats{}, nil
}

func (m *TraceQueryRepository) GetDBStats(ctx context.Context, service string, since time.Duration, startTime, endTime string) ([]apm.DBOperationStats, error) {
	if m.GetDBStatsFn != nil {
		return m.GetDBStatsFn(ctx, service, since, startTime, endTime)
	}
	return []apm.DBOperationStats{}, nil
}

func (m *TraceQueryRepository) GetServiceTimeSeries(ctx context.Context, service string, since time.Duration) ([]apm.TimePoint, error) {
	if m.GetServiceTimeSeriesFn != nil {
		return m.GetServiceTimeSeriesFn(ctx, service, since)
	}
	return []apm.TimePoint{}, nil
}

func (m *TraceQueryRepository) GetTraceCorrelations(ctx context.Context, service, operation, mode string, since time.Duration, startTime, endTime string) (*apm.CorrelationResult, error) {
	if m.GetTraceCorrelationsFn != nil {
		return m.GetTraceCorrelationsFn(ctx, service, operation, mode, since, startTime, endTime)
	}
	return &apm.CorrelationResult{Items: []apm.CorrelationItem{}}, nil
}

// LogQueryRepository mock
type LogQueryRepository struct {
	QueryLogsFn         func(ctx context.Context, opts repository.LogQueryOptions) (*log.QueryResult, error)
	QueryHistogramFn    func(ctx context.Context, opts repository.LogQueryOptions) (*log.HistogramResult, error)
	GetSummaryFn        func(ctx context.Context) (*log.Summary, error)
	ListRecentEntriesFn func(ctx context.Context, limit int) ([]log.Entry, error)
}

func (m *LogQueryRepository) QueryLogs(ctx context.Context, opts repository.LogQueryOptions) (*log.QueryResult, error) {
	if m.QueryLogsFn != nil {
		return m.QueryLogsFn(ctx, opts)
	}
	return &log.QueryResult{Logs: []log.Entry{}}, nil
}

func (m *LogQueryRepository) QueryHistogram(ctx context.Context, opts repository.LogQueryOptions) (*log.HistogramResult, error) {
	if m.QueryHistogramFn != nil {
		return m.QueryHistogramFn(ctx, opts)
	}
	return nil, nil
}

func (m *LogQueryRepository) GetSummary(ctx context.Context) (*log.Summary, error) {
	if m.GetSummaryFn != nil {
		return m.GetSummaryFn(ctx)
	}
	return nil, nil
}

func (m *LogQueryRepository) ListRecentEntries(ctx context.Context, limit int) ([]log.Entry, error) {
	if m.ListRecentEntriesFn != nil {
		return m.ListRecentEntriesFn(ctx, limit)
	}
	return []log.Entry{}, nil
}

// MetricsQueryRepository mock
type MetricsQueryRepository struct {
	ListAllNodeMetricsFn    func(ctx context.Context) ([]metrics.NodeMetrics, error)
	GetNodeMetricsFn        func(ctx context.Context, nodeName string) (*metrics.NodeMetrics, error)
	GetNodeMetricsSeriesFn  func(ctx context.Context, nodeName string, metric string, since time.Duration) ([]metrics.Point, error)
	GetMetricsSummaryFn     func(ctx context.Context) (*metrics.Summary, error)
	GetNodeMetricsHistoryFn func(ctx context.Context, nodeName string, since time.Duration) (map[string][]metrics.Point, error)
}

func (m *MetricsQueryRepository) ListAllNodeMetrics(ctx context.Context) ([]metrics.NodeMetrics, error) {
	if m.ListAllNodeMetricsFn != nil {
		return m.ListAllNodeMetricsFn(ctx)
	}
	return []metrics.NodeMetrics{}, nil
}

func (m *MetricsQueryRepository) GetNodeMetrics(ctx context.Context, nodeName string) (*metrics.NodeMetrics, error) {
	if m.GetNodeMetricsFn != nil {
		return m.GetNodeMetricsFn(ctx, nodeName)
	}
	return nil, nil
}

func (m *MetricsQueryRepository) GetNodeMetricsSeries(ctx context.Context, nodeName string, metric string, since time.Duration) ([]metrics.Point, error) {
	if m.GetNodeMetricsSeriesFn != nil {
		return m.GetNodeMetricsSeriesFn(ctx, nodeName, metric, since)
	}
	return []metrics.Point{}, nil
}

func (m *MetricsQueryRepository) GetMetricsSummary(ctx context.Context) (*metrics.Summary, error) {
	if m.GetMetricsSummaryFn != nil {
		return m.GetMetricsSummaryFn(ctx)
	}
	return nil, nil
}

func (m *MetricsQueryRepository) GetNodeMetricsHistory(ctx context.Context, nodeName string, since time.Duration) (map[string][]metrics.Point, error) {
	if m.GetNodeMetricsHistoryFn != nil {
		return m.GetNodeMetricsHistoryFn(ctx, nodeName, since)
	}
	return map[string][]metrics.Point{}, nil
}

// SLOQueryRepository mock
type SLOQueryRepository struct {
	ListIngressSLOFn         func(ctx context.Context, since time.Duration) ([]slo.IngressSLO, error)
	ListIngressSLOPreviousFn func(ctx context.Context, since time.Duration) ([]slo.IngressSLO, error)
	GetIngressSLOHistoryFn   func(ctx context.Context, since, bucket time.Duration) ([]slo.SLOHistoryPoint, error)
	GetSLOTimeSeriesFn       func(ctx context.Context, name string, since time.Duration) (*slo.TimeSeries, error)
	GetSLOSummaryFn          func(ctx context.Context) (*slo.SLOSummary, error)
}

func (m *SLOQueryRepository) ListIngressSLO(ctx context.Context, since time.Duration) ([]slo.IngressSLO, error) {
	if m.ListIngressSLOFn != nil {
		return m.ListIngressSLOFn(ctx, since)
	}
	return []slo.IngressSLO{}, nil
}

func (m *SLOQueryRepository) ListIngressSLOPrevious(ctx context.Context, since time.Duration) ([]slo.IngressSLO, error) {
	if m.ListIngressSLOPreviousFn != nil {
		return m.ListIngressSLOPreviousFn(ctx, since)
	}
	return []slo.IngressSLO{}, nil
}

func (m *SLOQueryRepository) GetIngressSLOHistory(ctx context.Context, since, bucket time.Duration) ([]slo.SLOHistoryPoint, error) {
	if m.GetIngressSLOHistoryFn != nil {
		return m.GetIngressSLOHistoryFn(ctx, since, bucket)
	}
	return []slo.SLOHistoryPoint{}, nil
}

func (m *SLOQueryRepository) GetSLOTimeSeries(ctx context.Context, name string, since time.Duration) (*slo.TimeSeries, error) {
	if m.GetSLOTimeSeriesFn != nil {
		return m.GetSLOTimeSeriesFn(ctx, name, since)
	}
	return nil, nil
}

func (m *SLOQueryRepository) GetSLOSummary(ctx context.Context) (*slo.SLOSummary, error) {
	if m.GetSLOSummaryFn != nil {
		return m.GetSLOSummaryFn(ctx)
	}
	return nil, nil
}

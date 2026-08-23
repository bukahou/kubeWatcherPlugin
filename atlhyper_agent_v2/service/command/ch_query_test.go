package command

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/testutil/mock"
	"AtlHyper/model_v3/apm"
	"AtlHyper/model_v3/command"
	"AtlHyper/model_v3/log"
	"AtlHyper/model_v3/metrics"
	"AtlHyper/model_v3/slo"
)

// newTestService 创建包含 ClickHouse mock 的测试服务
func newTestService(
	traceRepo repository.TraceQueryRepository,
	logRepo repository.LogQueryRepository,
	metricsRepo repository.MetricsQueryRepository,
	sloRepo repository.SLOQueryRepository,
) *commandService {
	return &commandService{
		podRepo:          &mock.PodRepository{},
		genericRepo:      &mock.GenericRepository{},
		traceQueryRepo:   traceRepo,
		logQueryRepo:     logRepo,
		metricsQueryRepo: metricsRepo,
		sloQueryRepo:     sloRepo,
	}
}

// =============================================================================
// TestExecute_QueryTraces
// =============================================================================

func TestExecute_QueryTraces_ListTraces(t *testing.T) {
	traceRepo := &mock.TraceQueryRepository{
		ListTracesFn: func(ctx context.Context, service, operation string, minDurationMs float64, limit int, since time.Duration, sort string, startTime, endTime string, statusCode, method string) ([]apm.TraceSummary, error) {
			return []apm.TraceSummary{
				{TraceId: "abc123", RootService: "api", SpanCount: 5},
			}, nil
		},
	}

	svc := newTestService(traceRepo, nil, nil, nil)
	cmd := &command.Command{
		ID:     "cmd-traces-1",
		Action: command.ActionQueryTraces,
		Params: map[string]any{"sub_action": "list_traces", "service": "api"},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "abc123") {
		t.Errorf("expected output to contain trace ID, got: %s", result.Output)
	}
}

func TestExecute_QueryTraces_ListServices(t *testing.T) {
	traceRepo := &mock.TraceQueryRepository{
		ListServicesFn: func(ctx context.Context, since time.Duration, startTime, endTime string) ([]apm.APMService, error) {
			return []apm.APMService{{Name: "frontend", RPS: 10.5}}, nil
		},
	}

	svc := newTestService(traceRepo, nil, nil, nil)
	cmd := &command.Command{
		ID:     "cmd-traces-2",
		Action: command.ActionQueryTraces,
		Params: map[string]any{"sub_action": "list_services"},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "frontend") {
		t.Errorf("expected output to contain service name, got: %s", result.Output)
	}
}

func TestExecute_QueryTraces_NilRepo(t *testing.T) {
	svc := newTestService(nil, nil, nil, nil)
	cmd := &command.Command{
		ID:     "cmd-traces-nil",
		Action: command.ActionQueryTraces,
		Params: map[string]any{},
	}

	result := svc.Execute(context.Background(), cmd)

	if result.Success {
		t.Fatal("expected failure when repo is nil")
	}
	if !strings.Contains(result.Error, "ClickHouse not configured") {
		t.Errorf("expected 'ClickHouse not configured', got: %s", result.Error)
	}
}

// =============================================================================
// TestExecute_QueryTraceDetail
// =============================================================================

func TestExecute_QueryTraceDetail_Success(t *testing.T) {
	traceRepo := &mock.TraceQueryRepository{
		GetTraceDetailFn: func(ctx context.Context, traceID string) (*apm.TraceDetail, error) {
			return &apm.TraceDetail{
				TraceId:    traceID,
				DurationMs: 42.5,
				SpanCount:  3,
			}, nil
		},
	}

	svc := newTestService(traceRepo, nil, nil, nil)
	cmd := &command.Command{
		ID:     "cmd-trace-detail-1",
		Action: command.ActionQueryTraceDetail,
		Params: map[string]any{"trace_id": "abc123"},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "abc123") {
		t.Errorf("expected output to contain trace ID, got: %s", result.Output)
	}
}

func TestExecute_QueryTraceDetail_MissingID(t *testing.T) {
	traceRepo := &mock.TraceQueryRepository{}
	svc := newTestService(traceRepo, nil, nil, nil)
	cmd := &command.Command{
		ID:     "cmd-trace-detail-2",
		Action: command.ActionQueryTraceDetail,
		Params: map[string]any{},
	}

	result := svc.Execute(context.Background(), cmd)

	if result.Success {
		t.Fatal("expected failure for missing trace_id")
	}
	if !strings.Contains(result.Error, "trace_id is required") {
		t.Errorf("expected 'trace_id is required', got: %s", result.Error)
	}
}

// =============================================================================
// TestExecute_QueryLogs
// =============================================================================

func TestExecute_QueryLogs_Success(t *testing.T) {
	logRepo := &mock.LogQueryRepository{
		QueryLogsFn: func(ctx context.Context, opts repository.LogQueryOptions) (*log.QueryResult, error) {
			if opts.Service != "api" {
				t.Errorf("expected service 'api', got '%s'", opts.Service)
			}
			if opts.Level != "ERROR" {
				t.Errorf("expected level 'ERROR', got '%s'", opts.Level)
			}
			return &log.QueryResult{
				Logs:  []log.Entry{{Body: "something failed", ServiceName: "api"}},
				Total: 1,
			}, nil
		},
	}

	svc := newTestService(nil, logRepo, nil, nil)
	cmd := &command.Command{
		ID:     "cmd-logs-ch-1",
		Action: command.ActionQueryLogs,
		Params: map[string]any{
			"service": "api",
			"level":   "ERROR",
			"limit":   float64(10),
		},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "something failed") {
		t.Errorf("expected output to contain log body, got: %s", result.Output)
	}
}

// =============================================================================
// TestExecute_QueryMetrics
// =============================================================================

func TestExecute_QueryMetrics_ListAll(t *testing.T) {
	metricsRepo := &mock.MetricsQueryRepository{
		ListAllNodeMetricsFn: func(ctx context.Context) ([]metrics.NodeMetrics, error) {
			return []metrics.NodeMetrics{
				{NodeName: "node-1", CPU: metrics.NodeCPU{UsagePct: 45.2}},
			}, nil
		},
	}

	svc := newTestService(nil, nil, metricsRepo, nil)
	cmd := &command.Command{
		ID:     "cmd-metrics-1",
		Action: command.ActionQueryMetrics,
		Params: map[string]any{"sub_action": "list_all"},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "node-1") {
		t.Errorf("expected output to contain node name, got: %s", result.Output)
	}
}

func TestExecute_QueryMetrics_GetNode(t *testing.T) {
	metricsRepo := &mock.MetricsQueryRepository{
		GetNodeMetricsFn: func(ctx context.Context, nodeName string) (*metrics.NodeMetrics, error) {
			if nodeName != "worker-1" {
				return nil, fmt.Errorf("unexpected node: %s", nodeName)
			}
			return &metrics.NodeMetrics{NodeName: "worker-1"}, nil
		},
	}

	svc := newTestService(nil, nil, metricsRepo, nil)
	cmd := &command.Command{
		ID:     "cmd-metrics-2",
		Action: command.ActionQueryMetrics,
		Params: map[string]any{"sub_action": "get_node", "node_name": "worker-1"},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
}

func TestExecute_QueryMetrics_GetSeries(t *testing.T) {
	metricsRepo := &mock.MetricsQueryRepository{
		GetNodeMetricsSeriesFn: func(ctx context.Context, nodeName string, metric string, since time.Duration) ([]metrics.Point, error) {
			return []metrics.Point{{Value: 42.0}}, nil
		},
	}

	svc := newTestService(nil, nil, metricsRepo, nil)
	cmd := &command.Command{
		ID:     "cmd-metrics-3",
		Action: command.ActionQueryMetrics,
		Params: map[string]any{
			"sub_action": "get_series",
			"node_name":  "worker-1",
			"metric":     "node_load1",
			"since":      "30m",
		},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
}

func TestExecute_QueryMetrics_MissingNodeName(t *testing.T) {
	metricsRepo := &mock.MetricsQueryRepository{}
	svc := newTestService(nil, nil, metricsRepo, nil)
	cmd := &command.Command{
		ID:     "cmd-metrics-4",
		Action: command.ActionQueryMetrics,
		Params: map[string]any{"sub_action": "get_node"},
	}

	result := svc.Execute(context.Background(), cmd)

	if result.Success {
		t.Fatal("expected failure for missing node_name")
	}
}

// =============================================================================
// TestExecute_QuerySLO
// =============================================================================

func TestExecute_QuerySLO_ListIngress(t *testing.T) {
	sloRepo := &mock.SLOQueryRepository{
		ListIngressSLOFn: func(ctx context.Context, since time.Duration) ([]slo.IngressSLO, error) {
			return []slo.IngressSLO{
				{ServiceKey: "web@docker", RPS: 100},
			}, nil
		},
	}

	svc := newTestService(nil, nil, nil, sloRepo)
	cmd := &command.Command{
		ID:     "cmd-slo-1",
		Action: command.ActionQuerySLO,
		Params: map[string]any{"sub_action": "list_ingress"},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "web@docker") {
		t.Errorf("expected output to contain service key, got: %s", result.Output)
	}
}

func TestExecute_QuerySLO_GetSummary(t *testing.T) {
	sloRepo := &mock.SLOQueryRepository{
		GetSLOSummaryFn: func(ctx context.Context) (*slo.SLOSummary, error) {
			return &slo.SLOSummary{TotalServices: 5, HealthyServices: 4}, nil
		},
	}

	svc := newTestService(nil, nil, nil, sloRepo)
	cmd := &command.Command{
		ID:     "cmd-slo-3",
		Action: command.ActionQuerySLO,
		Params: map[string]any{"sub_action": "get_summary"},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
}

// =============================================================================
// TestParamHelpers
// =============================================================================

func TestGetStringParam(t *testing.T) {
	params := map[string]any{"key": "value"}
	if getStringParam(params, "key") != "value" {
		t.Error("expected 'value'")
	}
	if getStringParam(params, "missing") != "" {
		t.Error("expected empty string for missing key")
	}
	if getStringParam(nil, "key") != "" {
		t.Error("expected empty string for nil params")
	}
}

func TestGetIntParam(t *testing.T) {
	params := map[string]any{"count": float64(42)}
	if getIntParam(params, "count", 10) != 42 {
		t.Error("expected 42")
	}
	if getIntParam(params, "missing", 10) != 10 {
		t.Error("expected default 10")
	}
}

func TestExecute_QueryTraces_HTTPStats(t *testing.T) {
	traceRepo := &mock.TraceQueryRepository{
		GetHTTPStatsFn: func(ctx context.Context, service string, since time.Duration, startTime, endTime string) ([]apm.HTTPStats, error) {
			if service != "api" {
				t.Errorf("expected service 'api', got '%s'", service)
			}
			return []apm.HTTPStats{
				{StatusCode: 200, Method: "GET", Count: 100},
				{StatusCode: 404, Method: "GET", Count: 5},
			}, nil
		},
	}

	svc := newTestService(traceRepo, nil, nil, nil)
	cmd := &command.Command{
		ID:     "cmd-traces-http-stats",
		Action: command.ActionQueryTraces,
		Params: map[string]any{"sub_action": "http_stats", "service": "api"},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "200") {
		t.Errorf("expected output to contain status code 200, got: %s", result.Output)
	}
}

func TestExecute_QueryTraces_HTTPStats_MissingService(t *testing.T) {
	traceRepo := &mock.TraceQueryRepository{}
	svc := newTestService(traceRepo, nil, nil, nil)
	cmd := &command.Command{
		ID:     "cmd-traces-http-stats-2",
		Action: command.ActionQueryTraces,
		Params: map[string]any{"sub_action": "http_stats"},
	}

	result := svc.Execute(context.Background(), cmd)

	if result.Success {
		t.Fatal("expected failure for missing service")
	}
	if !strings.Contains(result.Error, "service is required") {
		t.Errorf("expected 'service is required', got: %s", result.Error)
	}
}

func TestExecute_QueryTraces_DBStats(t *testing.T) {
	traceRepo := &mock.TraceQueryRepository{
		GetDBStatsFn: func(ctx context.Context, service string, since time.Duration, startTime, endTime string) ([]apm.DBOperationStats, error) {
			return []apm.DBOperationStats{
				{DBSystem: "mysql", DBName: "mydb", Operation: "SELECT", CallCount: 50},
			}, nil
		},
	}

	svc := newTestService(traceRepo, nil, nil, nil)
	cmd := &command.Command{
		ID:     "cmd-traces-db-stats",
		Action: command.ActionQueryTraces,
		Params: map[string]any{"sub_action": "db_stats", "service": "api"},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "mysql") {
		t.Errorf("expected output to contain 'mysql', got: %s", result.Output)
	}
}

func TestGetDurationParam(t *testing.T) {
	params := map[string]any{"since": "5m"}
	d := getDurationParam(params, "since", time.Minute)
	if d != 5*time.Minute {
		t.Errorf("expected 5m, got %v", d)
	}

	// Numeric seconds
	params2 := map[string]any{"since": float64(300)}
	d2 := getDurationParam(params2, "since", time.Minute)
	if d2 != 300*time.Second {
		t.Errorf("expected 300s, got %v", d2)
	}

	// Default
	d3 := getDurationParam(nil, "since", 10*time.Minute)
	if d3 != 10*time.Minute {
		t.Errorf("expected 10m default, got %v", d3)
	}
}

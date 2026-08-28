package command

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/model_v3/command"
)

// =============================================================================
// ClickHouse 查询指令处理
// =============================================================================

// handleQueryTraces 处理 Trace 查询指令
func (s *commandService) handleQueryTraces(ctx context.Context, cmd *command.Command) (any, error) {
	if s.traceQueryRepo == nil {
		return nil, fmt.Errorf("ClickHouse not configured")
	}

	subAction := getStringParam(cmd.Params, "sub_action")
	startTime := getStringParam(cmd.Params, "start_time")
	endTime := getStringParam(cmd.Params, "end_time")

	switch subAction {
	case "list_traces", "":
		service := getStringParam(cmd.Params, "service")
		operation := getStringParam(cmd.Params, "operation")
		minDuration := getFloat64Param(cmd.Params, "min_duration_ms")
		limit := getIntParam(cmd.Params, "limit", 50)
		since := getDurationParam(cmd.Params, "since", 5*time.Minute)
		sortBy := getStringParam(cmd.Params, "sort")
		statusCode := getStringParam(cmd.Params, "status_code")
		method := getStringParam(cmd.Params, "method")
		return s.traceQueryRepo.ListTraces(ctx, service, operation, minDuration, limit, since, sortBy, startTime, endTime, statusCode, method)

	case "list_services":
		since := getDurationParam(cmd.Params, "since", 15*time.Minute)
		return s.traceQueryRepo.ListServices(ctx, since, startTime, endTime)

	case "get_correlations":
		service := getStringParam(cmd.Params, "service")
		operation := getStringParam(cmd.Params, "operation")
		mode := getStringParam(cmd.Params, "mode")
		if mode == "" {
			mode = "failure"
		}
		since := getDurationParam(cmd.Params, "since", 60*time.Minute)
		return s.traceQueryRepo.GetTraceCorrelations(ctx, service, operation, mode, since, startTime, endTime)

	case "get_topology":
		since := getDurationParam(cmd.Params, "since", 15*time.Minute)
		return s.traceQueryRepo.GetTopology(ctx, since, startTime, endTime)

	case "list_operations":
		since := getDurationParam(cmd.Params, "since", 15*time.Minute)
		return s.traceQueryRepo.ListOperations(ctx, since, startTime, endTime)

	case "http_stats":
		service := getStringParam(cmd.Params, "service")
		if service == "" {
			return nil, fmt.Errorf("service is required for http_stats")
		}
		since := getDurationParam(cmd.Params, "since", 15*time.Minute)
		return s.traceQueryRepo.GetHTTPStats(ctx, service, since, startTime, endTime)

	case "db_stats":
		service := getStringParam(cmd.Params, "service")
		if service == "" {
			return nil, fmt.Errorf("service is required for db_stats")
		}
		since := getDurationParam(cmd.Params, "since", 15*time.Minute)
		return s.traceQueryRepo.GetDBStats(ctx, service, since, startTime, endTime)

	case "service_series":
		service := getStringParam(cmd.Params, "service")
		if service == "" {
			return nil, fmt.Errorf("service is required for service_series")
		}
		since := getDurationParam(cmd.Params, "since", 60*time.Minute)
		points, err := s.traceQueryRepo.GetServiceTimeSeries(ctx, service, since)
		if err != nil {
			return nil, err
		}
		// 包装为与 Concentrator 路径一致的格式: { service, points }
		return map[string]interface{}{
			"service": service,
			"points":  points,
		}, nil

	default:
		return nil, fmt.Errorf("unknown trace sub_action: %s", subAction)
	}
}

// handleQueryTraceDetail 处理 Trace 详情查询指令
func (s *commandService) handleQueryTraceDetail(ctx context.Context, cmd *command.Command) (any, error) {
	if s.traceQueryRepo == nil {
		return nil, fmt.Errorf("ClickHouse not configured")
	}

	traceID := getStringParam(cmd.Params, "trace_id")
	if traceID == "" {
		return nil, fmt.Errorf("trace_id is required")
	}
	return s.traceQueryRepo.GetTraceDetail(ctx, traceID)
}

// handleQueryLogs 处理日志查询指令（通过 sub_action 区分列表/直方图）
func (s *commandService) handleQueryLogs(ctx context.Context, cmd *command.Command) (any, error) {
	if s.logQueryRepo == nil {
		return nil, fmt.Errorf("ClickHouse not configured")
	}

	subAction := getStringParam(cmd.Params, "sub_action")
	if subAction == "histogram" {
		return s.handleQueryLogHistogram(ctx, cmd)
	}

	opts := repository.LogQueryOptions{
		Query:     getStringParam(cmd.Params, "query"),
		Service:   getStringParam(cmd.Params, "service"),
		Level:     getStringParam(cmd.Params, "level"),
		Scope:     getStringParam(cmd.Params, "scope"),
		TraceId:   getStringParam(cmd.Params, "trace_id"),
		SpanId:    getStringParam(cmd.Params, "span_id"),
		Limit:     getIntParam(cmd.Params, "limit", 50),
		Offset:    getIntParam(cmd.Params, "offset", 0),
		Since:     getDurationParam(cmd.Params, "since", 15*time.Minute),
		StartTime: getStringParam(cmd.Params, "start_time"),
		EndTime:   getStringParam(cmd.Params, "end_time"),
	}

	return s.logQueryRepo.QueryLogs(ctx, opts)
}

// handleQueryLogHistogram 处理日志直方图查询指令
func (s *commandService) handleQueryLogHistogram(ctx context.Context, cmd *command.Command) (any, error) {
	opts := repository.LogQueryOptions{
		Query:     getStringParam(cmd.Params, "query"),
		Service:   getStringParam(cmd.Params, "service"),
		Level:     getStringParam(cmd.Params, "level"),
		Scope:     getStringParam(cmd.Params, "scope"),
		Since:     getDurationParam(cmd.Params, "since", 15*time.Minute),
		StartTime: getStringParam(cmd.Params, "start_time"),
		EndTime:   getStringParam(cmd.Params, "end_time"),
	}
	return s.logQueryRepo.QueryHistogram(ctx, opts)
}

// handleQueryMetrics 处理指标查询指令
func (s *commandService) handleQueryMetrics(ctx context.Context, cmd *command.Command) (any, error) {
	if s.metricsQueryRepo == nil {
		return nil, fmt.Errorf("ClickHouse not configured")
	}

	subAction := getStringParam(cmd.Params, "sub_action")

	switch subAction {
	case "list_all":
		return s.metricsQueryRepo.ListAllNodeMetrics(ctx)

	case "get_node":
		nodeName := getStringParam(cmd.Params, "node_name")
		if nodeName == "" {
			return nil, fmt.Errorf("node_name is required")
		}
		return s.metricsQueryRepo.GetNodeMetrics(ctx, nodeName)

	case "get_series":
		nodeName := getStringParam(cmd.Params, "node_name")
		metric := getStringParam(cmd.Params, "metric")
		since := getDurationParam(cmd.Params, "since", 30*time.Minute)
		if nodeName == "" || metric == "" {
			return nil, fmt.Errorf("node_name and metric are required")
		}
		return s.metricsQueryRepo.GetNodeMetricsSeries(ctx, nodeName, metric, since)

	case "get_history":
		nodeName := getStringParam(cmd.Params, "node_name")
		if nodeName == "" {
			return nil, fmt.Errorf("node_name is required")
		}
		since := getDurationParam(cmd.Params, "since", 24*time.Hour)
		return s.metricsQueryRepo.GetNodeMetricsHistory(ctx, nodeName, since)

	case "get_summary", "":
		return s.metricsQueryRepo.GetMetricsSummary(ctx)

	default:
		return nil, fmt.Errorf("unknown metrics sub_action: %s", subAction)
	}
}

// handleQuerySLO 处理 SLO 查询指令
func (s *commandService) handleQuerySLO(ctx context.Context, cmd *command.Command) (any, error) {
	if s.sloQueryRepo == nil {
		return nil, fmt.Errorf("ClickHouse not configured")
	}

	subAction := getStringParam(cmd.Params, "sub_action")
	since := getDurationParam(cmd.Params, "since", 5*time.Minute)

	switch subAction {
	case "list_ingress":
		return s.sloQueryRepo.ListIngressSLO(ctx, since)

	// list_service / list_edges 已移除：SLO 只做 ingress 外部视角，
	// 服务间调用关系改由 APM 指令 (query_traces 的 topology) 提供。
	// 见 docs/design/active/slo-ingress-contract-design.md

	case "get_time_series":
		name := getStringParam(cmd.Params, "name")
		if name == "" {
			return nil, fmt.Errorf("name is required")
		}
		return s.sloQueryRepo.GetSLOTimeSeries(ctx, name, since)

	case "get_summary", "":
		return s.sloQueryRepo.GetSLOSummary(ctx)

	default:
		return nil, fmt.Errorf("unknown SLO sub_action: %s", subAction)
	}
}

// =============================================================================
// 参数提取辅助
// =============================================================================

func getStringParam(params map[string]any, key string) string {
	if params == nil {
		return ""
	}
	v, ok := params[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func getFloat64Param(params map[string]any, key string) float64 {
	if params == nil {
		return 0
	}
	v, ok := params[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case string:
		f, _ := strconv.ParseFloat(n, 64)
		return f
	default:
		return 0
	}
}

func getIntParam(params map[string]any, key string, defaultVal int) int {
	if params == nil {
		return defaultVal
	}
	v, ok := params[key]
	if !ok {
		return defaultVal
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case string:
		if i, err := strconv.Atoi(n); err == nil {
			return i
		}
		return defaultVal
	default:
		return defaultVal
	}
}

func getDurationParam(params map[string]any, key string, defaultVal time.Duration) time.Duration {
	if params == nil {
		return defaultVal
	}
	v, ok := params[key]
	if !ok {
		return defaultVal
	}
	switch d := v.(type) {
	case string:
		parsed, err := time.ParseDuration(d)
		if err != nil {
			return defaultVal
		}
		return parsed
	case float64:
		// 秒数
		if d > 0 {
			return time.Duration(d) * time.Second
		}
		return defaultVal
	default:
		return defaultVal
	}
}

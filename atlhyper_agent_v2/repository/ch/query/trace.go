// Package query ClickHouse 按需查询仓库实现
package query

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	model_v3 "AtlHyper/model_v3"
	"AtlHyper/model_v3/apm"
)

// traceRepository Trace 查询仓库
type traceRepository struct {
	client sdk.ClickHouseClient
}

// NewTraceQueryRepository 创建 Trace 查询仓库
func NewTraceQueryRepository(client sdk.ClickHouseClient) repository.TraceQueryRepository {
	return &traceRepository{client: client}
}

// traceTimeCondition 构建时间条件和参数
// 绝对时间（startTime/endTime）优先于相对时间（since）
func traceTimeCondition(col string, since time.Duration, startTime, endTime string) ([]string, []any) {
	if startTime != "" && endTime != "" {
		if startT, err := time.Parse(time.RFC3339Nano, startTime); err == nil {
			if endT, err := time.Parse(time.RFC3339Nano, endTime); err == nil {
				return []string{col + " >= ?", col + " <= ?"}, []any{startT, endT}
			}
		}
	}
	sec := sinceSeconds(since)
	return []string{fmt.Sprintf("%s >= now() - INTERVAL %d SECOND", col, sec)}, nil
}

// traceTimeSec 返回绝对时间范围的秒数，或 since 秒数（用于 RPS 分母等）
func traceTimeSec(since time.Duration, startTime, endTime string) int64 {
	if startTime != "" && endTime != "" {
		if startT, err := time.Parse(time.RFC3339Nano, startTime); err == nil {
			if endT, err := time.Parse(time.RFC3339Nano, endTime); err == nil {
				sec := int64(endT.Sub(startT).Seconds())
				if sec > 0 {
					return sec
				}
			}
		}
	}
	return sinceSeconds(since)
}

// entrySpanRootExpr 生成「入口 span」字段取值表达式（≈ Elastic APM 的 Transaction）。
//
// 规则：优先 SpanKind = Server 的最早 span；trace 没有 Server span 时
// 退化到无父 span 的最早 span。所有需要「trace 的根是谁」的查询必须
// 用本函数，禁止各自内联变体 —— 2026-08-28 曾因该逻辑作用在被过滤的
// 残缺集合上，把叶子服务错判成根服务。
func entrySpanRootExpr(column string) string {
	server := "argMinIf(" + column + ", Timestamp, SpanKind = " + apm.SQLSpanKindServer + ")"
	orphan := "argMinIf(" + column + ", Timestamp, ParentSpanId = '')"
	return "if(" + server + " != '', " + server + ", " + orphan + ")"
}

// buildListTracesQuery 构造 trace 列表聚合查询。
//
// ⚠️ 两段式结构是硬约束（有测试守护）：过滤条件（service / operation /
// status_code …）只用于内层子查询**定位 TraceId 集合**，外层对完整 trace
// 聚合。条件若直接写进聚合层，spanCount / serviceCount / rootService
// 都会基于残缺 span 集合计算（实测失真：1 span/1 服务 vs 真实 3/2）。
//
// 外层不带时间窗：otel_traces 走 7 天 TTL（当前万级行数），全表按
// TraceId IN 过滤的代价可忽略；若未来量级上升，再给外层加放宽的时间窗
// （必须比内层宽出单条 trace 的最大时长，否则边缘 span 会被截掉）。
func buildListTracesQuery(where, orderBy string, limit int) string {
	return fmt.Sprintf(`
		SELECT TraceId,
		       min(Timestamp) AS ts,
		       %s AS rootSvc,
		       %s AS rootOp,
		       max(Duration) / 1e6 AS durationMs,
		       count() AS spanCount,
		       count(DISTINCT ServiceName) AS serviceCount,
		       groupArray(DISTINCT ServiceName) AS services,
		       countIf(StatusCode = `+apm.SQLStatusCodeError+`) > 0 AS hasError,
		       anyIf(
		           Events.Attributes[indexOf(Events.Name, 'exception')]['exception.type'],
		           indexOf(Events.Name, 'exception') > 0
		       ) AS errorType,
		       anyIf(
		           Events.Attributes[indexOf(Events.Name, 'exception')]['exception.message'],
		           indexOf(Events.Name, 'exception') > 0
		       ) AS errorMessage
		FROM otel_traces
		WHERE TraceId IN (
		    SELECT DISTINCT TraceId FROM otel_traces WHERE %s
		)
		GROUP BY TraceId
		ORDER BY %s
		LIMIT %d
	`, entrySpanRootExpr("ServiceName"), entrySpanRootExpr("SpanName"), where, orderBy, limit)
}

// ListTraces 查询 Trace 列表（按 TraceId 聚合）
func (r *traceRepository) ListTraces(ctx context.Context, service, operation string, minDurationMs float64, limit int, since time.Duration, sortBy string, startTime, endTime string, statusCode, method string) ([]apm.TraceSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 2000 {
		limit = 2000
	}

	// 构建 WHERE
	timeConds, timeArgs := traceTimeCondition("Timestamp", since, startTime, endTime)
	conditions := timeConds
	args := timeArgs

	if service != "" {
		conditions = append(conditions, "ServiceName = ?")
		args = append(args, service)
	}
	if operation != "" {
		conditions = append(conditions, "SpanName = ?")
		args = append(args, operation)
	}
	if minDurationMs > 0 {
		conditions = append(conditions, "Duration >= ?")
		args = append(args, int64(minDurationMs*1e6)) // ms → ns
	}
	if statusCode != "" {
		conditions = append(conditions, "SpanAttributes['http.response.status_code'] = ?")
		args = append(args, statusCode)
	}
	if method != "" {
		conditions = append(conditions, "SpanAttributes['http.request.method'] = ?")
		args = append(args, method)
	}

	where := strings.Join(conditions, " AND ")

	orderBy := "ts DESC"
	if sortBy == "duration_desc" {
		orderBy = "durationMs DESC"
	}

	query := buildListTracesQuery(where, orderBy, limit)

	rows, err := r.client.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list traces: %w", err)
	}
	defer rows.Close()

	var result []apm.TraceSummary
	for rows.Next() {
		var t apm.TraceSummary
		var hasErr uint8
		if err := rows.Scan(
			&t.TraceId, &t.Timestamp, &t.RootService, &t.RootOperation,
			&t.DurationMs, &t.SpanCount, &t.ServiceCount, &t.Services, &hasErr,
			&t.ErrorType, &t.ErrorMessage,
		); err != nil {
			continue
		}
		t.HasError = hasErr > 0
		t.DurationMs = roundTo(t.DurationMs, 2)
		result = append(result, t)
	}
	if result == nil {
		result = []apm.TraceSummary{}
	}
	return result, rows.Err()
}

// GetTraceDetail 查询 Trace 详情（所有 Span）
func (r *traceRepository) GetTraceDetail(ctx context.Context, traceID string) (*apm.TraceDetail, error) {
	if traceID == "" {
		return nil, fmt.Errorf("traceID is required")
	}

	query := `
		SELECT Timestamp, TraceId, SpanId, ParentSpanId, SpanName, SpanKind,
		       ServiceName, Duration, StatusCode, StatusMessage,
		       SpanAttributes, ResourceAttributes, Events.Timestamp, Events.Name, Events.Attributes
		FROM otel_traces
		WHERE TraceId = ?
		ORDER BY Timestamp
	`

	rows, err := r.client.Query(ctx, query, traceID)
	if err != nil {
		return nil, fmt.Errorf("get trace detail: %w", err)
	}
	defer rows.Close()

	var spans []apm.Span
	serviceSet := make(map[string]bool)

	for rows.Next() {
		var s apm.Span
		var spanAttrs map[string]string
		var resAttrs map[string]string
		var eventTimestamps []time.Time
		var eventNames []string
		var eventAttrs []map[string]string

		if err := rows.Scan(
			&s.Timestamp, &s.TraceId, &s.SpanId, &s.ParentSpanId,
			&s.SpanName, &s.SpanKind, &s.ServiceName, &s.Duration,
			&s.StatusCode, &s.StatusMessage,
			&spanAttrs, &resAttrs,
			&eventTimestamps, &eventNames, &eventAttrs,
		); err != nil {
			continue
		}

		s.DurationMs = parseDurationNanos(s.Duration)
		serviceSet[s.ServiceName] = true

		// 解析 HTTP 属性
		if method := spanAttrs["http.method"]; method != "" {
			s.HTTP = &apm.SpanHTTP{
				Method: method,
				Route:  spanAttrs["http.route"],
				URL:    spanAttrs["http.url"],
				Server: spanAttrs["server.address"],
			}
			if code := spanAttrs["http.status_code"]; code != "" {
				fmt.Sscanf(code, "%d", &s.HTTP.StatusCode)
			}
			if port := spanAttrs["server.port"]; port != "" {
				fmt.Sscanf(port, "%d", &s.HTTP.ServerPort)
			}
		}

		// 解析 DB 属性
		if sys := spanAttrs["db.system"]; sys != "" {
			s.DB = &apm.SpanDB{
				System:    sys,
				Name:      spanAttrs["db.name"],
				Operation: spanAttrs["db.operation"],
				Table:     spanAttrs["db.sql.table"],
				Statement: spanAttrs["db.statement"],
			}
		}

		// 解析 Resource 属性
		s.Resource = apm.SpanResource{
			ServiceVersion: resAttrs["service.version"],
			InstanceId:     resAttrs["service.instance.id"],
			PodName:        resAttrs["k8s.pod.name"],
			NodeName:       resAttrs["k8s.node.name"],
			DeploymentName: resAttrs["k8s.deployment.name"],
			NamespaceName:  resAttrs["k8s.namespace.name"],
			ClusterName:    resAttrs["k8s.cluster.name"],
		}

		// 解析 Events
		for i := 0; i < len(eventNames) && i < len(eventTimestamps); i++ {
			evt := apm.SpanEvent{
				Timestamp: eventTimestamps[i],
				Name:      eventNames[i],
			}
			if i < len(eventAttrs) {
				evt.Attributes = eventAttrs[i]
			}
			s.Events = append(s.Events, evt)
		}

		// 从 Events 提取结构化错误信息
		for _, evt := range s.Events {
			if evt.Name == "exception" {
				s.Error = &apm.SpanError{
					Type:       evt.Attributes["exception.type"],
					Message:    evt.Attributes["exception.message"],
					Stacktrace: evt.Attributes["exception.stacktrace"],
				}
				break
			}
		}

		spans = append(spans, s)
	}

	if len(spans) == 0 {
		return nil, fmt.Errorf("trace not found: %s", traceID)
	}

	// 计算整体 duration
	var maxDuration float64
	for _, s := range spans {
		if s.DurationMs > maxDuration {
			maxDuration = s.DurationMs
		}
	}

	return &apm.TraceDetail{
		TraceId:      traceID,
		DurationMs:   roundTo(maxDuration, 2),
		ServiceCount: len(serviceSet),
		SpanCount:    len(spans),
		Spans:        spans,
	}, rows.Err()
}

// ListServices 查询服务列表（聚合统计）
func (r *traceRepository) ListServices(ctx context.Context, since time.Duration, startTime, endTime string) ([]apm.APMService, error) {
	sec := traceTimeSec(since, startTime, endTime)
	timeConds, timeArgs := traceTimeCondition("Timestamp", since, startTime, endTime)

	whereClause := "SpanKind = " + apm.SQLSpanKindServer
	for _, c := range timeConds {
		whereClause += " AND " + c
	}

	query := fmt.Sprintf(`
		SELECT ServiceName,
		       if(ResourceAttributes['service.namespace'] != '', ResourceAttributes['service.namespace'], if(ResourceAttributes['k8s.namespace.name'] != '', ResourceAttributes['k8s.namespace.name'], 'default')) AS ns,
		       anyLast(ResourceAttributes['deployment.environment']) AS env,
		       count() AS spanCount,
		       countIf(StatusCode = `+apm.SQLStatusCodeError+`) AS errorCount,
		       1 - countIf(StatusCode = `+apm.SQLStatusCodeError+`) / count() AS successRate,
		       avg(Duration) / 1e6 AS avgMs,
		       quantile(0.50)(Duration) / 1e6 AS p50Ms,
		       quantile(0.99)(Duration) / 1e6 AS p99Ms,
		       count() / %d AS rps
		FROM otel_traces
		WHERE %s
		GROUP BY ServiceName, ns
	`, sec, whereClause)

	rows, err := r.client.Query(ctx, query, timeArgs...)
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}
	defer rows.Close()

	var result []apm.APMService
	for rows.Next() {
		var s apm.APMService
		if err := rows.Scan(
			&s.Name, &s.Namespace, &s.Environment, &s.SpanCount, &s.ErrorCount,
			&s.SuccessRate, &s.AvgDurationMs, &s.P50Ms, &s.P99Ms, &s.RPS,
		); err != nil {
			continue
		}
		s.SuccessRate = roundTo(s.SuccessRate, 4)
		s.AvgDurationMs = roundTo(s.AvgDurationMs, 2)
		s.P50Ms = roundTo(s.P50Ms, 2)
		s.P99Ms = roundTo(s.P99Ms, 2)
		s.RPS = roundRPS(s.RPS)
		result = append(result, s)
	}
	if result == nil {
		result = []apm.APMService{}
	}
	return result, rows.Err()
}

// ListOperations 查询操作级聚合统计（GROUP BY ServiceName, SpanName）
func (r *traceRepository) ListOperations(ctx context.Context, since time.Duration, startTime, endTime string) ([]apm.OperationStats, error) {
	sec := traceTimeSec(since, startTime, endTime)
	timeConds, timeArgs := traceTimeCondition("Timestamp", since, startTime, endTime)

	whereClause := "SpanKind = " + apm.SQLSpanKindServer
	for _, c := range timeConds {
		whereClause += " AND " + c
	}

	query := fmt.Sprintf(`
		SELECT ServiceName,
		       SpanName AS operation,
		       count() AS spanCount,
		       countIf(StatusCode = `+apm.SQLStatusCodeError+`) AS errorCount,
		       1 - countIf(StatusCode = `+apm.SQLStatusCodeError+`) / count() AS successRate,
		       avg(Duration) / 1e6 AS avgMs,
		       quantile(0.50)(Duration) / 1e6 AS p50Ms,
		       quantile(0.99)(Duration) / 1e6 AS p99Ms,
		       count() / %d AS rps
		FROM otel_traces
		WHERE %s
		GROUP BY ServiceName, SpanName
		ORDER BY spanCount DESC
	`, sec, whereClause)

	rows, err := r.client.Query(ctx, query, timeArgs...)
	if err != nil {
		return nil, fmt.Errorf("list operations: %w", err)
	}
	defer rows.Close()

	var result []apm.OperationStats
	for rows.Next() {
		var o apm.OperationStats
		if err := rows.Scan(
			&o.ServiceName, &o.OperationName, &o.SpanCount, &o.ErrorCount,
			&o.SuccessRate, &o.AvgDurationMs, &o.P50Ms, &o.P99Ms, &o.RPS,
		); err != nil {
			continue
		}
		o.SuccessRate = roundTo(o.SuccessRate, 4)
		o.AvgDurationMs = roundTo(o.AvgDurationMs, 2)
		o.P50Ms = roundTo(o.P50Ms, 2)
		o.P99Ms = roundTo(o.P99Ms, 2)
		o.RPS = roundRPS(o.RPS)
		result = append(result, o)
	}
	if result == nil {
		result = []apm.OperationStats{}
	}
	return result, rows.Err()
}

// GetTopology 获取服务拓扑图
func (r *traceRepository) GetTopology(ctx context.Context, since time.Duration, startTime, endTime string) (*apm.Topology, error) {
	sec := traceTimeSec(since, startTime, endTime)
	timeConds, timeArgs := traceTimeCondition("Timestamp", since, startTime, endTime)

	svcWhere := "SpanKind = " + apm.SQLSpanKindServer
	for _, c := range timeConds {
		svcWhere += " AND " + c
	}

	// 节点: 所有服务
	serviceQuery := fmt.Sprintf(`
		SELECT ServiceName,
		       if(ResourceAttributes['service.namespace'] != '', ResourceAttributes['service.namespace'], if(ResourceAttributes['k8s.namespace.name'] != '', ResourceAttributes['k8s.namespace.name'], 'default')) AS ns,
		       count() / %d AS rps,
		       1 - countIf(StatusCode = `+apm.SQLStatusCodeError+`) / count() AS successRate,
		       quantile(0.99)(Duration) / 1e6 AS p99Ms
		FROM otel_traces
		WHERE %s
		GROUP BY ServiceName, ns
	`, sec, svcWhere)
	svcRows, err := r.client.Query(ctx, serviceQuery, timeArgs...)
	if err != nil {
		return nil, fmt.Errorf("topology nodes: %w", err)
	}
	defer svcRows.Close()

	var nodes []apm.TopologyNode
	nodeSet := make(map[string]bool)

	for svcRows.Next() {
		var name, ns string
		var rps, successRate, p99Ms float64
		if err := svcRows.Scan(&name, &ns, &rps, &successRate, &p99Ms); err != nil {
			continue
		}
		if ns == "" {
			ns = "default"
		}
		id := ns + "/" + name
		nodeSet[id] = true
		node := apm.TopologyNode{
			Id:          id,
			Name:        name,
			Namespace:   ns,
			Type:        "service",
			RPS:         roundRPS(rps),
			SuccessRate: roundTo(successRate, 4),
			P99Ms:       roundTo(p99Ms, 2),
			Status:      nodeStatus(successRate),
		}
		nodes = append(nodes, node)
	}

	// 边: 跨服务调用（parent-child 关系）
	edgeTimeConds1, _ := traceTimeCondition("t1.Timestamp", since, startTime, endTime)
	edgeTimeConds2, _ := traceTimeCondition("t2.Timestamp", since, startTime, endTime)
	edgeWhere := "t1.ServiceName != t2.ServiceName"
	for _, c := range edgeTimeConds1 {
		edgeWhere += " AND " + c
	}
	for _, c := range edgeTimeConds2 {
		edgeWhere += " AND " + c
	}

	// 边查询需要 t1 和 t2 两组时间参数
	var edgeArgs []any
	edgeArgs = append(edgeArgs, timeArgs...)
	edgeArgs = append(edgeArgs, timeArgs...)

	edgeQuery := fmt.Sprintf(`
		SELECT concat(if(t1.ResourceAttributes['service.namespace'] != '', t1.ResourceAttributes['service.namespace'], if(t1.ResourceAttributes['k8s.namespace.name'] != '', t1.ResourceAttributes['k8s.namespace.name'], 'default')), '/', t1.ServiceName) AS source,
		       concat(if(t2.ResourceAttributes['service.namespace'] != '', t2.ResourceAttributes['service.namespace'], if(t2.ResourceAttributes['k8s.namespace.name'] != '', t2.ResourceAttributes['k8s.namespace.name'], 'default')), '/', t2.ServiceName) AS target,
		       count() AS callCount,
		       avg(t2.Duration) / 1e6 AS avgMs,
		       countIf(t2.StatusCode = `+apm.SQLStatusCodeError+`) / count() AS errorRate
		FROM otel_traces t1
		JOIN otel_traces t2 ON t1.SpanId = t2.ParentSpanId AND t1.TraceId = t2.TraceId
		WHERE %s
		GROUP BY source, target
	`, edgeWhere)

	edgeRows, err := r.client.Query(ctx, edgeQuery, edgeArgs...)
	if err != nil {
		return &apm.Topology{Nodes: nodes, Edges: []apm.TopologyEdge{}}, nil
	}
	defer edgeRows.Close()

	var edges []apm.TopologyEdge
	for edgeRows.Next() {
		var e apm.TopologyEdge
		var source, target string
		if err := edgeRows.Scan(&source, &target, &e.CallCount, &e.AvgMs, &e.ErrorRate); err != nil {
			continue
		}
		if !nodeSet[source] || !nodeSet[target] {
			continue
		}
		e.Source = source
		e.Target = target
		e.AvgMs = roundTo(e.AvgMs, 2)
		e.ErrorRate = roundTo(e.ErrorRate, 4)
		edges = append(edges, e)
	}
	if edges == nil {
		edges = []apm.TopologyEdge{}
	}

	// DB 节点: 从 CLIENT span 中提取数据库调用
	dbWhere := "SpanKind = " + apm.SQLSpanKindClient + " AND SpanAttributes['db.system'] != ''"
	for _, c := range timeConds {
		dbWhere += " AND " + c
	}

	dbQuery := fmt.Sprintf(`
		SELECT ServiceName,
		       if(ResourceAttributes['service.namespace'] != '', ResourceAttributes['service.namespace'], if(ResourceAttributes['k8s.namespace.name'] != '', ResourceAttributes['k8s.namespace.name'], 'default')) AS ns,
		       SpanAttributes['db.system'] AS dbSystem,
		       SpanAttributes['db.name'] AS dbName,
		       count() AS callCount,
		       avg(Duration) / 1e6 AS avgMs,
		       countIf(StatusCode = `+apm.SQLStatusCodeError+`) / count() AS errorRate
		FROM otel_traces
		WHERE %s
		GROUP BY ServiceName, ns, dbSystem, dbName
	`, dbWhere)

	dbRows, err := r.client.Query(ctx, dbQuery, timeArgs...)
	if err == nil {
		defer dbRows.Close()
		dbNodeSet := make(map[string]bool)
		for dbRows.Next() {
			var svcName, svcNs, dbSystem, dbName string
			var callCount int64
			var avgMs, errorRate float64
			if err := dbRows.Scan(&svcName, &svcNs, &dbSystem, &dbName, &callCount, &avgMs, &errorRate); err != nil {
				continue
			}
			// 创建 DB 节点
			dbID := dbSystem + ":" + dbName
			if !dbNodeSet[dbID] {
				dbNodeSet[dbID] = true
				successRate := 1 - errorRate
				nodes = append(nodes, apm.TopologyNode{
					Id:          dbID,
					Name:        dbID,
					Type:        "database",
					RPS:         roundRPS(float64(callCount) / float64(sec)),
					SuccessRate: roundTo(successRate, 4),
					P99Ms:       roundTo(avgMs, 2),
					Status:      nodeStatus(successRate),
				})
			}
			// 创建 service → DB 边
			svcID := svcNs + "/" + svcName
			if nodeSet[svcID] {
				edges = append(edges, apm.TopologyEdge{
					Source:    svcID,
					Target:    dbID,
					CallCount: callCount,
					AvgMs:     roundTo(avgMs, 2),
					ErrorRate: roundTo(errorRate, 4),
				})
			}
		}
	}

	// 排序节点
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})

	return &apm.Topology{Nodes: nodes, Edges: edges}, nil
}

// GetHTTPStats 获取 HTTP 状态码分布
func (r *traceRepository) GetHTTPStats(ctx context.Context, service string, since time.Duration, startTime, endTime string) ([]apm.HTTPStats, error) {
	timeConds, timeArgs := traceTimeCondition("Timestamp", since, startTime, endTime)

	whereClause := "SpanKind = " + apm.SQLSpanKindServer + " AND SpanAttributes['http.response.status_code'] != '' AND ServiceName = ?"
	for _, c := range timeConds {
		whereClause += " AND " + c
	}

	query := fmt.Sprintf(`
		SELECT toInt32OrZero(SpanAttributes['http.response.status_code']) AS statusCode,
		       SpanAttributes['http.request.method'] AS method,
		       SpanName AS operation,
		       count() AS cnt
		FROM otel_traces
		WHERE %s
		GROUP BY statusCode, method, operation
		ORDER BY statusCode, method, operation
	`, whereClause)

	args := append([]any{service}, timeArgs...)
	rows, err := r.client.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get http stats: %w", err)
	}
	defer rows.Close()

	var result []apm.HTTPStats
	for rows.Next() {
		var s apm.HTTPStats
		if err := rows.Scan(&s.StatusCode, &s.Method, &s.Operation, &s.Count); err != nil {
			continue
		}
		result = append(result, s)
	}
	if result == nil {
		result = []apm.HTTPStats{}
	}
	return result, rows.Err()
}

// GetDBStats 获取数据库操作统计
func (r *traceRepository) GetDBStats(ctx context.Context, service string, since time.Duration, startTime, endTime string) ([]apm.DBOperationStats, error) {
	timeConds, timeArgs := traceTimeCondition("Timestamp", since, startTime, endTime)

	whereClause := "SpanKind = " + apm.SQLSpanKindClient + " AND SpanAttributes['db.system'] != '' AND ServiceName = ?"
	for _, c := range timeConds {
		whereClause += " AND " + c
	}

	query := fmt.Sprintf(`
		SELECT SpanAttributes['db.system'] AS dbSystem,
		       SpanAttributes['db.name'] AS dbName,
		       SpanAttributes['db.operation'] AS operation,
		       SpanAttributes['db.sql.table'] AS tableName,
		       count() AS callCount,
		       avg(Duration) / 1e6 AS avgMs,
		       quantile(0.99)(Duration) / 1e6 AS p99Ms,
		       countIf(StatusCode = `+apm.SQLStatusCodeError+`) / count() AS errorRate
		FROM otel_traces
		WHERE %s
		GROUP BY dbSystem, dbName, operation, tableName
		ORDER BY callCount DESC
	`, whereClause)

	args := append([]any{service}, timeArgs...)
	rows, err := r.client.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get db stats: %w", err)
	}
	defer rows.Close()

	var result []apm.DBOperationStats
	for rows.Next() {
		var s apm.DBOperationStats
		if err := rows.Scan(&s.DBSystem, &s.DBName, &s.Operation, &s.Table, &s.CallCount, &s.AvgMs, &s.P99Ms, &s.ErrorRate); err != nil {
			continue
		}
		s.AvgMs = roundTo(s.AvgMs, 2)
		s.P99Ms = roundTo(s.P99Ms, 2)
		s.ErrorRate = roundTo(s.ErrorRate, 4)
		result = append(result, s)
	}
	if result == nil {
		result = []apm.DBOperationStats{}
	}
	return result, rows.Err()
}

// GetServiceTimeSeries 获取服务时序趋势（ClickHouse 按需聚合）
//
// 根据 since 自动选择时间桶大小，目标 60-180 个数据点。
func (r *traceRepository) GetServiceTimeSeries(ctx context.Context, service string, since time.Duration) ([]apm.TimePoint, error) {
	sec := sinceSeconds(since)

	// 根据总时长选择桶大小（秒），目标 60-180 点
	bucketSec := int64(60) // 默认 1 分钟
	switch {
	case sec <= 3600: // ≤1h → 1min 桶
		bucketSec = 60
	case sec <= 21600: // ≤6h → 5min 桶
		bucketSec = 300
	case sec <= 86400: // ≤24h → 15min 桶
		bucketSec = 900
	case sec <= 604800: // ≤7d → 1h 桶
		bucketSec = 3600
	default: // >7d → 4h 桶
		bucketSec = 14400
	}

	query := fmt.Sprintf(`
		SELECT toStartOfInterval(Timestamp, INTERVAL %d SECOND) AS bucket,
		       count() / %d AS rps,
		       1 - countIf(StatusCode = `+apm.SQLStatusCodeError+`) / count() AS successRate,
		       avg(Duration) / 1e6 AS avgMs,
		       quantile(0.99)(Duration) / 1e6 AS p99Ms,
		       countIf(StatusCode = `+apm.SQLStatusCodeError+`) AS errorCount
		FROM otel_traces
		WHERE SpanKind = `+apm.SQLSpanKindServer+`
		  AND ServiceName = ?
		  AND Timestamp >= now() - INTERVAL %d SECOND
		GROUP BY bucket
		ORDER BY bucket ASC
	`, bucketSec, bucketSec, sec)

	rows, err := r.client.Query(ctx, query, service)
	if err != nil {
		return nil, fmt.Errorf("get service time series: %w", err)
	}
	defer rows.Close()

	var result []apm.TimePoint
	for rows.Next() {
		var pt apm.TimePoint
		if err := rows.Scan(&pt.Timestamp, &pt.RPS, &pt.SuccessRate, &pt.AvgMs, &pt.P99Ms, &pt.ErrorCount); err != nil {
			continue
		}
		pt.RPS = roundRPS(pt.RPS)
		pt.SuccessRate = roundTo(pt.SuccessRate, 4)
		pt.AvgMs = roundTo(pt.AvgMs, 2)
		pt.P99Ms = roundTo(pt.P99Ms, 2)
		result = append(result, pt)
	}
	if result == nil {
		result = []apm.TimePoint{}
	}
	return result, rows.Err()
}

// nodeStatus 根据成功率判断节点健康状态
func nodeStatus(successRate float64) model_v3.HealthStatus {
	switch {
	case successRate < 0.95:
		return model_v3.HealthStatusCritical
	case successRate < 0.99:
		return model_v3.HealthStatusWarning
	default:
		return model_v3.HealthStatusHealthy
	}
}

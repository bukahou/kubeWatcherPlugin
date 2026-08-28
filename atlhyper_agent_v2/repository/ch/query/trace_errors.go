package query

import (
	"context"
	"time"

	"AtlHyper/model_v3/apm"
)

// trace_errors.go — 错误证据链回填
//
// 背景（2026-08-28 register 500）：connect-go 等框架的 Error span 常常
// 既没有 exception 事件也没有 StatusMessage，异常详情只存在于**同 trace
// 的 ERROR 日志属性**里 —— 且往往在上游服务（BFF/网关）手里，而排查者
// 点开的是下游报错服务的 span。此处把日志证据回填到 span 上并标注来源，
// 让任何一个 Error span 都能回答「哪一层、什么异常、什么消息」。

// traceErrorLog 一条候选证据日志（仅取回填需要的字段）
type traceErrorLog struct {
	Timestamp   time.Time
	SpanId      string
	ServiceName string
	Body        string
	Attributes  map[string]string
}

func (l *traceErrorLog) hasException() bool {
	return l.Attributes["exception.message"] != "" || l.Attributes["exception.type"] != ""
}

// attachLogErrors 按证据优先级为每个 Error span 填充 SpanError：
//
//	span_event（Events 里的 exception，已由 GetTraceDetail 解析）
//	  > status_message（span 自带）
//	  > trace_log（SpanId 精确匹配 > 同服务日志 > 跨服务日志）
//
// 跨服务日志作为最后手段是有意的：下游 span 的错误详情往往只在上游
// 日志里，宁可给出标注了来源的跨服务证据，也不留一句空白。
func attachLogErrors(spans []apm.Span, logs []traceErrorLog) {
	// 候选证据：只有含 exception 字段的日志才算
	var evidence []traceErrorLog
	for _, l := range logs {
		if l.hasException() {
			evidence = append(evidence, l)
		}
	}

	for i := range spans {
		s := &spans[i]
		if s.StatusCode != apm.StatusCodeError {
			continue
		}
		// 1. span 自带 exception 事件（GetTraceDetail 已填）—— 只补 Source
		if s.Error != nil {
			if s.Error.Source == "" {
				s.Error.Source = apm.ErrorSourceSpanEvent
			}
			continue
		}
		// 2. StatusMessage
		if s.StatusMessage != "" {
			s.Error = &apm.SpanError{
				Message: s.StatusMessage,
				Source:  apm.ErrorSourceStatusMessage,
			}
			continue
		}
		// 3. trace 日志：SpanId 精确 > 同服务 > 跨服务
		var best *traceErrorLog
		bestRank := 99
		for j := range evidence {
			l := &evidence[j]
			rank := 3
			switch {
			case l.SpanId != "" && l.SpanId == s.SpanId:
				rank = 1
			case l.ServiceName == s.ServiceName:
				rank = 2
			}
			if rank < bestRank {
				best, bestRank = l, rank
			}
		}
		if best != nil {
			s.Error = &apm.SpanError{
				Type:          best.Attributes["exception.type"],
				Message:       best.Attributes["exception.message"],
				Source:        apm.ErrorSourceTraceLog,
				SourceService: best.ServiceName,
			}
		}
	}
}

// queryTraceErrorLogs 拉取一条 trace 的全部 ERROR 日志（证据候选）。
// 失败时返回空切片而非错误 —— 证据链是增强信息，不应让整个详情查询失败。
func (r *traceRepository) queryTraceErrorLogs(ctx context.Context, traceID string) []traceErrorLog {
	query := `
		SELECT Timestamp, SpanId, ServiceName, Body, LogAttributes
		FROM otel_logs
		WHERE TraceId = ? AND SeverityText = 'ERROR'
		ORDER BY Timestamp
		LIMIT 50
	`
	rows, err := r.client.Query(ctx, query, traceID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var logs []traceErrorLog
	for rows.Next() {
		var l traceErrorLog
		if err := rows.Scan(&l.Timestamp, &l.SpanId, &l.ServiceName, &l.Body, &l.Attributes); err != nil {
			continue
		}
		logs = append(logs, l)
	}
	return logs
}

// atlhyper_master_v2/aiops/enricher/context_builder.go
// 将结构化事件数据转换为 LLM 可理解的文本描述
package enricher

import (
	"fmt"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/prompts"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/model_v3/cluster"
)

// 条目上限常量 — 防止超长 Prompt
const (
	MaxTimelineEntries   = 30 // 时间线最多 30 条
	MaxHistoricalEntries = 10 // 历史事件最多 10 条
	MaxEntityEntries     = 50 // 受影响实体最多 50 个
)

// BuildIncidentContext 从结构化数据构建 LLM 上下文
func BuildIncidentContext(
	incident *database.AIOpsIncident,
	entities []*database.AIOpsIncidentEntity,
	timeline []*database.AIOpsIncidentTimeline,
	historical []*database.AIOpsIncident,
) *prompts.IncidentPromptContext {
	return &prompts.IncidentPromptContext{
		IncidentSummary:   buildSummary(incident),
		RootCauseEntity:   buildRootCause(entities),
		AffectedEntities:  buildEntities(entities),
		TimelineText:      buildTimeline(timeline),
		HistoricalContext: buildHistorical(historical),
	}
}

// buildSummary 构建事件概要
func buildSummary(inc *database.AIOpsIncident) string {
	if inc == nil {
		return "事件概要: 无数据"
	}

	duration := formatDuration(inc.DurationS)
	resolved := "进行中"
	if inc.ResolvedAt != nil {
		resolved = fmt.Sprintf("已解决 (%s)", inc.ResolvedAt.Format("2006-01-02 15:04"))
	}

	return fmt.Sprintf(`事件概要:
  ID: %s
  状态: %s | 严重度: %s | 持续: %s
  集群: %s
  开始时间: %s
  解决状态: %s
  峰值风险: %.0f | 复发次数: %d`,
		inc.ID,
		inc.State, inc.Severity, duration,
		inc.ClusterID,
		inc.StartedAt.Format("2006-01-02 15:04:05"),
		resolved,
		inc.PeakRisk, inc.Recurrence,
	)
}

// buildRootCause 构建根因实体描述
func buildRootCause(entities []*database.AIOpsIncidentEntity) string {
	for _, e := range entities {
		if e.Role == "root_cause" {
			return fmt.Sprintf(`根因实体:
  %s (角色: root_cause)
  R_local: %.2f | R_final: %.2f`,
				e.EntityKey, e.RLocal, e.RFinal)
		}
	}
	return "根因实体: 未识别"
}

// buildEntities 构建受影响实体列表（上限 MaxEntityEntries）
func buildEntities(entities []*database.AIOpsIncidentEntity) string {
	if len(entities) == 0 {
		return "受影响实体: 无"
	}

	total := len(entities)
	truncated := entities
	if len(truncated) > MaxEntityEntries {
		truncated = truncated[:MaxEntityEntries]
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("受影响实体 (%d 个):\n", total))
	for i, e := range truncated {
		b.WriteString(fmt.Sprintf("  %d. %-30s %-12s R=%.2f\n",
			i+1, e.EntityKey, e.Role, e.RFinal))
	}
	if total > MaxEntityEntries {
		b.WriteString(fmt.Sprintf("  ... 省略 %d 个实体\n", total-MaxEntityEntries))
	}
	return b.String()
}

// buildTimeline 构建时间线描述（上限 MaxTimelineEntries，保留最新的）
func buildTimeline(timeline []*database.AIOpsIncidentTimeline) string {
	if len(timeline) == 0 {
		return "时间线: 无记录"
	}

	total := len(timeline)
	entries := timeline
	if len(entries) > MaxTimelineEntries {
		// 保留最新的条目（时间线按时间排序，尾部更新）
		entries = entries[total-MaxTimelineEntries:]
	}

	var b strings.Builder
	b.WriteString("时间线:\n")
	if total > MaxTimelineEntries {
		b.WriteString(fmt.Sprintf("  ... 省略前 %d 条记录\n", total-MaxTimelineEntries))
	}
	for _, t := range entries {
		ts := t.Timestamp.Format("15:04:05")
		detail := t.Detail
		if len(detail) > 200 {
			detail = detail[:200] + "..."
		}
		b.WriteString(fmt.Sprintf("  %s [%s] %s %s\n",
			ts, t.EventType, t.EntityKey, detail))
	}
	return b.String()
}

// buildHistorical 构建历史事件描述（上限 MaxHistoricalEntries，保留最近的）
func buildHistorical(incidents []*database.AIOpsIncident) string {
	if len(incidents) == 0 {
		return "历史相似事件: 无"
	}

	total := len(incidents)
	entries := incidents
	if len(entries) > MaxHistoricalEntries {
		entries = entries[:MaxHistoricalEntries]
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("历史相似事件 (%d 个):\n", total))
	for i, inc := range entries {
		duration := formatDuration(inc.DurationS)
		b.WriteString(fmt.Sprintf("  %d. %s (%s) — %s, 持续 %s\n",
			i+1, inc.ID, inc.StartedAt.Format("2006-01-02"), inc.RootCause, duration))
	}
	if total > MaxHistoricalEntries {
		b.WriteString(fmt.Sprintf("  ... 省略 %d 个事件\n", total-MaxHistoricalEntries))
	}
	return b.String()
}

// buildOTelContext 从 OTelSnapshot 过滤受影响服务的错误 Traces/Logs/SLO
func buildOTelContext(otel *cluster.OTelSnapshot, entities []*database.AIOpsIncidentEntity) (traces, logs, sloCtx string) {
	if otel == nil {
		return "", "", ""
	}

	// 提取受影响的服务名列表
	affectedServices := extractAffectedServices(entities)
	if len(affectedServices) == 0 {
		return "", "", ""
	}

	// 1. 错误 Traces（Top 5）
	var traceLines []string
	count := 0
	for _, t := range otel.RecentTraces {
		if count >= 5 {
			break
		}
		if t.HasError && containsService(affectedServices, t.RootService) {
			traceLines = append(traceLines, fmt.Sprintf(
				"- [%s] %s %s 耗时=%.0fms Spans=%d 错误=%s",
				t.Timestamp.Format("15:04:05"),
				t.RootService, t.RootOperation,
				t.DurationMs, t.SpanCount, t.ErrorMessage,
			))
			count++
		}
	}
	if len(traceLines) > 0 {
		traces = strings.Join(traceLines, "\n")
	}

	// 2. ERROR 日志（Top 10）
	var logLines []string
	count = 0
	for _, l := range otel.RecentLogs {
		if count >= 10 {
			break
		}
		if l.Severity == "ERROR" && containsService(affectedServices, l.ServiceName) {
			body := l.Body
			if len(body) > 200 {
				body = body[:200] + "..."
			}
			logLines = append(logLines, fmt.Sprintf(
				"- [%s] %s: %s",
				l.Timestamp.Format("15:04:05"),
				l.ServiceName, body,
			))
			count++
		}
	}
	if len(logLines) > 0 {
		logs = strings.Join(logLines, "\n")
	}

	// 3. SLO 摘要
	var sloLines []string
	serviceSet := make(map[string]bool)
	for _, s := range affectedServices {
		serviceSet[s] = true
	}

	// 从实时 SLO 列表中匹配
	for _, s := range otel.SLOIngress {
		if serviceSet[s.ServiceKey] || serviceSet[s.DisplayName] {
			sloLines = append(sloLines, fmt.Sprintf(
				"- %s: SuccessRate=%.2f%% ErrorRate=%.4f%% P99=%.1fms RPS=%.1f",
				s.ServiceKey, s.SuccessRate*100, s.ErrorRate*100, s.P99Ms, s.RPS,
			))
		}
	}
	if len(sloLines) > 0 {
		sloCtx = strings.Join(sloLines, "\n")
	}

	return traces, logs, sloCtx
}

// extractAffectedServices 从事件实体中提取服务名列表
func extractAffectedServices(entities []*database.AIOpsIncidentEntity) []string {
	seen := make(map[string]bool)
	var services []string
	for _, e := range entities {
		// entityKey 格式: "type:namespace/name" 或 "type:name"
		name := extractServiceName(e.EntityKey)
		if name != "" && !seen[name] {
			seen[name] = true
			services = append(services, name)
		}
	}
	return services
}

// extractServiceName 从 entityKey 提取服务/Pod 名称
func extractServiceName(entityKey string) string {
	// 格式: "default/service/geass-gateway" → "geass-gateway"
	// 格式: "default/pod/geass-auth-xxx" → "geass-auth-xxx"
	// 格式: "_cluster/node/worker-3" → "worker-3"
	parts := strings.Split(entityKey, "/")
	if len(parts) < 3 {
		return ""
	}
	return parts[len(parts)-1]
}

// containsService 检查服务名是否在列表中（支持前缀匹配）
func containsService(services []string, target string) bool {
	for _, s := range services {
		if s == target || strings.HasPrefix(target, s) || strings.HasPrefix(s, target) {
			return true
		}
	}
	return false
}

// formatDuration 格式化持续时间（秒 → 人类可读）
func formatDuration(seconds int64) string {
	if seconds <= 0 {
		return "进行中"
	}
	d := time.Duration(seconds) * time.Second
	if d < time.Minute {
		return fmt.Sprintf("%d 秒", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d 分钟", int(d.Minutes()))
	}
	return fmt.Sprintf("%.1f 小时", d.Hours())
}

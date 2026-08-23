// atlhyper_master_v2/ai/prompts/background.go
// background 角色提示词（后台事件摘要）
package prompts

import "fmt"

// IncidentPromptContext LLM 输入上下文（纯文本，不含领域类型）
// 由 aiops 包构建后传入，prompts 包不依赖 aiops
type IncidentPromptContext struct {
	IncidentSummary   string // 事件基本信息
	TimelineText      string // 时间线叙述
	AffectedEntities  string // 受影响实体及其风险评分
	RootCauseEntity   string // 根因实体详情
	HistoricalContext string // 历史相似事件

	// OTel 上下文（Phase 3 新增）
	RecentErrorTraces string // 受影响服务的最近错误 Traces（Top 5）
	RecentErrorLogs   string // 受影响服务的最近 ERROR 日志（Top 10）
	SLOContext        string // 受影响服务的 SLO 变化摘要
}

// PromptPair 系统提示词 + 用户消息
type PromptPair struct {
	System string
	User   string
}

// backgroundSystem background 角色系统提示词
const backgroundSystem = `你是 AtlHyper 平台的 AIOps 分析引擎。你的任务是分析 Kubernetes 集群的运维事件，
提供根因分析、处置建议和历史模式匹配。

要求:
1. 根因分析必须基于提供的数据，不要臆测
2. 处置建议必须具体可执行，按优先级排列
3. 如果有历史相似事件，指出模式和趋势
4. 使用技术精确的语言，避免模糊表述
5. 输出格式严格遵循 JSON Schema
6. 如果提供的数据不足以判断根因（如只有风险分数无具体异常指标），在 rootCauseAnalysis 中说明 "数据不足，建议进一步调查"
7. 如果无历史相似事件，similarPattern 填写 "无匹配的历史模式"

输出格式:
{
  "summary": "一段话概述事件（什么时间，什么实体，什么问题，什么影响）",
  "rootCauseAnalysis": "详细分析根因链路（从源头到影响面）",
  "recommendations": [
    {
      "priority": 1,
      "action": "具体操作步骤",
      "reason": "为什么这样做",
      "impact": "预期效果"
    }
  ],
  "similarPattern": "如果有历史相似事件，描述模式和建议"
}`

// backgroundUserTemplate 用户消息模板
const backgroundUserTemplate = `请分析以下 Kubernetes 集群事件:

%s

请按照指定的 JSON 格式输出分析结果。`

// BuildBackgroundPrompt 构建 background 角色完整 Prompt
func BuildBackgroundPrompt(ctx *IncidentPromptContext) *PromptPair {
	// 基础上下文
	content := ctx.IncidentSummary + "\n\n" +
		ctx.RootCauseEntity + "\n\n" +
		ctx.AffectedEntities + "\n\n" +
		ctx.TimelineText + "\n\n" +
		ctx.HistoricalContext

	// OTel 上下文（非空时追加）
	if ctx.RecentErrorTraces != "" {
		content += "\n\n## 最近错误 Traces\n" + ctx.RecentErrorTraces
	}
	if ctx.RecentErrorLogs != "" {
		content += "\n\n## 最近 ERROR 日志\n" + ctx.RecentErrorLogs
	}
	if ctx.SLOContext != "" {
		content += "\n\n## SLO 指标\n" + ctx.SLOContext
	}

	userContent := fmt.Sprintf(backgroundUserTemplate, content)
	return &PromptPair{
		System: backgroundSystem,
		User:   userContent,
	}
}

// atlhyper_master_v2/ai/analyze.go
// 通用多轮 Tool Calling 循环（非交互式，后台执行）
// 从 aiops/ai/analysis.go 提取的通用逻辑
package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/ai/prompts"
)

const (
	defaultAnalyzeMaxRounds = 8               // 默认最大调查轮次
	defaultAnalyzeTimeout   = 5 * time.Minute // 默认分析超时
	maxToolCallsPerAnalyze  = 5               // 每轮最多 Tool Call 数
)

// Analyze 执行非交互式多轮 Tool Calling 分析
func (s *aiServiceImpl) Analyze(ctx context.Context, req *AnalyzeRequest) (*AnalyzeResult, error) {
	// 默认值
	maxRounds := req.MaxRounds
	if maxRounds <= 0 {
		maxRounds = defaultAnalyzeMaxRounds
	}
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = defaultAnalyzeTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 获取角色配置（Provider + 预算检查）
	role := req.Role
	if role == "" {
		role = RoleAnalysis
	}
	roleCfg, err := s.loadAIConfigForRole(ctx, role)
	if err != nil {
		return nil, fmt.Errorf("AI 配置错误: %w", err)
	}

	// 创建 LLM 客户端
	llmClient, err := llm.NewLLMClient(roleCfg.Config)
	if err != nil {
		s.recordProviderError(ctx, roleCfg.ProviderID, fmt.Sprintf("创建客户端失败: %v", err))
		return nil, fmt.Errorf("创建 LLM 客户端失败: %w", err)
	}
	defer llmClient.Close()

	maxToolResult := toolResultMaxLen(roleCfg.ContextWindow)
	tools := prompts.GetToolDefinitions()

	messages := []llm.Message{
		{Role: "user", Content: req.UserPrompt},
	}

	var steps []AnalyzeStep
	var totalInputTokens, totalOutputTokens, totalToolCalls int

	log.Info("分析开始",
		"cluster", req.ClusterID,
		"role", role,
		"provider", roleCfg.ProviderName,
		"model", roleCfg.Config.Model,
		"maxRounds", maxRounds,
	)

	for round := 1; round <= maxRounds; round++ {
		remaining := maxRounds - round + 1
		roundHint := fmt.Sprintf(
			"\n\n[系统提示] 剩余调查轮次: %d/%d。如果信息已足够，请返回最终分析报告，不要调用 Tool。",
			remaining, maxRounds,
		)

		llmReq := &llm.Request{
			SystemPrompt: req.SystemPrompt + roundHint,
			Messages:     messages,
			Tools:        tools,
		}

		// 最后一轮不提供 tools，强制输出
		if round == maxRounds {
			llmReq.SystemPrompt = req.SystemPrompt + "\n\n[系统提示] 这是最后一轮。请基于已有信息输出最终分析报告。不要再调用 Tool。"
			llmReq.Tools = nil
		}

		stream, err := llmClient.ChatStream(ctx, llmReq)
		if err != nil {
			s.recordProviderError(ctx, roleCfg.ProviderID, fmt.Sprintf("第 %d 轮调用失败: %v", round, err))
			return nil, fmt.Errorf("第 %d 轮 LLM 调用失败: %w", round, err)
		}

		// 收集响应
		text, toolCalls, usage := collectStreamResponse(stream)

		if usage != nil {
			totalInputTokens += usage.InputTokens
			totalOutputTokens += usage.OutputTokens
		}

		log.Debug("分析轮次完成",
			"cluster", req.ClusterID, "round", round,
			"text_len", len(text), "tool_calls", len(toolCalls))

		// 没有 Tool Call → 分析结束
		if len(toolCalls) == 0 {
			s.clearProviderError(ctx, roleCfg.ProviderID)
			s.RecordUsage(ctx, role, roleCfg.ProviderID, totalInputTokens, totalOutputTokens)
			log.Info("分析完成",
				"cluster", req.ClusterID,
				"rounds", round,
				"tools", totalToolCalls,
				"tokens", totalInputTokens+totalOutputTokens,
			)
			return &AnalyzeResult{
				Response:     text,
				ToolCalls:    totalToolCalls,
				InputTokens:  totalInputTokens,
				OutputTokens: totalOutputTokens,
				ProviderName: roleCfg.ProviderName,
				Model:        roleCfg.Config.Model,
				Steps:        steps,
			}, nil
		}

		// 限制每轮 Tool Call 数量
		if len(toolCalls) > maxToolCallsPerAnalyze {
			toolCalls = toolCalls[:maxToolCallsPerAnalyze]
		}
		totalToolCalls += len(toolCalls)

		step := AnalyzeStep{
			Round:    round,
			Thinking: text,
		}

		messages = append(messages, llm.Message{
			Role:      "assistant",
			Content:   text,
			ToolCalls: toolCalls,
		})

		for _, tc := range toolCalls {
			result, err := s.executor.Execute(ctx, req.ClusterID, &tc)
			if err != nil {
				result = fmt.Sprintf("执行失败: %v", err)
			}

			step.ToolCalls = append(step.ToolCalls, ToolCallRecord{
				Tool:          tc.Name,
				Params:        tc.Params,
				ResultSummary: truncate(result, 500),
			})

			messages = append(messages, llm.Message{
				Role: "tool",
				ToolResult: &llm.ToolResult{
					CallID:  tc.ID,
					Name:    tc.Name,
					Content: truncate(result, maxToolResult),
				},
			})
		}

		steps = append(steps, step)
	}

	// 循环用尽
	s.RecordUsage(ctx, role, roleCfg.ProviderID, totalInputTokens, totalOutputTokens)
	return &AnalyzeResult{
		ToolCalls:    totalToolCalls,
		InputTokens:  totalInputTokens,
		OutputTokens: totalOutputTokens,
		ProviderName: roleCfg.ProviderName,
		Model:        roleCfg.Config.Model,
		Steps:        steps,
	}, nil
}

// collectStreamResponse 收集流式响应（不推送 SSE，静默收集）
func collectStreamResponse(stream <-chan *llm.Chunk) (string, []llm.ToolCall, *llm.Usage) {
	var text strings.Builder
	var toolCalls []llm.ToolCall

	for chunk := range stream {
		switch chunk.Type {
		case llm.ChunkText:
			text.WriteString(chunk.Content)
		case llm.ChunkToolCall:
			if chunk.ToolCall != nil {
				toolCalls = append(toolCalls, *chunk.ToolCall)
			}
		case llm.ChunkDone:
			return text.String(), toolCalls, chunk.Usage
		case llm.ChunkError:
			if chunk.Error != nil {
				log.Warn("analyze LLM 流错误", "err", chunk.Error)
			}
			return text.String(), toolCalls, nil
		}
	}
	return text.String(), toolCalls, nil
}

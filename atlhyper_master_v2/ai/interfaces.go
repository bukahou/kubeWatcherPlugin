// atlhyper_master_v2/ai/interfaces.go
// AIService 对外接口 + 类型定义
package ai

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
)

// AIService AI 服务接口
type AIService interface {
	// CreateConversation 创建对话
	CreateConversation(ctx context.Context, userID int64, clusterID, title string) (*Conversation, error)

	// Chat 发送消息并获取流式响应
	// 返回 ChatChunk channel，通过 SSE 推送给前端
	Chat(ctx context.Context, req *ChatRequest) (<-chan *ChatChunk, error)

	// GetConversations 获取用户对话列表
	GetConversations(ctx context.Context, userID int64, limit, offset int) ([]*Conversation, error)

	// GetMessages 获取对话历史消息
	GetMessages(ctx context.Context, conversationID int64) ([]*Message, error)

	// DeleteConversation 删除对话及其所有消息
	DeleteConversation(ctx context.Context, conversationID int64) error

	// RegisterTool 注册自定义 Tool（AIOps 等扩展模块使用）
	RegisterTool(name string, handler ToolHandler)

	// Analyze 非交互式多轮 Tool Calling 分析（后台执行，无 SSE）
	Analyze(ctx context.Context, req *AnalyzeRequest) (*AnalyzeResult, error)

	// Complete 单轮 LLM 调用（无 Tool，用于 background 摘要等）
	Complete(ctx context.Context, req *CompleteRequest) (*CompleteResult, error)

	// GetToolExecuteFunc 获取 Tool 执行函数（供 analysis 深度分析使用）
	GetToolExecuteFunc() func(ctx context.Context, clusterID string, tc *llm.ToolCall) (string, error)

	// GetToolDefinitions 获取 Tool 定义列表
	GetToolDefs() []llm.ToolDefinition
}

// ChatRequest 发送消息请求
type ChatRequest struct {
	ConversationID int64  // 对话 ID
	ClusterID      string // 目标集群
	UserID         int64  // 用户 ID
	Message        string // 用户消息
}

// ChatChunk SSE 流式响应块
type ChatChunk struct {
	Type    string     `json:"type"`              // text / tool_call / tool_result / done / error
	Content string     `json:"content,omitempty"` // 文本内容
	Tool    string     `json:"tool,omitempty"`    // tool 名称
	Params  string     `json:"params,omitempty"`  // tool 参数 JSON
	Stats   *ChatStats `json:"stats,omitempty"`   // 统计信息（done 时返回）
}

// ChatStats 对话统计信息
type ChatStats struct {
	Rounds         int `json:"rounds"`         // 思考轮次（AI 调用次数）
	TotalToolCalls int `json:"totalToolCalls"` // 总指令数（所有轮次的 Tool 调用总数）
	InputTokens    int `json:"inputTokens"`    // 输入 Token 数
	OutputTokens   int `json:"outputTokens"`   // 输出 Token 数
}

// Conversation 对话
type Conversation struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"userId"`
	ClusterID    string `json:"clusterId"`
	Title        string `json:"title"`
	MessageCount int    `json:"messageCount"`
	// 累计统计
	TotalInputTokens  int64     `json:"totalInputTokens"`  // 累计输入 Token
	TotalOutputTokens int64     `json:"totalOutputTokens"` // 累计输出 Token
	TotalToolCalls    int       `json:"totalToolCalls"`    // 累计指令数
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// Message 消息
type Message struct {
	ID             int64     `json:"id"`
	ConversationID int64     `json:"conversationId"`
	Role           string    `json:"role"` // user / assistant / tool
	Content        string    `json:"content"`
	ToolCalls      string    `json:"toolCalls,omitempty"` // JSON
	CreatedAt      time.Time `json:"createdAt"`
}

// ==================== Analyze（多轮 Tool Calling）====================

// AnalyzeRequest 非交互式分析请求
type AnalyzeRequest struct {
	ClusterID    string        // 目标集群
	Role         string        // 角色名（用于预算扣减，如 "analysis"）
	SystemPrompt string        // 调用方提供的系统提示词
	UserPrompt   string        // 调用方构建的用户提示词
	MaxRounds    int           // 最大 Tool 调用轮次（默认 8）
	Timeout      time.Duration // 全局超时（默认 5min）
}

// AnalyzeResult 分析结果
type AnalyzeResult struct {
	Response     string        // LLM 最终文本输出
	ToolCalls    int           // 总 Tool 调用次数
	InputTokens  int           // 总输入 Token
	OutputTokens int           // 总输出 Token
	ProviderName string        // Provider 名称
	Model        string        // 模型名称
	Steps        []AnalyzeStep // 调查步骤记录
}

// AnalyzeStep 单轮调查步骤
type AnalyzeStep struct {
	Round     int              `json:"round"`
	Thinking  string           `json:"thinking"`
	ToolCalls []ToolCallRecord `json:"toolCalls"`
}

// ToolCallRecord Tool 调用记录
type ToolCallRecord struct {
	Tool          string `json:"tool"`
	Params        string `json:"params"`
	ResultSummary string `json:"resultSummary"`
}

// ==================== Complete（单轮 LLM 调用）====================

// CompleteRequest 单轮 LLM 调用请求（无 Tool）
type CompleteRequest struct {
	Role         string // 角色名（用于角色路由、预算扣减，如 "background"）
	SystemPrompt string
	UserPrompt   string
}

// CompleteResult 单轮 LLM 调用结果
type CompleteResult struct {
	Response     string
	InputTokens  int
	OutputTokens int
	ProviderID   int64  // Provider ID（供调用方记录）
	ProviderName string // Provider 名称（供调用方日志）
	Model        string // 模型名称
}

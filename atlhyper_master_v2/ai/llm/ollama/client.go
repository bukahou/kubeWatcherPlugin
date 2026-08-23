// atlhyper_master_v2/ai/llm/ollama/client.go
// Ollama LLM 客户端（原生 /api/chat 端点）
// 使用 Ollama 原生 API 而非 OpenAI 兼容层，避免兼容性问题
package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/common/logger"
)

var log = logger.Module("Ollama")

func init() {
	llm.Register("ollama", func(cfg llm.Config) (llm.LLMClient, error) {
		if cfg.BaseURL == "" {
			return nil, fmt.Errorf("ollama requires base_url (e.g. http://ollama.atlhyper.svc:11434)")
		}
		if cfg.Model == "" {
			cfg.Model = "qwen2.5:14b"
		}
		return NewClient(cfg.BaseURL, cfg.Model)
	})
}

// Client Ollama 原生客户端
type Client struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewClient 创建 Ollama 客户端
// baseURL 只需要 host:port（如 http://192.168.0.121:11434）
// 会自动清理尾部多余路径（如 /v1）
func NewClient(baseURL, model string) (*Client, error) {
	// 清理尾部路径：用户可能填了 http://host:11434/v1 或 http://host:11434/
	cleaned := strings.TrimRight(baseURL, "/")
	cleaned = strings.TrimSuffix(cleaned, "/v1")
	cleaned = strings.TrimSuffix(cleaned, "/api")
	return &Client{
		baseURL:    cleaned,
		model:      model,
		httpClient: &http.Client{},
	}, nil
}

// ==================== Native API Types ====================

// chatRequest Ollama /api/chat 请求
type chatRequest struct {
	Model    string         `json:"model"`
	Messages []messageParam `json:"messages"`
	Stream   bool           `json:"stream"`
	Tools    []toolParam    `json:"tools,omitempty"`
}

type messageParam struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []toolCall `json:"tool_calls,omitempty"`
}

type toolCall struct {
	Function toolCallFunction `json:"function"`
}

type toolCallFunction struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type toolParam struct {
	Type     string      `json:"type"`
	Function functionDef `json:"function"`
}

type functionDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

// streamChunk Ollama NDJSON 流式响应
type streamChunk struct {
	Model     string       `json:"model"`
	CreatedAt string       `json:"created_at"`
	Message   chunkMessage `json:"message"`
	Done      bool         `json:"done"`
	// 最终 chunk (done=true) 包含统计
	PromptEvalCount int `json:"prompt_eval_count"`
	EvalCount       int `json:"eval_count"`
}

type chunkMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []toolCall `json:"tool_calls,omitempty"`
}

// ==================== LLMClient Implementation ====================

// ChatStream 发送流式对话请求
func (c *Client) ChatStream(ctx context.Context, req *llm.Request) (<-chan *llm.Chunk, error) {
	apiReq := chatRequest{
		Model:  c.model,
		Stream: true,
	}

	// 消息转换
	apiReq.Messages = convertMessages(req.Messages, req.SystemPrompt)

	// 工具转换
	if len(req.Tools) > 0 {
		apiReq.Tools = convertTools(req.Tools)
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := c.baseURL + "/api/chat"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API error: %d - %s", resp.StatusCode, string(errBody))
	}

	ch := make(chan *llm.Chunk, 32)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		c.readStream(ctx, resp.Body, ch)
	}()

	return ch, nil
}

// readStream 读取 Ollama NDJSON 流式响应
func (c *Client) readStream(ctx context.Context, body io.Reader, ch chan<- *llm.Chunk) {
	scanner := bufio.NewScanner(body)
	// Ollama 响应可能包含较长的 tool call JSON
	scanner.Buffer(make([]byte, 0, 64*1024), 512*1024)

	var toolCallCounter int

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			ch <- &llm.Chunk{Type: llm.ChunkError, Error: ctx.Err()}
			return
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			log.Error("JSON 解析失败", "err", err, "line", line)
			continue
		}

		// 文本内容
		if chunk.Message.Content != "" {
			ch <- &llm.Chunk{Type: llm.ChunkText, Content: chunk.Message.Content}
		}

		// Tool Calls
		for _, tc := range chunk.Message.ToolCalls {
			toolCallCounter++
			paramsJSON, _ := json.Marshal(tc.Function.Arguments)
			ch <- &llm.Chunk{
				Type: llm.ChunkToolCall,
				ToolCall: &llm.ToolCall{
					ID:     fmt.Sprintf("ollama_call_%d", toolCallCounter),
					Name:   tc.Function.Name,
					Params: string(paramsJSON),
				},
			}
		}

		// 完成
		if chunk.Done {
			ch <- &llm.Chunk{
				Type: llm.ChunkDone,
				Usage: &llm.Usage{
					InputTokens:  chunk.PromptEvalCount,
					OutputTokens: chunk.EvalCount,
				},
			}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		log.Error("Scanner 读取错误", "err", err)
		ch <- &llm.Chunk{Type: llm.ChunkError, Error: err}
	}
}

// Close 关闭客户端
func (c *Client) Close() error {
	return nil
}

// ==================== Helpers ====================

// convertMessages 将 llm.Message 转换为 Ollama 格式
func convertMessages(msgs []llm.Message, systemPrompt string) []messageParam {
	var result []messageParam

	if systemPrompt != "" {
		result = append(result, messageParam{Role: "system", Content: systemPrompt})
	}

	for _, msg := range msgs {
		switch msg.Role {
		case "user":
			result = append(result, messageParam{Role: "user", Content: msg.Content})

		case "assistant":
			param := messageParam{Role: "assistant", Content: msg.Content}
			for _, tc := range msg.ToolCalls {
				var args map[string]any
				if err := json.Unmarshal([]byte(tc.Params), &args); err != nil {
					log.Warn("Tool Call 参数解析失败，使用空参数", "tool", tc.Name, "err", err)
					args = map[string]any{}
				}
				param.ToolCalls = append(param.ToolCalls, toolCall{
					Function: toolCallFunction{Name: tc.Name, Arguments: args},
				})
			}
			result = append(result, param)

		case "tool":
			if msg.ToolResult != nil {
				result = append(result, messageParam{
					Role:    "tool",
					Content: msg.ToolResult.Content,
				})
			}
		}
	}

	return result
}

// convertTools 将 llm.ToolDefinition 转换为 Ollama 格式
func convertTools(tools []llm.ToolDefinition) []toolParam {
	var result []toolParam
	for _, t := range tools {
		var params any
		if err := json.Unmarshal(t.Parameters, &params); err != nil {
			log.Warn("Tool 参数 Schema 解析失败", "tool", t.Name, "err", err)
			params = map[string]any{"type": "object"}
		}
		result = append(result, toolParam{
			Type: "function",
			Function: functionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}
	return result
}

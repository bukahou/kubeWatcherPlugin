// atlhyper_master_v2/ai/llm/openai/client.go
// OpenAI LLM 客户端实现
package openai

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

var log = logger.Module("OpenAI")

const (
	defaultEndpoint = "https://api.openai.com/v1/chat/completions"
)

func init() {
	llm.Register("openai", func(cfg llm.Config) (llm.LLMClient, error) {
		return NewOpenAIClient(cfg.APIKey, cfg.Model, cfg.BaseURL)
	})
}

// Client OpenAI 客户端
type Client struct {
	apiKey     string
	model      string
	endpoint   string
	httpClient *http.Client
}

// NewOpenAIClient 创建 OpenAI 客户端
// baseURL 可选，为空时使用 OpenAI 官方地址
func NewOpenAIClient(apiKey, model, baseURL string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("openai api key is required")
	}
	if model == "" {
		model = "gpt-4o"
	}
	endpoint := defaultEndpoint
	if baseURL != "" {
		endpoint = strings.TrimRight(baseURL, "/") + "/v1/chat/completions"
	}
	return &Client{
		apiKey:     apiKey,
		model:      model,
		endpoint:   endpoint,
		httpClient: &http.Client{},
	}, nil
}

// ==================== API Request/Response Types ====================

// chatRequest OpenAI Chat Completions API リクエスト
type chatRequest struct {
	Model         string         `json:"model"`
	Messages      []messageParam `json:"messages"`
	Tools         []toolParam    `json:"tools,omitempty"`
	Stream        bool           `json:"stream"`
	StreamOptions *streamOptions `json:"stream_options,omitempty"`
}

// streamOptions ストリームオプション
type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// messageParam メッセージパラメータ
type messageParam struct {
	Role       string          `json:"role"`
	Content    any             `json:"content,omitempty"`
	ToolCalls  []toolCallParam `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

// toolCallParam ツール呼び出しパラメータ
type toolCallParam struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function functionCall `json:"function"`
}

// functionCall 関数呼び出し
type functionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// toolParam ツール定義
type toolParam struct {
	Type     string      `json:"type"`
	Function functionDef `json:"function"`
}

// functionDef 関数定義
type functionDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

// streamChunk SSE チャンク
type streamChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Choices []streamChoice `json:"choices"`
	Usage   *usageInfo     `json:"usage,omitempty"`
}

// usageInfo Token 使用量
type usageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

// streamChoice ストリームチョイス
type streamChoice struct {
	Index        int          `json:"index"`
	Delta        deltaContent `json:"delta"`
	FinishReason string       `json:"finish_reason,omitempty"`
}

// deltaContent デルタコンテンツ
type deltaContent struct {
	Role      string          `json:"role,omitempty"`
	Content   string          `json:"content,omitempty"`
	ToolCalls []toolCallDelta `json:"tool_calls,omitempty"`
}

// toolCallDelta ツール呼び出しデルタ
type toolCallDelta struct {
	Index    int           `json:"index"`
	ID       string        `json:"id,omitempty"`
	Type     string        `json:"type,omitempty"`
	Function functionDelta `json:"function,omitempty"`
}

// functionDelta 関数デルタ
type functionDelta struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// ==================== LLMClient Implementation ====================

// ChatStream 发送流式对话请求
func (c *Client) ChatStream(ctx context.Context, req *llm.Request) (<-chan *llm.Chunk, error) {
	// リクエスト構築
	apiReq := chatRequest{
		Model:         c.model,
		Stream:        true,
		StreamOptions: &streamOptions{IncludeUsage: true},
	}

	// メッセージ変換
	apiReq.Messages = convertMessages(req.Messages, req.SystemPrompt)

	// ツール変換
	if len(req.Tools) > 0 {
		apiReq.Tools = convertTools(req.Tools)
	}

	// JSON エンコード
	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// HTTP リクエスト作成
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	// リクエスト送信
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai API error: %d - %s", resp.StatusCode, string(body))
	}

	// ストリーム読み取り開始
	ch := make(chan *llm.Chunk, 32)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		c.readStream(ctx, resp.Body, ch)
	}()

	return ch, nil
}

// readStream SSE ストリームを読み取り
func (c *Client) readStream(ctx context.Context, body io.Reader, ch chan<- *llm.Chunk) {
	scanner := bufio.NewScanner(body)

	// 現在処理中のツール呼び出し（インデックスでマップ）
	toolCalls := make(map[int]*llm.ToolCall)
	var lastUsage *llm.Usage

	for scanner.Scan() {
		line := scanner.Text()

		// SSE データ行のみ処理
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			// 未送信のツール呼び出しを送信
			for _, tc := range toolCalls {
				select {
				case ch <- &llm.Chunk{Type: llm.ChunkToolCall, ToolCall: tc}:
				case <-ctx.Done():
					ch <- &llm.Chunk{Type: llm.ChunkError, Error: ctx.Err()}
					return
				}
			}
			ch <- &llm.Chunk{Type: llm.ChunkDone, Usage: lastUsage}
			return
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			log.Error("JSON parse error", "err", err, "data", data)
			continue
		}

		// 记录 usage（stream_options.include_usage=true 时最后一个 chunk 会包含）
		if chunk.Usage != nil {
			lastUsage = &llm.Usage{
				InputTokens:  chunk.Usage.PromptTokens,
				OutputTokens: chunk.Usage.CompletionTokens,
			}
		}

		for _, choice := range chunk.Choices {
			// テキストコンテンツ
			if choice.Delta.Content != "" {
				select {
				case ch <- &llm.Chunk{Type: llm.ChunkText, Content: choice.Delta.Content}:
				case <-ctx.Done():
					ch <- &llm.Chunk{Type: llm.ChunkError, Error: ctx.Err()}
					return
				}
			}

			// ツール呼び出しデルタ
			for _, tcDelta := range choice.Delta.ToolCalls {
				tc, ok := toolCalls[tcDelta.Index]
				if !ok {
					tc = &llm.ToolCall{}
					toolCalls[tcDelta.Index] = tc
				}

				if tcDelta.ID != "" {
					tc.ID = tcDelta.ID
				}
				if tcDelta.Function.Name != "" {
					tc.Name = tcDelta.Function.Name
				}
				if tcDelta.Function.Arguments != "" {
					tc.Params += tcDelta.Function.Arguments
				}
			}

			// 終了理由
			if choice.FinishReason == "tool_calls" {
				// ツール呼び出し完了時に送信
				for _, tc := range toolCalls {
					select {
					case ch <- &llm.Chunk{Type: llm.ChunkToolCall, ToolCall: tc}:
					case <-ctx.Done():
						ch <- &llm.Chunk{Type: llm.ChunkError, Error: ctx.Err()}
						return
					}
				}
				toolCalls = make(map[int]*llm.ToolCall)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Error("Scanner error", "err", err)
		ch <- &llm.Chunk{Type: llm.ChunkError, Error: err}
	}
}

// Close 关闭客户端
func (c *Client) Close() error {
	return nil
}

// ==================== Helpers ====================

// convertMessages llm.Message を OpenAI 形式に変換
func convertMessages(msgs []llm.Message, systemPrompt string) []messageParam {
	var result []messageParam

	// システムプロンプト
	if systemPrompt != "" {
		result = append(result, messageParam{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	for _, msg := range msgs {
		switch msg.Role {
		case "user":
			result = append(result, messageParam{
				Role:    "user",
				Content: msg.Content,
			})

		case "assistant":
			param := messageParam{
				Role: "assistant",
			}
			if msg.Content != "" {
				param.Content = msg.Content
			}
			for _, tc := range msg.ToolCalls {
				param.ToolCalls = append(param.ToolCalls, toolCallParam{
					ID:   tc.ID,
					Type: "function",
					Function: functionCall{
						Name:      tc.Name,
						Arguments: tc.Params,
					},
				})
			}
			result = append(result, param)

		case "tool":
			if msg.ToolResult != nil {
				result = append(result, messageParam{
					Role:       "tool",
					Content:    msg.ToolResult.Content,
					ToolCallID: msg.ToolResult.CallID,
				})
			}
		}
	}

	return result
}

// convertTools llm.ToolDefinition を OpenAI 形式に変換
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

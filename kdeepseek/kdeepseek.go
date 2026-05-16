// Package kdeepseek 提供 DeepSeek API 的 Go 客户端封装。
// 支持对话补全、思考模式、工具调用、流式输出、多轮对话等功能。
package kdeepseek

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// 思考模式常量
const (
	ThinkingEnabled  = "enabled"
	ThinkingDisabled = "disabled"
)

// 思考强度常量
const (
	ReasoningEffortHigh = "high"
	ReasoningEffortMax  = "max"
)

// Message 表示对话中的一条消息。
// Role 可以是 system、user、assistant 或 tool。
type Message struct {
	Role             string     `json:"role"`
	Content          string     `json:"content"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
	Name             string     `json:"name,omitempty"`
}

// Tool 表示模型可调用的工具定义。
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction 描述一个可被模型调用的函数。
type ToolFunction struct {
	Description string `json:"description,omitempty"`
	Name        string `json:"name"`
	Parameters  any    `json:"parameters,omitempty"`
	Strict      bool   `json:"strict,omitempty"`
}

// ToolCall 表示响应中的工具调用。
type ToolCall struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function ToolFunc `json:"function"`
}

// ToolFunc 表示工具调用的函数名和参数。
type ToolFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// CompletionTokensDetails 提供完成 token 的细分统计。
type CompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}

// ResponseFormat 指定模型输出格式。
type ResponseFormat struct {
	Type string `json:"type"`
}

// ThinkingConfig 控制思考模式的启用/禁用。
type ThinkingConfig struct {
	Type string `json:"type"`
}

// StreamOptions 流式选项，仅在 stream: true 时可用。
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// ChatCompletionRequest 是对话补全的请求结构。
type ChatCompletionRequest struct {
	Messages        []Message       `json:"messages"`
	Model           string          `json:"model"`
	MaxTokens       int             `json:"max_tokens,omitempty"`
	ResponseFormat  *ResponseFormat `json:"response_format,omitempty"`
	Stop            []string        `json:"stop,omitempty"`
	Stream          bool            `json:"stream,omitempty"`
	StreamOptions   *StreamOptions  `json:"stream_options,omitempty"`
	Temperature     float64         `json:"temperature,omitempty"`
	TopP            float64         `json:"top_p,omitempty"`
	Thinking        *ThinkingConfig `json:"thinking,omitempty"`
	ReasoningEffort string          `json:"reasoning_effort,omitempty"`
	Tools           []Tool          `json:"tools,omitempty"`
	ToolChoice      any             `json:"tool_choice,omitempty"`
	Logprobs        bool            `json:"logprobs,omitempty"`
	TopLogprobs     int             `json:"top_logprobs,omitempty"`
	UserID          string          `json:"user_id,omitempty"`
}

// Choice 表示一个补全结果。
type Choice struct {
	FinishReason string          `json:"finish_reason"`
	Index        int             `json:"index"`
	Message      ResponseMessage `json:"message"`
	Logprobs     *LogprobsInfo   `json:"logprobs"`
}

// ResponseMessage 表示助手返回的消息。
type ResponseMessage struct {
	Role             string     `json:"role"`
	Content          string     `json:"content"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

// LogprobsInfo 包含输出 token 的对数概率信息。
type LogprobsInfo struct {
	Content          []LogprobToken `json:"content"`
	ReasoningContent []LogprobToken `json:"reasoning_content"`
}

// LogprobToken 表示单个 token 的对数概率信息。
type LogprobToken struct {
	Token       string       `json:"token"`
	Logprob     float64      `json:"logprob"`
	Bytes       []int        `json:"bytes"`
	TopLogprobs []LogprobTop `json:"top_logprobs"`
}

// LogprobTop 表示概率最高的备选 token。
type LogprobTop struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
	Bytes   []int   `json:"bytes"`
}

// Usage 表示 token 用量统计。
type Usage struct {
	PromptTokens            int                      `json:"prompt_tokens"`
	PromptCacheHitTokens    int                      `json:"prompt_cache_hit_tokens"`
	PromptCacheMissTokens   int                      `json:"prompt_cache_miss_tokens"`
	CompletionTokens        int                      `json:"completion_tokens"`
	TotalTokens             int                      `json:"total_tokens"`
	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
}

// ChatCompletionResponse 是非流式对话补全的响应结构。
type ChatCompletionResponse struct {
	ID                string   `json:"id"`
	Choices           []Choice `json:"choices"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Object            string   `json:"object"`
	SystemFingerprint string   `json:"system_fingerprint"`
	Usage             Usage    `json:"usage"`
}

// StreamDelta 表示流式响应中每个 chunk 的增量内容。
type StreamDelta struct {
	Role             string           `json:"role,omitempty"`
	Content          string           `json:"content,omitempty"`
	ReasoningContent string           `json:"reasoning_content,omitempty"`
	ToolCalls        []StreamToolCall `json:"tool_calls,omitempty"`
}

// StreamToolCall 表示流式响应中工具调用的一个增量片段。
type StreamToolCall struct {
	Index    *int               `json:"index"`
	ID       string             `json:"id,omitempty"`
	Type     string             `json:"type,omitempty"`
	Function StreamToolCallFunc `json:"function"`
}

// StreamToolCallFunc 表示流式工具调用的函数信息。
type StreamToolCallFunc struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// StreamChoice 表示流式响应中的一个选项。
type StreamChoice struct {
	Index        int         `json:"index"`
	Delta        StreamDelta `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

// StreamChunk 表示流式响应的一个 SSE 事件块。
type StreamChunk struct {
	ID                string         `json:"id"`
	Object            string         `json:"object"`
	Created           int64          `json:"created"`
	Model             string         `json:"model"`
	SystemFingerprint string         `json:"system_fingerprint"`
	Choices           []StreamChoice `json:"choices"`
	Usage             *Usage         `json:"usage,omitempty"`
}

// ResponseToMessage 将 ResponseMessage 转为可追加到历史的消息。
func ResponseToMessage(msg ResponseMessage) Message {
	return Message{
		Role:             msg.Role,
		Content:          msg.Content,
		ReasoningContent: msg.ReasoningContent,
		ToolCalls:        msg.ToolCalls,
	}
}

// Client 是与 DeepSeek API 交互的 HTTP 客户端。
type Client struct {
	token           string
	baseURL         string
	httpClient      *http.Client
	model           string
	temperature     float64
	system          string
	reasoningEffort string
}

// ClientOption 是配置 Client 的函数类型。
type ClientOption func(*Client)

// WithModel 设置默认模型。
func WithModel(model string) ClientOption {
	return func(c *Client) {
		c.model = model
	}
}

// WithTemperature 设置默认采样温度，范围 0–2。
func WithTemperature(temp float64) ClientOption {
	return func(c *Client) {
		c.temperature = temp
	}
}

// WithBaseURL 设置自定义 API 基础 URL。
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithHTTPClient 设置自定义 HTTP 客户端。
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithSystem 设置默认系统提示，供 SimpleChat/ThinkingChat 使用。
func WithSystem(system string) ClientOption {
	return func(c *Client) {
		c.system = system
	}
}

// WithThinking 设置默认启用思考模式。
func WithThinking(enabled bool) ClientOption {
	return func(c *Client) {
		if enabled {
			c.reasoningEffort = ReasoningEffortHigh
		} else {
			c.reasoningEffort = ""
		}
	}
}

// New 创建新的 DeepSeek 客户端。
func New(token string, opts ...ClientOption) *Client {
	c := &Client{
		token:       token,
		baseURL:     "https://api.deepseek.com",
		httpClient:  &http.Client{Timeout: 300 * time.Second},
		model:       "deepseek-v4-flash",
		temperature: 1,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// do 发送 HTTP 请求并检查状态码，返回响应体。所有 HTTP 调用共享此管道。
func (c *Client) do(ctx context.Context, req *ChatCompletionRequest, accept string) (*http.Response, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/chat/completions", c.baseURL),
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", accept)
	httpReq.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

func (c *Client) chatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("messages cannot be empty")
	}
	if req.Model == "" {
		req.Model = c.model
	}
	if req.Temperature == 0 {
		req.Temperature = c.temperature
	}

	resp, err := c.do(ctx, req, "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}
	return &response, nil
}

// ChatCompletion 发送非流式对话补全请求。
func (c *Client) ChatCompletion(req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	return c.chatCompletion(context.Background(), req)
}

// ChatCompletionStream 发送流式对话补全请求，返回 chunk 通道和错误通道。
// ctx 取消会立即终止流并关闭通道。
func (c *Client) ChatCompletionStream(ctx context.Context, req *ChatCompletionRequest) (<-chan StreamChunk, <-chan error) {
	chunkCh := make(chan StreamChunk, 16)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunkCh)
		defer close(errCh)

		if len(req.Messages) == 0 {
			errCh <- fmt.Errorf("messages cannot be empty")
			return
		}
		if req.Model == "" {
			req.Model = c.model
		}
		if req.Temperature == 0 {
			req.Temperature = c.temperature
		}

		req.Stream = true

		resp, err := c.do(ctx, req, "text/event-stream")
		if err != nil {
			errCh <- err
			return
		}
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				return
			}

			var chunk StreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			select {
			case chunkCh <- chunk:
			case <-ctx.Done():
				return
			}
		}

		if err := scanner.Err(); err != nil {
			errCh <- fmt.Errorf("error reading stream: %w", err)
		}
	}()

	return chunkCh, errCh
}

// SimpleChat 发送一条用户消息并返回助手回复文本。
// 适用于单轮对话。多轮对话请使用 Chat。
func (c *Client) SimpleChat(prompt string) (string, error) {
	chat := &Chat{
		client: c,
		config: ChatConfig{
			Model:           c.model,
			Temperature:     c.temperature,
			ReasoningEffort: c.reasoningEffort,
		},
	}
	if c.system != "" {
		chat.history = []Message{{Role: "system", Content: c.system}}
	}
	return chat.Send(prompt)
}

// ThinkingChat 在思考模式下发送消息，同时返回回复内容和推理内容。
// 适用于单轮对话。多轮对话请使用 Chat。
func (c *Client) ThinkingChat(prompt string) (content string, reasoning string, err error) {
	chat := &Chat{
		client: c,
		config: ChatConfig{
			Model:       c.model,
			Temperature: c.temperature,
		},
	}
	if c.system != "" {
		chat.history = []Message{{Role: "system", Content: c.system}}
	}
	return chat.SendWithThinking(prompt)
}

// JSONResponseFormat 返回设置为 json_object 的 ResponseFormat，用于约束模型输出合法 JSON。
func JSONResponseFormat() *ResponseFormat {
	return &ResponseFormat{Type: "json_object"}
}

// JSONPrompt 构建指导模型输出 JSON 格式的 system prompt。
// systemPrompt 描述输出格式要求；exampleInput 为示例输入；
// exampleOutput 为期望的 JSON 输出样例（任意类型，自动缩进序列化）。
func JSONPrompt(systemPrompt, exampleInput string, exampleOutput any) (string, error) {
	jsonBytes, err := json.MarshalIndent(exampleOutput, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error marshaling example output: %w", err)
	}
	return fmt.Sprintf("%s\n\nEXAMPLE INPUT:\n%s\n\nEXAMPLE JSON OUTPUT:\n```json\n%s\n```",
		systemPrompt, exampleInput, string(jsonBytes)), nil
}

// JSONCompletion 发送一条消息并确保模型输出合法 JSON 字符串。
// systemPrompt 为描述输出要求的系统提示（留空则不带 system 消息）；
// prompt 为用户输入。
func (c *Client) JSONCompletion(systemPrompt, prompt string) (string, error) {
	chat := &Chat{
		client: c,
		config: ChatConfig{
			Model:          c.model,
			Temperature:    c.temperature,
			ResponseFormat: &ResponseFormat{Type: "json_object"},
		},
	}
	if systemPrompt != "" {
		chat.history = []Message{{Role: "system", Content: systemPrompt}}
	}
	return chat.Send(prompt)
}

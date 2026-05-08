// Package kdeepseek 提供 DeepSeek API 的 Go 客户端封装。
// 支持对话补全、思考模式、工具调用、流式输出、JSON 结构化输出等功能。
package kdeepseek

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client 是与 DeepSeek API 交互的客户端。
type Client struct {
	token       string
	baseURL     string
	httpClient  *http.Client
	model       string // 默认模型
	system      string // 默认系统提示
	temperature float64
	thinking    bool // 是否默认启用思考模式
}

// SetModel 设置当前使用的模型。
func (c *Client) SetModel(model string) {
	c.model = model
}

// GetModel 获取当前使用的模型。
func (c *Client) GetModel() string {
	return c.model
}

// ClientOption 是配置客户端的函数类型。
type ClientOption func(*Client)

// WithTemperature 设置采样温度，范围 0–2，默认 1。值越高输出越随机。
func WithTemperature(temp float64) ClientOption {
	return func(c *Client) {
		c.temperature = temp
	}
}

// NewClient 创建新的 DeepSeek 客户端。
// token 为 API 密钥；options 为可选配置项。
func NewClient(token string, options ...ClientOption) *Client {
	httpClient := &http.Client{
		Timeout: 300 * time.Second,
	}
	c := &Client{
		token:       token,
		baseURL:     "https://api.deepseek.com",
		httpClient:  httpClient,
		model:       "deepseek-v4-flash",
		temperature: 1,
	}

	for _, opt := range options {
		opt(c)
	}
	return c
}

// WithSystem 设置默认的系统提示，供 SimpleChat 等方法自动使用。
func WithSystem(system string) ClientOption {
	return func(c *Client) {
		c.system = system
	}
}

// WithHTTPClient 设置自定义 HTTP 客户端。
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithBaseURL 设置自定义 API 基础 URL。
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithModel 设置默认模型。
func WithModel(model string) ClientOption {
	return func(c *Client) {
		c.model = model
	}
}

// WithThinking 设置是否默认启用思考模式。
func WithThinking(enabled bool) ClientOption {
	return func(c *Client) {
		c.thinking = enabled
	}
}

// Message 表示对话中的一条消息。
// Role 可以是 system、user、assistant 或 tool。
type Message struct {
	Role             string     `json:"role"`
	Content          string     `json:"content"`
	ReasoningContent string     `json:"reasoning_content,omitempty"` // 思考模式下的推理内容，有工具调用时必须回传
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`        // 工具调用列表
	ToolCallID       string     `json:"tool_call_id,omitempty"`      // 工具调用 ID（role=tool 时必填）
	Name             string     `json:"name,omitempty"`              // 可选的参与者名称
}

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

// ResponseFormat 指定模型输出格式。
// Type 可选值：text（默认）、json_object。
type ResponseFormat struct {
	Type string `json:"type"`
}

// ThinkingConfig 控制思考模式与非思考模式的转换。
// Type 可选值：enabled（默认，使用思考模式）、disabled（不使用）。
type ThinkingConfig struct {
	Type string `json:"type"`
}

// StreamOptions 流式选项，仅在 stream: true 时可用。
type StreamOptions struct {
	// IncludeUsage 若为 true，在最后一个 chunk 中会携带 token 用量统计。
	IncludeUsage bool `json:"include_usage"`
}

// Tool 表示模型可调用的工具定义。
type Tool struct {
	Type     string       `json:"type"` // 固定为 "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction 描述一个可被模型调用的函数。
type ToolFunction struct {
	Description string `json:"description,omitempty"` // 函数功能描述
	Name        string `json:"name"`                  // 函数名，最长 64 字符
	Parameters  any    `json:"parameters,omitempty"`  // 函数参数，符合 JSON Schema 规范
	Strict      bool   `json:"strict,omitempty"`      // 是否启用严格模式（Beta 功能）
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
	Arguments string `json:"arguments"` // JSON 序列化的参数
}

// CompletionTokensDetails 提供完成 token 的细分统计。
type CompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"` // 思维链 token 数（思考模式）
}

// ChatCompletionRequest 是对话补全的请求结构。
// 详见 DeepSeek 对话补全 API 文档。
type ChatCompletionRequest struct {
	Messages       []Message       `json:"messages"`                  // 对话消息列表，长度 >= 1
	Model          string          `json:"model"`                     // 模型 ID，如 deepseek-v4-flash
	MaxTokens      int             `json:"max_tokens,omitempty"`      // 最大输出 token 数
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"` // 输出格式
	Stop           []string        `json:"stop,omitempty"`            // 停止词，最多 16 个
	Stream         bool            `json:"stream,omitempty"`          // 是否启用 SSE 流式输出
	StreamOptions  *StreamOptions  `json:"stream_options,omitempty"`  // 流式选项
	Temperature    float64         `json:"temperature,omitempty"`     // 采样温度，0–2，默认 1
	TopP           float64         `json:"top_p,omitempty"`           // 核心采样概率，≤1，默认 1
	Thinking       *ThinkingConfig `json:"thinking,omitempty"`        // 思考模式配置
	ReasoningEffort string         `json:"reasoning_effort,omitempty"` // 推理强度：high 或 max
	Tools          []Tool          `json:"tools,omitempty"`           // 可调用的工具列表，最多 128 个
	ToolChoice     any             `json:"tool_choice,omitempty"`     // 工具调用行为：none、auto、required 或指定函数
	Logprobs       bool            `json:"logprobs,omitempty"`        // 是否返回每个输出 token 的对数概率
	TopLogprobs    int             `json:"top_logprobs,omitempty"`    // 返回概率最高的 N 个 token，0–20
	UserID         string          `json:"user_id,omitempty"`         // 自定义用户 ID，用于内容安全审查及缓存隔离
}

// Choice 表示一个补全结果。
type Choice struct {
	FinishReason string          `json:"finish_reason"` // 停止原因：stop、length、content_filter、tool_calls、insufficient_system_resource
	Index        int             `json:"index"`         // 选择列表索引
	Message      ResponseMessage `json:"message"`       // 助手消息
	Logprobs     *LogprobsInfo   `json:"logprobs"`      // 对数概率信息（需请求时启用）
}

// ResponseMessage 表示助手返回的消息。
type ResponseMessage struct {
	Role             string     `json:"role"`                        // 固定为 "assistant"
	Content          string     `json:"content"`                     // 回复正文，可为空字符串（对应 API 的 null）
	ReasoningContent string     `json:"reasoning_content,omitempty"` // 思考模式下的推理内容
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`        // 工具调用列表
}

// LogprobsInfo 包含输出 token 的对数概率信息。
type LogprobsInfo struct {
	Content          []LogprobToken `json:"content"`           // 正文 token 的对数概率
	ReasoningContent []LogprobToken `json:"reasoning_content"` // 推理文本的对数概率（思考模式）
}

// LogprobToken 表示单个 token 的对数概率信息。
type LogprobToken struct {
	Token       string       `json:"token"`        // token 文本
	Logprob     float64      `json:"logprob"`      // 对数概率，极低概率用 -9999.0
	Bytes       []int        `json:"bytes"`        // UTF-8 字节表示
	TopLogprobs []LogprobTop `json:"top_logprobs"` // 该位置概率最高的 N 个 token
}

// LogprobTop 表示概率最高的备选 token。
type LogprobTop struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
	Bytes   []int   `json:"bytes"`
}

// Usage 表示 token 用量统计。
type Usage struct {
	PromptTokens            int                       `json:"prompt_tokens"`              // 提示消耗的 token 数（= cache_hit + cache_miss）
	PromptCacheHitTokens    int                       `json:"prompt_cache_hit_tokens"`    // 命中缓存的 token 数
	PromptCacheMissTokens   int                       `json:"prompt_cache_miss_tokens"`   // 未命中缓存的 token 数
	CompletionTokens        int                       `json:"completion_tokens"`          // 完成生成的 token 数
	TotalTokens             int                       `json:"total_tokens"`               // 总计 token 数
	CompletionTokensDetails *CompletionTokensDetails  `json:"completion_tokens_details,omitempty"` // 完成 token 细分
}

// ChatCompletionResponse 是非流式对话补全的响应结构。
type ChatCompletionResponse struct {
	ID                string  `json:"id"`                 // 对话唯一标识符
	Choices           []Choice `json:"choices"`           // 生成结果列表
	Created           int64   `json:"created"`            // 创建时的 Unix 时间戳（秒）
	Model             string  `json:"model"`              // 实际使用的模型名
	Object            string  `json:"object"`             // 固定为 "chat.completion"
	SystemFingerprint string  `json:"system_fingerprint"` // 后端配置指纹
	Usage             Usage   `json:"usage"`              // Token 用量统计
}

// CreateChatCompletion 发送非流式对话补全请求。
// req 中消息列表不能为空；若未指定模型则使用客户端默认模型。
func (c *Client) CreateChatCompletion(req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("messages cannot be empty")
	}
	if req.Model == "" {
		req.Model = c.model
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	httpReq, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/chat/completions", c.baseURL),
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var response ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &response, nil
}

// StreamDelta 表示流式响应中每个 chunk 的增量内容。
type StreamDelta struct {
	Role             string `json:"role,omitempty"`              // 角色（通常仅在首个 chunk 出现）
	Content          string `json:"content,omitempty"`           // 增量正文
	ReasoningContent string `json:"reasoning_content,omitempty"` // 增量推理内容（思考模式）
}

// StreamChoice 表示流式响应中的一个选项。
type StreamChoice struct {
	Index        int         `json:"index"`         // 选项索引
	Delta        StreamDelta `json:"delta"`         // 增量内容
	FinishReason *string     `json:"finish_reason"` // 停止原因，仅在最后 chunk 有值
}

// StreamChunk 表示流式响应的一个 SSE 事件块。
type StreamChunk struct {
	ID                string         `json:"id"`                 // 对话唯一标识符
	Object            string         `json:"object"`             // 固定为 "chat.completion.chunk"
	Created           int64          `json:"created"`            // 创建时间戳
	Model             string         `json:"model"`              // 模型名
	SystemFingerprint string         `json:"system_fingerprint"` // 后端配置指纹
	Choices           []StreamChoice `json:"choices"`            // 增量结果列表
	Usage             *Usage         `json:"usage,omitempty"`    // Token 用量统计（需启用 stream_options.include_usage）
}

// CreateChatCompletionStream 发送流式对话补全请求，返回 chunk 通道和错误通道。
// 调用方通过 range 遍历 chunk 通道获取实时增量，通过 err 通道获取异步错误。
// 流结束时两个通道都会关闭。
func (c *Client) CreateChatCompletionStream(req *ChatCompletionRequest) (<-chan StreamChunk, <-chan error) {
	chunkCh := make(chan StreamChunk)
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

		req.Stream = true

		payload, err := json.Marshal(req)
		if err != nil {
			errCh <- fmt.Errorf("error marshaling request: %w", err)
			return
		}

		httpReq, err := http.NewRequest(
			"POST",
			fmt.Sprintf("%s/chat/completions", c.baseURL),
			bytes.NewReader(payload),
		)
		if err != nil {
			errCh <- fmt.Errorf("error creating request: %w", err)
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "text/event-stream")
		httpReq.Header.Set("Authorization", "Bearer "+c.token)

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			errCh <- fmt.Errorf("error sending request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			errCh <- fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
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
			chunkCh <- chunk
		}

		if err := scanner.Err(); err != nil {
			errCh <- fmt.Errorf("error reading stream: %w", err)
		}
	}()

	return chunkCh, errCh
}

// ResponseToMessage 将 API 响应的 ResponseMessage 转为可追加到消息列表的 Message。
// 用于多轮对话拼接，自动携带 reasoning_content 和 tool_calls。
func ResponseToMessage(msg ResponseMessage) Message {
	return Message{
		Role:             msg.Role,
		Content:          msg.Content,
		ReasoningContent: msg.ReasoningContent,
		ToolCalls:        msg.ToolCalls,
	}
}

// JSONStructureConfig 封装 JSON 结构化输出的配置。
// 用于指导模型按照预设的输入输出格式生成符合要求的 JSON 响应。
type JSONStructureConfig struct {
	SystemPrompt      string // 系统提示内容，描述输出要求
	ExampleInput      string // 示例输入
	ExampleJSONOutput string // 示例 JSON 输出格式
	JsonSchema        string // JSON Schema（预留字段）
}

// SetExampleOutput 将任意值序列化为格式化的 JSON 字符串，作为示例输出。
func (c *JSONStructureConfig) SetExampleOutput(example any) error {
	jsonBytes, err := json.MarshalIndent(example, "", "  ")
	if err != nil {
		return err
	}
	c.ExampleJSONOutput = string(jsonBytes)
	return nil
}

// FormatSystemPrompt 将配置格式化为完整的系统提示文本。
func (c *JSONStructureConfig) FormatSystemPrompt() string {
	prompt := fmt.Sprintf("%s\n\nEXAMPLE INPUT:\n%s\n\nEXAMPLE JSON OUTPUT:\n```json\n%s\n```",
		c.SystemPrompt, c.ExampleInput, c.ExampleJSONOutput)
	return prompt
}

// CreateJSONStructuredCompletion 创建生成 JSON 结构化输出的对话补全请求。
// config 定义了输出格式要求；userPrompt 为用户的输入；model 指定模型（留空则使用客户端默认值）。
func (c *Client) CreateJSONStructuredCompletion(
	config JSONStructureConfig,
	userPrompt string,
	model string,
) (*ChatCompletionResponse, error) {
	fullSystemPrompt := config.FormatSystemPrompt()

	messages := []Message{
		{Role: "system", Content: fullSystemPrompt},
		{Role: "user", Content: userPrompt},
	}

	modelToUse := model
	if modelToUse == "" {
		modelToUse = c.model
	}

	req := &ChatCompletionRequest{
		Messages: messages,
		Model:    modelToUse,
		ResponseFormat: &ResponseFormat{
			Type: "json_object",
		},
	}

	if req.Temperature == 0 {
		req.Temperature = c.temperature
	}

	return c.CreateChatCompletion(req)
}

// SimpleChat 提供简化的对话接口，发送一条用户消息并返回助手回复文本。
// 如果客户端配置了默认系统提示，会自动添加到消息列表中。
// 如果客户端配置了思考模式（WithThinking），请求会自动携带 thinking 参数。
func (c *Client) SimpleChat(prompt string) (string, error) {
	messages := []Message{}
	if c.system != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: c.system,
		})
	}

	messages = append(messages, Message{
		Role:    "user",
		Content: prompt,
	})

	req := &ChatCompletionRequest{
		Messages: messages,
		Model:    c.model,
	}

	if c.thinking {
		req.Thinking = &ThinkingConfig{Type: ThinkingEnabled}
		req.ReasoningEffort = ReasoningEffortHigh
	}

	if req.Temperature == 0 {
		req.Temperature = c.temperature
	}

	resp, err := c.CreateChatCompletion(req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response received")
	}

	return resp.Choices[0].Message.Content, nil
}

// ThinkingChat 类似 SimpleChat，但在思考模式下同时返回回复内容和推理内容。
// 客户端需通过 WithThinking(true) 启用思考模式，否则与 SimpleChat 行为一致。
func (c *Client) ThinkingChat(prompt string) (content string, reasoning string, err error) {
	messages := []Message{}
	if c.system != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: c.system,
		})
	}

	messages = append(messages, Message{
		Role:    "user",
		Content: prompt,
	})

	req := &ChatCompletionRequest{
		Messages:       messages,
		Model:          c.model,
		Thinking:       &ThinkingConfig{Type: ThinkingEnabled},
		ReasoningEffort: ReasoningEffortHigh,
	}

	if req.Temperature == 0 {
		req.Temperature = c.temperature
	}

	resp, err := c.CreateChatCompletion(req)
	if err != nil {
		return "", "", err
	}

	if len(resp.Choices) == 0 {
		return "", "", fmt.Errorf("no response received")
	}

	return resp.Choices[0].Message.Content, resp.Choices[0].Message.ReasoningContent, nil
}

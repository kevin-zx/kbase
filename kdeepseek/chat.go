package kdeepseek

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// ChatConfig 是会话级配置，创建 Chat 时一次性设置，后续所有 Send/SendStream 共用。
type ChatConfig struct {
	Model           string
	Temperature     float64
	MaxTokens       int
	TopP            float64
	Tools           []Tool
	ToolChoice      any
	ReasoningEffort string
	ResponseFormat  *ResponseFormat
	Stop            []string
}

// ChatOption 用于在创建 Chat 时配置 ChatConfig。
type ChatOption func(*ChatConfig)

// WithChatModel 设置会话使用的模型。
func WithChatModel(model string) ChatOption {
	return func(c *ChatConfig) {
		c.Model = model
	}
}

// WithChatTemperature 设置采样温度，范围 0–2。
func WithChatTemperature(temp float64) ChatOption {
	return func(c *ChatConfig) {
		c.Temperature = temp
	}
}

// WithChatMaxTokens 设置最大输出 token 数。
func WithChatMaxTokens(n int) ChatOption {
	return func(c *ChatConfig) {
		c.MaxTokens = n
	}
}

// WithChatTopP 设置核心采样概率。
func WithChatTopP(p float64) ChatOption {
	return func(c *ChatConfig) {
		c.TopP = p
	}
}

// WithChatTools 设置可调用的工具列表。
func WithChatTools(tools ...Tool) ChatOption {
	return func(c *ChatConfig) {
		c.Tools = tools
	}
}

// WithChatToolChoice 设置工具调用行为：none、auto、required 或指定函数名。
func WithChatToolChoice(choice any) ChatOption {
	return func(c *ChatConfig) {
		c.ToolChoice = choice
	}
}

// WithChatReasoningEffort 设置推理强度：high 或 max。
func WithChatReasoningEffort(effort string) ChatOption {
	return func(c *ChatConfig) {
		c.ReasoningEffort = effort
	}
}

// WithChatResponseFormat 设置输出格式（如 json_object）。
func WithChatResponseFormat(format *ResponseFormat) ChatOption {
	return func(c *ChatConfig) {
		c.ResponseFormat = format
	}
}

// WithChatStop 设置停止词列表。
func WithChatStop(stop ...string) ChatOption {
	return func(c *ChatConfig) {
		c.Stop = stop
	}
}

// toolAccumulator 在流式响应中按 index 拼接工具调用分片。
type toolAccumulator struct {
	calls map[int]*ToolCall
}

func newToolAccumulator() *toolAccumulator {
	return &toolAccumulator{
		calls: make(map[int]*ToolCall),
	}
}

func (a *toolAccumulator) add(tc StreamToolCall) {
	if tc.Index == nil {
		return
	}
	idx := *tc.Index
	call, ok := a.calls[idx]
	if !ok {
		call = &ToolCall{}
		a.calls[idx] = call
	}
	if tc.ID != "" {
		call.ID = tc.ID
	}
	if tc.Type != "" {
		call.Type = tc.Type
	}
	if tc.Function.Name != "" {
		call.Function.Name = tc.Function.Name
	}
	call.Function.Arguments += tc.Function.Arguments
}

func (a *toolAccumulator) toolCalls() []ToolCall {
	result := make([]ToolCall, 0, len(a.calls))
	for i := 0; i < len(a.calls); i++ {
		if tc, ok := a.calls[i]; ok {
			result = append(result, *tc)
		}
	}
	return result
}

// Chat 维护多轮对话的消息历史和配置，提供 Send/SendStream 等方法。
// 所有公共方法均为并发安全。
type Chat struct {
	client  *Client
	config  ChatConfig
	history []Message
	mu      sync.Mutex
}

// Chat 创建新的多轮会话。
// systemPrompt 为可选系统提示，留空则不带 system 消息；
// opts 为一次性配置项，后续可通过 SetXxx 方法动态修改。
func (c *Client) Chat(systemPrompt string, opts ...ChatOption) *Chat {
	config := ChatConfig{
		Model:           c.model,
		Temperature:     c.temperature,
		ReasoningEffort: c.reasoningEffort,
	}
	for _, opt := range opts {
		opt(&config)
	}
	chat := &Chat{
		client: c,
		config: config,
	}
	if systemPrompt != "" {
		chat.history = append(chat.history, Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}
	return chat
}

func (ch *Chat) buildRequest(messages []Message) *ChatCompletionRequest {
	req := &ChatCompletionRequest{
		Messages: messages,
	}
	if ch.config.Model != "" {
		req.Model = ch.config.Model
	}
	if ch.config.Temperature > 0 {
		req.Temperature = ch.config.Temperature
	}
	if ch.config.MaxTokens > 0 {
		req.MaxTokens = ch.config.MaxTokens
	}
	if ch.config.TopP > 0 {
		req.TopP = ch.config.TopP
	}
	if len(ch.config.Tools) > 0 {
		req.Tools = ch.config.Tools
	}
	if ch.config.ToolChoice != nil {
		req.ToolChoice = ch.config.ToolChoice
	}
	if ch.config.ReasoningEffort != "" {
		req.Thinking = &ThinkingConfig{Type: ThinkingEnabled}
		req.ReasoningEffort = ch.config.ReasoningEffort
	}
	if ch.config.ResponseFormat != nil {
		req.ResponseFormat = ch.config.ResponseFormat
	}
	if len(ch.config.Stop) > 0 {
		req.Stop = ch.config.Stop
	}
	return req
}

func (ch *Chat) send(prompt string, forceThinking bool) (Message, error) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.history = append(ch.history, Message{
		Role:    "user",
		Content: prompt,
	})

	req := ch.buildRequest(ch.history)
	if forceThinking {
		req.Thinking = &ThinkingConfig{Type: ThinkingEnabled}
		if req.ReasoningEffort == "" {
			req.ReasoningEffort = ReasoningEffortHigh
		}
	}

	resp, err := ch.client.ChatCompletion(req)
	if err != nil {
		ch.history = ch.history[:len(ch.history)-1]
		return Message{}, err
	}

	if len(resp.Choices) == 0 {
		ch.history = ch.history[:len(ch.history)-1]
		return Message{}, fmt.Errorf("no response received")
	}

	msg := ResponseToMessage(resp.Choices[0].Message)
	ch.history = append(ch.history, msg)
	return msg, nil
}

// Send 发送一条用户消息并返回助手回复文本，自动维护历史。
func (ch *Chat) Send(prompt string) (string, error) {
	msg, err := ch.send(prompt, false)
	if err != nil {
		return "", err
	}
	return msg.Content, nil
}

// Continue 继续当前对话，将现有历史发送给模型并返回回复。
// 不会追加新的用户消息。用于工具调用循环：Send → AddToolResult → Continue → ...
func (ch *Chat) Continue() (string, error) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	req := ch.buildRequest(ch.history)
	resp, err := ch.client.ChatCompletion(req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response received")
	}

	msg := ResponseToMessage(resp.Choices[0].Message)
	ch.history = append(ch.history, msg)
	return msg.Content, nil
}

// SendStream 以流式发送消息，实时返回 chunk；流正常结束后自动追加完整回复到历史。
// ctx 取消会终止流式并回滚本次用户消息。
func (ch *Chat) SendStream(ctx context.Context, prompt string) (<-chan StreamChunk, <-chan error) {
	return ch.stream(ctx, true, prompt)
}

// ContinueStream 以流式继续当前对话，不会追加新的用户消息。
// 用于工具调用循环的流式场景：SendStream → AddToolResult → ContinueStream → ...
func (ch *Chat) ContinueStream(ctx context.Context) (<-chan StreamChunk, <-chan error) {
	return ch.stream(ctx, false, "")
}

func (ch *Chat) stream(ctx context.Context, addUser bool, prompt string) (<-chan StreamChunk, <-chan error) {
	ch.mu.Lock()

	chunkCh := make(chan StreamChunk, 16)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunkCh)
		defer close(errCh)
		defer ch.mu.Unlock()

		if addUser {
			ch.history = append(ch.history, Message{
				Role:    "user",
				Content: prompt,
			})
		}

		req := ch.buildRequest(ch.history)

		srcChunks, srcErrs := ch.client.ChatCompletionStream(ctx, req)

		var (
			fullContent   strings.Builder
			fullReasoning strings.Builder
			toolAccum     = newToolAccumulator()
			gotFinish     bool
		)

		for {
			select {
			case chunk, ok := <-srcChunks:
				if !ok {
					goto done
				}
				for _, choice := range chunk.Choices {
					if choice.Delta.Content != "" {
						fullContent.WriteString(choice.Delta.Content)
					}
					if choice.Delta.ReasoningContent != "" {
						fullReasoning.WriteString(choice.Delta.ReasoningContent)
					}
					for _, tc := range choice.Delta.ToolCalls {
						toolAccum.add(tc)
					}
					if choice.FinishReason != nil {
						gotFinish = true
					}
				}
				select {
				case chunkCh <- chunk:
				case <-ctx.Done():
					if addUser {
						ch.history = ch.history[:len(ch.history)-1]
					}
					return
				}

			case err := <-srcErrs:
				if err != nil {
					errCh <- err
				}

			case <-ctx.Done():
				if addUser {
					ch.history = ch.history[:len(ch.history)-1]
				}
				return
			}
		}

	done:
		for range srcErrs {
		}

		if gotFinish {
			ch.history = append(ch.history, Message{
				Role:             "assistant",
				Content:          fullContent.String(),
				ReasoningContent: fullReasoning.String(),
				ToolCalls:        toolAccum.toolCalls(),
			})
			return
		}

		if addUser {
			ch.history = ch.history[:len(ch.history)-1]
		}
	}()

	return chunkCh, errCh
}

// SendWithThinking 在思考模式下发送消息，同时返回回复内容和推理内容。
// 无论 ChatConfig 是否配置了 ReasoningEffort，本方法都会强制启用思考。
func (ch *Chat) SendWithThinking(prompt string) (content string, reasoning string, err error) {
	msg, err := ch.send(prompt, true)
	if err != nil {
		return "", "", err
	}
	return msg.Content, msg.ReasoningContent, nil
}

// AddToolResult 追加工具调用结果到历史。需在收到 assistant 工具调用后、下一次 Send 前调用。
func (ch *Chat) AddToolResult(toolCallID, name, result string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.history = append(ch.history, Message{
		Role:       "tool",
		Content:    result,
		ToolCallID: toolCallID,
		Name:       name,
	})
}

// History 返回当前历史的副本，可用于持久化或调试。
func (ch *Chat) History() []Message {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	h := make([]Message, len(ch.history))
	copy(h, ch.history)
	return h
}

// SetHistory 用外部历史替换当前历史，可用于从持久化恢复会话。
func (ch *Chat) SetHistory(history []Message) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.history = make([]Message, len(history))
	copy(ch.history, history)
}

// SetModel 动态修改会话使用的模型。
func (ch *Chat) SetModel(model string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.config.Model = model
}

// SetTemperature 动态修改采样温度。
func (ch *Chat) SetTemperature(temp float64) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.config.Temperature = temp
}

// SetMaxTokens 动态修改最大输出 token 数。
func (ch *Chat) SetMaxTokens(n int) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.config.MaxTokens = n
}

// SetTopP 动态修改核心采样概率。
func (ch *Chat) SetTopP(p float64) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.config.TopP = p
}

// SetTools 动态修改可调用的工具列表。
func (ch *Chat) SetTools(tools ...Tool) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.config.Tools = tools
}

// SetToolChoice 动态修改工具调用行为。
func (ch *Chat) SetToolChoice(choice any) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.config.ToolChoice = choice
}

// SetReasoningEffort 动态修改推理强度。
func (ch *Chat) SetReasoningEffort(effort string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.config.ReasoningEffort = effort
}

// SetResponseFormat 动态修改输出格式。
func (ch *Chat) SetResponseFormat(format *ResponseFormat) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.config.ResponseFormat = format
}

// SetStop 动态修改停止词列表。
func (ch *Chat) SetStop(stop ...string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.config.Stop = stop
}

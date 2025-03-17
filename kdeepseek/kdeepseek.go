package kdeepseek

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client 用于与DeepSeek API交互的客户端
type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
	model      string // 新增默认模型字段
}

// ClientOption 是配置客户端的函数类型
type ClientOption func(*Client)

// NewClient 创建新的DeepSeek客户端
func NewClient(token string, options ...ClientOption) *Client {
	c := &Client{
		token:      token,
		baseURL:    "https://api.deepseek.com",
		httpClient: http.DefaultClient,
		model:      "deepseek-chat", // 设置默认模型
	}

	for _, opt := range options {
		opt(c)
	}
	return c
}

// WithHTTPClient 设置自定义HTTP客户端
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithBaseURL 设置自定义基础URL
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithModel 设置默认模型
func WithModel(model string) ClientOption {
	return func(c *Client) {
		c.model = model
	}
}

// Message 表示对话中的消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ResponseFormat 定义响应格式
type ResponseFormat struct {
	Type string `json:"type"`
}

// ChatCompletionRequest 聊天补全请求结构
type ChatCompletionRequest struct {
	Messages         []Message       `json:"messages"`
	Model            string          `json:"model"`
	FrequencyPenalty float64         `json:"frequency_penalty,omitempty"`
	MaxTokens        int             `json:"max_tokens,omitempty"`
	PresencePenalty  float64         `json:"presence_penalty,omitempty"`
	ResponseFormat   *ResponseFormat `json:"response_format,omitempty"`
	Stop             []string        `json:"stop,omitempty"`
	Stream           bool            `json:"stream,omitempty"`
	Temperature      float64         `json:"temperature,omitempty"`
	TopP             float64         `json:"top_p,omitempty"`
}

// ChatCompletionResponse 聊天补全响应结构
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Index        int    `json:"index"`
		Message      struct {
			Content string `json:"content"`
			Role    string `json:"role"`
		} `json:"message"`
	} `json:"choices"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Object  string `json:"object"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// CreateChatCompletion 创建聊天补全请求
func (c *Client) CreateChatCompletion(req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	// 验证必要参数
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("messages cannot be empty")
	}
	if req.Model == "" {
		req.Model = c.model // 如果未指定模型，使用客户端默认模型
	}

	// 序列化请求体
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/chat/completions", c.baseURL),
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.token)

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// 处理响应
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var response ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &response, nil
}

// JSONStructureConfig 封装JSON结构化输出的配置
type JSONStructureConfig struct {
	SystemPrompt      string // 系统提示内容，描述输出要求
	ExampleInput      string // 示例输入
	ExampleJSONOutput string // 示例JSON输出格式
}

func (c *JSONStructureConfig) SetExampleOutput(example interface{}) error {
	jsonBytes, err := json.MarshalIndent(example, "", "  ")
	if err != nil {
		return err
	}
	c.ExampleJSONOutput = string(jsonBytes)
	return nil
}

// FormatSystemPrompt 将配置格式化为完整的系统提示
func (c *JSONStructureConfig) FormatSystemPrompt() string {
	return fmt.Sprintf("%s\n\nEXAMPLE INPUT:\n%s\n\nEXAMPLE JSON OUTPUT:\n%s",
		c.SystemPrompt, c.ExampleInput, c.ExampleJSONOutput)
}

// CreateJSONStructuredCompletion 创建生成JSON结构化输出的聊天补全
// config: JSON结构化输出的配置
// userPrompt: 用户实际输入的提示
// model: 模型名称
func (c *Client) CreateJSONStructuredCompletion(
	config JSONStructureConfig,
	userPrompt string,
	model string,
) (*ChatCompletionResponse, error) {
	// 使用JSONStructureConfig的方法来获取完整的系统提示
	fullSystemPrompt := config.FormatSystemPrompt()

	// 构建消息
	messages := []Message{
		{Role: "system", Content: fullSystemPrompt},
		{Role: "user", Content: userPrompt},
	}

	// 如果未指定模型，使用客户端默认模型
	modelToUse := model
	if modelToUse == "" {
		modelToUse = c.model
	}

	// 创建请求
	req := &ChatCompletionRequest{
		Messages: messages,
		Model:    modelToUse,
		ResponseFormat: &ResponseFormat{
			Type: "json_object",
		},
	}

	// 发送请求并直接返回响应
	return c.CreateChatCompletion(req)
}

// SimpleChat 提供简化的聊天接口，只需提供提示文本即可获取回复
func (c *Client) SimpleChat(prompt string) (string, error) {
	// 创建请求消息
	messages := []Message{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// 创建聊天补全请求
	req := &ChatCompletionRequest{
		Messages: messages,
		Model:    c.model,
	}

	// 调用API
	resp, err := c.CreateChatCompletion(req)
	if err != nil {
		return "", err
	}

	// 检查响应是否有内容
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response received")
	}

	// 返回助手的回复文本
	return resp.Choices[0].Message.Content, nil
}

package kollama

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/invopop/jsonschema"
)

// ChatMessage represents a single message in the conversation
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Chat represents the entire conversation with the API and handles message sending
type Chat struct {
	Model        string        `json:"model"`
	Messages     []ChatMessage `json:"messages"`
	Stream       bool          `json:"stream"` // Controls whether we use streaming or not
	Format       *Format       `json:"format"`
	SystemPrompt string        `json:"-"` // System prompt to add at the beginning of each conversation
	// Host         string        `json:"-"` // API服务主机地址，默认"localhost"
	// Port         string        `json:"-"` // API服务端口号，默认"11434"
	// Protocol     string        `json:"-"` // API服务协议，默认"http"
	APIURL string `json:"-"` // 完整的API请求URL
}

// GetModel returns the model used for the chat
func (c *Chat) GetModel() string {
	return c.Model
}

// SetModel sets the model for the chat
func (c *Chat) SetModel(model string) {
	if model != "" {
		c.Model = model
	}
}

// clear messages
func (c *Chat) ClearMessages() {
	c.Messages = []ChatMessage{}
}

// ChatResponse represents the response from the API
type ChatResponse struct {
	Model         string      `json:"model"`
	CreatedAt     time.Time   `json:"created_at"`
	Message       ChatMessage `json:"message"`
	Done          bool        `json:"done"`
	TotalDuration int64       `json:"total_duration"`
	LoadDuration  int64       `json:"load_duration"`
	EvalCount     int         `json:"eval_count"`
	EvalDuration  int64       `json:"eval_duration"`
}

type ChatOption func(*Chat)

func WithModel(model string) ChatOption {
	return func(c *Chat) {
		if model != "" {
			c.Model = model
		} else {
			c.Model = "qwen2.5:7B" // Default model if none
		}
	}
}
func WithStream(stream bool) ChatOption {
	return func(c *Chat) {
		c.Stream = stream
	}
}

// WithFormat sets the format for the chat
func WithFormat(format *Format) ChatOption {
	return func(c *Chat) {
		if format != nil {
			c.Format = format
		} else {
			c.Format = &Format{
				Type:       "object",
				Properties: make(map[string]Property),
			} // Default format if none provided
		}
	}
}

// WithSystemPrompt sets the system prompt for the chat
func WithSystemPrompt(prompt string) ChatOption {
	return func(c *Chat) {
		if prompt != "" {
			c.SystemPrompt = prompt
		} else {
			c.SystemPrompt = "" // Default to empty string if none provided
		}
	}
}

// WithAPIURL sets the API URL for the chat
func WithAPIURL(apiURL string) ChatOption {
	return func(c *Chat) {
		if apiURL != "" {
			c.APIURL = apiURL
		} else {
			c.APIURL = "http://localhost:11434/api/chat" // Default API URL if none provided
		}
	}
}

// NewChat initializes a new chat session with the provided model and format
func NewChat(model string, format *Format, chatOpts ...ChatOption) *Chat {
	if model == "" {
		model = "qwen2.5:7B"
	}
	c := &Chat{
		Model:        model,
		Messages:     []ChatMessage{},
		Stream:       false,
		Format:       format,
		SystemPrompt: "",                                // Default to empty string
		APIURL:       "http://localhost:11434/api/chat", // Default API URL
		// Host:         "localhost", // 默认host
		// Port:         "11434",     // 默认port
		// Protocol:     "http",      // 默认协议
	}
	for _, opt := range chatOpts {
		opt(c)
	}
	return c
}

// NewChatWithOptions initializes a new chat session with the provided model, format, and options
func NewChatWithOptions(opts ...ChatOption) *Chat {
	defaultChat := &Chat{
		Model:        "qwen2.5:7B",
		Messages:     []ChatMessage{},
		Stream:       false,
		Format:       nil,
		SystemPrompt: "",                                // Default to empty string
		APIURL:       "http://localhost:11434/api/chat", // Default API URL
	}

	for _, opt := range opts {
		opt(defaultChat)
	}
	return defaultChat
}

// SetSystemPrompt sets the system prompt to be used in conversations
func (c *Chat) SetSystemPrompt(prompt string) {
	c.SystemPrompt = prompt
}

// SetAPIURL sets the API URL for the chat service
func (c *Chat) SetAPIURL(apiURL string) {
	if apiURL != "" {
		c.APIURL = apiURL
	} else {
		c.APIURL = "http://localhost:11434/api/chat" // Default API URL
	}

}

// // SetHost 设置API服务主机地址
// // 如果host为空则使用默认值"localhost"
// func (c *Chat) SetHost(host string) {
// 	if host != "" {
// 		c.Host = host
// 	} else {
// 		c.Host = "localhost"
// 	}
// }

// // SetPort 设置API服务端口号
// // 如果port为空则使用默认值"11434"
// func (c *Chat) SetPort(port string) {
// 	if port != "" {
// 		c.Port = port
// 	} else {
// 		c.Port = "11434"
// 	}
// }

// // apiURL 生成完整的API请求URL
// func (c *Chat) apiURL() string {
// 	return fmt.Sprintf("http://%s:%s/api/chat", c.Host, c.Port)
// }

// prepareMessagesForAPI prepares messages for API call, adding system prompt if set
func (c *Chat) prepareMessagesForAPI() {

	// Add system message if set
	if c.SystemPrompt != "" && len(c.Messages) == 0 {
		c.Messages = append(c.Messages, ChatMessage{
			Role:    "system",
			Content: c.SystemPrompt,
		})
	}

}

type SchemaPayload struct {
	Model    string             `json:"model"`
	Messages []ChatMessage      `json:"messages"`
	Stream   bool               `json:"stream"`
	Format   *jsonschema.Schema `json:"format,omitempty"`
}

func GenerateSchema[T any]() *jsonschema.Schema {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false, // 禁止额外字段
		DoNotReference:            true,  // 内联而非引用
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

// send schema message
func (c *Chat) SendSchemaMessage(
	schema *jsonschema.Schema,
	config JSONStructureConfig,
	userMessage string,
) (ChatMessage, error) {

	c.Messages = append(c.Messages, ChatMessage{
		Role:    "system",
		Content: config.FormatSystemPrompt(),
	})

	c.Messages = append(c.Messages, ChatMessage{
		Role:    "user",
		Content: userMessage,
	})
	defer c.ClearMessages()

	// Prepare the request schema Schema
	payload := SchemaPayload{
		Model:    c.Model,
		Messages: c.Messages,
		Stream:   false, // False for non-streaming
		Format:   schema,
	}
	// Convert payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return ChatMessage{}, fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Send the HTTP POST request
	resp, err := http.Post(c.APIURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return ChatMessage{}, fmt.Errorf("failed to send POST request: %v", err)
	}
	defer resp.Body.Close()

	// Decode the response
	var chatResponse ChatResponse
	err = json.NewDecoder(resp.Body).Decode(&chatResponse)
	if err != nil {
		return ChatMessage{}, fmt.Errorf("failed to decode response: %v", err)
	}

	// Add assistant's response to the conversation history
	c.Messages = append(c.Messages, chatResponse.Message)

	return chatResponse.Message, nil
}

type Payload struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Format   *Format       `json:"format,omitempty"`
}

// SendMessage sends a user message to the chat API and returns the assistant's response (non-streaming)
func (c *Chat) SendMessage(userMessage string) (ChatMessage, error) {
	c.prepareMessagesForAPI()
	// Append the user's message to the conversation
	c.Messages = append(c.Messages, ChatMessage{
		Role:    "user",
		Content: userMessage,
	})

	// Prepare the request payload
	payload := Payload{
		Model:    c.Model,
		Messages: c.Messages,
		Stream:   false, // False for non-streaming
		Format:   c.Format,
	}

	// Convert payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return ChatMessage{}, fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Send the HTTP POST request
	resp, err := http.Post(c.APIURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return ChatMessage{}, fmt.Errorf("failed to send POST request: %v", err)
	}
	defer resp.Body.Close()

	// Decode the response
	var chatResponse ChatResponse
	err = json.NewDecoder(resp.Body).Decode(&chatResponse)
	if err != nil {
		return ChatMessage{}, fmt.Errorf("failed to decode response: %v", err)
	}

	// Add assistant's response to the conversation history
	c.Messages = append(c.Messages, chatResponse.Message)

	return chatResponse.Message, nil
}

type Format struct {
	Type       string              `json:"type"`
	Required   []string            `json:"required,omitempty`
	Properties map[string]Property `json:"properties,omitempty"`
}

type Property struct {
	Type string `json:"type"`
}

// SendStreamMessage sends a user message to the chat API and processes the response as a stream
func (c *Chat) SendStreamMessage(userMessage string) (*ChatMessage, error) {
	c.prepareMessagesForAPI()
	// Append the user's message to the conversation
	c.Messages = append(c.Messages, ChatMessage{
		Role:    "user",
		Content: userMessage,
	})

	// Prepare the request payload
	payload := Payload{
		Model:    c.Model,
		Messages: c.Messages,
		Stream:   true, // Set streaming to true
		Format:   c.Format,
	}

	// Convert payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Send the HTTP POST request
	resp, err := http.Post(c.APIURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to send POST request: %v", err)
	}
	defer resp.Body.Close()

	// Prepare to read the streaming response
	reader := bufio.NewReader(resp.Body)
	var streamMessages []ChatMessage

	maxSize := 3000

	for {
		// Read until a newline character
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break // Finished reading the stream
			}
			return nil, fmt.Errorf("failed to read stream: %v", err)
		}

		// Ignore empty lines
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		// Decode the streamed JSON object
		var chatResponse ChatResponse
		err = json.Unmarshal(line, &chatResponse)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal stream chunk: %v", err)
		}
		// fmt.Printf("%s", chatResponse.Message.Content)
		// Append the message to the result
		streamMessages = append(streamMessages, chatResponse.Message)

		// Check if this was the final message in the stream
		if chatResponse.Done {
			break
		}
		maxSize--
		if maxSize <= 0 {
			break
		}
	}
	if maxSize <= 0 {
		return nil, fmt.Errorf("stream message size is too large")
	}

	// Add all assistant responses to the conversation history
	// c.Messages = append(c.Messages, streamMessages...)
	m, err := c.MergeMessages(streamMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to merge messages: %v", err)
	}
	c.Messages = append(c.Messages, *m)

	return m, nil
}

// 合并 messages
func (c *Chat) MergeMessages(messages []ChatMessage) (*ChatMessage, error) {
	message := ChatMessage{
		Role:    "assistant",
		Content: "",
	}
	for _, m := range messages {
		message.Content += m.Content
	}

	return &message, nil
}

// SendSchemaMessageStream sends a schema message with streaming response and checks for max size
func (c *Chat) SendSchemaMessageStream(
	schema *jsonschema.Schema,
	config JSONStructureConfig,
	userMessage string,
	maxSize int,
) (*ChatMessage, error) {
	c.Messages = append(c.Messages, ChatMessage{
		Role:    "system",
		Content: config.FormatSystemPrompt(),
	})

	c.Messages = append(c.Messages, ChatMessage{
		Role:    "user",
		Content: userMessage,
	})
	defer c.ClearMessages()

	// Prepare the request schema with streaming
	payload := SchemaPayload{
		Model:    c.Model,
		Messages: c.Messages,
		Stream:   true, // Enable streaming
		Format:   schema,
	}

	// Convert payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Send the HTTP POST request
	resp, err := http.Post(c.APIURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to send POST request: %v", err)
	}
	defer resp.Body.Close()

	// Prepare to read the streaming response
	reader := bufio.NewReader(resp.Body)
	var streamMessages []ChatMessage

	for {
		// Read until a newline character
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break // Finished reading the stream
			}
			return nil, fmt.Errorf("failed to read stream: %v", err)
		}

		// Ignore empty lines
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		// Decode the streamed JSON object
		var chatResponse ChatResponse
		err = json.Unmarshal(line, &chatResponse)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal stream chunk: %v", err)
		}

		// Append the message to the result
		streamMessages = append(streamMessages, chatResponse.Message)

		// Check if this was the final message in the stream
		if chatResponse.Done {
			break
		}

		// Check max size limit
		maxSize--
		if maxSize <= 0 {
			return nil, fmt.Errorf("stream message size exceeds limit")
		}
	}

	// Merge all streamed messages
	m, err := c.MergeMessages(streamMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to merge messages: %v", err)
	}
	c.Messages = append(c.Messages, *m)

	return m, nil
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

package kollama

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ChatMessage represents a single message in the conversation
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Chat represents the entire conversation with the API and handles message sending
type Chat struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"` // Controls whether we use streaming or not
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

// NewChat initializes a new chat session with the provided model
func NewChat(model string) *Chat {
	if model == "" {
		model = "qwen2.5:7B"
	}
	return &Chat{
		Model:    model,
		Messages: []ChatMessage{},
		// Stream:   stream,
	}
}

// SendMessage sends a user message to the chat API and returns the assistant's response (non-streaming)
func (c *Chat) SendMessage(userMessage string) (ChatMessage, error) {
	// Append the user's message to the conversation
	c.Messages = append(c.Messages, ChatMessage{
		Role:    "user",
		Content: userMessage,
	})

	// Prepare the request payload
	payload := map[string]interface{}{
		"model":    c.Model,
		"messages": c.Messages,
		"stream":   false, // False for non-streaming
	}

	// Convert payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return ChatMessage{}, fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Send the HTTP POST request
	resp, err := http.Post("http://localhost:11434/api/chat", "application/json", bytes.NewBuffer(payloadBytes))
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

// SendStreamMessage sends a user message to the chat API and processes the response as a stream
func (c *Chat) SendStreamMessage(userMessage string) (*ChatMessage, error) {
	// Append the user's message to the conversation
	c.Messages = append(c.Messages, ChatMessage{
		Role:    "user",
		Content: userMessage,
	})

	// Prepare the request payload
	payload := map[string]interface{}{
		"model":    c.Model,
		"messages": c.Messages,
		"stream":   true, // Set streaming to true
	}

	// Convert payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Send the HTTP POST request
	resp, err := http.Post("http://localhost:11434/api/chat", "application/json", bytes.NewBuffer(payloadBytes))
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
		fmt.Printf("%s", chatResponse.Message.Content)
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

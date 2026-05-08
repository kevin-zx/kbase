package kdeepseek

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Example_multiTurnThinkingChat 演示多轮对话中的思考模式使用。
// 通过 ThinkingChat 获取 content 和 reasoning_content，
// 再通过 ResponseToMessage 将响应转为消息追加到多轮对话中。
func Example_multiTurnThinkingChat() {
	token := os.Getenv("DEEPSEEK_API_KEY")
	if token == "" {
		fmt.Println("请设置环境变量 DEEPSEEK_API_KEY")
		return
	}

	client := NewClient(token,
		WithModel("deepseek-v4-pro"),
		WithSystem("你是一个有帮助的助手，请用中文回答。"),
		WithThinking(true),
	)

	// Turn 1
	content, reasoning, err := client.ThinkingChat("9.11 和 9.8 哪个大？")
	if err != nil {
		fmt.Printf("Turn 1 error: %v\n", err)
		return
	}
	fmt.Printf("Turn 1 - 推理: %s\n", reasoning)
	fmt.Printf("Turn 1 - 回答: %s\n", content)

	// Turn 2: 拼接多轮对话
	messages := []Message{
		{Role: "user", Content: "9.11 和 9.8 哪个大？"},
	}
	// 将 Turn 1 的响应转为消息追加
	messages = append(messages, ResponseToMessage(ResponseMessage{
		Role:             "assistant",
		Content:          content,
		ReasoningContent: reasoning,
	}))

	// 添加新问题（reasoning_content 无工具调用时可传可不传，这里传了也无妨）
	messages = append(messages, Message{Role: "user", Content: "那么 9.9 和 9.11 呢？"})

	resp, err := client.CreateChatCompletion(&ChatCompletionRequest{
		Messages:       messages,
		Model:          "deepseek-v4-pro",
		Thinking:       &ThinkingConfig{Type: ThinkingEnabled},
		ReasoningEffort: ReasoningEffortHigh,
	})
	if err != nil {
		fmt.Printf("Turn 2 error: %v\n", err)
		return
	}

	choice := resp.Choices[0]
	fmt.Printf("Turn 2 - 推理: %s\n", choice.Message.ReasoningContent)
	fmt.Printf("Turn 2 - 回答: %s\n", choice.Message.Content)
}

// Example_toolCallWithThinking 演示思考模式下的工具调用。
// 模拟 get_date 和 get_weather 两个工具，
// 展示如何正确处理 thinking 模式下的 reasoning_content 回传。
func Example_toolCallWithThinking() {
	token := os.Getenv("DEEPSEEK_API_KEY")
	if token == "" {
		fmt.Println("请设置环境变量 DEEPSEEK_API_KEY")
		return
	}

	client := NewClient(token, WithModel("deepseek-v4-pro"))

	// 定义工具
	tools := []Tool{
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "get_date",
				Description: "获取当前日期",
				Parameters: map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "get_weather",
				Description: "获取指定城市和日期的天气",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]any{
							"type":        "string",
							"description": "城市名称",
						},
						"date": map[string]any{
							"type":        "string",
							"description": "日期，格式 YYYY-mm-dd",
						},
					},
					"required": []string{"location", "date"},
				},
			},
		},
	}

	// 工具实现（模拟）
	execTool := func(call ToolCall) string {
		switch call.Function.Name {
		case "get_date":
			return time.Now().Format("2006-01-02")
		case "get_weather":
			var args struct {
				Location string `json:"location"`
				Date     string `json:"date"`
			}
			json.Unmarshal([]byte(call.Function.Arguments), &args)
			return fmt.Sprintf("%s %s: 多云 7~13°C", args.Location, args.Date)
		}
		return ""
	}

	// 初始化消息列表
	messages := []Message{
		{Role: "user", Content: "杭州明天天气怎么样？"},
	}

	// 工具调用循环
	subTurn := 0
	for {
		subTurn++
		resp, err := client.CreateChatCompletion(&ChatCompletionRequest{
			Messages:       messages,
			Model:          "deepseek-v4-pro",
			Tools:          tools,
			Thinking:       &ThinkingConfig{Type: ThinkingEnabled},
			ReasoningEffort: ReasoningEffortHigh,
		})
		if err != nil {
			fmt.Printf("请求错误: %v\n", err)
			return
		}

		choice := resp.Choices[0]
		msg := choice.Message

		// 将 assistant 消息追加到消息列表（自动携带 reasoning_content 和 tool_calls）
		messages = append(messages, ResponseToMessage(msg))

		fmt.Printf("Turn 1.%d\n", subTurn)
		fmt.Printf("  推理: %s\n", msg.ReasoningContent)
		if msg.Content != "" {
			fmt.Printf("  内容: %s\n", msg.Content)
		}

		// 无工具调用说明模型已给出最终回答
		if len(msg.ToolCalls) == 0 {
			fmt.Printf("  最终回答: %s\n", msg.Content)
			break
		}

		// 执行工具调用并追加结果
		for _, tc := range msg.ToolCalls {
			result := execTool(tc)
			fmt.Printf("  工具调用: %s → %s\n", tc.Function.Name, result)
			messages = append(messages, Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    result,
			})
		}
	}
}

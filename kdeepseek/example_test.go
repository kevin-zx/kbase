package kdeepseek

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Example_simpleChat 演示最简单的单轮对话。
func Example_simpleChat() {
	token := os.Getenv("DEEPSEEK_API_KEY")
	if token == "" {
		fmt.Println("请设置环境变量 DEEPSEEK_API_KEY")
		return
	}

	client := New(token,
		WithModel("deepseek-v4-pro"),
		WithSystem("你是一个有帮助的助手，请用中文回答。"),
	)

	result, err := client.SimpleChat("为什么大海是蓝色的？")
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}
	fmt.Println(result)
}

// Example_thinkingChat 演示思考模式下的单轮对话，返回推理过程和回复内容。
func Example_thinkingChat() {
	token := os.Getenv("DEEPSEEK_API_KEY")
	if token == "" {
		fmt.Println("请设置环境变量 DEEPSEEK_API_KEY")
		return
	}

	client := New(token, WithModel("deepseek-v4-pro"))

	content, reasoning, err := client.ThinkingChat("9.11 和 9.8 哪个大？")
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}
	fmt.Printf("推理过程: %s\n", reasoning)
	fmt.Printf("回答: %s\n", content)
}

// Example_multiTurnChat 演示多轮对话，Chat 自动维护历史。
func Example_multiTurnChat() {
	token := os.Getenv("DEEPSEEK_API_KEY")
	if token == "" {
		fmt.Println("请设置环境变量 DEEPSEEK_API_KEY")
		return
	}

	client := New(token, WithModel("deepseek-v4-pro"))

	chat := client.Chat("你是一个风趣的助手，请用中文回答。")

	// Turn 1
	reply, err := chat.Send("给你讲个笑话。")
	if err != nil {
		fmt.Printf("Turn 1 错误: %v\n", err)
		return
	}
	fmt.Printf("Turn 1: %s\n", reply)

	// Turn 2 — Chat 自动携带上文
	reply, err = chat.Send("再来一个。")
	if err != nil {
		fmt.Printf("Turn 2 错误: %v\n", err)
		return
	}
	fmt.Printf("Turn 2: %s\n", reply)
}

// Example_multiTurnWithThinking 演示多轮对话 + 思考模式。
func Example_multiTurnWithThinking() {
	token := os.Getenv("DEEPSEEK_API_KEY")
	if token == "" {
		fmt.Println("请设置环境变量 DEEPSEEK_API_KEY")
		return
	}

	client := New(token, WithModel("deepseek-v4-pro"))

	chat := client.Chat("你是一个数学老师，请用中文回答。",
		WithChatReasoningEffort(ReasoningEffortHigh),
	)

	// Turn 1
	content, reasoning, err := chat.SendWithThinking("9.11 和 9.8 哪个大？")
	if err != nil {
		fmt.Printf("Turn 1 错误: %v\n", err)
		return
	}
	fmt.Printf("Turn 1 推理: %s\n", reasoning)
	fmt.Printf("Turn 1 回答: %s\n", content)

	// Turn 2 — 上下文自动携带
	content, reasoning, err = chat.SendWithThinking("那 9.9 和 9.11 呢？")
	if err != nil {
		fmt.Printf("Turn 2 错误: %v\n", err)
		return
	}
	fmt.Printf("Turn 2 推理: %s\n", reasoning)
	fmt.Printf("Turn 2 回答: %s\n", content)
}

// Example_streamChat 演示流式多轮对话，内容逐字实时输出。
func Example_streamChat() {
	token := os.Getenv("DEEPSEEK_API_KEY")
	if token == "" {
		fmt.Println("请设置环境变量 DEEPSEEK_API_KEY")
		return
	}

	client := New(token, WithModel("deepseek-v4-pro"))

	chat := client.Chat("你是一个有帮助的助手。")

	ctx := context.Background()
	chunks, errCh := chat.SendStream(ctx, "用 Go 写一个冒泡排序。")

	for chunk := range chunks {
		for _, choice := range chunk.Choices {
			fmt.Print(choice.Delta.Content)
		}
	}
	fmt.Println()

	if err := <-errCh; err != nil {
		fmt.Printf("流式错误: %v\n", err)
		return
	}

	// 流正常结束后，完整回复已自动追加到历史
	reply, err := chat.Send("能加注释吗？")
	if err != nil {
		fmt.Printf("Turn 2 错误: %v\n", err)
		return
	}
	fmt.Println(reply)
}

// Example_toolCall 演示工具调用 + 多轮对话。
// Send 返回工具调用时，通过 AddToolResult 追加结果，再用 Continue 继续。
func Example_toolCall() {
	token := os.Getenv("DEEPSEEK_API_KEY")
	if token == "" {
		fmt.Println("请设置环境变量 DEEPSEEK_API_KEY")
		return
	}

	client := New(token, WithModel("deepseek-v4-pro"))

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

	execTool := func(call ToolCall) string {
		switch call.Function.Name {
		case "get_date":
			return time.Now().Format("2006-01-02")
		case "get_weather":
			var args struct {
				Location string `json:"location"`
				Date     string `json:"date"`
			}
			if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
				return fmt.Sprintf("参数解析错误: %v", err)
			}
			return fmt.Sprintf("%s %s: 多云 7~13°C", args.Location, args.Date)
		}
		return ""
	}

	chat := client.Chat("",
		WithChatTools(tools...),
		WithChatReasoningEffort(ReasoningEffortHigh),
	)

	// 发送初始问题 — 可能触发工具调用
	reply, err := chat.Send("杭州明天天气怎么样？")
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}
	if reply != "" {
		fmt.Printf("回答: %s\n", reply)
	}

	// 工具调用循环
	for {
		history := chat.History()
		last := history[len(history)-1]
		if last.Role != "assistant" || len(last.ToolCalls) == 0 {
			break
		}

		// 执行工具并追加结果
		for _, tc := range last.ToolCalls {
			result := execTool(tc)
			fmt.Printf("工具调用: %s → %s\n", tc.Function.Name, result)
			chat.AddToolResult(tc.ID, tc.Function.Name, result)
		}

		// 继续对话（不追加新 user 消息）
		reply, err = chat.Continue()
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			return
		}
		fmt.Printf("回答: %s\n", reply)
	}
}

// Example_toolCallStreaming 演示流式工具调用 + 多轮对话。
// SendStream 返回工具调用时，实时看到推理过程，
// 通过 AddToolResult 追加结果，再用 ContinueStream 流式继续。
func Example_toolCallStreaming() {
	token := os.Getenv("DEEPSEEK_API_KEY")
	if token == "" {
		fmt.Println("请设置环境变量 DEEPSEEK_API_KEY")
		return
	}

	client := New(token, WithModel("deepseek-v4-pro"))

	getWeatherTool := Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "get_weather",
			Description: "获取指定城市的天气",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"location": map[string]any{
						"type":        "string",
						"description": "城市名称",
					},
				},
				"required": []string{"location"},
			},
		},
	}

	execWeather := func(call ToolCall) string {
		var args struct {
			Location string `json:"location"`
		}
		json.Unmarshal([]byte(call.Function.Arguments), &args)
		return fmt.Sprintf("%s: 晴 15~22°C", args.Location)
	}

	chat := client.Chat("",
		WithChatTools(getWeatherTool),
		WithChatReasoningEffort(ReasoningEffortHigh),
	)

	ctx := context.Background()
	firstRound := true

	for {
		var chunks <-chan StreamChunk
		var errCh <-chan error

		if firstRound {
			chunks, errCh = chat.SendStream(ctx, "北京天气怎么样？")
			firstRound = false
		} else {
			chunks, errCh = chat.ContinueStream(ctx)
		}

		for chunk := range chunks {
			for _, choice := range chunk.Choices {
				if choice.Delta.ReasoningContent != "" {
					fmt.Printf("[推理] %s", choice.Delta.ReasoningContent)
				}
				if choice.Delta.Content != "" {
					fmt.Print(choice.Delta.Content)
				}
			}
		}

		if err := <-errCh; err != nil {
			fmt.Printf("\n流式错误: %v\n", err)
			return
		}
		fmt.Println()

		history := chat.History()
		last := history[len(history)-1]
		if len(last.ToolCalls) == 0 {
			break
		}

		for _, tc := range last.ToolCalls {
			result := execWeather(tc)
			fmt.Printf("工具调用: %s → %s\n", tc.Function.Name, result)
			chat.AddToolResult(tc.ID, tc.Function.Name, result)
		}
	}
}

// Example_jsonOutput 演示 JSON 结构化输出。
func Example_jsonOutput() {
	token := os.Getenv("DEEPSEEK_API_KEY")
	if token == "" {
		fmt.Println("请设置环境变量 DEEPSEEK_API_KEY")
		return
	}

	client := New(token, WithModel("deepseek-v4-pro"))

	// 构建 system prompt（DeepSeek 要求 prompt 中含 "json" 字样和示例）
	systemPrompt, err := JSONPrompt(
		"解析文本中的问答对，输出 JSON。",
		"最高的山是什么？珠穆朗玛峰。",
		map[string]string{
			"question": "最高的山是什么？",
			"answer":   "珠穆朗玛峰",
		},
	)
	if err != nil {
		fmt.Printf("构建 prompt 错误: %v\n", err)
		return
	}

	// 单轮 JSON 完成
	result, err := client.JSONCompletion(systemPrompt, "最长的河是什么？尼罗河。")
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	var output map[string]string
	if err := json.Unmarshal([]byte(result), &output); err != nil {
		fmt.Printf("JSON 解析错误: %v\n", err)
		return
	}
	fmt.Printf("问题: %s\n", output["question"])
	fmt.Printf("答案: %s\n", output["answer"])
}

// Example_jsonOutputMultiTurn 演示 JSON 输出的多轮对话。
func Example_jsonOutputMultiTurn() {
	token := os.Getenv("DEEPSEEK_API_KEY")
	if token == "" {
		fmt.Println("请设置环境变量 DEEPSEEK_API_KEY")
		return
	}

	client := New(token, WithModel("deepseek-v4-pro"))

	systemPrompt, _ := JSONPrompt(
		"解析问答对，输出 JSON。",
		"谁写了《红楼梦》？曹雪芹。",
		map[string]string{"question": "谁写了《红楼梦》？", "answer": "曹雪芹"},
	)

	chat := client.Chat(systemPrompt,
		WithChatResponseFormat(JSONResponseFormat()),
		WithChatMaxTokens(4096),
	)

	var result map[string]string

	// Turn 1
	raw, _ := chat.Send("《三体》的作者是谁？刘慈欣。")
	json.Unmarshal([]byte(raw), &result)
	fmt.Printf("Turn 1 — Q: %s  A: %s\n", result["question"], result["answer"])

	// Turn 2
	raw, _ = chat.Send("《活着》的作者是谁？余华。")
	json.Unmarshal([]byte(raw), &result)
	fmt.Printf("Turn 2 — Q: %s  A: %s\n", result["question"], result["answer"])
}

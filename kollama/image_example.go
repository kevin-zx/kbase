package kollama

import (
	"fmt"
	"log"
)

// ExampleImageChatUsage 展示如何使用图像聊天功能
func ExampleImageChatUsage() {
	// 创建一个新的聊天会话
	chat := NewChat("gemma3", nil) // 使用支持图像的模型，如gemma3

	// 创建用户消息并添加图像
	userMessage := ChatMessage{
		Role:    "user",
		Content: "What is in this image?",
	}

	// 添加base64编码的图像（示例图像数据）
	// 实际使用时，您需要从文件或URL加载图像并编码为base64
	exampleImage := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="
	userMessage.AddImage(exampleImage)

	// 或者使用AddImages添加多个图像
	// userMessage.AddImages([]string{exampleImage1, exampleImage2})

	// 将消息添加到聊天历史
	chat.Messages = append(chat.Messages, userMessage)

	// 发送消息（非流式）
	response, err := chat.SendMessage("")
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	fmt.Printf("Assistant response: %s\n", response.Content)
}

// ExampleImageChatWithStreamUsage 展示如何使用流式图像聊天功能
func ExampleImageChatWithStreamUsage() {
	// 创建支持流式响应的聊天会话
	chat := NewChatWithOptions(
		WithModel("gemma3"),
		WithStream(true),
	)

	// 创建带图像的用户消息
	userMessage := ChatMessage{
		Role:    "user",
		Content: "Describe what you see in this image",
	}

	// 添加图像（这里使用简化的base64数据）
	exampleImage := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="
	userMessage.AddImage(exampleImage)

	// 将消息添加到聊天历史
	chat.Messages = append(chat.Messages, userMessage)

	// 发送流式消息
	response, err := chat.SendStreamMessage("")
	if err != nil {
		log.Fatalf("Failed to send stream message: %v", err)
	}

	fmt.Printf("Assistant stream response: %s\n", response.Content)
}

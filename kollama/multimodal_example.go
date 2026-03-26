package kollama

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

func ExampleMultimodalUsage() {
	chat := NewChatWithOptions(
		WithModel("gemma3:4b"),
	)

	imagePath := "path/to/your/image.jpg"
	imageBase64, err := loadImageAsBase64(imagePath)
	if err != nil {
		log.Fatalf("Failed to load image: %v", err)
	}

	userMsg := NewUserMessage("描述这张图片的内容", imageBase64)

	response, err := chat.SendChatMessage(userMsg)
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	fmt.Println("Response:", response.Content)
}

func loadImageAsBase64(imagePath string) (string, error) {
	imageData, err := ioutil.ReadFile(imagePath)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(imageData), nil
}

func ExampleMultiminalWithSchema() {
	type ImageDescription struct {
		Subject   string   `json:"subject"`
		Action    string   `json:"action"`
		Emotion   string   `json:"emotion"`
		Colors    []string `json:"colors"`
		SceneType string   `json:"scene_type"`
	}

	chat := NewChatWithOptions(WithModel("gemma3:4b"))

	imagePath := "path/to/your/image.jpg"
	imageBase64, err := loadImageAsBase64(imagePath)
	if err != nil {
		log.Fatalf("Failed to load image: %v", err)
	}

	schema := GenerateSchema[ImageDescription]()
	config := JSONStructureConfig{
		SystemPrompt:      "分析图片并提取结构化信息",
		ExampleInput:      "一张图片",
		ExampleJSONOutput: `{"subject":"人物","action":"走路","emotion":"开心","colors":["蓝色","白色"],"scene_type":"户外"}`,
	}

	userMsg := NewUserMessage("分析这张图片", imageBase64)

	response, err := chat.SendSchemaChatMessage(schema, config, userMsg)
	if err != nil {
		log.Fatalf("Failed to send schema message: %v", err)
	}

	fmt.Println("Structured Response:", response.Content)
}

func ExampleMultipleImages() {
	chat := NewChatWithOptions(WithModel("gemma3:4b"))

	images := []string{}
	for i := 1; i <= 2; i++ {
		imagePath := fmt.Sprintf("path/to/image%d.jpg", i)
		data, err := ioutil.ReadFile(imagePath)
		if err != nil {
			log.Printf("Warning: could not read %s: %v", imagePath, err)
			continue
		}
		images = append(images, base64.StdEncoding.EncodeToString(data))
	}

	userMsg := NewUserMessage("比较这两张图片的异同", images...)

	response, err := chat.SendChatMessage(userMsg)
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	fmt.Println("Response:", response.Content)
}

func ExampleMultimodalWithEnvImage() {
	imageURL := os.Getenv("IMAGE_URL")
	_ = imageURL

	chat := NewChatWithOptions(WithModel("gemma3:4b"))

	userMsg := NewUserMessage("这张图片里有什么?", os.Environ()[0])

	response, err := chat.SendChatMessage(userMsg)
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	_ = response
}

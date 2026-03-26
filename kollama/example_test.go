package kollama

import (
	"fmt"
	"os"
	"testing"
)

func TestAddImageFromFile(t *testing.T) {
	// 创建一个临时测试文件
	tempFile, err := os.CreateTemp("", "test_image_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// 写入一些测试数据（模拟图片数据）
	testData := []byte("fake image data for testing")
	if _, err := tempFile.Write(testData); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	tempFile.Close()

	// 测试 AddImageFromFile
	msg := &ChatMessage{}
	err = msg.AddImageFromFile(tempFile.Name())
	if err != nil {
		t.Errorf("AddImageFromFile failed: %v", err)
	}

	if len(msg.Images) != 1 {
		t.Errorf("Expected 1 image, got %d", len(msg.Images))
	}

	// 测试不存在的文件
	msg2 := &ChatMessage{}
	err = msg2.AddImageFromFile("/nonexistent/file.jpg")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestAddImageFromURL(t *testing.T) {
	// 注意：这个测试需要网络连接，并且可能会失败
	// 我们可以测试错误处理，或者使用一个已知会失败的URL
	msg := &ChatMessage{}

	// 测试无效URL
	err := msg.AddImageFromURL("http://invalid-url-that-does-not-exist.example.com/image.jpg")
	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	} else {
		fmt.Printf("Expected error for invalid URL: %v\n", err)
	}
}

func TestAddImageMethodsIntegration(t *testing.T) {
	// 测试所有添加图片的方法一起工作
	msg := &ChatMessage{}

	// 添加 base64 图片
	msg.AddImage("base64data1")
	if len(msg.Images) != 1 {
		t.Errorf("Expected 1 image after AddImage, got %d", len(msg.Images))
	}

	// 添加多个 base64 图片
	msg.AddImages([]string{"base64data2", "base64data3"})
	if len(msg.Images) != 3 {
		t.Errorf("Expected 3 images after AddImages, got %d", len(msg.Images))
	}

	// 验证图片顺序
	expected := []string{"base64data1", "base64data2", "base64data3"}
	for i, img := range msg.Images {
		if img != expected[i] {
			t.Errorf("Image %d: expected %s, got %s", i, expected[i], img)
		}
	}
}

func ExampleChatMessage_AddImageFromFile() {
	msg := &ChatMessage{}

	// 从文件添加图片
	err := msg.AddImageFromFile("path/to/your/image.jpg")
	if err != nil {
		fmt.Printf("Error adding image from file: %v\n", err)
		return
	}

	fmt.Printf("Added image from file. Total images: %d\n", len(msg.Images))
}

func ExampleChatMessage_AddImageFromURL() {
	msg := &ChatMessage{}

	// 从URL添加图片
	err := msg.AddImageFromURL("https://example.com/image.jpg")
	if err != nil {
		fmt.Printf("Error adding image from URL: %v\n", err)
		return
	}

	fmt.Printf("Added image from URL. Total images: %d\n", len(msg.Images))
}

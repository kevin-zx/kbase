package kopenai

import (
	"context"

	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type KOpenAI struct {
	client *openai.Client
	model  string
}

func (k *KOpenAI) SetModel(model string) {
	k.model = model
}

func NewKOpenAI(apiKey, baseUrl string) *KOpenAI {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseUrl),
	)
	return &KOpenAI{
		client: client,
	}
}

func (k *KOpenAI) CreateCompletion(prompt string) (*openai.ChatCompletion, error) {
	chatCompletion, err := k.client.Chat.Completions.New(
		context.TODO(),
		openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(prompt),
			}),
			Model: openai.F(k.model),
		},
	)
	return chatCompletion, err
}

func (k *KOpenAI) CreateStructuredCompletion(prompt string, schema interface{}, schemaName string, schemaDesc string) (*openai.ChatCompletion, error) {
	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        openai.F(schemaName),
		Description: openai.F(schemaDesc),
		Schema:      openai.F(schema),
		Strict:      openai.Bool(true),
	}

	chatCompletion, err := k.client.Chat.Completions.New(
		context.TODO(),
		openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(prompt),
			}),
			Model: openai.F(k.model),
			ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
				openai.ResponseFormatJSONSchemaParam{
					Type:       openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
					JSONSchema: openai.F(schemaParam),
				},
			),
		},
	)
	return chatCompletion, err
}

func GenerateSchema[T any]() interface{} {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false, // 禁止额外字段
		DoNotReference:            true,  // 内联而非引用
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

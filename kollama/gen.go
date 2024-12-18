package kollama

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

type OllamaGen interface {
	Generate(prompt string) (string, error)
	SetModel(model string)
	SetStream(stream bool)
}

type OllamaImpl struct {
	Model  string
	Stream bool
}

func (o *OllamaImpl) Generate(prompt string) (string, error) {
	resp, err := OllamaRequest(prompt)
	if err != nil {
		return "", err
	}
	return resp.Response, nil
}

func (o *OllamaImpl) SetModel(model string) {
	o.Model = model
}

func (o *OllamaImpl) SetStream(stream bool) {
	o.Stream = stream
}

func NewOllama() OllamaGen {
	return &OllamaImpl{
		Model:  "qwen2.5",
		Stream: false,
	}
}

type Setting struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Response           string    `json:"response"`
	Done               bool      `json:"done"`
	DoneReason         string    `json:"done_reason"`
	Context            []int     `json:"context"`
	TotalDuration      int64     `json:"total_duration"`
	LoadDuration       int64     `json:"load_duration"`
	PromptEvalCount    int       `json:"prompt_eval_count"`
	PromptEvalDuration int       `json:"prompt_eval_duration"`
	EvalCount          int       `json:"eval_count"`
	EvalDuration       int64     `json:"eval_duration"`
}

func OllamaRequest(prompt string) (*OllamaResponse, error) {
	url := "http://localhost:11434/api/generate"
	method := "POST"

	pl := Setting{
		Model:  "qwen2.5",
		Prompt: prompt,
		Stream: false,
	}

	payload, err := json.Marshal(pl)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	req, err := http.NewRequest(method, url, strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	// return body, nil
	var OllamaResponse OllamaResponse
	err = json.Unmarshal(body, &OllamaResponse)
	if err != nil {
		return nil, err
	}
	return &OllamaResponse, nil
}

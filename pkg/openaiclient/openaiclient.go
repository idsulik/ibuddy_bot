package openaiclient

import (
	"github.com/sashabaranov/go-openai"
)

type OpenAiClient struct {
	*openai.Client
}

func NewOpenAiClient(apiKey string) *OpenAiClient {
	client := openai.NewClient(apiKey)

	return &OpenAiClient{client}
}

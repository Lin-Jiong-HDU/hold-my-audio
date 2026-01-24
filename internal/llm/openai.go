package llm

import (
	"context"

	"github.com/Lin-Jiong-HDU/hold-my-audio/internal/ai"
)

// OpenAI implements LLMEngine using OpenAI API
type OpenAI struct {
	apiKey  string
	baseURL string
}

// NewOpenAI creates a new OpenAI LLM client
func NewOpenAI(apiKey, baseURL string) ai.LLMEngine {
	return &OpenAI{
		apiKey:  apiKey,
		baseURL: baseURL,
	}
}

// GenerateStream generates a podcast script stream from the given topic
func (o *OpenAI) GenerateStream(ctx context.Context, prompt string) <-chan string {
	ch := make(chan string)
	// TODO: Implement OpenAI streaming API call
	go func() {
		defer close(ch)
		// Implementation will go here
	}()
	return ch
}

// GenerateResponse generates a response to a user question with context
func (o *OpenAI) GenerateResponse(ctx context.Context, question, context string) <-chan string {
	ch := make(chan string)
	// TODO: Implement OpenAI streaming API call
	go func() {
		defer close(ch)
		// Implementation will go here
	}()
	return ch
}

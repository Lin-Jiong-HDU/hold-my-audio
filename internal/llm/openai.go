package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Lin-Jiong-HDU/hold-my-audio/internal/ai"
)

// Default configurations
const (
	defaultModel        = "glm-4.7-flashx"
	defaultChatEndpoint = "/paas/v4/chat/completions"
)

// OpenAI implements LLMEngine using OpenAI API
type OpenAI struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// chatRequest represents the request body for OpenAI Chat Completions API
type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Stream   bool      `json:"stream"`
}

// message represents a chat message
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse represents a streaming response chunk from OpenAI
type chatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
}

// choice represents a choice in the chat response
type choice struct {
	Index        int         `json:"index"`
	Delta        delta       `json:"delta"`
	FinishReason interface{} `json:"finish_reason"`
}

// delta represents the delta content in streaming response
type delta struct {
	Content string `json:"content,omitempty"`
}

// NewOpenAI creates a new OpenAI LLM client
func NewOpenAI(apiKey, baseURL string) ai.LLMEngine {
	return &OpenAI{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   defaultModel,
		client:  &http.Client{},
	}
}

// GenerateStream generates a podcast script stream from the given topic
func (o *OpenAI) GenerateStream(ctx context.Context, prompt string) <-chan string {
	ch := make(chan string)
	go func() {
		defer close(ch)
		if err := o.streamChat(ctx, []message{
			{Role: "system", Content: "You are a podcast script writer. Generate engaging podcast content on the given topic."},
			{Role: "user", Content: prompt},
		}, ch); err != nil {
			// Log error but don't send to channel to avoid breaking consumers
			fmt.Printf("GenerateStream error: %v\n", err)
		}
	}()
	return ch
}

// GenerateResponse generates a response to a user question with context
func (o *OpenAI) GenerateResponse(ctx context.Context, question, context string) <-chan string {
	ch := make(chan string)
	go func() {
		defer close(ch)
		systemPrompt := fmt.Sprintf("You are a helpful assistant answering questions about a podcast. Context: %s", context)
		messages := []message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: question},
		}
		if err := o.streamChat(ctx, messages, ch); err != nil {
			fmt.Printf("GenerateResponse error: %v\n", err)
		}
	}()
	return ch
}

// streamChat performs the actual streaming chat request
func (o *OpenAI) streamChat(ctx context.Context, messages []message, ch chan<- string) error {
	reqBody := chatRequest{
		Model:    o.model,
		Messages: messages,
		Stream:   true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := o.baseURL + defaultChatEndpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk chatResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// Skip malformed chunks
			continue
		}

		if len(chunk.Choices) > 0 {
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				select {
				case ch <- content:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}

	return scanner.Err()
}

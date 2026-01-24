package tts

import (
	"context"

	"github.com/Lin-Jiong-HDU/hold-my-audio/internal/ai"
)

// OpenAI implements TTSEngine using OpenAI TTS API
type OpenAI struct {
	apiKey  string
	baseURL string
	voice   string
}

// NewOpenAI creates a new OpenAI TTS client
func NewOpenAI(apiKey, baseURL string) ai.TTSEngine {
	return &OpenAI{
		apiKey:  apiKey,
		baseURL: baseURL,
		voice:   "alloy", // default voice
	}
}

// SynthesizeStream converts text to audio stream
func (o *OpenAI) SynthesizeStream(ctx context.Context, text string) <-chan []byte {
	ch := make(chan []byte)
	// TODO: Implement OpenAI TTS API call and stream audio chunks
	go func() {
		defer close(ch)
		// Implementation will go here
	}()
	return ch
}

package tts

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestGLM_SynthesizeStream(t *testing.T) {
	apiKey := os.Getenv("GLM_API_KEY")
	if apiKey == "" {
		t.Skip("GLM_API_KEY not set")
	}

	glm := NewGLM(apiKey)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream := glm.SynthesizeStream(ctx, "你好，这是一个测试。")

	received := false
	for chunk := range stream {
		if len(chunk) > 0 {
			received = true
			t.Logf("received chunk: %d bytes", len(chunk))
		}
	}

	if !received {
		t.Error("no audio data received")
	}
}

func TestGLM_Synthesize(t *testing.T) {
	apiKey := os.Getenv("GLM_API_KEY")
	if apiKey == "" {
		t.Skip("GLM_API_KEY not set")
	}

	glm := NewGLM(apiKey)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	data, err := glm.Synthesize(ctx, "你好，这是非流式测试。")
	if err != nil {
		t.Fatalf("Synthesize failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("no audio data received")
	}

	t.Logf("received %d bytes of audio data", len(data))
}

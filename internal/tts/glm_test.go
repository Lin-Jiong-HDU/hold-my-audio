package tts

import (
	"context"
	"os"
	"strings"
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

func TestGLM_SynthesizeStream_LongText(t *testing.T) {
	apiKey := os.Getenv("GLM_API_KEY")
	if apiKey == "" {
		t.Skip("GLM_API_KEY not set")
	}

	glm := NewGLM(apiKey)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 构造一个超过 1024 字符的长文本
	longText := strings.Repeat("这是一段很长的文本，用于测试自动分段功能。", 50)

	t.Logf("Testing with text length: %d characters", len(longText))

	stream := glm.SynthesizeStream(ctx, longText)

	chunkCount := 0
	totalBytes := 0
	for chunk := range stream {
		if len(chunk) > 0 {
			chunkCount++
			totalBytes += len(chunk)
			t.Logf("received chunk %d: %d bytes", chunkCount, len(chunk))
		}
	}

	if chunkCount == 0 {
		t.Error("no audio data received")
	}

	t.Logf("Total: %d chunks, %d bytes", chunkCount, totalBytes)
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

func TestSplitText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLess int // 每段都应该 <= 1024
	}{
		{
			name:     "短文本",
			input:    "你好",
			wantLess: 1024,
		},
		{
			name:     "刚好 1024 字符",
			input:    strings.Repeat("a", 1024),
			wantLess: 1024,
		},
		{
			name:     "超过 1024 字符",
			input:    strings.Repeat("这是一个测试句子。", 100),
			wantLess: 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := splitText(tt.input)
			for i, chunk := range chunks {
				if len(chunk) > tt.wantLess {
					t.Errorf("chunk %d length %d > %d", i, len(chunk), tt.wantLess)
				}
			}
		})
	}
}

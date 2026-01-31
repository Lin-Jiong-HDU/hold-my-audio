package tts

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// GLM implements TTSEngine using Zhipu AI GLM-TTS API
type GLM struct {
	apiKey  string
	voice   string
	speed   float64
	volume  float64
	baseURL string
}

// NewGLM creates a new GLM-TTS client
func NewGLM(apiKey string) *GLM {
	return &GLM{
		apiKey:  apiKey,
		voice:   "tongtong", // 默认音色
		speed:   1.0,
		volume:  1.0,
		baseURL: "https://open.bigmodel.cn/api/paas/v4/audio/speech",
	}
}

// SetVoice sets the voice for TTS
func (g *GLM) SetVoice(voice string) {
	g.voice = voice
}

// SetSpeed sets the speech speed [0.5, 2.0]
func (g *GLM) SetSpeed(speed float64) {
	g.speed = speed
}

// SetVolume sets the volume (0, 10]
func (g *GLM) SetVolume(volume float64) {
	g.volume = volume
}

// glmRequest represents the request body for GLM-TTS API
type glmRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format"`
	Stream         bool    `json:"stream"`
	Speed          float64 `json:"speed,omitempty"`
	Volume         float64 `json:"volume,omitempty"`
	EncodeFormat   string  `json:"encode_format,omitempty"`
}

// SynthesizeStream converts text to audio stream
func (g *GLM) SynthesizeStream(ctx context.Context, text string) <-chan []byte {
	ch := make(chan []byte)

	go func() {
		defer close(ch)

		// 构建请求
		reqBody := glmRequest{
			Model:          "glm-tts",
			Input:          text,
			Voice:          g.voice,
			ResponseFormat: "pcm",
			Stream:         true,
			EncodeFormat:   "base64",
		}
		if g.speed != 1.0 {
			reqBody.Speed = g.speed
		}
		if g.volume != 1.0 {
			reqBody.Volume = g.volume
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return
		}

		// 创建 HTTP 请求
		req, err := http.NewRequestWithContext(ctx, "POST", g.baseURL, strings.NewReader(string(jsonData)))
		if err != nil {
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+g.apiKey)

		// 发送请求
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return
		}

		// 手动读取 SSE 流（因为行可能很长，超过 scanner 的默认缓冲区）
		reader := bufio.NewReader(resp.Body)

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}

			line = strings.TrimSuffix(line, "\n")

			// SSE 格式: "data: {json}"
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			jsonStr := strings.TrimPrefix(line, "data: ")
			if jsonStr == "" || jsonStr == "[DONE]" {
				continue
			}

			// 解析 JSON，提取 content 字段
			var data struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}

			if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
				continue
			}

			if len(data.Choices) == 0 {
				continue
			}

			base64Data := data.Choices[0].Delta.Content
			if base64Data == "" {
				continue
			}

			// 解码 base64
			audioData, err := base64.StdEncoding.DecodeString(base64Data)
			if err != nil {
				continue
			}

			// 发送到 channel（需要检查 context 是否已取消）
			select {
			case ch <- audioData:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch
}

// Synthesize converts text to audio (non-streaming, returns all data at once)
func (g *GLM) Synthesize(ctx context.Context, text string) ([]byte, error) {
	reqBody := glmRequest{
		Model:          "glm-tts",
		Input:          text,
		Voice:          g.voice,
		ResponseFormat: "wav",
		Stream:         false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", g.baseURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

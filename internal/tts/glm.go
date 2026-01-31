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

const (
	maxInputLength = 1024 // GLM-TTS API 限制
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
	if speed < 0.5 {
		speed = 0.5
	} else if speed > 2.0 {
		speed = 2.0
	}
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

// splitText 将文本分段，每段不超过 maxInputLength 字节
// 优先按句子分割，避免在词语中间断开
func splitText(text string) []string {
	var chunks []string

	for len(text) > 0 {
		// 如果剩余文本小于限制，直接添加
		if len(text) <= maxInputLength {
			chunks = append(chunks, text)
			break
		}

		// 找到合适的分割点
		splitPos := findSplitPoint(text[:maxInputLength])

		// 添加这段
		chunks = append(chunks, text[:splitPos])
		text = text[splitPos:]
	}

	return chunks
}

// findSplitPoint 在文本中找到合适的分割点
// 优先找句号，其次逗号，最后强制在 80% 位置分割
func findSplitPoint(s string) int {
	// 常见中英文句末标点
	sentenceEnds := []string{"。", "！", "？", ".", "!", "?"}
	commas := []string{"，", ",", "；", ";"}

	// 先找句号
	for _, marker := range sentenceEnds {
		if idx := strings.LastIndex(s, marker); idx > len(s)/2 {
			return idx + len(marker)
		}
	}

	// 找不到句号，找逗号
	for _, marker := range commas {
		if idx := strings.LastIndex(s, marker); idx > len(s)/2 {
			return idx + len(marker)
		}
	}

	// 都找不到，在 80% 位置强制分割，确保不切在多字节字符中间
	splitPos := len(s) * 4 / 5
	for splitPos < len(s) {
		if s[splitPos]&0xC0 != 0x80 {
			return splitPos
		}
		splitPos++
	}

	return len(s)
}

// SynthesizeStream converts text to audio stream
func (g *GLM) SynthesizeStream(ctx context.Context, text string) <-chan []byte {
	ch := make(chan []byte)

	go func() {
		defer close(ch)

		// 如果文本为空，直接返回
		if len(text) == 0 {
			return
		}

		// 分段处理长文本
		chunks := splitText(text)

		for _, chunk := range chunks {
			// 检查 context 是否已取消
			select {
			case <-ctx.Done():
				return
			default:
			}

			// 构建请求
			reqBody := glmRequest{
				Model:          "glm-tts",
				Input:          chunk,
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

			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
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
					resp.Body.Close()
					return
				}
			}

			resp.Body.Close()
		}
	}()

	return ch
}

// Synthesize converts text to audio (non-streaming, returns all data at once)
func (g *GLM) Synthesize(ctx context.Context, text string) ([]byte, error) {
	if len(text) == 0 {
		return nil, fmt.Errorf("empty text")
	}

	// 分段处理长文本
	chunks := splitText(text)
	var allAudioData []byte

	for _, chunk := range chunks {
		reqBody := glmRequest{
			Model:          "glm-tts",
			Input:          chunk,
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

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("API error: status=%d, body=%s", resp.StatusCode, string(body))
		}

		audioData, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}

		allAudioData = append(allAudioData, audioData...)
	}

	return allAudioData, nil
}

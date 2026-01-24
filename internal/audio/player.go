package audio

import (
	"context"

	"github.com/Lin-Jiong-HDU/hold-my-audio/internal/ai"
)

// Player implements AudioPlayer using oto library
type Player struct {
	sampleRate int
	channels   int
	// TODO: Add oto.Player or similar audio device handle
}

// NewPlayer creates a new audio player
func NewPlayer() ai.AudioPlayer {
	return &Player{
		sampleRate: 24000, // default sample rate for OpenAI TTS
		channels:   1,     // mono
	}
}

// PlayStream plays audio from a stream channel
func (p *Player) PlayStream(ctx context.Context, audioStream <-chan []byte) error {
	// TODO: Implement audio playback using oto library
	// - Initialize oto player
	// - Read chunks from audioStream
	// - Write to audio device
	// - Handle context cancellation
	for chunk := range audioStream {
		_ = chunk // TODO: Play chunk
	}
	return nil
}

// Stop immediately stops the current playback
func (p *Player) Stop() error {
	// TODO: Stop audio playback and cleanup
	return nil
}

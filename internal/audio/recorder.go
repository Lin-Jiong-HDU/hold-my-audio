package audio

import (
	"context"

	"github.com/Lin-Jiong-HDU/hold-my-audio/internal/ai"
)

// Recorder implements AudioRecorder using microphone capture and Whisper API
type Recorder struct {
	apiKey string
	// TODO: Add audio capture device handle
}

// NewRecorder creates a new audio recorder
func NewRecorder(apiKey string) ai.AudioRecorder {
	return &Recorder{
		apiKey: apiKey,
	}
}

// Record captures audio and returns transcribed text
func (r *Recorder) Record(ctx context.Context) (string, error) {
	// TODO: Implement audio recording and STT
	// - Capture audio from microphone
	// - Detect silence/end of speech
	// - Send to OpenAI Whisper API for transcription
	// - Return transcribed text
	return "", nil
}

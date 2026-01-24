package ai

import "context"

// LLMEngine defines the interface for LLM providers
type LLMEngine interface {
	// GenerateStream generates a podcast script stream from the given topic
	GenerateStream(ctx context.Context, prompt string) <-chan string

	// GenerateResponse generates a response to a user question with context
	GenerateResponse(ctx context.Context, question, context string) <-chan string
}

// TTSEngine defines the interface for text-to-speech providers
type TTSEngine interface {
	// SynthesizeStream converts text to audio stream
	// Returns a channel of audio byte chunks
	SynthesizeStream(ctx context.Context, text string) <-chan []byte
}

// VADMonitor defines the interface for voice activity detection
type VADMonitor interface {
	// Start begins monitoring for voice activity
	// Sends a signal when voice is detected
	Start(ctx context.Context) <-chan struct{}

	// Stop stops the monitoring process
	Stop() error
}

// AudioPlayer defines the interface for audio playback
type AudioPlayer interface {
	// PlayStream plays audio from a stream channel
	// Blocks until playback is complete or context is cancelled
	PlayStream(ctx context.Context, audioStream <-chan []byte) error

	// Stop immediately stops the current playback
	Stop() error
}

// AudioRecorder defines the interface for audio recording and STT
type AudioRecorder interface {
	// Record captures audio and returns transcribed text
	// Blocks until recording is complete or context is cancelled
	Record(ctx context.Context) (string, error)
}

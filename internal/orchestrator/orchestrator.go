package orchestrator

import (
	"context"
	"log"
	"sync"

	"github.com/Lin-Jiong-HDU/hold-my-audio/internal/ai"
)

// Orchestrator manages the podcast playback and interruption flow
type Orchestrator struct {
	state      State
	stateMu    sync.RWMutex
	ctx        context.Context
	cancelFunc context.CancelFunc
	llm        ai.LLMEngine
	tts        ai.TTSEngine
	vad        ai.VADMonitor
	player     ai.AudioPlayer
	recorder   ai.AudioRecorder
}

// New creates a new Orchestrator instance
func New(llm ai.LLMEngine, tts ai.TTSEngine, vad ai.VADMonitor, player ai.AudioPlayer, recorder ai.AudioRecorder) *Orchestrator {
	return &Orchestrator{
		state:    IDLE,
		llm:      llm,
		tts:      tts,
		vad:      vad,
		player:   player,
		recorder: recorder,
	}
}

// GetState returns the current state (thread-safe)
func (o *Orchestrator) GetState() State {
	o.stateMu.RLock()
	defer o.stateMu.RUnlock()
	return o.state
}

// setState updates the current state (thread-safe)
func (o *Orchestrator) setState(newState State) {
	o.stateMu.Lock()
	defer o.stateMu.Unlock()
	log.Printf("[Orchestrator] State transition: %s -> %s", o.state, newState)
	o.state = newState
}

// Start begins the podcast with the given topic
func (o *Orchestrator) Start(topic string) error {
	o.setState(PLAYING)

	ctx, cancel := context.WithCancel(context.Background())
	o.ctx = ctx
	o.cancelFunc = cancel

	// Generate initial podcast script stream
	scriptStream := o.llm.GenerateStream(ctx, topic)

	// Start playback loop
	go o.playbackLoop(scriptStream)

	// Start monitoring for interruptions
	go o.monitorInterruption()

	return nil
}

// playbackLoop manages the continuous playback of the podcast
func (o *Orchestrator) playbackLoop(scriptStream <-chan string) {
	for {
		select {
		case text, ok := <-scriptStream:
			if !ok {
				// Script stream ended
				log.Println("[Orchestrator] Script stream ended")
				o.setState(IDLE)
				return
			}

			log.Printf("[Orchestrator] Processing text chunk: %s", text)

			// Convert text to audio stream
			audioStream := o.tts.SynthesizeStream(o.ctx, text)

			// Play audio
			if err := o.player.PlayStream(o.ctx, audioStream); err != nil {
				log.Printf("[Orchestrator] Playback error: %v", err)
			}

		case <-o.ctx.Done():
			log.Println("[Orchestrator] Playback loop cancelled")
			return
		}
	}
}

// monitorInterruption listens for voice activity and triggers interruption handling
func (o *Orchestrator) monitorInterruption() {
	vadSignal := o.vad.Start(o.ctx)

	for {
		select {
		case _, ok := <-vadSignal:
			if !ok {
				log.Println("[Orchestrator] VAD monitor closed")
				return
			}
			log.Println("[Orchestrator] Voice detected, triggering interruption")
			o.handleInterruption()

		case <-o.ctx.Done():
			log.Println("[Orchestrator] Monitor interruption cancelled")
			return
		}
	}
}

// handleInterruption processes user interruption and generates response
func (o *Orchestrator) handleInterruption() {
	o.setState(INTERRUPTED)

	// Stop current playback immediately
	log.Println("[Orchestrator] Stopping current playback")
	o.cancelFunc()
	o.player.Stop()

	// Record user question
	log.Println("[Orchestrator] Recording user question")
	o.setState(THINKING)
	question, err := o.recorder.Record(o.ctx)
	if err != nil {
		log.Printf("[Orchestrator] Recording error: %v", err)
		o.setState(PLAYING)
		return
	}

	log.Printf("[Orchestrator] User question: %s", question)

	// Generate response stream
	responseStream := o.llm.GenerateResponse(o.ctx, question, "")

	// Play response
	for text := range responseStream {
		log.Printf("[Orchestrator] Response chunk: %s", text)

		audioStream := o.tts.SynthesizeStream(o.ctx, text)
		if err := o.player.PlayStream(o.ctx, audioStream); err != nil {
			log.Printf("[Orchestrator] Response playback error: %v", err)
		}
	}

	// Resume or update remaining script
	o.setState(PLAYING)
	log.Println("[Orchestrator] Response completed, resuming playback")
}

// Stop stops the orchestrator
func (o *Orchestrator) Stop() {
	log.Println("[Orchestrator] Stopping orchestrator")
	if o.cancelFunc != nil {
		o.cancelFunc()
	}
	o.player.Stop()
	o.vad.Stop()
	o.setState(IDLE)
}

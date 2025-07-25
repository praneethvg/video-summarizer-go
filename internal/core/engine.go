package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"video-summarizer-go/internal/config"
	"video-summarizer-go/internal/core/tasks"
	"video-summarizer-go/internal/interfaces"
)

type ProcessingEngine struct {
	store      interfaces.StateStore
	eventBus   interfaces.EventBus
	taskQueue  interfaces.TaskQueue
	workerPool *WorkerPool

	videoProvider         interfaces.VideoProvider
	audioProcessor        interfaces.AudioProcessor
	transcriptionProvider interfaces.TranscriptionProvider
	summarizationProvider interfaces.SummarizationProvider
	outputProvider        interfaces.OutputProvider
	promptManager         *config.PromptManager
	taskProcessorRegistry *tasks.TaskProcessorRegistry

	mu sync.Mutex
}

func NewProcessingEngine(
	store interfaces.StateStore,
	eventBus interfaces.EventBus,
	taskQueue interfaces.TaskQueue,
	workerPool *WorkerPool,
	videoProvider interfaces.VideoProvider,
	audioProcessor interfaces.AudioProcessor,
	transcriptionProvider interfaces.TranscriptionProvider,
	summarizationProvider interfaces.SummarizationProvider,
	outputProvider interfaces.OutputProvider,
	promptManager *config.PromptManager,
) *ProcessingEngine {
	engine := &ProcessingEngine{
		store:                 store,
		eventBus:              eventBus,
		taskQueue:             taskQueue,
		workerPool:            workerPool,
		videoProvider:         videoProvider,
		audioProcessor:        audioProcessor,
		transcriptionProvider: transcriptionProvider,
		summarizationProvider: summarizationProvider,
		outputProvider:        outputProvider,
		promptManager:         promptManager,
		taskProcessorRegistry: tasks.NewTaskProcessorRegistry(),
	}
	engine.registerEventHandlers()
	return engine
}

func (e *ProcessingEngine) registerEventHandlers() {
	e.eventBus.Subscribe("VideoProcessingRequested", e.onVideoProcessingRequested)
	e.eventBus.Subscribe("VideoInfoFetched", e.onVideoInfoFetched)
	e.eventBus.Subscribe("AudioDownloaded", e.onAudioDownloaded)
	e.eventBus.Subscribe(interfaces.EventTypeTranscriptionCompleted, e.onTranscriptionCompleted)
	e.eventBus.Subscribe(interfaces.EventTypeSummarizationCompleted, e.onSummarizationCompleted)
	e.eventBus.Subscribe(interfaces.EventTypeOutputCompleted, e.onOutputCompleted)
}

// Entry point: create a new request and emit VideoProcessingRequested
func (e *ProcessingEngine) StartRequest(requestID, url string, prompt interfaces.Prompt, sourceType string, category string, maxTokens int) error {
	log.Debugf("[Engine] StartRequest called for requestID: %s, url: %s, sourceType: %s, category: %s", requestID, url, sourceType, category)
	state := &interfaces.ProcessingState{
		RequestID:  requestID,
		Status:     interfaces.StatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		SourceType: sourceType,
		URL:        url,
		Prompt:     prompt,
		MaxTokens:  maxTokens,
		// Add category as a top-level field if you want, or handle it elsewhere
	}
	e.store.SaveRequestState(requestID, state)
	log.Debugf("Publishing VideoProcessingRequested event for requestID: %s", requestID)
	e.eventBus.Publish(interfaces.Event{
		ID:        fmt.Sprintf("evt-%s-%d", requestID, time.Now().UnixNano()),
		RequestID: requestID,
		Type:      "VideoProcessingRequested",
		Data:      map[string]interface{}{"url": url},
		Timestamp: time.Now(),
	})
	return nil
}

// GetRequestState gets the current state of a processing request
func (e *ProcessingEngine) GetRequestState(requestID string) (*interfaces.ProcessingState, error) {
	return e.store.GetRequestState(requestID)
}

// CancelRequest cancels a processing request
func (e *ProcessingEngine) CancelRequest(requestID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	state, err := e.store.GetRequestState(requestID)
	if err != nil {
		return fmt.Errorf("request not found: %s", requestID)
	}

	if state.Status == interfaces.StatusCompleted || state.Status == interfaces.StatusFailed || state.Status == interfaces.StatusCancelled {
		return fmt.Errorf("request %s is already in final state: %s", requestID, state.Status)
	}

	// Update state to cancelled
	err = e.store.UpdateRequestState(requestID, map[string]interface{}{
		"status":       interfaces.StatusCancelled,
		"completed_at": time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed to update request state: %w", err)
	}

	// Publish cancellation event
	e.eventBus.Publish(interfaces.Event{
		ID:        fmt.Sprintf("evt-%s-cancelled-%d", requestID, time.Now().UnixNano()),
		RequestID: requestID,
		Type:      "RequestCancelled",
		Data:      map[string]interface{}{"cancelled_at": time.Now()},
		Timestamp: time.Now(),
	})

	return nil
}

// GetRequestCountsByStatus returns a map of status to count
func (e *ProcessingEngine) GetRequestCountsByStatus() map[string]int {
	return e.store.GetRequestCountsByStatus()
}

// Start starts the processing engine (workers are already started when limits are set)
func (e *ProcessingEngine) Start() {
	// Workers are already running when concurrency limits are set
	// This method is kept for API consistency
}

// Stop stops the processing engine
func (e *ProcessingEngine) Stop() {
	e.workerPool.Stop()
}

// GetVideoProvider returns the video provider
func (e *ProcessingEngine) GetVideoProvider() interfaces.VideoProvider {
	return e.videoProvider
}

// GetTranscriptionProvider returns the transcription provider
func (e *ProcessingEngine) GetTranscriptionProvider() interfaces.TranscriptionProvider {
	return e.transcriptionProvider
}

// GetSummarizationProvider returns the summarization provider
func (e *ProcessingEngine) GetSummarizationProvider() interfaces.SummarizationProvider {
	return e.summarizationProvider
}

// GetOutputProvider returns the output provider
func (e *ProcessingEngine) GetOutputProvider() interfaces.OutputProvider {
	return e.outputProvider
}

// GetPromptManager returns the prompt manager
func (e *ProcessingEngine) GetPromptManager() *config.PromptManager {
	return e.promptManager
}

// GetStore returns the state store
func (e *ProcessingEngine) GetStore() interfaces.StateStore {
	return e.store
}

// GetEventBus returns the event bus
func (e *ProcessingEngine) GetEventBus() interfaces.EventBus {
	return e.eventBus
}

// GetTaskQueue returns the task queue
func (e *ProcessingEngine) GetTaskQueue() interfaces.TaskQueue {
	return e.taskQueue
}

func (e *ProcessingEngine) onVideoProcessingRequested(event interfaces.Event) {
	log.Debugf("[Engine] Received VideoProcessingRequested event for request: %s", event.RequestID)
	state, err := e.store.GetRequestState(event.RequestID)
	if err != nil {
		log.Errorf("Could not get state for request: %s", event.RequestID)
		return
	}
	url := state.URL
	log.Debugf("[Engine] Enqueueing video info task for request: %s, URL: %s", event.RequestID, url)
	e.taskQueue.Enqueue(&interfaces.Task{
		ID:        fmt.Sprintf("task-%s-video-%d", event.RequestID, time.Now().UnixNano()),
		Type:      interfaces.TaskVideoInfo,
		RequestID: event.RequestID,
		Data:      map[string]interface{}{"url": url},
		CreatedAt: time.Now(),
	})
	e.store.UpdateRequestState(event.RequestID, map[string]interface{}{
		"status": interfaces.StatusRunning,
	})
}

func (e *ProcessingEngine) onVideoInfoFetched(event interfaces.Event) {
	state, err := e.store.GetRequestState(event.RequestID)
	if err != nil {
		log.Errorf("Could not get state for request: %s", event.RequestID)
		return
	}
	url := state.URL
	e.taskQueue.Enqueue(&interfaces.Task{
		ID:        fmt.Sprintf("task-%s-audio-%d", event.RequestID, time.Now().UnixNano()),
		Type:      interfaces.TaskAudioDownload,
		RequestID: event.RequestID,
		Data:      map[string]interface{}{"url": url},
		CreatedAt: time.Now(),
	})
	// Optionally update video_info if needed
}

func (e *ProcessingEngine) onAudioDownloaded(event interfaces.Event) {
	state, err := e.store.GetRequestState(event.RequestID)
	if err != nil {
		log.Errorf("Could not get state for request: %s", event.RequestID)
		return
	}
	audioPath := state.AudioPath
	if audioPath == "" {
		audioPath = event.Data["audio_path"].(string)
	}
	e.taskQueue.Enqueue(&interfaces.Task{
		ID:        fmt.Sprintf("task-%s-transcribe-%d", event.RequestID, time.Now().UnixNano()),
		Type:      interfaces.TaskTranscription,
		RequestID: event.RequestID,
		Data:      map[string]interface{}{"audio_path": audioPath},
		CreatedAt: time.Now(),
	})
}

func (e *ProcessingEngine) onTranscriptionCompleted(event interfaces.Event) {
	state, err := e.store.GetRequestState(event.RequestID)
	if err != nil {
		log.Errorf("Could not get state for request: %s", event.RequestID)
		return
	}
	transcriptPath := state.Transcript
	if transcriptPath == "" {
		transcriptPath = event.Data["transcript"].(string)
	}
	e.taskQueue.Enqueue(&interfaces.Task{
		ID:        fmt.Sprintf("task-%s-summarize-%d", event.RequestID, time.Now().UnixNano()),
		Type:      interfaces.TaskSummarization,
		RequestID: event.RequestID,
		Data:      map[string]interface{}{"transcript_path": transcriptPath},
		CreatedAt: time.Now(),
	})
}

func (e *ProcessingEngine) onSummarizationCompleted(event interfaces.Event) {
	state, err := e.store.GetRequestState(event.RequestID)
	if err != nil {
		log.Errorf("Could not get state for request: %s", event.RequestID)
		return
	}
	summaryPath := state.Summary
	if summaryPath == "" {
		summaryPath = event.Data["summary"].(string)
	}
	log.Debugf("onSummarizationCompleted called for request: %s, summaryPath: %v", event.RequestID, summaryPath)
	e.taskQueue.Enqueue(&interfaces.Task{
		ID:        fmt.Sprintf("task-%s-output-%d", event.RequestID, time.Now().UnixNano()),
		Type:      interfaces.TaskOutput,
		RequestID: event.RequestID,
		Data:      map[string]interface{}{"summary_path": summaryPath},
		CreatedAt: time.Now(),
	})
}

func (e *ProcessingEngine) onOutputCompleted(event interfaces.Event) {
	log.Debugf("onOutputCompleted called for request: %s", event.RequestID)
	e.taskQueue.Enqueue(&interfaces.Task{
		ID:        fmt.Sprintf("task-%s-cleanup-%d", event.RequestID, time.Now().UnixNano()),
		Type:      interfaces.TaskCleanup,
		RequestID: event.RequestID,
		Data:      map[string]interface{}{},
		CreatedAt: time.Now(),
	})
}

// Worker processing logic (real plugins where available)
func (e *ProcessingEngine) WorkerProcess(task *interfaces.Task) {
	log.Infof("WorkerProcess called for task: %s, request: %s", task.Type, task.RequestID)

	// Use task processor
	if processor, exists := e.taskProcessorRegistry.GetProcessor(task.Type); exists {
		if err := processor.Process(context.Background(), task, e); err != nil {
			log.Errorf("Task processor failed for %s: %v", task.Type, err)
		}
		return
	}

	// Fallback for unknown task types
	log.Errorf("No processor found for task type: %s", task.Type)
}

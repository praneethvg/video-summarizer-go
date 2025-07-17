package core

import (
	"fmt"
	"os"
	"sync"
	"time"

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
	}
	engine.registerEventHandlers()
	return engine
}

func (e *ProcessingEngine) registerEventHandlers() {
	e.eventBus.Subscribe("VideoProcessingRequested", e.onVideoProcessingRequested)
	e.eventBus.Subscribe("VideoInfoFetched", e.onVideoInfoFetched)
	e.eventBus.Subscribe("AudioDownloaded", e.onAudioDownloaded)
	e.eventBus.Subscribe("TranscriptionCompleted", e.onTranscriptionCompleted)
	e.eventBus.Subscribe("SummarizationCompleted", e.onSummarizationCompleted)
}

// Entry point: create a new request and emit VideoProcessingRequested
func (e *ProcessingEngine) StartRequest(requestID, url string) error {
	state := &interfaces.ProcessingState{
		RequestID: requestID,
		Status:    interfaces.StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  map[string]interface{}{"url": url},
	}
	e.store.SaveRequestState(requestID, state)
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

func (e *ProcessingEngine) onVideoProcessingRequested(event interfaces.Event) {
	url := event.Data["url"].(string)
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
	videoInfo := event.Data
	e.taskQueue.Enqueue(&interfaces.Task{
		ID:        fmt.Sprintf("task-%s-audio-%d", event.RequestID, time.Now().UnixNano()),
		Type:      interfaces.TaskAudioDownload,
		RequestID: event.RequestID,
		Data:      videoInfo,
		CreatedAt: time.Now(),
	})
	e.store.UpdateRequestState(event.RequestID, map[string]interface{}{
		"video_info": videoInfo,
	})
}

func (e *ProcessingEngine) onAudioDownloaded(event interfaces.Event) {
	audioPath := event.Data["audio_path"].(string)
	e.taskQueue.Enqueue(&interfaces.Task{
		ID:        fmt.Sprintf("task-%s-transcribe-%d", event.RequestID, time.Now().UnixNano()),
		Type:      interfaces.TaskTranscription,
		RequestID: event.RequestID,
		Data:      map[string]interface{}{"audio_path": audioPath},
		CreatedAt: time.Now(),
	})
	e.store.UpdateRequestState(event.RequestID, map[string]interface{}{
		"audio_path": audioPath,
	})
}

func (e *ProcessingEngine) onTranscriptionCompleted(event interfaces.Event) {
	transcriptPath := event.Data["transcript"].(string)
	e.taskQueue.Enqueue(&interfaces.Task{
		ID:        fmt.Sprintf("task-%s-summarize-%d", event.RequestID, time.Now().UnixNano()),
		Type:      interfaces.TaskSummarization,
		RequestID: event.RequestID,
		Data:      map[string]interface{}{"transcript_path": transcriptPath},
		CreatedAt: time.Now(),
	})
	e.store.UpdateRequestState(event.RequestID, map[string]interface{}{
		"transcript": transcriptPath,
	})
}

func (e *ProcessingEngine) onSummarizationCompleted(event interfaces.Event) {
	summaryPath := event.Data["summary"].(string)
	e.taskQueue.Enqueue(&interfaces.Task{
		ID:        fmt.Sprintf("task-%s-output-%d", event.RequestID, time.Now().UnixNano()),
		Type:      interfaces.TaskOutput,
		RequestID: event.RequestID,
		Data:      map[string]interface{}{"summary_path": summaryPath},
		CreatedAt: time.Now(),
	})
	e.store.UpdateRequestState(event.RequestID, map[string]interface{}{
		"summary": summaryPath,
	})
}

// Worker processing logic (real plugins where available)
func (e *ProcessingEngine) WorkerProcess(task *interfaces.Task) {
	fmt.Printf("[Engine] WorkerProcess called for task: %s, request: %s\n", task.Type, task.RequestID)
	switch task.Type {
	case interfaces.TaskVideoInfo:
		fmt.Printf("[Engine] Processing TaskVideoInfo for request: %s\n", task.RequestID)
		// Use real video provider
		url := task.Data.(map[string]interface{})["url"].(string)
		videoInfo, err := e.videoProvider.GetVideoInfo(url)
		if err != nil {
			e.store.UpdateRequestState(task.RequestID, map[string]interface{}{
				"status": interfaces.StatusFailed,
				"error":  fmt.Sprintf("Failed to get video info: %v", err),
			})
			return
		}

		audioPath, err := e.videoProvider.DownloadAudio(url)
		if err != nil {
			e.store.UpdateRequestState(task.RequestID, map[string]interface{}{
				"status": interfaces.StatusFailed,
				"error":  fmt.Sprintf("Failed to download audio: %v", err),
			})
			return
		}

		// Publish video info event
		e.eventBus.Publish(interfaces.Event{
			ID:        fmt.Sprintf("evt-%s-videoinfo-%d", task.RequestID, time.Now().UnixNano()),
			RequestID: task.RequestID,
			Type:      "VideoInfoFetched",
			Data:      videoInfo,
			Timestamp: time.Now(),
		})

		// Publish audio downloaded event
		e.eventBus.Publish(interfaces.Event{
			ID:        fmt.Sprintf("evt-%s-audio-%d", task.RequestID, time.Now().UnixNano()),
			RequestID: task.RequestID,
			Type:      "AudioDownloaded",
			Data:      map[string]interface{}{"audio_path": audioPath},
			Timestamp: time.Now(),
		})

	case interfaces.TaskAudioDownload:
		fmt.Printf("[Engine] Processing TaskAudioDownload for request: %s\n", task.RequestID)
		// This step is now handled in TaskVideoInfo, so this should not be reached
		// Keep for backward compatibility but log a warning
		fmt.Printf("Warning: TaskAudioDownload received but audio download is now handled in TaskVideoInfo\n")

	case interfaces.TaskTranscription:
		fmt.Printf("[Engine] Processing TaskTranscription for request: %s\n", task.RequestID)
		// Use real transcription provider
		audioPath := task.Data.(map[string]interface{})["audio_path"].(string)
		transcriptPath, err := e.transcriptionProvider.TranscribeAudio(audioPath)
		if err != nil {
			e.store.UpdateRequestState(task.RequestID, map[string]interface{}{
				"status": interfaces.StatusFailed,
				"error":  fmt.Sprintf("Failed to transcribe audio: %v", err),
			})
			return
		}
		e.eventBus.Publish(interfaces.Event{
			ID:        fmt.Sprintf("evt-%s-transcript-%d", task.RequestID, time.Now().UnixNano()),
			RequestID: task.RequestID,
			Type:      "TranscriptionCompleted",
			Data:      map[string]interface{}{"transcript": transcriptPath},
			Timestamp: time.Now(),
		})

	case interfaces.TaskSummarization:
		fmt.Printf("[Engine] Processing TaskSummarization for request: %s\n", task.RequestID)
		// Use real summarization provider
		transcriptPath := task.Data.(map[string]interface{})["transcript_path"].(string)
		transcriptBytes, err := os.ReadFile(transcriptPath)
		if err != nil {
			e.store.UpdateRequestState(task.RequestID, map[string]interface{}{
				"status": interfaces.StatusFailed,
				"error":  fmt.Sprintf("Failed to read transcript file: %v", err),
			})
			return
		}

		// Get prompt from request metadata
		state, err := e.store.GetRequestState(task.RequestID)
		if err != nil {
			e.store.UpdateRequestState(task.RequestID, map[string]interface{}{
				"status": interfaces.StatusFailed,
				"error":  fmt.Sprintf("Failed to get request state: %v", err),
			})
			return
		}

		prompt := "general" // default prompt
		if state.Metadata != nil {
			if promptVal, ok := state.Metadata["prompt"].(string); ok && promptVal != "" {
				prompt = promptVal
			}
		}

		summaryPath, err := e.summarizationProvider.SummarizeText(string(transcriptBytes), prompt)
		if err != nil {
			e.store.UpdateRequestState(task.RequestID, map[string]interface{}{
				"status": interfaces.StatusFailed,
				"error":  fmt.Sprintf("Failed to summarize text: %v", err),
			})
			return
		}
		e.eventBus.Publish(interfaces.Event{
			ID:        fmt.Sprintf("evt-%s-summary-%d", task.RequestID, time.Now().UnixNano()),
			RequestID: task.RequestID,
			Type:      "SummarizationCompleted",
			Data:      map[string]interface{}{"summary": summaryPath},
			Timestamp: time.Now(),
		})

	case interfaces.TaskOutput:
		fmt.Printf("[Engine] Processing TaskOutput for request: %s\n", task.RequestID)
		// Upload summary and/or transcript if outputProvider is set
		if e.outputProvider != nil {
			state, _ := e.store.GetRequestState(task.RequestID)
			if state != nil {
				videoInfo := state.VideoInfo
				if state.Summary != "" && videoInfo != nil {
					err := e.outputProvider.UploadSummary(task.RequestID, videoInfo, state.Summary)
					if err != nil {
						fmt.Println("GDrive upload summary error:", err)
					}
				}
				if state.Transcript != "" && videoInfo != nil {
					err := e.outputProvider.UploadTranscript(task.RequestID, videoInfo, state.Transcript)
					if err != nil {
						fmt.Println("GDrive upload transcript error:", err)
					}
				}
			}
		}
		// Mock: pretend to generate output
		summaryPath := task.Data.(map[string]interface{})["summary_path"].(string)
		outputPath := fmt.Sprintf("/tmp/output-%s.txt", task.RequestID)
		// In a real implementation, you would write the summary to a file
		// or generate some other output format

		e.store.UpdateRequestState(task.RequestID, map[string]interface{}{
			"status":       interfaces.StatusCompleted,
			"output_path":  outputPath,
			"completed_at": time.Now(),
		})

		e.eventBus.Publish(interfaces.Event{
			ID:        fmt.Sprintf("evt-%s-completed-%d", task.RequestID, time.Now().UnixNano()),
			RequestID: task.RequestID,
			Type:      "ProcessingCompleted",
			Data:      map[string]interface{}{"output_path": outputPath, "summary": summaryPath},
			Timestamp: time.Now(),
		})
	}
}

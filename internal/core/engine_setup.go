package core

import (
	"fmt"
	"video-summarizer-go/internal/config"
	"video-summarizer-go/internal/interfaces"
	"video-summarizer-go/internal/providers/output"
	"video-summarizer-go/internal/providers/summarization"
	"video-summarizer-go/internal/providers/transcription"
	"video-summarizer-go/internal/providers/video"
)

// SetupEngine wires up the event bus, state store, task queue, worker pool, providers, and processing engine.
// Returns the engine, worker pool, and prompt manager.
func SetupEngine(appCfg *config.AppConfig) (*ProcessingEngine, *WorkerPool, *config.PromptManager, error) {
	store := NewInMemoryStore()
	eventBus := NewInMemoryEventBus()
	taskQueue := NewInMemoryTaskQueue()

	concurrencyLimits := map[interfaces.TaskType]int{
		interfaces.TaskVideoInfo:     appCfg.Concurrency.VideoInfo,
		interfaces.TaskTranscription: appCfg.Concurrency.Transcription,
		interfaces.TaskSummarization: appCfg.Concurrency.Summarization,
		interfaces.TaskOutput:        appCfg.Concurrency.Output,
		interfaces.TaskCleanup:       appCfg.Concurrency.Cleanup,
		interfaces.TaskAudioDownload: appCfg.Concurrency.AudioDownload,
	}

	workerPool := NewWorkerPool(taskQueue, concurrencyLimits, nil)

	videoProvider := video.NewYtDlpVideoProvider(appCfg.YtDlpPath, appCfg.TmpDir)
	transcriptionProvider := transcription.NewWhisperCppTranscriptionProvider(appCfg.WhisperPath, appCfg.WhisperModelPath)

	// Initialize prompt manager
	promptManager := config.NewPromptManager()
	promptsDir := appCfg.PromptsDir
	if promptsDir == "" {
		promptsDir = "prompts"
	}
	if err := promptManager.LoadPrompts(promptsDir); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load prompts: %w", err)
	}

	summarizationProvider, err := summarization.NewConfigurableSummarizationProviderFromConfig(appCfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create summarization provider: %w", err)
	}

	var outputProvider interfaces.OutputProvider
	if appCfg.OutputProvider == "gdrive" {
		outputProvider, err = output.NewGDriveOutputProvider(appCfg)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create output provider: %w", err)
		}
	}

	engine := NewProcessingEngine(
		store,
		eventBus,
		taskQueue,
		workerPool,
		videoProvider,
		nil, // audioProcessor
		transcriptionProvider,
		summarizationProvider,
		outputProvider,
		promptManager,
	)
	workerPool.SetProcessFunc(engine.WorkerProcess)

	return engine, workerPool, promptManager, nil
}

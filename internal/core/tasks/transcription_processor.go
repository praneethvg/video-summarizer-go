package tasks

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"video-summarizer-go/internal/interfaces"
)

// TranscriptionProcessor handles audio transcription
type TranscriptionProcessor struct{}

// NewTranscriptionProcessor creates a new TranscriptionProcessor
func NewTranscriptionProcessor() *TranscriptionProcessor {
	return &TranscriptionProcessor{}
}

// GetTaskType returns the task type this processor handles
func (p *TranscriptionProcessor) GetTaskType() interfaces.TaskType {
	return interfaces.TaskTranscription
}

// Process handles the transcription task
func (p *TranscriptionProcessor) Process(ctx context.Context, task *interfaces.Task, engine interfaces.Engine) error {
	log.Infof("Processing TaskTranscription for request: %s", task.RequestID)

	audioPath := task.Data.(map[string]interface{})["audio_path"].(string)
	transcriptPath, err := engine.GetTranscriptionProvider().TranscribeAudio(audioPath)
	if err != nil {
		engine.GetStore().UpdateRequestState(task.RequestID, map[string]interface{}{
			"status": interfaces.StatusFailed,
			"error":  fmt.Sprintf("Failed to transcribe audio: %v", err),
		})
		return err
	}

	// Write transcript path to state
	err = engine.GetStore().UpdateRequestState(task.RequestID, map[string]interface{}{
		"transcript": transcriptPath,
	})
	if err != nil {
		log.Errorf("Failed to update state with transcript: %v", err)
		return err
	}

	// Publish transcription completed event
	engine.GetEventBus().Publish(interfaces.Event{
		ID:        fmt.Sprintf("evt-%s-transcript-%d", task.RequestID, time.Now().UnixNano()),
		RequestID: task.RequestID,
		Type:      "TranscriptionCompleted",
		Data:      map[string]interface{}{"transcript": transcriptPath},
		Timestamp: time.Now(),
	})

	return nil
}

package tasks

import (
	"context"
	"fmt"
	"os"
	"time"

	"video-summarizer-go/internal/interfaces"

	log "github.com/sirupsen/logrus"
)

// CleanupProcessor handles cleanup operations
type CleanupProcessor struct{}

// NewCleanupProcessor creates a new CleanupProcessor
func NewCleanupProcessor() *CleanupProcessor {
	return &CleanupProcessor{}
}

// GetTaskType returns the task type this processor handles
func (p *CleanupProcessor) GetTaskType() interfaces.TaskType {
	return interfaces.TaskCleanup
}

// Process handles the cleanup task
func (p *CleanupProcessor) Process(ctx context.Context, task *interfaces.Task, engine interfaces.Engine) error {
	log.Infof("Processing TaskCleanup for request: %s", task.RequestID)

	// Get request state for cleanup
	state, err := engine.GetStore().GetRequestState(task.RequestID)
	if err != nil {
		log.Errorf("Failed to get request state for cleanup: %v", err)
		return err
	}

	// Clean up temporary files
	log.Debugf("Starting cleanup for request: %s", task.RequestID)
	cleanupErrors := []string{}

	// Clean up audio file
	if state.AudioPath != "" {
		if err := os.Remove(state.AudioPath); err != nil {
			cleanupError := fmt.Sprintf("Failed to remove audio file %s: %v", state.AudioPath, err)
			log.Warnf("%s", cleanupError)
			cleanupErrors = append(cleanupErrors, cleanupError)
		} else {
			log.Debugf("Removed audio file: %s", state.AudioPath)
		}
	}

	// Clean up transcript file
	if state.Transcript != "" {
		if err := os.Remove(state.Transcript); err != nil {
			cleanupError := fmt.Sprintf("Failed to remove transcript file %s: %v", state.Transcript, err)
			log.Warnf("%s", cleanupError)
			cleanupErrors = append(cleanupErrors, cleanupError)
		} else {
			log.Debugf("Removed transcript file: %s", state.Transcript)
		}
	}

	// Clean up summary file
	if state.Summary != "" {
		if err := os.Remove(state.Summary); err != nil {
			cleanupError := fmt.Sprintf("Failed to remove summary file %s: %v", state.Summary, err)
			log.Warnf("%s", cleanupError)
			cleanupErrors = append(cleanupErrors, cleanupError)
		} else {
			log.Debugf("Removed summary file: %s", state.Summary)
		}
	}

	// Mark request as fully completed
	updateData := map[string]interface{}{
		"completed_at": time.Now(),
	}

	if len(cleanupErrors) > 0 {
		// Cleanup errors are warnings, don't fail the request but log them
		log.Warnf("Cleanup completed with warnings for request: %s", task.RequestID)
	}

	engine.GetStore().UpdateRequestState(task.RequestID, updateData)

	log.Debugf("TaskCleanup completed for request: %s", task.RequestID)

	// Publish final completion event
	engine.GetEventBus().Publish(interfaces.Event{
		ID:        fmt.Sprintf("evt-%s-completed-%d", task.RequestID, time.Now().UnixNano()),
		RequestID: task.RequestID,
		Type:      "ProcessingCompleted",
		Data:      map[string]interface{}{"status": "completed"},
		Timestamp: time.Now(),
	})

	return nil
}

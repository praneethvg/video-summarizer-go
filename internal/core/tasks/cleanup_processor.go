package tasks

import (
	"context"
	"fmt"
	"os"
	"time"

	"video-summarizer-go/internal/interfaces"
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
	fmt.Printf("[CleanupProcessor] Processing TaskCleanup for request: %s\n", task.RequestID)

	// Get request state for cleanup
	state, err := engine.GetStore().GetRequestState(task.RequestID)
	if err != nil {
		fmt.Printf("[CleanupProcessor][ERROR] Failed to get request state for cleanup: %v\n", err)
		return err
	}

	// Clean up temporary files
	fmt.Printf("[CleanupProcessor][DEBUG] Starting cleanup for request: %s\n", task.RequestID)
	cleanupErrors := []string{}

	// Clean up audio file
	if state.AudioPath != "" {
		if err := os.Remove(state.AudioPath); err != nil {
			cleanupError := fmt.Sprintf("Failed to remove audio file %s: %v", state.AudioPath, err)
			fmt.Printf("[CleanupProcessor][WARNING] %s\n", cleanupError)
			cleanupErrors = append(cleanupErrors, cleanupError)
		} else {
			fmt.Printf("[CleanupProcessor][DEBUG] Removed audio file: %s\n", state.AudioPath)
		}
	}

	// Clean up transcript file
	if state.Transcript != "" {
		if err := os.Remove(state.Transcript); err != nil {
			cleanupError := fmt.Sprintf("Failed to remove transcript file %s: %v", state.Transcript, err)
			fmt.Printf("[CleanupProcessor][WARNING] %s\n", cleanupError)
			cleanupErrors = append(cleanupErrors, cleanupError)
		} else {
			fmt.Printf("[CleanupProcessor][DEBUG] Removed transcript file: %s\n", state.Transcript)
		}
	}

	// Clean up summary file
	if state.Summary != "" {
		if err := os.Remove(state.Summary); err != nil {
			cleanupError := fmt.Sprintf("Failed to remove summary file %s: %v", state.Summary, err)
			fmt.Printf("[CleanupProcessor][WARNING] %s\n", cleanupError)
			cleanupErrors = append(cleanupErrors, cleanupError)
		} else {
			fmt.Printf("[CleanupProcessor][DEBUG] Removed summary file: %s\n", state.Summary)
		}
	}

	// Mark request as fully completed
	updateData := map[string]interface{}{
		"completed_at": time.Now(),
	}

	if len(cleanupErrors) > 0 {
		// Cleanup errors are warnings, don't fail the request but log them
		fmt.Printf("[CleanupProcessor][WARNING] Cleanup completed with warnings for request: %s\n", task.RequestID)
	}

	engine.GetStore().UpdateRequestState(task.RequestID, updateData)

	fmt.Printf("[CleanupProcessor][DEBUG] TaskCleanup completed for request: %s\n", task.RequestID)

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

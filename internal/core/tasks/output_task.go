package tasks

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"video-summarizer-go/internal/interfaces"
)

// OutputTask handles output operations (uploads, etc.)
type OutputTask struct{}

// NewOutputTask creates a new OutputTask
func NewOutputTask() *OutputTask {
	return &OutputTask{}
}

// GetTaskType returns the task type this processor handles
func (p *OutputTask) GetTaskType() interfaces.TaskType {
	return interfaces.TaskOutput
}

// Process handles the output task
func (p *OutputTask) Process(ctx context.Context, task *interfaces.Task, engine interfaces.Engine) error {
	log.Infof("Processing TaskOutput for request: %s", task.RequestID)

	// Get request state for upload
	state, err := engine.GetStore().GetRequestState(task.RequestID)
	if err != nil {
		log.Errorf("Failed to get request state for output: %v", err)
		return err
	}

	// Use category from state
	category := state.Category
	if category == "" {
		category = "general"
	}

	// Use hardcoded values for user and category for now
	user := "admin"

	// Upload summary and/or transcript if outputProvider is set
	uploadErrors := []string{}
	if engine.GetOutputProvider() != nil {
		videoInfo := state.VideoInfo
		if state.Summary != "" && videoInfo != nil {
			log.Debugf("Uploading summary for request: %s to user: %s, category: %s", task.RequestID, user, category)
			err := engine.GetOutputProvider().UploadSummary(task.RequestID, videoInfo, state.Summary, category, user)
			if err != nil {
				uploadError := fmt.Sprintf("GDrive upload summary error: %v", err)
				log.Errorf("%s", uploadError)
				uploadErrors = append(uploadErrors, uploadError)
			} else {
				log.Debugf("Summary uploaded successfully for request: %s", task.RequestID)
			}
		}
		if state.Transcript != "" && videoInfo != nil {
			log.Debugf("Uploading transcript for request: %s to user: %s, category: %s", task.RequestID, user, category)
			err := engine.GetOutputProvider().UploadTranscript(task.RequestID, videoInfo, state.Transcript, category, user)
			if err != nil {
				uploadError := fmt.Sprintf("GDrive upload transcript error: %v", err)
				log.Errorf("%s", uploadError)
				uploadErrors = append(uploadErrors, uploadError)
			} else {
				log.Debugf("Transcript uploaded successfully for request: %s", task.RequestID)
			}
		}
	}

	// Determine final status based on upload results
	finalStatus := interfaces.StatusCompleted
	finalError := ""

	if len(uploadErrors) > 0 {
		finalStatus = interfaces.StatusFailed
		finalError = fmt.Sprintf("Upload errors: %s", strings.Join(uploadErrors, "; "))
	}

	// Update state with upload results
	updateData := map[string]interface{}{
		"status": finalStatus,
	}

	if finalError != "" {
		updateData["error"] = finalError
	}

	err = engine.GetStore().UpdateRequestState(task.RequestID, updateData)
	if err != nil {
		log.Errorf("Failed to update state after output: %v", err)
	}

	log.Debugf("TaskOutput completed for request: %s with status: %s", task.RequestID, finalStatus)

	// Publish output completion event (cleanup will be triggered by this)
	summaryPath := task.Data.(map[string]interface{})["summary_path"].(string)
	engine.GetEventBus().Publish(interfaces.Event{
		ID:        fmt.Sprintf("evt-%s-output-%d", task.RequestID, time.Now().UnixNano()),
		RequestID: task.RequestID,
		Type:      interfaces.EventTypeOutputCompleted,
		Data:      map[string]interface{}{"summary": summaryPath, "status": finalStatus},
		Timestamp: time.Now(),
	})

	return nil
}

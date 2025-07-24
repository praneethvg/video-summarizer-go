package tasks

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"video-summarizer-go/internal/interfaces"
)

// VideoInfoProcessor handles video information extraction and audio download
type VideoInfoProcessor struct{}

// NewVideoInfoProcessor creates a new VideoInfoProcessor
func NewVideoInfoProcessor() *VideoInfoProcessor {
	return &VideoInfoProcessor{}
}

// GetTaskType returns the task type this processor handles
func (p *VideoInfoProcessor) GetTaskType() interfaces.TaskType {
	return interfaces.TaskVideoInfo
}

// Process handles the video info task
func (p *VideoInfoProcessor) Process(ctx context.Context, task *interfaces.Task, engine interfaces.Engine) error {
	log.Printf("[VideoInfoProcessor] Processing TaskVideoInfo for request: %s", task.RequestID)

	url := task.Data.(map[string]interface{})["url"].(string)
	videoInfo, err := engine.GetVideoProvider().GetVideoInfo(url)
	if err != nil {
		engine.GetStore().UpdateRequestState(task.RequestID, map[string]interface{}{
			"status": interfaces.StatusFailed,
			"error":  fmt.Sprintf("Failed to get video info: %v", err),
		})
		return err
	}

	// Write video info to state
	err = engine.GetStore().UpdateRequestState(task.RequestID, map[string]interface{}{
		"video_info": videoInfo,
	})
	if err != nil {
		log.Printf("[VideoInfoProcessor][ERROR] Failed to update state with video info: %v", err)
		return err
	}

	// Publish video info event
	engine.GetEventBus().Publish(interfaces.Event{
		ID:        fmt.Sprintf("evt-%s-videoinfo-%d", task.RequestID, time.Now().UnixNano()),
		RequestID: task.RequestID,
		Type:      "VideoInfoFetched",
		Data:      videoInfo,
		Timestamp: time.Now(),
	})

	return nil
}

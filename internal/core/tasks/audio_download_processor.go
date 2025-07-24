package tasks

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"video-summarizer-go/internal/interfaces"
)

// AudioDownloadProcessor handles audio download for a video
type AudioDownloadProcessor struct{}

func NewAudioDownloadProcessor() *AudioDownloadProcessor {
	return &AudioDownloadProcessor{}
}

func (p *AudioDownloadProcessor) GetTaskType() interfaces.TaskType {
	return interfaces.TaskAudioDownload
}

func (p *AudioDownloadProcessor) Process(ctx context.Context, task *interfaces.Task, engine interfaces.Engine) error {
	log.Infof("Processing TaskAudioDownload for request: %s", task.RequestID)

	url, ok := task.Data.(map[string]interface{})["url"].(string)
	if !ok || url == "" {
		return fmt.Errorf("audio_download task missing url in data")
	}

	audioPath, err := engine.GetVideoProvider().DownloadAudio(url)
	if err != nil {
		engine.GetStore().UpdateRequestState(task.RequestID, map[string]interface{}{
			"status": interfaces.StatusFailed,
			"error":  fmt.Sprintf("Failed to download audio: %v", err),
		})
		return err
	}

	// Write audio path to state
	err = engine.GetStore().UpdateRequestState(task.RequestID, map[string]interface{}{
		"audio_path": audioPath,
	})
	if err != nil {
		log.Errorf("Failed to update state with audio path: %v", err)
		return err
	}

	// Publish audio downloaded event
	engine.GetEventBus().Publish(interfaces.Event{
		ID:        fmt.Sprintf("evt-%s-audio-%d", task.RequestID, time.Now().UnixNano()),
		RequestID: task.RequestID,
		Type:      "AudioDownloaded",
		Data:      map[string]interface{}{"audio_path": audioPath},
		Timestamp: time.Now(),
	})

	return nil
}

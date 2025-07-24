package tasks

import (
	"context"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"video-summarizer-go/internal/interfaces"
)

// SummarizationProcessor handles text summarization
type SummarizationProcessor struct{}

// NewSummarizationProcessor creates a new SummarizationProcessor
func NewSummarizationProcessor() *SummarizationProcessor {
	return &SummarizationProcessor{}
}

// GetTaskType returns the task type this processor handles
func (p *SummarizationProcessor) GetTaskType() interfaces.TaskType {
	return interfaces.TaskSummarization
}

// Process handles the summarization task
func (p *SummarizationProcessor) Process(ctx context.Context, task *interfaces.Task, engine interfaces.Engine) error {
	log.Printf("[SummarizationProcessor] Processing TaskSummarization for request: %s", task.RequestID)

	transcriptPath := task.Data.(map[string]interface{})["transcript_path"].(string)
	transcriptBytes, err := os.ReadFile(transcriptPath)
	if err != nil {
		engine.GetStore().UpdateRequestState(task.RequestID, map[string]interface{}{
			"status": interfaces.StatusFailed,
			"error":  fmt.Sprintf("Failed to read transcript file: %v", err),
		})
		return err
	}

	// Read promptID and maxTokens from state
	state, err := engine.GetStore().GetRequestState(task.RequestID)
	if err != nil {
		log.Printf("[SummarizationProcessor][ERROR] Failed to get state: %v", err)
		return err
	}
	prompt := state.Prompt
	var promptText string
	switch prompt.Type {
	case interfaces.PromptTypeID:
		pm := engine.GetPromptManager()
		if pm != nil {
			resolved, err := pm.ResolvePrompt(prompt.Prompt)
			if err != nil {
				log.WithError(err).Warn("Failed to resolve prompt ID, using ID as text")
				promptText = prompt.Prompt
			} else {
				promptText = resolved
			}
		} else {
			promptText = prompt.Prompt
		}
	case interfaces.PromptTypeText:
		promptText = prompt.Prompt
	}
	if promptText == "" {
		promptText = "general"
	}
	maxTokens := state.MaxTokens
	if maxTokens == 0 {
		maxTokens = 10000
	}

	summaryPath, err := engine.GetSummarizationProvider().SummarizeText(ctx, string(transcriptBytes), promptText, maxTokens)
	if err != nil {
		engine.GetStore().UpdateRequestState(task.RequestID, map[string]interface{}{
			"status": interfaces.StatusFailed,
			"error":  fmt.Sprintf("Failed to summarize text: %v", err),
		})
		return err
	}

	// Write summary path to state
	err = engine.GetStore().UpdateRequestState(task.RequestID, map[string]interface{}{
		"summary": summaryPath,
	})
	if err != nil {
		log.Printf("[SummarizationProcessor][ERROR] Failed to update state with summary: %v", err)
		return err
	}

	// Publish summarization completed event
	engine.GetEventBus().Publish(interfaces.Event{
		ID:        fmt.Sprintf("evt-%s-summary-%d", task.RequestID, time.Now().UnixNano()),
		RequestID: task.RequestID,
		Type:      "SummarizationCompleted",
		Data:      map[string]interface{}{"summary": summaryPath},
		Timestamp: time.Now(),
	})

	return nil
}

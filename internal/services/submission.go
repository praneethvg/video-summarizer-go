package services

import (
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"video-summarizer-go/internal/core"
	"video-summarizer-go/internal/interfaces"
)

// VideoSubmissionService provides a unified interface for submitting videos to the processing queue
type VideoSubmissionService struct {
	engine    *core.ProcessingEngine
	mu        sync.RWMutex
	requestID string
}

// NewVideoSubmissionService creates a new video submission service
func NewVideoSubmissionService(engine *core.ProcessingEngine) *VideoSubmissionService {
	return &VideoSubmissionService{
		engine: engine,
	}
}

// SubmitVideo submits a single video for processing
func (s *VideoSubmissionService) SubmitVideo(url string, prompt interfaces.Prompt, sourceType string, category string, maxTokens int) (string, error) {
	model := "gpt-4o" // TODO: Make this configurable or pass as argument
	dedupKey := core.MakeDedupKey(url, prompt.Prompt, model)

	// Prepare the state for possible creation
	requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())
	state := &interfaces.ProcessingState{
		RequestID:  requestID,
		Status:     interfaces.StatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		SourceType: sourceType,
		URL:        url,
		Prompt:     prompt,
		MaxTokens:  maxTokens,
		Category:   category,
	}

	// Use the store's deduplication method
	id, alreadyExists, err := s.engine.GetStore().CreateOrGetDedupRequest(dedupKey, state)
	if err != nil {
		return "", fmt.Errorf("failed to create or get dedup request: %w", err)
	}
	if alreadyExists {
		log.WithFields(log.Fields{
			"dedupKey":  dedupKey,
			"requestID": id,
		}).Info("Deduplication hit")
		return id, nil
	}

	// Start the request (stores state and publishes event)
	err = s.engine.StartRequest(state.RequestID, state.URL, state.Prompt, state.SourceType, category, state.MaxTokens)
	if err != nil {
		return "", fmt.Errorf("failed to start request: %w", err)
	}

	log.WithFields(log.Fields{
		"url":        url,
		"prompt":     prompt.Prompt,
		"promptType": prompt.Type,
		"sourceType": sourceType,
		"category":   category,
		"maxTokens":  maxTokens,
	}).Info("SubmitVideo created new request")
	return state.RequestID, nil
}

// SubmitBatch submits multiple videos for processing
func (s *VideoSubmissionService) SubmitBatch(urls []string, prompt interfaces.Prompt, sourceType, category string, maxTokens int) ([]string, error) {
	log.WithField("prompt", prompt).Info("SubmitBatch called")
	var requestIDs []string
	var errors []error

	for _, url := range urls {
		log.WithField("url", url).WithField("prompt", prompt).Info("Submitting url")
		requestID, err := s.SubmitVideo(url, prompt, sourceType, category, maxTokens)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to submit %s: %w", url, err))
			continue
		}
		requestIDs = append(requestIDs, requestID)
	}

	if len(errors) > 0 {
		return requestIDs, fmt.Errorf("some submissions failed: %v", errors)
	}

	return requestIDs, nil
}

// GetRequestStatus gets the status of a processing request
func (s *VideoSubmissionService) GetRequestStatus(requestID string) (*interfaces.ProcessingState, error) {
	return s.engine.GetRequestState(requestID)
}

// CancelRequest cancels a processing request
func (s *VideoSubmissionService) CancelRequest(requestID string) error {
	return s.engine.CancelRequest(requestID)
}

// GetRequestCountsByStatus returns a map of status to count
func (s *VideoSubmissionService) GetRequestCountsByStatus() map[string]int {
	return s.engine.GetRequestCountsByStatus()
}

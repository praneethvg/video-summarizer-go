package services

import (
	"fmt"
	"sync"
	"time"

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
func (s *VideoSubmissionService) SubmitVideo(url string, prompt string, metadata map[string]interface{}) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate unique request ID
	requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())

	// Add metadata to the request
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["url"] = url
	metadata["prompt"] = prompt
	metadata["submitted_at"] = time.Now()
	metadata["source"] = "api" // or "background" depending on caller

	// Start the processing request
	err := s.engine.StartRequest(requestID, url)
	if err != nil {
		return "", fmt.Errorf("failed to start request: %w", err)
	}

	return requestID, nil
}

// SubmitBatch submits multiple videos for processing
func (s *VideoSubmissionService) SubmitBatch(urls []string, prompt string, metadata map[string]interface{}) ([]string, error) {
	var requestIDs []string
	var errors []error

	for _, url := range urls {
		requestID, err := s.SubmitVideo(url, prompt, metadata)
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

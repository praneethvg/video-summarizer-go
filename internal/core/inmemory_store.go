package core

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"video-summarizer-go/internal/interfaces"
)

type InMemoryStateStore struct {
	requests map[string]*interfaces.ProcessingState
	events   map[string][]interfaces.Event // keyed by requestID
	dedup    map[string]string             // dedupKey -> requestID
	mu       sync.RWMutex
}

func NewInMemoryStore() *InMemoryStateStore {
	return &InMemoryStateStore{
		requests: make(map[string]*interfaces.ProcessingState),
		events:   make(map[string][]interfaces.Event),
		dedup:    make(map[string]string),
	}
}

// MakeDedupKey creates a unique key for deduplication
func MakeDedupKey(resource, promptID, model string) string {
	return fmt.Sprintf("%s|%s|%s", resource, promptID, model)
}

// GetRequestIDByDedupKey returns the requestID for a dedup key, if any
func (s *InMemoryStateStore) GetRequestIDByDedupKey(dedupKey string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.dedup[dedupKey]
	return id, ok
}

// AddDedupKey stores a dedup key -> requestID mapping
func (s *InMemoryStateStore) AddDedupKey(dedupKey, requestID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dedup[dedupKey] = requestID
}

func (s *InMemoryStateStore) SaveRequestState(requestID string, state *interfaces.ProcessingState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests[requestID] = state
	return nil
}

func (s *InMemoryStateStore) GetRequestState(requestID string) (*interfaces.ProcessingState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.requests[requestID]
	if !ok {
		return nil, errors.New("request not found")
	}
	return state, nil
}

func (s *InMemoryStateStore) UpdateRequestState(requestID string, updates map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.requests[requestID]
	if !ok {
		return errors.New("request not found")
	}
	for k, v := range updates {
		switch k {
		case "status":
			if val, ok := v.(interfaces.ProcessingStatus); ok {
				state.Status = val
			} else if val, ok := v.(string); ok {
				state.Status = interfaces.ProcessingStatus(val)
			}
		case "video_info":
			if val, ok := v.(map[string]interface{}); ok {
				state.VideoInfo = val
			}
		case "audio_path":
			if val, ok := v.(string); ok {
				state.AudioPath = val
			}
		case "transcript":
			if val, ok := v.(string); ok {
				state.Transcript = val
			}
		case "summary":
			if val, ok := v.(string); ok {
				state.Summary = val
			}
		case "error":
			if val, ok := v.(string); ok {
				state.Error = val
			}
		case "output_path":
			if val, ok := v.(string); ok {
				state.OutputPath = val
			}
		case "completed_at":
			if val, ok := v.(time.Time); ok {
				state.CompletedAt = &val
			}
		case "source_type":
			if val, ok := v.(string); ok {
				state.SourceType = val
			}
		case "url":
			if val, ok := v.(string); ok {
				state.URL = val
			}
		case "document_info":
			if val, ok := v.(map[string]interface{}); ok {
				state.DocumentInfo = val
			}
		case "text_path":
			if val, ok := v.(string); ok {
				state.TextPath = val
			}
		}
	}
	state.UpdatedAt = time.Now()
	return nil
}

func (s *InMemoryStateStore) DeleteRequestState(requestID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.requests, requestID)
	delete(s.events, requestID)
	return nil
}

func (s *InMemoryStateStore) LogEvent(event interfaces.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events[event.RequestID] = append(s.events[event.RequestID], event)
	return nil
}

func (s *InMemoryStateStore) GetEventsForRequest(requestID string) ([]interfaces.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	events, ok := s.events[requestID]
	if !ok {
		return nil, errors.New("no events for request")
	}
	return events, nil
}

func (s *InMemoryStateStore) GetAllActiveRequests() ([]*interfaces.ProcessingState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var active []*interfaces.ProcessingState
	for _, state := range s.requests {
		if state.Status != interfaces.StatusCompleted && state.Status != interfaces.StatusCancelled && state.Status != interfaces.StatusFailed {
			active = append(active, state)
		}
	}
	return active, nil
}

func (s *InMemoryStateStore) CleanupOldRequests(olderThan time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, state := range s.requests {
		if (state.Status == interfaces.StatusCompleted || state.Status == interfaces.StatusCancelled || state.Status == interfaces.StatusFailed) && state.UpdatedAt.Before(olderThan) {
			delete(s.requests, id)
			delete(s.events, id)
		}
	}
	return nil
}

// GetRequestCountsByStatus returns a map of status to count
func (s *InMemoryStateStore) GetRequestCountsByStatus() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	counts := make(map[string]int)
	for _, state := range s.requests {
		status := string(state.Status)
		counts[status]++
	}
	return counts
}

func (s *InMemoryStateStore) CreateOrGetDedupRequest(dedupKey string, state *interfaces.ProcessingState) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if reqID, ok := s.dedup[dedupKey]; ok {
		existing := s.requests[reqID]
		if existing != nil && existing.Status != interfaces.StatusFailed {
			return reqID, true, nil
		}
		// If failed, allow a new request (replace mapping)
	}
	s.requests[state.RequestID] = state
	s.dedup[dedupKey] = state.RequestID
	return state.RequestID, false, nil
}

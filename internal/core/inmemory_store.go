package core

import (
	"errors"
	"sync"
	"time"

	"video-summarizer-go/internal/interfaces"
)

type InMemoryStateStore struct {
	requests map[string]*interfaces.ProcessingState
	events   map[string][]interfaces.Event // keyed by requestID
	mu       sync.RWMutex
}

func NewInMemoryStore() *InMemoryStateStore {
	return &InMemoryStateStore{
		requests: make(map[string]*interfaces.ProcessingState),
		events:   make(map[string][]interfaces.Event),
	}
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
		case "metadata":
			if val, ok := v.(map[string]interface{}); ok {
				state.Metadata = val
			}
		case "output_path":
			if val, ok := v.(string); ok {
				state.OutputPath = val
			}
		case "completed_at":
			if val, ok := v.(time.Time); ok {
				state.CompletedAt = &val
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

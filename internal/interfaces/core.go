package interfaces

import (
	"context"
	"time"
	"video-summarizer-go/internal/config"
)

// StateStore defines methods for request state and event persistence
type StateStore interface {
	SaveRequestState(requestID string, state *ProcessingState) error
	GetRequestState(requestID string) (*ProcessingState, error)
	UpdateRequestState(requestID string, updates map[string]interface{}) error
	DeleteRequestState(requestID string) error

	LogEvent(event Event) error
	GetEventsForRequest(requestID string) ([]Event, error)

	GetAllActiveRequests() ([]*ProcessingState, error)
	CleanupOldRequests(olderThan time.Time) error
	GetRequestCountsByStatus() map[string]int

	// Deduplication: create or get a request for a dedup key
	CreateOrGetDedupRequest(dedupKey string, state *ProcessingState) (requestID string, alreadyExists bool, err error)
}

// EventBus defines pub/sub for events
type EventHandler func(event Event)

type EventBus interface {
	Publish(event Event) error
	Subscribe(eventType EventType, handler EventHandler)
}

// TaskQueue defines enqueue/dequeue for tasks
type TaskQueue interface {
	Enqueue(task *Task) error
	Dequeue(taskType TaskType) (*Task, error)
	QueueLength(taskType TaskType) int
	RemoveTasksForRequest(requestID string) error
}

// AudioProcessor defines methods for audio processing
type AudioProcessor interface {
	ProcessAudio(inputPath string) (string, error)
	GetSupportedFormats() []string
}

// TaskProcessor defines the interface for processing tasks
type TaskProcessor interface {
	Process(ctx context.Context, task *Task, engine Engine) error
	GetTaskType() TaskType
}

// Engine defines the interface for the processing engine
type Engine interface {
	GetVideoProvider() VideoProvider
	GetTranscriptionProvider() TranscriptionProvider
	GetSummarizationProvider() SummarizationProvider
	GetOutputProvider() OutputProvider
	GetPromptManager() *config.PromptManager
	GetStore() StateStore
	GetEventBus() EventBus
	GetTaskQueue() TaskQueue
}

// PromptType is an enum for prompt type
// PromptTypeID: use prompt as an ID
// PromptTypeText: use prompt as direct text
type PromptType string

const (
	PromptTypeID   PromptType = "id"
	PromptTypeText PromptType = "text"
)

type Prompt struct {
	Type   PromptType `json:"type" yaml:"type"`
	Prompt string     `json:"prompt" yaml:"prompt"`
}

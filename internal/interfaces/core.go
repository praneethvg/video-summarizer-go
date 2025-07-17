package interfaces

import (
	"time"
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
}

// EventBus defines pub/sub for events
type EventHandler func(event Event)

type EventBus interface {
	Publish(event Event) error
	Subscribe(eventType string, handler EventHandler)
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

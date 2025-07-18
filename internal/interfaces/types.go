package interfaces

import (
	"time"
)

// TaskType represents different types of processing tasks
type TaskType string

const (
	TaskVideoInfo     TaskType = "video_info"
	TaskAudioDownload TaskType = "audio_download"
	TaskTranscription TaskType = "transcription"
	TaskSummarization TaskType = "summarization"
	TaskOutput        TaskType = "output"
)

// Task represents a processing task
type Task struct {
	ID        string                 `json:"id"`
	Type      TaskType               `json:"type"`
	RequestID string                 `json:"request_id"`
	Data      interface{}            `json:"data"`
	CreatedAt time.Time              `json:"created_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Event represents a system event
type Event struct {
	ID        string                 `json:"id"`
	RequestID string                 `json:"request_id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// ProcessingStatus represents the status of a request
type ProcessingStatus string

const (
	StatusPending   ProcessingStatus = "pending"
	StatusRunning   ProcessingStatus = "running"
	StatusCompleted ProcessingStatus = "completed"
	StatusFailed    ProcessingStatus = "failed"
	StatusCancelled ProcessingStatus = "cancelled"
)

// ProcessingState represents the state of a video processing request
type ProcessingState struct {
	RequestID   string                 `json:"request_id"`
	VideoURL    string                 `json:"video_url"`
	Status      ProcessingStatus       `json:"status"`
	Progress    float64                `json:"progress"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
	VideoInfo   map[string]interface{} `json:"video_info,omitempty"`
	AudioPath   string                 `json:"audio_path,omitempty"`
	Transcript  string                 `json:"transcript_path,omitempty"`
	Summary     string                 `json:"summary_path,omitempty"`
	OutputPath  string                 `json:"output_path,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

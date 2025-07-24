package tasks

import (
	"video-summarizer-go/internal/interfaces"
)

// TaskProcessorRegistry manages task tasks
type TaskProcessorRegistry struct {
	processors map[interfaces.TaskType]interfaces.TaskProcessor
}

// NewTaskProcessorRegistry creates a new task task registry
func NewTaskProcessorRegistry() *TaskProcessorRegistry {
	registry := &TaskProcessorRegistry{
		processors: make(map[interfaces.TaskType]interfaces.TaskProcessor),
	}
	// Register all task tasks
	registry.Register(NewVideoInfoTask())
	registry.Register(NewTranscriptionTask())
	registry.Register(NewSummarizationTask())
	registry.Register(NewOutputTask())
	registry.Register(NewCleanupTask())
	registry.Register(NewAudioDownloadTask())
	return registry
}

// Register adds a task processor to the registry
func (r *TaskProcessorRegistry) Register(processor interfaces.TaskProcessor) {
	r.processors[processor.GetTaskType()] = processor
}

// GetProcessor returns the processor for a given task type
func (r *TaskProcessorRegistry) GetProcessor(taskType interfaces.TaskType) (interfaces.TaskProcessor, bool) {
	processor, exists := r.processors[taskType]
	return processor, exists
}

// GetAllProcessors returns all registered processors
func (r *TaskProcessorRegistry) GetAllProcessors() map[interfaces.TaskType]interfaces.TaskProcessor {
	return r.processors
}

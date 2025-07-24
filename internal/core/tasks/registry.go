package tasks

import (
	"video-summarizer-go/internal/interfaces"
)

// TaskProcessorRegistry manages task processors
type TaskProcessorRegistry struct {
	processors map[interfaces.TaskType]interfaces.TaskProcessor
}

// NewTaskProcessorRegistry creates a new task processor registry
func NewTaskProcessorRegistry() *TaskProcessorRegistry {
	registry := &TaskProcessorRegistry{
		processors: make(map[interfaces.TaskType]interfaces.TaskProcessor),
	}

	// Register all processors
	registry.Register(NewVideoInfoProcessor())
	registry.Register(NewTranscriptionProcessor())
	registry.Register(NewSummarizationProcessor())
	registry.Register(NewOutputProcessor())
	registry.Register(NewCleanupProcessor())
	registry.Register(NewAudioDownloadProcessor())

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

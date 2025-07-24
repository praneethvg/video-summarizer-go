package core

import (
	"errors"
	"log"
	"sync"

	"video-summarizer-go/internal/interfaces"
)

type InMemoryTaskQueue struct {
	queues map[interfaces.TaskType][]*interfaces.Task
	mu     sync.RWMutex
}

func NewInMemoryTaskQueue() *InMemoryTaskQueue {
	return &InMemoryTaskQueue{
		queues: make(map[interfaces.TaskType][]*interfaces.Task),
	}
}

func (q *InMemoryTaskQueue) Enqueue(task *interfaces.Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.queues[task.Type] = append(q.queues[task.Type], task)
	log.Printf("[TaskQueue] Enqueued task: %s for request: %s", task.Type, task.RequestID)
	// Debug: print current queue for this type
	queueIDs := make([]string, len(q.queues[task.Type]))
	for i, t := range q.queues[task.Type] {
		queueIDs[i] = t.ID
	}
	log.Printf("[TaskQueue][DEBUG] Current queue for %s: %v", task.Type, queueIDs)
	return nil
}

func (q *InMemoryTaskQueue) Dequeue(taskType interfaces.TaskType) (*interfaces.Task, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	queue, exists := q.queues[taskType]
	if !exists || len(queue) == 0 {
		return nil, errors.New("no tasks available")
	}
	task := queue[0]
	q.queues[taskType] = queue[1:]
	return task, nil
}

func (q *InMemoryTaskQueue) Size(taskType interfaces.TaskType) int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.queues[taskType])
}

func (q *InMemoryTaskQueue) QueueLength(taskType interfaces.TaskType) int {
	return q.Size(taskType)
}

// RemoveTasksForRequest removes all tasks for a specific request ID from all queues
func (q *InMemoryTaskQueue) RemoveTasksForRequest(requestID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for taskType, queue := range q.queues {
		var newQueue []*interfaces.Task
		for _, task := range queue {
			if task.RequestID != requestID {
				newQueue = append(newQueue, task)
			}
		}
		q.queues[taskType] = newQueue
	}

	return nil
}

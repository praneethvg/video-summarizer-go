package core

import (
	"fmt"
	"sync"
	"time"

	"video-summarizer-go/internal/interfaces"
)

type WorkerPool struct {
	queue       interfaces.TaskQueue
	limits      map[interfaces.TaskType]int
	workers     map[interfaces.TaskType][]chan struct{}
	stopChans   map[interfaces.TaskType]chan struct{}
	processFunc func(task *interfaces.Task)
	mu          sync.Mutex
}

func NewWorkerPool(queue interfaces.TaskQueue, limits map[interfaces.TaskType]int, processFunc func(task *interfaces.Task)) *WorkerPool {
	wp := &WorkerPool{
		queue:       queue,
		limits:      limits,
		workers:     make(map[interfaces.TaskType][]chan struct{}),
		stopChans:   make(map[interfaces.TaskType]chan struct{}),
		processFunc: processFunc,
	}
	for taskType, limit := range limits {
		wp.startWorkers(taskType, limit)
	}
	return wp
}

func (wp *WorkerPool) startWorkers(taskType interfaces.TaskType, count int) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	if wp.stopChans[taskType] != nil {
		close(wp.stopChans[taskType])
	}
	wp.stopChans[taskType] = make(chan struct{})
	wp.workers[taskType] = nil
	for i := 0; i < count; i++ {
		workerDone := make(chan struct{})
		wp.workers[taskType] = append(wp.workers[taskType], workerDone)
		go wp.worker(taskType, wp.stopChans[taskType], workerDone)
	}
}

func (wp *WorkerPool) worker(taskType interfaces.TaskType, stopChan chan struct{}, done chan struct{}) {
	fmt.Printf("[WorkerPool] Worker goroutine started for task type: %s\n", taskType)
	defer close(done)
	for {
		select {
		case <-stopChan:
			return
		default:
			task, err := wp.queue.Dequeue(taskType)
			if err != nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			// Debug: log when a worker picks up a task
			fmt.Printf("[WorkerPool] Worker picked up task: %s for request: %s\n", task.Type, task.RequestID)
			wp.mu.Lock()
			processFunc := wp.processFunc
			wp.mu.Unlock()
			if processFunc != nil {
				processFunc(task)
				// Debug: log after processing function returns
				fmt.Printf("[WorkerPool] Worker finished task: %s for request: %s\n", task.Type, task.RequestID)
			} else {
				fmt.Printf("[WorkerPool] No process function set for task: %s\n", task.Type)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

func (wp *WorkerPool) SetConcurrencyLimit(taskType interfaces.TaskType, limit int) {
	wp.startWorkers(taskType, limit)
}

// SetProcessFunc sets the task processing function
func (wp *WorkerPool) SetProcessFunc(processFunc func(task *interfaces.Task)) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	wp.processFunc = processFunc
}

func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	for _, stopChan := range wp.stopChans {
		close(stopChan)
	}
	// Optionally wait for all workers to finish
	for _, workerChans := range wp.workers {
		for _, done := range workerChans {
			<-done
		}
	}
}

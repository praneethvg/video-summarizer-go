package core

import (
	"sync"

	"video-summarizer-go/internal/interfaces"
)

type InMemoryEventBus struct {
	handlers map[string][]interfaces.EventHandler
	mu       sync.RWMutex
}

func NewInMemoryEventBus() *InMemoryEventBus {
	return &InMemoryEventBus{
		handlers: make(map[string][]interfaces.EventHandler),
	}
}

func (b *InMemoryEventBus) Publish(event interfaces.Event) error {
	b.mu.RLock()
	handlers := b.handlers[event.Type]
	b.mu.RUnlock()

	for _, handler := range handlers {
		handler(event)
	}
	return nil
}

func (b *InMemoryEventBus) Subscribe(eventType string, handler interfaces.EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

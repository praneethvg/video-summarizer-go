package core

import (
	"sync"

	"video-summarizer-go/internal/interfaces"

	log "github.com/sirupsen/logrus"
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
	log.Debugf("[EventBus] Publishing event: type=%s, requestID=%s", event.Type, event.RequestID)
	b.mu.RLock()
	handlers := b.handlers[event.Type]
	b.mu.RUnlock()
	log.Debugf("[EventBus] Found %d handler(s) for event type: %s", len(handlers), event.Type)
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

package eventbus

import (
	"context"
	"log"
	"sync"
)

type LocalBus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
	queue    chan Event
}

func NewLocalBus(buffer int) *LocalBus {
	if buffer <= 0 {
		buffer = 1
	}
	b := &LocalBus{
		handlers: make(map[string][]Handler),
		queue:    make(chan Event, buffer),
	}
	go b.loop()
	return b
}

func (b *LocalBus) Publish(ctx context.Context, event Event) {
	if event == nil {
		return
	}
	select {
	case b.queue <- event:
	case <-ctx.Done():
	default:
		log.Printf("[eventbus] drop event: %s", event.Name())
	}
}

func (b *LocalBus) Subscribe(eventName string, handler Handler) {
	if eventName == "" || handler == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventName] = append(b.handlers[eventName], handler)
}

func (b *LocalBus) loop() {
	for event := range b.queue {
		b.dispatch(event)
	}
}

func (b *LocalBus) dispatch(event Event) {
	b.mu.RLock()
	handlers := append([]Handler(nil), b.handlers[event.Name()]...)
	b.mu.RUnlock()

	for _, handler := range handlers {
		go func(h Handler) {
			if err := h.Handle(context.Background(), event); err != nil {
				log.Printf("[eventbus] handle %s failed: %v", event.Name(), err)
			}
		}(handler)
	}
}

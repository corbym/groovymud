package event

import (
	"sync"
)

// Scope determines who receives an event.
type Scope int

const (
	GlobalScope    Scope = iota // everyone in the world
	ContainerScope              // everyone in the same container/area
	LocalScope                  // everyone in the same room
)

// EventType identifies the kind of event.
type EventType int

const (
	Movement EventType = iota
	Message
)

// Event carries data about something that happened.
type Event struct {
	Type    EventType
	Scope   Scope
	RoomID  string
	ActorID string // player name
	Payload interface{}
}

// Handler is a function that processes an event.
type Handler func(e Event)

// Bus is a simple publish/subscribe event bus.
type Bus struct {
	mu       sync.RWMutex
	handlers []Handler
}

// NewBus creates a new Bus.
func NewBus() *Bus {
	return &Bus{}
}

// Subscribe registers a handler to receive all events.
func (b *Bus) Subscribe(h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, h)
}

// Publish dispatches an event to all registered handlers.
func (b *Bus) Publish(e Event) {
	b.mu.RLock()
	handlers := make([]Handler, len(b.handlers))
	copy(handlers, b.handlers)
	b.mu.RUnlock()
	for _, h := range handlers {
		h(e)
	}
}

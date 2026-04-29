package events

import (
	"log"
	"sync"
)

// Event represents a domain event
type Event struct {
	Type     string
	TenantID string
	UserID   string
	Payload  map[string]any
}

// Handler processes an event
type Handler func(event Event)

// Bus is a simple in-process event bus for module decoupling
type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

func NewBus() *Bus {
	return &Bus{
		handlers: make(map[string][]Handler),
	}
}

// Subscribe registers a handler for an event type
func (b *Bus) Subscribe(eventType string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Publish dispatches an event to all registered handlers (async)
func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	handlers := b.handlers[event.Type]
	b.mu.RUnlock()

	for _, h := range handlers {
		go func(handler Handler) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("event handler panic for %s: %v", event.Type, r)
				}
			}()
			handler(event)
		}(h)
	}
}

// Event type constants
const (
	EventTenantCreated      = "tenant.created"
	EventUserRegistered     = "user.registered"
	EventUserLoggedIn       = "user.logged_in"
	EventSubscriptionChange = "subscription.changed"
	EventRoleAssigned       = "role.assigned"
)

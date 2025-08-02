package types

import (
	"sync"
	"time"
)

// EventType represents the type of event
type EventType string

const (
	// Configuration events
	EventTypeConfigChanged EventType = "config.changed"
	EventTypeMappingAdded  EventType = "mapping.added"
	EventTypeMappingRemoved EventType = "mapping.removed"
	
	// Network events
	EventTypeNetworkDiscovered EventType = "network.discovered"
	EventTypeNATDetected       EventType = "nat.detected"
	EventTypeConnectionEstablished EventType = "connection.established"
	EventTypeConnectionLost    EventType = "connection.lost"
	
	// Forwarding events
	EventTypeForwardingStarted EventType = "forwarding.started"
	EventTypeForwardingStopped EventType = "forwarding.stopped"
	EventTypeForwardingError   EventType = "forwarding.error"
	
	// Signaling events
	EventTypeSignalingConnected    EventType = "signaling.connected"
	EventTypeSignalingDisconnected EventType = "signaling.disconnected"
	EventTypePeerRegistered        EventType = "peer.registered"
	
	// System events
	EventTypeShutdown EventType = "system.shutdown"
	EventTypeError    EventType = "system.error"
)

// Event represents a system event
type Event interface {
	Type() EventType
	Data() interface{}
	Timestamp() time.Time
	Source() string
}

// EventHandler handles events
type EventHandler func(event Event)

// EventBus manages event publishing and subscription
type EventBus interface {
	Publish(event Event)
	Subscribe(eventType EventType, handler EventHandler) func() // Returns unsubscribe function
	SubscribeAll(handler EventHandler) func()
	Close()
}

// BaseEvent provides a basic implementation of Event
type BaseEvent struct {
	EventType EventType   `json:"type"`
	EventData interface{} `json:"data"`
	EventTime time.Time   `json:"timestamp"`
	EventSource string    `json:"source"`
}

// Type returns the event type
func (e *BaseEvent) Type() EventType {
	return e.EventType
}

// Data returns the event data
func (e *BaseEvent) Data() interface{} {
	return e.EventData
}

// Timestamp returns the event timestamp
func (e *BaseEvent) Timestamp() time.Time {
	return e.EventTime
}

// Source returns the event source
func (e *BaseEvent) Source() string {
	return e.EventSource
}

// NewEvent creates a new event
func NewEvent(eventType EventType, data interface{}, source string) Event {
	return &BaseEvent{
		EventType:   eventType,
		EventData:   data,
		EventTime:   time.Now(),
		EventSource: source,
	}
}

// SimpleEventBus provides a simple in-memory event bus implementation
type SimpleEventBus struct {
	handlers    map[EventType][]EventHandler
	allHandlers []EventHandler
	mutex       sync.RWMutex
	closed      bool
}

// NewSimpleEventBus creates a new simple event bus
func NewSimpleEventBus() *SimpleEventBus {
	return &SimpleEventBus{
		handlers:    make(map[EventType][]EventHandler),
		allHandlers: make([]EventHandler, 0),
	}
}

// Publish publishes an event to all subscribers
func (eb *SimpleEventBus) Publish(event Event) {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()
	
	if eb.closed {
		return
	}
	
	// Send to specific event type handlers
	if handlers, exists := eb.handlers[event.Type()]; exists {
		for _, handler := range handlers {
			go handler(event) // Handle asynchronously
		}
	}
	
	// Send to all-event handlers
	for _, handler := range eb.allHandlers {
		go handler(event) // Handle asynchronously
	}
}

// Subscribe subscribes to events of a specific type
func (eb *SimpleEventBus) Subscribe(eventType EventType, handler EventHandler) func() {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()
	
	if eb.closed {
		return func() {} // No-op unsubscribe function
	}
	
	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
	
	// Return unsubscribe function
	return func() {
		eb.mutex.Lock()
		defer eb.mutex.Unlock()
		
		if handlers, exists := eb.handlers[eventType]; exists {
			for i, h := range handlers {
				if &h == &handler { // Compare function pointers
					eb.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
					break
				}
			}
		}
	}
}

// SubscribeAll subscribes to all events
func (eb *SimpleEventBus) SubscribeAll(handler EventHandler) func() {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()
	
	if eb.closed {
		return func() {} // No-op unsubscribe function
	}
	
	eb.allHandlers = append(eb.allHandlers, handler)
	
	// Return unsubscribe function
	return func() {
		eb.mutex.Lock()
		defer eb.mutex.Unlock()
		
		for i, h := range eb.allHandlers {
			if &h == &handler { // Compare function pointers
				eb.allHandlers = append(eb.allHandlers[:i], eb.allHandlers[i+1:]...)
				break
			}
		}
	}
}

// Close closes the event bus
func (eb *SimpleEventBus) Close() {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()
	
	eb.closed = true
	eb.handlers = make(map[EventType][]EventHandler)
	eb.allHandlers = make([]EventHandler, 0)
}
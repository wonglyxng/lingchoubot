package service

import (
	"encoding/json"
	"sync"
	"time"
)

// Event represents a real-time event pushed to SSE clients.
type Event struct {
	ID        string          `json:"id"`
	Topic     string          `json:"topic"`      // workflow, approval, tool_call, audit
	EventType string          `json:"event_type"` // e.g. workflow.started, approval.decided
	TargetID  string          `json:"target_id"`
	ProjectID string          `json:"project_id,omitempty"`
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
}

// Subscriber receives events via a channel.
type Subscriber struct {
	ID     string
	Ch     chan *Event
	Topics map[string]bool // empty = all topics
	Done   chan struct{}
}

// EventHub is a simple in-memory pub/sub for server-sent events.
type EventHub struct {
	mu          sync.RWMutex
	subscribers map[string]*Subscriber
	nextID      int
}

func NewEventHub() *EventHub {
	return &EventHub{
		subscribers: make(map[string]*Subscriber),
	}
}

// Subscribe creates a new subscriber. Pass nil topics to receive all events.
func (h *EventHub) Subscribe(topics []string) *Subscriber {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextID++
	id := time.Now().Format("20060102150405") + "-" + string(rune('0'+h.nextID%10))

	topicMap := make(map[string]bool)
	for _, t := range topics {
		topicMap[t] = true
	}

	sub := &Subscriber{
		ID:     id,
		Ch:     make(chan *Event, 64),
		Topics: topicMap,
		Done:   make(chan struct{}),
	}
	h.subscribers[id] = sub
	return sub
}

// Unsubscribe removes a subscriber.
func (h *EventHub) Unsubscribe(sub *Subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.subscribers[sub.ID]; ok {
		close(sub.Done)
		delete(h.subscribers, sub.ID)
	}
}

// Publish sends an event to all matching subscribers (non-blocking).
func (h *EventHub) Publish(event *Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, sub := range h.subscribers {
		if len(sub.Topics) > 0 && !sub.Topics[event.Topic] {
			continue
		}
		select {
		case sub.Ch <- event:
		default:
			// drop if subscriber is slow — SSE will auto-reconnect
		}
	}
}

// ActiveSubscribers returns count of active subscribers.
func (h *EventHub) ActiveSubscribers() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subscribers)
}

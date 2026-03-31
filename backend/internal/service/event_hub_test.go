package service

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEventHub_PublishSubscribe(t *testing.T) {
	hub := NewEventHub()

	// Subscribe to "workflow" topic
	sub := hub.Subscribe([]string{"workflow"})
	defer hub.Unsubscribe(sub)

	if hub.ActiveSubscribers() != 1 {
		t.Fatalf("expected 1 subscriber, got %d", hub.ActiveSubscribers())
	}

	// Publish matching event
	hub.Publish(&Event{
		ID:        "e1",
		Topic:     "workflow",
		EventType: "workflow.started",
		TargetID:  "run-1",
		Data:      json.RawMessage(`{"status":"running"}`),
		Timestamp: time.Now(),
	})

	select {
	case evt := <-sub.Ch:
		if evt.ID != "e1" || evt.Topic != "workflow" {
			t.Fatalf("unexpected event: %+v", evt)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}

	// Publish non-matching event — should not arrive
	hub.Publish(&Event{
		ID:    "e2",
		Topic: "approval",
	})

	select {
	case evt := <-sub.Ch:
		t.Fatalf("should not receive non-matching event, got: %+v", evt)
	case <-time.After(100 * time.Millisecond):
		// expected
	}
}

func TestEventHub_AllTopics(t *testing.T) {
	hub := NewEventHub()

	// Subscribe to all topics (nil)
	sub := hub.Subscribe(nil)
	defer hub.Unsubscribe(sub)

	hub.Publish(&Event{ID: "e1", Topic: "workflow"})
	hub.Publish(&Event{ID: "e2", Topic: "approval"})
	hub.Publish(&Event{ID: "e3", Topic: "audit"})

	received := 0
	timeout := time.After(time.Second)
	for received < 3 {
		select {
		case <-sub.Ch:
			received++
		case <-timeout:
			t.Fatalf("timed out, received only %d/3", received)
		}
	}
}

func TestEventHub_Unsubscribe(t *testing.T) {
	hub := NewEventHub()
	sub := hub.Subscribe(nil)

	hub.Unsubscribe(sub)
	if hub.ActiveSubscribers() != 0 {
		t.Fatalf("expected 0 subscribers after unsubscribe, got %d", hub.ActiveSubscribers())
	}

	// Publish should not panic after unsubscribe
	hub.Publish(&Event{ID: "e1", Topic: "test"})
}

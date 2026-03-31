package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lingchou/lingchoubot/backend/internal/service"
)

// SSEHandler provides a Server-Sent Events endpoint for real-time updates.
type SSEHandler struct {
	hub *service.EventHub
}

func NewSSEHandler(hub *service.EventHub) *SSEHandler {
	return &SSEHandler{hub: hub}
}

func (h *SSEHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/events/stream", h.Stream)
}

// Stream handles SSE connections.
// Query params:
//   - topics: comma-separated list of topics (workflow,approval,tool_call,audit). Empty = all.
func (h *SSEHandler) Stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Disable write deadline for this long-lived connection
	rc := http.NewResponseController(w)
	rc.SetWriteDeadline(time.Time{})

	// Parse topic filter
	var topics []string
	if t := r.URL.Query().Get("topics"); t != "" {
		topics = strings.Split(t, ",")
	}

	sub := h.hub.Subscribe(topics)
	defer h.hub.Unsubscribe(sub)

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // nginx
	flusher.Flush()

	// Heartbeat ticker
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-sub.Done:
			return
		case event := <-sub.Ch:
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", event.ID, event.Topic, data)
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprintf(w, ": heartbeat %d\n\n", time.Now().Unix())
			flusher.Flush()
		}
	}
}

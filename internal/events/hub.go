package events

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type Event struct {
	Type string `json:"type"`
	At   int64  `json:"at"`
	Data any    `json:"data"`
}

type Hub struct {
	mu      sync.Mutex
	seq     uint64
	ring    []Event
	waiters []chan Event
}

func New() *Hub {
	return &Hub{ring: make([]Event, 0, 128)}
}

func (h *Hub) Publish(kind string, data any) {
	ev := Event{Type: kind, At: time.Now().Unix(), Data: data}
	h.mu.Lock()
	h.seq++
	if len(h.ring) == 128 {
		h.ring = h.ring[1:]
	}
	h.ring = append(h.ring, ev)
	for _, ch := range h.waiters {
		select {
		case ch <- ev:
		default:
		}
	}
	h.mu.Unlock()
}

func (h *Hub) Subscribe() (chan Event, func()) {
	ch := make(chan Event, 8)
	h.mu.Lock()
	h.waiters = append(h.waiters, ch)
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		out := h.waiters[:0]
		for _, w := range h.waiters {
			if w != ch {
				out = append(out, w)
			}
		}
		h.waiters = out
		h.mu.Unlock()
		close(ch)
	}
}

func (h *Hub) Recent() []Event {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]Event, len(h.ring))
	copy(out, h.ring)
	return out
}

func pass(allow func(Event) bool, ev Event) bool {
	return allow != nil && allow(ev)
}

func (h *Hub) SSE(w http.ResponseWriter, r *http.Request, allow func(Event) bool) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ch, cancel := h.Subscribe()
	defer cancel()
	for _, ev := range h.Recent() {
		if pass(allow, ev) {
			writeSSE(w, ev)
		}
	}
	fl.Flush()
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if !pass(allow, ev) {
				continue
			}
			writeSSE(w, ev)
			fl.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, ev Event) {
	b, err := json.Marshal(ev)
	if err != nil {
		return
	}
	_, _ = w.Write([]byte("data: "))
	_, _ = w.Write(b)
	_, _ = w.Write([]byte("\n\n"))
}

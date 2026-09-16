package events

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type sseBuf struct {
	mu      sync.Mutex
	b       strings.Builder
	header  http.Header
	flushed chan struct{}
}

func newSSEBuf() *sseBuf {
	return &sseBuf{header: http.Header{}, flushed: make(chan struct{}, 8)}
}

func (w *sseBuf) Header() http.Header { return w.header }

func (w *sseBuf) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.Write(p)
}

func (w *sseBuf) WriteHeader(int) {}

func (w *sseBuf) Flush() {
	select {
	case w.flushed <- struct{}{}:
	default:
	}
}

func (w *sseBuf) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.String()
}

func waitFlush(t *testing.T, w *sseBuf) {
	t.Helper()
	select {
	case <-w.flushed:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for flush")
	}
}

func waitBody(t *testing.T, w *sseBuf, want string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(w.String(), want) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("missing %q in %q", want, w.String())
}

func shownOnly(ev Event) bool {
	m, _ := ev.Data.(map[string]any)
	repo, _ := m["repo"].(string)
	return repo == "saidc/shown"
}

func TestPassNilDeny(t *testing.T) {
	if pass(nil, Event{Type: "pipeline.updated"}) {
		t.Fatal("nil allow must deny")
	}
	if pass(func(Event) bool { return false }, Event{Type: "x"}) {
		t.Fatal("false")
	}
	if !pass(func(Event) bool { return true }, Event{Type: "x"}) {
		t.Fatal("true")
	}
}

func TestSSEReplayFiltered(t *testing.T) {
	h := New()
	h.Publish("pipeline.updated", map[string]any{"repo": "saidc/hidden", "number": 1})
	h.Publish("pipeline.updated", map[string]any{"repo": "saidc/shown", "number": 2})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	w := newSSEBuf()
	done := make(chan struct{})
	go func() {
		h.SSE(w, req, shownOnly)
		close(done)
	}()
	waitFlush(t, w)
	body := w.String()
	if strings.Contains(body, "saidc/hidden") {
		t.Fatalf("leaked hidden replay: %s", body)
	}
	if !strings.Contains(body, "saidc/shown") {
		t.Fatalf("missing shown replay: %s", body)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("sse did not return")
	}
}

func TestSSELiveFiltered(t *testing.T) {
	h := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	w := newSSEBuf()
	done := make(chan struct{})
	go func() {
		h.SSE(w, req, shownOnly)
		close(done)
	}()
	waitFlush(t, w)

	h.Publish("pipeline.updated", map[string]any{"repo": "saidc/hidden", "number": 3})
	h.Publish("pipeline.updated", map[string]any{"repo": "saidc/shown", "number": 4})
	waitBody(t, w, `"number":4`)
	body := w.String()
	if strings.Contains(body, "saidc/hidden") {
		t.Fatalf("leaked hidden live: %s", body)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("sse did not return")
	}
}

func TestSSENilAllowWritesNothing(t *testing.T) {
	h := New()
	h.Publish("pipeline.updated", map[string]any{"repo": "saidc/hidden"})
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	w := newSSEBuf()
	done := make(chan struct{})
	go func() {
		h.SSE(w, req, nil)
		close(done)
	}()
	waitFlush(t, w)
	if body := w.String(); body != "" {
		t.Fatalf("nil allow leaked %q", body)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("sse did not return")
	}
}

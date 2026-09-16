package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"acahti/internal/auth"
	"acahti/internal/catalog"
	"acahti/internal/config"
	"acahti/internal/events"
	"acahti/internal/forgejo"
	"acahti/internal/woodpecker"
)

type flushBuf struct {
	mu      sync.Mutex
	b       strings.Builder
	header  http.Header
	flushed chan struct{}
}

func newFlushBuf() *flushBuf {
	return &flushBuf{header: http.Header{}, flushed: make(chan struct{}, 8)}
}

func (w *flushBuf) Header() http.Header { return w.header }
func (w *flushBuf) WriteHeader(int)     {}
func (w *flushBuf) Flush() {
	select {
	case w.flushed <- struct{}{}:
	default:
	}
}

func (w *flushBuf) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.Write(p)
}

func (w *flushBuf) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.String()
}

func eventsPages(t *testing.T) (*Pages, *events.Hub) {
	t.Helper()
	fj := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/user" {
			sudo := r.Header.Get("Sudo")
			_ = json.NewEncoder(w).Encode(map[string]any{"login": sudo, "is_admin": sudo == "alice"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(fj.Close)
	cfg := config.Config{Org: "saidc", AdminUser: "alice", SessionSecret: "test"}
	a := auth.New([]byte("test"), "alice")
	hub := events.New()
	cat := catalog.New(cfg, forgejo.New(fj.URL, "t"), nil, nil)
	return New(cfg, cat, forgejo.New(fj.URL, "t"), nil, a, nil, nil, hub), hub
}

func replayEvents(t *testing.T, p *Pages, user string) string {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cookie := httptest.NewRecorder()
	p.SetSession(cookie, user)
	req := httptest.NewRequest(http.MethodGet, "/ui/events", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Cookie", cookie.Header().Get("Set-Cookie"))
	w := newFlushBuf()
	done := make(chan struct{})
	go func() {
		p.Events(w, req)
		close(done)
	}()
	select {
	case <-w.flushed:
	case <-time.After(2 * time.Second):
		t.Fatal("no sse flush")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("sse hang")
	}
	return w.String()
}

func TestEventsHidesForeignPipelines(t *testing.T) {
	p, hub := eventsPages(t)
	hub.Publish("catalog.updated", map[string]any{"ok": true})
	hub.Publish("pipeline.updated", woodpecker.Pipeline{Repo: "saidc/secret", Number: 9, Title: "hidden"})

	member := replayEvents(t, p, "bob")
	if !strings.Contains(member, `"type":"catalog.updated"`) {
		t.Fatalf("member missing catalog.updated: %s", member)
	}
	if strings.Contains(member, "saidc/secret") {
		t.Fatalf("member leaked pipeline: %s", member)
	}

	admin := replayEvents(t, p, "alice")
	if !strings.Contains(admin, "saidc/secret") {
		t.Fatalf("admin missing pipeline: %s", admin)
	}
}

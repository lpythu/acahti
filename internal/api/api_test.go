package api

import (
	"context"
	"net/http"
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

func TestRoute(t *testing.T) {
	cases := []struct {
		path, method, tool string
		extra              map[string]any
	}{
		{"/repos/acme/demo/pipelines", http.MethodGet, "pipeline_list", map[string]any{"owner": "acme", "name": "demo", "repo": "acme/demo"}},
		{"/repos/acme/demo/pipelines", http.MethodPost, "pipeline_trigger", map[string]any{"owner": "acme", "name": "demo", "repo": "acme/demo"}},
		{"/repos/acme/demo/pipelines/12", http.MethodGet, "pipeline_get", map[string]any{"owner": "acme", "name": "demo", "repo": "acme/demo", "number": int64(12)}},
		{"/repos/acme/demo/pipelines/12/log", http.MethodGet, "pipeline_log", map[string]any{"owner": "acme", "name": "demo", "repo": "acme/demo", "number": int64(12)}},
		{"/repos/acme/demo/pipelines/12/rerun", http.MethodPost, "pipeline_rerun", map[string]any{"owner": "acme", "name": "demo", "repo": "acme/demo", "number": int64(12)}},
		{"/repos/acme/demo/pipelines/12/cancel", http.MethodPost, "pipeline_cancel", map[string]any{"owner": "acme", "name": "demo", "repo": "acme/demo", "number": int64(12)}},
		{"/repos/acme/demo/pipelines/12", http.MethodDelete, "pipeline_delete", map[string]any{"owner": "acme", "name": "demo", "repo": "acme/demo", "number": int64(12)}},
		{"/repos/acme/demo/pipelines/12/approve", http.MethodPost, "deploy_approve", map[string]any{"owner": "acme", "name": "demo", "repo": "acme/demo", "number": int64(12)}},
		{"/repos/acme/demo/pulls/3/comments", http.MethodGet, "pr_comments", map[string]any{"owner": "acme", "name": "demo", "number": 3}},
		{"/repos/acme/demo/pulls/3/comments", http.MethodPost, "pr_comment", map[string]any{"owner": "acme", "name": "demo", "number": 3}},
		{"/repos/acme/demo/checks/abc", http.MethodGet, "checks_wait", map[string]any{"owner": "acme", "name": "demo", "sha": "abc"}},
		{"/repos/acme/demo/secrets", http.MethodGet, "secret_list", map[string]any{"owner": "acme", "name": "demo", "scope": "repo"}},
		{"/repos/acme/demo/secrets/kubeconfig_office", http.MethodPut, "secret_put", map[string]any{"owner": "acme", "name": "demo", "scope": "repo", "secret": "kubeconfig_office"}},
		{"/secrets", http.MethodGet, "secret_list", map[string]any{"scope": "org"}},
		{"/secrets/harbor_password", http.MethodDelete, "secret_delete", map[string]any{"scope": "org", "name": "harbor_password"}},
		{"/inbox", http.MethodGet, "inbox", map[string]any{}},
		{"/pipelines/acme/12/log", http.MethodGet, "", nil},
	}
	for _, tc := range cases {
		tool, extra := route(tc.path, tc.method, map[string]any{})
		if tool != tc.tool {
			t.Fatalf("%s %s tool=%q want %q", tc.method, tc.path, tool, tc.tool)
		}
		for k, want := range tc.extra {
			if extra[k] != want {
				t.Fatalf("%s %s extra[%s]=%v want %v", tc.method, tc.path, k, extra[k], want)
			}
		}
	}
}

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

func TestEventsHidesForeignPipelines(t *testing.T) {
	a := auth.New([]byte("test"), "alice")
	hub := events.New()
	hub.Publish("pipeline.updated", woodpecker.Pipeline{Repo: "saidc/secret", Number: 9, Title: "hidden"})
	cat := catalog.New(config.Config{Org: "saidc", AdminUser: "alice"}, forgejo.New("http://127.0.0.1:9", "t"), nil, nil)
	api := &API{Cfg: config.Config{RootURL: "http://127.0.0.1", Org: "saidc", AdminUser: "alice"}, Auth: a, Hub: hub, Cat: cat}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptestReq(t, ctx, a.Issue("bob"))
	w := newFlushBuf()
	done := make(chan struct{})
	go func() {
		api.ServeHTTP(w, req)
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
	if body := w.String(); strings.Contains(body, "saidc/secret") {
		t.Fatalf("member leaked pipeline: %s", body)
	}
}

func httptestReq(t *testing.T, ctx context.Context, token string) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/acahti/v1/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

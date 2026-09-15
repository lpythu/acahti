package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"acahti/internal/forgejo"
	"acahti/internal/httperr"
)

func TestToolSet(t *testing.T) {
	need := []string{
		"whoami",
		"repo_list", "repo_get", "repo_create",
		"branch_list", "ref_delete",
		"pr_create", "pr_list", "pr_get", "pr_comment", "pr_comments", "pr_merge", "pr_close",
		"checks_wait", "pipeline_list", "pipeline_get", "pipeline_log",
		"pipeline_rerun", "pipeline_trigger", "pipeline_cancel", "pipeline_delete", "inbox",
		"pkg_publish", "pkg_list", "agent_status", "deploy_approve",
	}
	have := map[string]bool{}
	for _, n := range ToolNames() {
		have[n] = true
	}
	for _, n := range need {
		if !have[n] {
			t.Fatalf("missing tool %s", n)
		}
	}
}

func TestToolSchemasObjectProperties(t *testing.T) {
	raw, err := json.Marshal(tools())
	if err != nil {
		t.Fatal(err)
	}
	var listed []map[string]any
	if err := json.Unmarshal(raw, &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) == 0 {
		t.Fatal("empty")
	}
	for _, tool := range listed {
		schema, _ := tool["inputSchema"].(map[string]any)
		if _, ok := schema["properties"].(map[string]any); !ok {
			t.Fatalf("%s properties=%T %v", tool["name"], schema["properties"], schema["properties"])
		}
	}
}

func TestCodeForStatus(t *testing.T) {
	if httperr.CodeForStatus(404) != "not_found" {
		t.Fatal(httperr.CodeForStatus(404))
	}
}

func TestAsInt64(t *testing.T) {
	if asInt64(int64(2)) != 2 || asInt64(2) != 2 || asInt64(2.0) != 2 || asInt64("2") != 2 {
		t.Fatal(asInt64(int64(2)), asInt64(2), asInt64(2.0), asInt64("2"))
	}
	if asInt64(nil) != 0 {
		t.Fatal("nil")
	}
}

func TestWaitChecksEmptyIsDone(t *testing.T) {
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(hs.Close)
	s := &Server{FJ: forgejo.New(hs.URL, "t")}
	out, err := s.waitChecks("acme", "demo", "abc")
	if err != nil {
		t.Fatal(err)
	}
	m, _ := out.(map[string]any)
	if m["ok"] != true || m["done"] != true {
		t.Fatalf("%v", out)
	}
}

func TestWaitChecksLatestPerContext(t *testing.T) {
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{"status":"success","context":"ci","created_at":"2026-01-01T02:00:00Z"},
			{"status":"pending","context":"ci","created_at":"2026-01-01T01:00:00Z"}
		]`))
	}))
	t.Cleanup(hs.Close)
	s := &Server{FJ: forgejo.New(hs.URL, "t")}
	out, err := s.waitChecks("acme", "demo", "abc")
	if err != nil {
		t.Fatal(err)
	}
	m, _ := out.(map[string]any)
	if m["ok"] != true || m["done"] != true {
		t.Fatalf("%v", out)
	}
}

func TestWaitChecksSnapshot(t *testing.T) {
	var hits int
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte(`[{"status":"pending","context":"ci"}]`))
	}))
	t.Cleanup(hs.Close)
	s := &Server{FJ: forgejo.New(hs.URL, "t")}
	start := time.Now()
	out, err := s.waitChecks("acme", "demo", "abc")
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("snapshot slept")
	}
	if hits != 1 {
		t.Fatalf("hits=%d", hits)
	}
	m, _ := out.(map[string]any)
	if m["ok"] != false || m["done"] != false {
		t.Fatalf("%v", out)
	}
	if _, ok := m["timeout"]; ok {
		t.Fatalf("snapshot must not set timeout: %v", out)
	}
}

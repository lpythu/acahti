package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestStructuredError(t *testing.T) {
	err := fail("failed_precondition", "checks not green")
	b, _ := json.Marshal(err.Data)
	var info httperr.Info
	if json.Unmarshal(b, &info) != nil {
		t.Fatal("data must be httperr.Info")
	}
	if info.Code != "failed_precondition" {
		t.Fatalf("code=%s", info.Code)
	}
	if !strings.Contains(info.Message, "checks") {
		t.Fatalf("message=%s", info.Message)
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

func TestInitializeInstructions(t *testing.T) {
	s := &Server{Cfg: ConfigView{RootURL: "https://acahti.saidc.ai", Domain: "acahti.saidc.ai", Org: "acme", Version: "0.2.36"}}
	res, err := s.dispatch("", rpcReq{Method: "initialize"})
	if err != nil {
		t.Fatal(err)
	}
	m, _ := res.(map[string]any)
	inst, _ := m["instructions"].(string)
	if !strings.Contains(inst, "apply_when_remote_host") || !strings.Contains(inst, "acahti.saidc.ai") || !strings.Contains(inst, "/skill.md") {
		t.Fatalf("instructions=%s", inst)
	}
	info, _ := m["serverInfo"].(map[string]any)
	if info["name"] != "acahti" || info["title"] != "Acahti" || info["version"] != "0.2.36" {
		t.Fatalf("serverInfo=%v", info)
	}
	icons, _ := info["icons"].([]map[string]any)
	if len(icons) == 0 {
		t.Fatalf("icons=%v", info["icons"])
	}
	src, _ := icons[0]["src"].(string)
	if !strings.HasPrefix(src, "data:image/png;base64,") {
		t.Fatalf("icon src=%s", src)
	}
}

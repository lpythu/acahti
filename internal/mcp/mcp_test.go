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
	"acahti/internal/woodpecker"
)

func TestToolSet(t *testing.T) {
	need := []string{
		"whoami",
		"repo_list", "repo_get", "repo_create",
		"branch_list", "ref_delete",
		"pr_create", "pr_list", "pr_get", "pr_comment", "pr_comments", "pr_merge",
		"checks_wait", "pipeline_list", "pipeline_get", "pipeline_log",
		"pipeline_rerun", "pipeline_trigger", "pipeline_cancel", "inbox",
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

func TestCheckTimeout(t *testing.T) {
	if checkTimeout(nil) != 600 {
		t.Fatal("omit")
	}
	if checkTimeout(map[string]any{}) != 600 {
		t.Fatal("empty")
	}
	if checkTimeout(map[string]any{"timeout_sec": 0}) != 0 {
		t.Fatal("zero")
	}
	if checkTimeout(map[string]any{"timeout_sec": "0"}) != 0 {
		t.Fatal("zero string")
	}
	if checkTimeout(map[string]any{"timeout_sec": 45.0}) != 45 {
		t.Fatal("45")
	}
	if checkTimeout(map[string]any{"timeout_sec": -1}) != 600 {
		t.Fatal("neg")
	}
}

func TestFilterPipes(t *testing.T) {
	items := []woodpecker.Pipeline{
		{Number: 1, Commit: "abcdef", Branch: "dev", Status: "success"},
		{Number: 2, Commit: "abc999", Branch: "test", Status: "failure"},
		{Number: 3, Commit: "fff", Branch: "dev", Status: "running"},
	}
	got := filterPipes(items, "abc", "", "")
	if len(got) != 2 || got[0].Number != 1 || got[1].Number != 2 {
		t.Fatalf("sha=%+v", got)
	}
	got = filterPipes(items, "abcdef", "dev", "success")
	if len(got) != 1 || got[0].Number != 1 {
		t.Fatalf("all=%+v", got)
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
	out, err := s.waitChecks("acme", "demo", "abc", 0)
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
	if m["ok"] != false {
		t.Fatalf("%v", out)
	}
	if _, ok := m["timeout"]; ok {
		t.Fatalf("snapshot must not set timeout: %v", out)
	}
}

func TestInitializeInstructions(t *testing.T) {
	s := &Server{Cfg: ConfigView{RootURL: "https://acahti.saidc.ai", Domain: "acahti.saidc.ai", Org: "acme"}}
	res, err := s.dispatch("", rpcReq{Method: "initialize"})
	if err != nil {
		t.Fatal(err)
	}
	m, _ := res.(map[string]any)
	inst, _ := m["instructions"].(string)
	if !strings.Contains(inst, "apply_when_remote_host") || !strings.Contains(inst, "acahti.saidc.ai") {
		t.Fatalf("instructions=%s", inst)
	}
}

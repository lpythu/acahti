package mcp

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"acahti/internal/catalog"
	"acahti/internal/config"
	"acahti/internal/forgejo"
	"acahti/internal/httperr"
	"acahti/internal/woodpecker"
)

func TestToolSet(t *testing.T) {
	need := []string{
		"whoami",
		"repo_list", "repo_get", "repo_create",
		"branch_list", "ref_delete",
		"pr_create", "pr_list", "pr_get", "pr_comment", "pr_comments", "pr_merge", "pr_close",
		"checks_wait", "pipeline_list", "pipeline_get", "pipeline_log",
		"pipeline_rerun", "pipeline_trigger", "pipeline_cancel", "pipeline_delete", "inbox",
		"pkg_publish", "pkg_list", "pkg_delete", "agent_status", "deploy_approve",
		"secret_list", "secret_put", "secret_delete",
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

func secretCatalog(t *testing.T) *catalog.Catalog {
	t.Helper()
	fj := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sudo := r.Header.Get("Sudo")
		switch {
		case r.URL.Path == "/api/v1/user":
			_ = json.NewEncoder(w).Encode(map[string]any{"login": sudo, "is_admin": sudo == "alice"})
		case strings.HasPrefix(r.URL.Path, "/api/v1/repos/saidc/tm-web"):
			admin := sudo == "alice" || sudo == "repoadmin"
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name": "tm-web", "full_name": "saidc/tm-web",
				"permissions": map[string]bool{"admin": admin, "push": true, "pull": true},
			})
		case strings.HasPrefix(r.URL.Path, "/api/v1/orgs/saidc/teams"):
			_ = json.NewEncoder(w).Encode([]any{})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(fj.Close)
	wp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/web-config.js"):
			_, _ = w.Write([]byte(`WOODPECKER_CSRF = "tok";`))
		case r.URL.Path == "/api/orgs/lookup/saidc":
			_, _ = w.Write([]byte(`{"id":1,"name":"saidc"}`))
		case r.URL.Path == "/api/orgs/1/secrets":
			_, _ = w.Write([]byte(`[{"name":"harbor_password","value":"nope","events":["push"]}]`))
		case strings.Contains(r.URL.Path, "/api/repos/") && strings.HasSuffix(r.URL.Path, "/secrets"):
			_, _ = w.Write([]byte(`[{"name":"svc_token","value":"nope"}]`))
		case r.URL.Path == "/api/repos/lookup/saidc/tm-web":
			_, _ = w.Write([]byte(`{"id":9,"full_name":"saidc/tm-web"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(wp.Close)
	fjClient := forgejo.New(fj.URL, "t")
	return catalog.New(config.Config{Org: "saidc", AdminUser: "alice"}, fjClient, woodpecker.New(wp.URL, "t"), nil)
}

func TestAgentStatusIncludesQueue(t *testing.T) {
	now := time.Now().Unix()
	wp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/agents":
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"id": 1, "name": "buildof", "capacity": 4, "last_contact": now,
			}})
		case r.URL.Path == "/api/queue/info":
			_, _ = w.Write([]byte(`{"paused":false,"pending":[{"name":"ci","repo_id":7,"pipeline_number":12}],"waiting_on_deps":[],"running":[{"name":"ci","repo_id":7,"pipeline_number":11,"agent_name":"buildof"}],"stats":{"pending_count":1,"running_count":1,"waiting_on_deps_count":0}}`))
		case r.URL.Path == "/api/repos/7":
			_, _ = w.Write([]byte(`{"id":7,"full_name":"saidc/tm-web"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(wp.Close)
	cat := catalog.New(config.Config{Org: "saidc"}, nil, woodpecker.New(wp.URL, "t"), nil)
	s := &Server{Cat: cat}
	out, err := s.CallForAPI("alice", "agent_status", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	st, ok := out.(catalog.AgentStatus)
	if !ok {
		t.Fatalf("%T", out)
	}
	if st.Queue.Stats.PendingCount != 1 || len(st.Queue.Pending) != 1 || st.Queue.Pending[0].Repo != "saidc/tm-web" {
		t.Fatalf("queue %+v", st.Queue)
	}
	if len(st.Items) != 1 || st.Items[0].Capacity != 4 || st.Items[0].Running != 1 {
		t.Fatalf("agents %+v", st.Items)
	}
}

func TestSecretListMemberNotFound(t *testing.T) {
	cat := secretCatalog(t)
	s := &Server{Cfg: ConfigView{Org: "saidc", RootURL: "https://acahti.example"}, FJ: cat.FJ, Cat: cat}

	if _, err := s.CallForAPI("bob", "secret_list", map[string]any{"scope": "org"}); !errors.Is(err, catalog.ErrNotFound) {
		t.Fatalf("member org list: %v", err)
	}
	if _, err := s.CallForAPI("bob", "secret_list", map[string]any{"scope": "repo", "repo": "saidc/tm-web"}); !errors.Is(err, catalog.ErrNotFound) {
		t.Fatalf("member repo list: %v", err)
	}

	listed, err := s.CallForAPI("alice", "secret_list", map[string]any{"scope": "org"})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(listed)
	if strings.Contains(string(raw), "nope") || strings.Contains(string(raw), `"value"`) {
		t.Fatalf("value leaked: %s", raw)
	}
	if !strings.Contains(string(raw), "harbor_password") {
		t.Fatalf("missing name: %s", raw)
	}

	who, err := s.CallForAPI("alice", "whoami", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	m, _ := who.(map[string]any)
	if m["org_admin"] != true {
		t.Fatalf("alice org_admin=%v", m["org_admin"])
	}
	who, err = s.CallForAPI("bob", "whoami", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	m, _ = who.(map[string]any)
	if m["org_admin"] != false {
		t.Fatalf("bob org_admin=%v", m["org_admin"])
	}

	repo, err := s.CallForAPI("bob", "repo_get", map[string]any{"owner": "saidc", "name": "tm-web"})
	if err != nil {
		t.Fatal(err)
	}
	r, _ := repo.(forgejo.Repo)
	if r.CanManageSecrets {
		t.Fatal("pusher must not manage secrets")
	}
	repo, err = s.CallForAPI("repoadmin", "repo_get", map[string]any{"owner": "saidc", "name": "tm-web"})
	if err != nil {
		t.Fatal(err)
	}
	r, _ = repo.(forgejo.Repo)
	if !r.CanManageSecrets {
		t.Fatal("repo admin can_manage_secrets")
	}
}

package woodpecker

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"acahti/internal/page"
)

func TestPipelineJobsFromKernelWorkflows(t *testing.T) {
	var p Pipeline
	if err := json.Unmarshal([]byte(`{"number":2,"status":"success","workflows":[{"name":"ci","state":"success"}]}`), &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Jobs) != 1 || p.Jobs[0].Name != "ci" {
		t.Fatalf("jobs=%v", p.Jobs)
	}
	out, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"jobs"`) || strings.Contains(string(out), `"workflows"`) {
		t.Fatalf("public json=%s", out)
	}
}

func TestListReposOmitsAllTrue(t *testing.T) {
	var raw string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw = r.URL.RawQuery
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(s.Close)
	c := New(s.URL, "t")
	if _, err := c.ListRepos(page.Query{Page: 1, Size: 20}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw, "all=true") {
		t.Fatalf("query %q", raw)
	}
}

func TestRepoKeyUsesCache(t *testing.T) {
	c := New("http://127.0.0.1", "t")
	c.ids["acme/demo"] = 42
	key, err := c.repoKey("acme/demo")
	if err != nil || key != "42" {
		t.Fatalf("%q %v", key, err)
	}
}

func TestActivatePostsForgeRemoteID(t *testing.T) {
	var posts []string
	var patch string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		posts = append(posts, r.Method+" "+r.URL.RequestURI())
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/web-config.js"):
			_, _ = w.Write([]byte(`WOODPECKER_CSRF = "tok";`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/repos":
			if r.URL.Query().Get("forge_remote_id") != "99" {
				t.Errorf("forge_remote_id=%q", r.URL.Query().Get("forge_remote_id"))
			}
			_, _ = w.Write([]byte(`{"id":7,"full_name":"saidc/demo","forge_remote_id":"99"}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/repos/7":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			patch, _ = body["config_file"].(string)
			if _, ok := body["config"]; ok {
				t.Fatal("Woodpecker 3.18 patch field is config_file")
			}
			_, _ = w.Write([]byte(`{"id":7,"config_file":".acahti/pipelines/"}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.RequestURI())
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(s.Close)
	c := New(s.URL, "t")
	if err := c.Activate("saidc/demo", "99"); err != nil {
		t.Fatal(err)
	}
	if len(posts) < 2 || !strings.Contains(posts[1], "POST /api/repos?forge_remote_id=99") {
		t.Fatalf("posts=%v", posts)
	}
	if patch != ".acahti/pipelines/" {
		t.Fatalf("config_file=%q", patch)
	}
}

func TestActivateConflictLooksUp(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/web-config.js"):
			_, _ = w.Write([]byte(`WOODPECKER_CSRF = "tok";`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/repos":
			http.Error(w, "Repository is already active.", http.StatusConflict)
		case r.Method == http.MethodGet && r.URL.Path == "/api/repos/lookup/saidc/demo":
			_, _ = w.Write([]byte(`{"id":4,"full_name":"saidc/demo"}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/repos/4":
			_, _ = w.Write([]byte(`{"id":4}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.RequestURI())
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(s.Close)
	c := New(s.URL, "t")
	if err := c.Activate("saidc/demo", "99"); err != nil {
		t.Fatal(err)
	}
}

func TestCancelPostsPath(t *testing.T) {
	var got string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Method + " " + r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(s.Close)
	c := New(s.URL, "t")
	c.ids["acme/demo"] = 7
	if err := c.Cancel("acme/demo", 12); err != nil {
		t.Fatal(err)
	}
	if got != "POST /api/repos/7/pipelines/12/cancel" {
		t.Fatalf("got %q", got)
	}
}



package woodpecker

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	c.reposAt = time.Now()
	key, err := c.repoKey("acme/demo")
	if err != nil || key != "42" {
		t.Fatalf("%q %v", key, err)
	}
}

func TestLatestPipelinesPreservesOrder(t *testing.T) {
	c := New("http://127.0.0.1", "t")
	c.storeLatest("a/one", Pipeline{Number: 1, Repo: "a/one", Status: "success"})
	c.storeLatest("a/two", Pipeline{Number: 2, Repo: "a/two", Status: "failure"})
	c.storeLatest("a/skip", Pipeline{Number: 3, Repo: "a/skip"})
	got := c.LatestPipelines([]string{"a/two", "a/one"}, false)
	if len(got) != 2 || got[0].Repo != "a/two" || got[1].Repo != "a/one" {
		t.Fatalf("%+v", got)
	}
}


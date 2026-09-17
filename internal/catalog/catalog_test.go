package catalog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"acahti/internal/config"
	"acahti/internal/forgejo"
	"acahti/internal/store"
	"acahti/internal/woodpecker"
)

func TestOwnersTeam(t *testing.T) {
	if !ownersTeam("Owners") || !ownersTeam("owners") {
		t.Fatal("Owners is reserved")
	}
	if ownersTeam("Platform") {
		t.Fatal("Platform is a code team")
	}
}

func TestValidTeamName(t *testing.T) {
	if err := ValidTeamName("Platform"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", "Owners", "1bad", "foo.bar", "has space"} {
		if ValidTeamName(name) == nil {
			t.Fatalf("accepted %q", name)
		}
	}
}

func TestParseRoleTeam(t *testing.T) {
	name, perm, ok := parseRoleTeam("Platform")
	if !ok || name != "Platform" || perm != permWrite {
		t.Fatalf("%s %s %v", name, perm, ok)
	}
	name, perm, ok = parseRoleTeam("Platform.read")
	if !ok || name != "Platform" || perm != permRead {
		t.Fatalf("%s %s %v", name, perm, ok)
	}
	name, perm, ok = parseRoleTeam("Platform.admin")
	if !ok || name != "Platform" || perm != permAdmin {
		t.Fatalf("%s %s %v", name, perm, ok)
	}
	if _, _, ok = parseRoleTeam("Owners"); ok {
		t.Fatal("owners")
	}
	if roleTeamName("Express", "admin") != "Express.admin" || roleTeamName("Express", "write") != "Express" {
		t.Fatal("roleTeamName")
	}
}

func TestMarkTeamVisible(t *testing.T) {
	repos := []forgejo.Repo{
		{FullName: "saidc/api-gateway"},
		{FullName: "saidc/ejp"},
	}
	got := markTeam(repos, "Platform", map[string]bool{"saidc/api-gateway": true})
	if len(got) != 1 || got[0].FullName != "saidc/api-gateway" || got[0].Team != "Platform" {
		t.Fatalf("%+v", got)
	}
}

func TestWriteID(t *testing.T) {
	t0 := teamRoles{roles: map[string]forgejo.Team{permRead: {ID: 1}, permWrite: {ID: 2}}}
	if t0.writeID() != 2 {
		t.Fatal(t0.writeID())
	}
}

func TestDecoratePipeKeepsKernelJobs(t *testing.T) {
	c := New(config.Config{}, nil, nil, nil)
	p := c.decoratePipe(woodpecker.Pipeline{Status: "error", Error: "bad yaml", Repo: "saidc/demo"})
	if p.Error != "bad yaml" {
		t.Fatalf("%+v", p)
	}
	if len(p.Jobs) != 0 {
		t.Fatalf("jobs=%+v", p.Jobs)
	}
}

func TestStepFailedAndTailLog(t *testing.T) {
	if !stepFailed("failure") || !stepFailed("killed") || stepFailed("success") {
		t.Fatal("stepFailed")
	}
	text := "a\nb\nc\nd"
	if got := tailLog(text, 2); got != "c\nd" {
		t.Fatalf("tail=%q", got)
	}
	if got := tailLog(text, 0); got != text {
		t.Fatalf("all=%q", got)
	}
}

func TestAsRepoUpdated(t *testing.T) {
	c := New(config.Config{RootURL: "https://acahti.example.com", Org: "saidc"}, nil, nil, nil)
	got := c.asRepo(store.OrgRepo{FullName: "saidc/argos-pack", DefaultBranch: "dev", Updated: 1710000000}, "")
	if got.Name != "argos-pack" || got.Updated != 1710000000 || got.Team != "" {
		t.Fatalf("%+v", got)
	}
}

func TestPublicRepoCloneHTTPS(t *testing.T) {
	c := New(config.Config{RootURL: "https://acahti.example.com", Org: "acme"}, nil, nil, nil)
	got := c.PublicRepo(forgejo.Repo{FullName: "acme/demo", CloneURL: "http://forgejo:3000/acme/demo.git"})
	if got.CloneURL != "https://acahti.example.com/acme/demo.git" {
		t.Fatalf("clone_url=%q", got.CloneURL)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "ssh_url") || strings.Contains(string(raw), "ssh://") {
		t.Fatalf("ssh leaked: %s", raw)
	}
}

func TestLiveAgentsDropsStale(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	got := liveAgents([]woodpecker.Agent{
		{ID: 1, Name: "live", LastSeen: now.Add(-time.Minute).Unix()},
		{ID: 2, Name: "dead", LastSeen: now.Add(-10 * time.Minute).Unix()},
		{ID: 3, Name: "zero", LastSeen: 0},
	}, now)
	if len(got) != 1 || got[0].Name != "live" {
		t.Fatalf("%+v", got)
	}
}

func TestExpandPipeStatus(t *testing.T) {
	if got := expandPipeStatus(""); got != nil {
		t.Fatalf("%v", got)
	}
	if got := expandPipeStatus("all"); got != nil {
		t.Fatalf("%v", got)
	}
	got := expandPipeStatus("failed")
	if len(got) != 4 || got[0] != "failure" || got[3] != "declined" {
		t.Fatalf("%v", got)
	}
	got = expandPipeStatus("failed,blocked")
	if len(got) != 5 || got[4] != "blocked" {
		t.Fatalf("%v", got)
	}
	if got := expandPipeStatus("running"); len(got) != 2 || got[0] != "running" || got[1] != "pending" {
		t.Fatalf("%v", got)
	}
}

func TestMergeReadyPRsNilIndex(t *testing.T) {
	c := New(config.Config{}, nil, nil, nil)
	p := forgejo.PR{Repo: "saidc/acahti"}
	p.Head.SHA = "abc"
	if got := c.mergeReadyPRs([]forgejo.PR{p}); got != nil {
		t.Fatalf("%+v", got)
	}
}

func TestMergeQueueHeadsReplacesStaleSuccess(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/queue/info"):
			_, _ = w.Write([]byte(`{"pending":[],"waiting_on_deps":[],"running":[{"name":"ci","repo_id":9,"pipeline_number":47}],"stats":{}}`))
		case r.URL.Path == "/api/repos/9":
			_, _ = w.Write([]byte(`{"id":9,"full_name":"saidc/voidgate"}`))
		case strings.Contains(r.URL.Path, "/lookup/"):
			_, _ = w.Write([]byte(`{"id":9,"full_name":"saidc/voidgate"}`))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/pipelines/47"):
			_, _ = w.Write([]byte(`{"number":47,"status":"running","created":200,"workflows":[{"name":"ci","state":"running"}]}`))
		case strings.HasSuffix(r.URL.Path, "/web-config.js"):
			_, _ = w.Write([]byte(`WOODPECKER_CSRF = "tok";`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(s.Close)
	c := New(config.Config{}, nil, woodpecker.New(s.URL, "t"), nil)
	got := c.mergeQueueHeads([]woodpecker.Pipeline{{
		Repo:    "saidc/voidgate",
		Number:  46,
		Status:  "success",
		Created: 100,
	}}, nil, nil, 1)
	if len(got) != 1 || got[0].Number != 47 || got[0].Status != "running" {
		t.Fatalf("%+v", got)
	}
}

func TestRefreshCancelsPendingAfterFailure(t *testing.T) {
	var canceled bool
	gets := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/queue/info"):
			_, _ = w.Write([]byte(`{"pending":[],"waiting_on_deps":[],"running":[],"stats":{}}`))
		case strings.Contains(r.URL.Path, "/lookup/"):
			_, _ = w.Write([]byte(`{"id":9,"full_name":"saidc/voidgate"}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/cancel"):
			canceled = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/pipelines/47"):
			gets++
			if canceled {
				_, _ = w.Write([]byte(`{"number":47,"status":"failure","workflows":[{"name":"ci","state":"failure"},{"name":"cd.office","state":"skipped"}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"number":47,"status":"running","workflows":[{"name":"ci","state":"failure"},{"name":"cd.office","state":"pending"}]}`))
		case strings.HasSuffix(r.URL.Path, "/web-config.js"):
			_, _ = w.Write([]byte(`WOODPECKER_CSRF = "tok";`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(s.Close)
	c := New(config.Config{}, nil, woodpecker.New(s.URL, "t"), nil)
	got, err := c.Refresh("saidc/voidgate", 47)
	if err != nil {
		t.Fatal(err)
	}
	if !canceled || gets < 2 {
		t.Fatalf("canceled=%v gets=%d", canceled, gets)
	}
	if got.Status != "failure" || len(got.Jobs) != 2 || got.Jobs[1].State != "skipped" {
		t.Fatalf("%+v", got)
	}
}

func TestPaintPipesSkipsKernelRefresh(t *testing.T) {
	var gotGet bool
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/queue/info"):
			_, _ = w.Write([]byte(`{"pending":[],"waiting_on_deps":[],"running":[],"stats":{}}`))
		case strings.Contains(r.URL.Path, "/lookup/"):
			_, _ = w.Write([]byte(`{"id":1,"full_name":"saidc/tm-cs"}`))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/pipelines/56"):
			gotGet = true
			_, _ = w.Write([]byte(`{"number":56,"status":"failure"}`))
		case strings.HasSuffix(r.URL.Path, "/web-config.js"):
			_, _ = w.Write([]byte(`WOODPECKER_CSRF = "tok";`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(s.Close)
	c := New(config.Config{}, nil, woodpecker.New(s.URL, "t"), nil)
	out := c.paintPipes([]woodpecker.Pipeline{{
		Repo:   "saidc/tm-cs",
		Number: 56,
		Status: "running",
		Jobs:   []woodpecker.Job{{Name: "ci", State: "running"}},
	}})
	if gotGet {
		t.Fatal("list paint must not GetPipeline")
	}
	if len(out) != 1 {
		t.Fatalf("%+v", out)
	}
}

func TestPermFromRole(t *testing.T) {
	if p := permFromRole("admin"); !p.Admin || !p.Push || !p.Pull {
		t.Fatalf("%+v", p)
	}
	if p := permFromRole("write"); p.Admin || !p.Push || !p.Pull {
		t.Fatalf("%+v", p)
	}
	if p := permFromRole("read"); p.Admin || p.Push || !p.Pull {
		t.Fatalf("%+v", p)
	}
}

func TestAcahtiCheckURL(t *testing.T) {
	c := New(config.Config{RootURL: "https://acahti.example.com"}, nil, nil, nil)
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"https://acahti.example.com/ci", "https://acahti.example.com/pipelines"},
		{"https://acahti.example.com/ci/", "https://acahti.example.com/pipelines"},
		{"https://acahti.example.com/ci/acme/demo/12", "https://acahti.example.com/repos/acme/demo/pipelines/12"},
		{"https://acahti.example.com/ci/repos/acme/demo/pipeline/12", "https://acahti.example.com/repos/acme/demo/pipelines/12"},
		{"https://acahti.example.com/repos/acme/demo", "https://acahti.example.com/repos/acme/demo"},
	}
	for _, tc := range cases {
		if got := c.acahtiCheckURL(tc.in); got != tc.want {
			t.Fatalf("acahtiCheckURL(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

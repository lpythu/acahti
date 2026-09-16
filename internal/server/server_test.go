package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"acahti/internal/auth"
	"acahti/internal/config"
	"acahti/internal/events"
	"acahti/internal/forgejo"
	"acahti/internal/woodpecker"
)

func testHandler(t *testing.T, fjURL string) http.Handler {
	t.Helper()
	return New(config.Config{
		SessionSecret: "test",
		RootURL:       "http://127.0.0.1",
		Org:           "acme",
		ForgejoURL:    fjURL,
		DataDir:       t.TempDir(),
	},
		forgejo.New("http://127.0.0.1:9", ""),
		woodpecker.New("http://127.0.0.1:9", ""),
		events.New())
}

func TestHooksAndNavTree(t *testing.T) {
	h := testHandler(t, "http://127.0.0.1:9")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/hooks/woodpecker", strings.NewReader(`{"repo":{"full_name":"a/b"},"pipeline":{"number":1,"status":"success"}}`)))
	if rr.Code != http.StatusNoContent {
		t.Fatalf("woodpecker hook %d", rr.Code)
	}
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/ui/nav/tree", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("nav tree %d", rr.Code)
	}
}

func TestForgejoHookPublishesPoke(t *testing.T) {
	hub := events.New()
	h := New(config.Config{
		SessionSecret: "test",
		RootURL:       "http://127.0.0.1",
		Org:           "acme",
		DataDir:       t.TempDir(),
	}, forgejo.New("http://127.0.0.1:9", ""), woodpecker.New("http://127.0.0.1:9", ""), hub)
	raw := `{"action":"opened","pull_request":{"title":"secret"},"repository":{"full_name":"saidc/hidden"},"pusher":{"email":"leak@example.com"}}`
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/hooks/forgejo", strings.NewReader(raw)))
	if rr.Code != http.StatusNoContent {
		t.Fatalf("forgejo hook %d", rr.Code)
	}
	got := hub.Recent()
	if len(got) != 1 || got[0].Type != "forgejo" {
		t.Fatalf("%+v", got)
	}
	body, err := json.Marshal(got[0].Data)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "leak@example.com") || strings.Contains(string(body), "saidc/hidden") {
		t.Fatalf("raw webhook leaked: %s", body)
	}
	data, _ := got[0].Data.(map[string]any)
	if data["ok"] != true {
		t.Fatalf("%s", body)
	}
}

func TestMuxRegisters(t *testing.T) {
	h := testHandler(t, "http://127.0.0.1:9")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("healthz %d", rr.Code)
	}
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/login", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("login %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Acahti") {
		t.Fatalf("login page missing brand")
	}
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/repos/acme/demo", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("spa /repos %d", rr.Code)
	}
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("spa /admin %d", rr.Code)
	}
}

func TestPublicAllowlist(t *testing.T) {
	var hit string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = r.URL.Path
		w.WriteHeader(http.StatusTeapot)
	}))
	t.Cleanup(backend.Close)
	h := testHandler(t, backend.URL)

	hit = ""
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/acme/demo.git/info/refs", nil))
	if hit != "" || rr.Code != http.StatusUnauthorized {
		t.Fatalf("git without token hit=%q code=%d", hit, rr.Code)
	}

	tok := auth.New([]byte("test"), "acahti").Issue("alice")
	gitReq := httptest.NewRequest(http.MethodGet, "/acme/demo.git/info/refs", nil)
	gitReq.SetBasicAuth("alice", tok)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, gitReq)
	if hit != "/acme/demo.git/info/refs" || rr.Code != http.StatusTeapot {
		t.Fatalf("git proxy hit=%q code=%d", hit, rr.Code)
	}

	hit = ""
	forge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/user" {
			u, p, ok := r.BasicAuth()
			if ok && u == "alice" && p == "secret" {
				_ = json.NewEncoder(w).Encode(map[string]string{"login": "alice"})
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		hit = r.URL.Path
		w.WriteHeader(http.StatusTeapot)
	}))
	t.Cleanup(forge.Close)
	hForge := New(config.Config{
		SessionSecret: "test",
		RootURL:       "http://127.0.0.1",
		Org:           "acme",
		ForgejoURL:    forge.URL,
		DataDir:       t.TempDir(),
	}, forgejo.New(forge.URL, ""), woodpecker.New("http://127.0.0.1:9", ""), events.New())

	gitPw := httptest.NewRequest(http.MethodGet, "/acme/demo.git/info/refs", nil)
	gitPw.SetBasicAuth("alice", "secret")
	rr = httptest.NewRecorder()
	hForge.ServeHTTP(rr, gitPw)
	if hit != "/acme/demo.git/info/refs" || rr.Code != http.StatusTeapot {
		t.Fatalf("git password hit=%q code=%d", hit, rr.Code)
	}

	hit = ""
	gitBad := httptest.NewRequest(http.MethodGet, "/acme/demo.git/info/refs", nil)
	gitBad.SetBasicAuth("alice", "wrong")
	rr = httptest.NewRecorder()
	hForge.ServeHTTP(rr, gitBad)
	if hit != "" || rr.Code != http.StatusUnauthorized {
		t.Fatalf("git bad password hit=%q code=%d", hit, rr.Code)
	}

	hit = ""
	gitAlias := httptest.NewRequest(http.MethodGet, "/acme/demo.git/info/refs", nil)
	gitAlias.SetBasicAuth("git", "secret")
	rr = httptest.NewRecorder()
	hForge.ServeHTTP(rr, gitAlias)
	if hit != "" || rr.Code != http.StatusUnauthorized {
		t.Fatalf("git alias password hit=%q code=%d", hit, rr.Code)
	}

	hit = ""
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/packages/acme/pypi/simple/", nil))
	if hit != "" || rr.Code != http.StatusUnauthorized {
		t.Fatalf("packages without token hit=%q code=%d", hit, rr.Code)
	}

	hit = ""
	pkgReq := httptest.NewRequest(http.MethodGet, "/api/packages/acme/pypi/simple/", nil)
	pkgReq.SetBasicAuth("alice", tok)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, pkgReq)
	if hit != "/api/packages/acme/pypi/simple/" || rr.Code != http.StatusTeapot {
		t.Fatalf("packages proxy hit=%q code=%d", hit, rr.Code)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/ci", nil))
	if rr.Code != http.StatusFound || rr.Header().Get("Location") != "/pipelines" {
		t.Fatalf("GET /ci -> %d %s", rr.Code, rr.Header().Get("Location"))
	}

	for _, path := range []string{"/ci/api/user", "/login/oauth/authorize", "/user/login", "/api/v1/user"} {
		hit = ""
		rr = httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		if hit != "" {
			t.Fatalf("%s leaked to kernel: %s", path, hit)
		}
		if rr.Code != http.StatusNotFound {
			t.Fatalf("%s code %d", path, rr.Code)
		}
		if !strings.HasPrefix(rr.Header().Get("Content-Type"), "application/json") {
			t.Fatalf("%s content-type %s", path, rr.Header().Get("Content-Type"))
		}
		var body map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil || body["error"] == "" {
			t.Fatalf("%s body %q", path, rr.Body.String())
		}
		if strings.Contains(rr.Body.String(), "<html") {
			t.Fatalf("%s returned HTML", path)
		}
	}
}

func TestSkillAndOAuth(t *testing.T) {
	h := testHandler(t, "http://127.0.0.1:9")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/skill.md", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("skill %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Install http://127.0.0.1/skill.md") || !strings.Contains(body, "Join    http://127.0.0.1/join") || !strings.Contains(body, "Pipeline secrets") || !strings.Contains(body, "secret_put") {
		t.Fatalf("skill contract missing: %s", body)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "/oauth/authorize") {
		t.Fatalf("as metadata %d %s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`)))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("mcp unauth %d", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("WWW-Authenticate"), "resource_metadata") {
		t.Fatalf("challenge %s", rr.Header().Get("WWW-Authenticate"))
	}
	if !strings.Contains(rr.Header().Get("Link"), "/acahti.svg") {
		t.Fatalf("mcp link %s", rr.Header().Get("Link"))
	}

	for _, path := range []string{"/favicon.ico", "/acahti.png", "/apple-touch-icon.png"} {
		rr = httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		if rr.Code != http.StatusOK || rr.Header().Get("Content-Type") != "image/png" {
			t.Fatalf("%s %d %s", path, rr.Code, rr.Header().Get("Content-Type"))
		}
		if strings.Contains(rr.Body.String(), "<html") {
			t.Fatalf("%s returned HTML", path)
		}
	}
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/acahti.svg", nil))
	if rr.Code != http.StatusOK || !strings.HasPrefix(rr.Header().Get("Content-Type"), "image/svg+xml") {
		t.Fatalf("acahti.svg %d %s", rr.Code, rr.Header().Get("Content-Type"))
	}
}

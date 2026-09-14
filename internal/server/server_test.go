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
		if r.URL.Path == "/api/v1/user" && r.Header.Get("Authorization") == "token forge-oauth" {
			_ = json.NewEncoder(w).Encode(map[string]string{"login": "acahti"})
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
	gitForge := httptest.NewRequest(http.MethodGet, "/acme/demo.git/info/refs", nil)
	gitForge.SetBasicAuth("acahti", "forge-oauth")
	rr = httptest.NewRecorder()
	hForge.ServeHTTP(rr, gitForge)
	if hit != "/acme/demo.git/info/refs" || rr.Code != http.StatusTeapot {
		t.Fatalf("git forge token hit=%q code=%d", hit, rr.Code)
	}

	hit = ""
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/packages/acme/pypi/simple/", nil))
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
	if !strings.Contains(body, "Install http://127.0.0.1/skill.md") || !strings.Contains(body, "Join    http://127.0.0.1/join") {
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

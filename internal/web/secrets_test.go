package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"acahti/internal/auth"
	"acahti/internal/catalog"
	"acahti/internal/config"
	"acahti/internal/forgejo"
	"acahti/internal/woodpecker"
)

func secretsPages(t *testing.T) *Pages {
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
	cfg := config.Config{Org: "saidc", AdminUser: "alice", SessionSecret: "test"}
	a := auth.New([]byte("test"), "alice")
	cat := catalog.New(cfg, forgejo.New(fj.URL, "t"), woodpecker.New(wp.URL, "t"), nil)
	return New(cfg, cat, forgejo.New(fj.URL, "t"), woodpecker.New(wp.URL, "t"), a, nil, nil, nil)
}

func TestSecretsUIMember404(t *testing.T) {
	p := secretsPages(t)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ui/secrets", p.OrgSecrets)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}/secrets", p.RepoSecrets)

	get := func(user, path string) *httptest.ResponseRecorder {
		t.Helper()
		cookie := httptest.NewRecorder()
		p.SetSession(cookie, user)
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Cookie", cookie.Header().Get("Set-Cookie"))
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		return rr
	}

	if rr := get("bob", "/ui/secrets"); rr.Code != http.StatusNotFound {
		t.Fatalf("member org list %d %s", rr.Code, rr.Body.String())
	}
	if rr := get("bob", "/ui/repos/saidc/tm-web/secrets"); rr.Code != http.StatusNotFound {
		t.Fatalf("member repo list %d %s", rr.Code, rr.Body.String())
	}

	admin := get("alice", "/ui/secrets")
	if admin.Code != http.StatusOK {
		t.Fatalf("admin org list %d %s", admin.Code, admin.Body.String())
	}
	body, _ := io.ReadAll(admin.Body)
	if strings.Contains(string(body), "nope") || strings.Contains(string(body), `"value"`) {
		t.Fatalf("value leaked: %s", body)
	}
	if !strings.Contains(string(body), "harbor_password") {
		t.Fatalf("missing name: %s", body)
	}

	repo := get("repoadmin", "/ui/repos/saidc/tm-web/secrets")
	if repo.Code != http.StatusOK {
		t.Fatalf("repo admin list %d %s", repo.Code, repo.Body.String())
	}
	raw, _ := io.ReadAll(repo.Body)
	if strings.Contains(string(raw), "nope") {
		t.Fatalf("value leaked: %s", raw)
	}
}

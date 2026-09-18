package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"acahti/internal/auth"
	"acahti/internal/catalog"
	"acahti/internal/config"
	"acahti/internal/forgejo"
)

func orgPages(t *testing.T) *Pages {
	t.Helper()
	fj := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/user":
			sudo := r.Header.Get("Sudo")
			_ = json.NewEncoder(w).Encode(map[string]any{"login": sudo, "full_name": sudo, "is_admin": sudo == "alice"})
		case r.URL.Path == "/api/v1/admin/orgs":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"username": "saidc", "full_name": "SAIDC"},
				{"username": "acme", "full_name": "Acme"},
			})
		case r.URL.Path == "/api/v1/user/orgs":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"username": "saidc", "full_name": "SAIDC"},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/orgs":
			_ = json.NewEncoder(w).Encode(map[string]any{"username": "labs", "full_name": "Labs"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(fj.Close)
	cfg := config.Config{Org: "saidc", AdminUser: "alice", SessionSecret: "test"}
	a := auth.New([]byte("test"), "alice")
	fjClient := forgejo.New(fj.URL, "t")
	cat := catalog.New(cfg, fjClient, nil, nil)
	return New(cfg, cat, fjClient, nil, a, nil, nil, nil)
}

func withSession(t *testing.T, p *Pages, user, org, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	cookie := httptest.NewRecorder()
	p.SetSession(cookie, user)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Cookie", cookie.Header().Get("Set-Cookie"))
	if org != "" {
		req.AddCookie(&http.Cookie{Name: orgCookie, Value: org})
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	switch path {
	case "/ui/me":
		p.Me(w, req)
	case "/ui/orgs":
		p.Orgs(w, req)
	default:
		t.Fatalf("path %s", path)
	}
	return w
}

func TestMeListsOrgs(t *testing.T) {
	p := orgPages(t)
	w := withSession(t, p, "alice", "", http.MethodGet, "/ui/me", "")
	if w.Code != http.StatusOK {
		t.Fatalf("me %d %s", w.Code, w.Body.String())
	}
	var got struct {
		Org  string `json:"org"`
		Orgs []Org  `json:"orgs"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Org != "saidc" || len(got.Orgs) != 2 || got.Orgs[1].Name != "acme" {
		t.Fatalf("%+v", got)
	}
}

func TestSwitchOrg(t *testing.T) {
	p := orgPages(t)
	w := withSession(t, p, "alice", "", http.MethodPut, "/ui/orgs", `{"org":"acme"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("switch %d %s", w.Code, w.Body.String())
	}
	var got struct {
		Org string `json:"org"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Org != "acme" {
		t.Fatalf("org=%s", got.Org)
	}
	found := false
	for _, c := range w.Result().Cookies() {
		if c.Name == orgCookie && c.Value == "acme" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing org cookie")
	}
}

func TestCreateOrg(t *testing.T) {
	p := orgPages(t)
	w := withSession(t, p, "alice", "", http.MethodPost, "/ui/orgs", `{"name":"Labs","full_name":"Labs"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("create %d %s", w.Code, w.Body.String())
	}
	var got Org
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "labs" {
		t.Fatalf("%+v", got)
	}
}

func TestMemberCannotCreateOrg(t *testing.T) {
	p := orgPages(t)
	w := withSession(t, p, "bob", "", http.MethodPost, "/ui/orgs", `{"name":"labs"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("create %d %s", w.Code, w.Body.String())
	}
}

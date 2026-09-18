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
	"acahti/internal/passwd"
)

func TestPasswordEnsureReusesStored(t *testing.T) {
	var patches int
	fj := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/admin/users":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"login": "gaowenrong", "login_name": "gaowenrong", "source_id": 0}})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/admin/users/gaowenrong":
			patches++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(fj.Close)
	store, err := passwd.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Set("gaowenrong", "kept"); err != nil {
		t.Fatal(err)
	}
	p := &Pages{
		Auth:      auth.New([]byte("test"), "alice"),
		Cat: catalog.New(config.Config{AdminUser: "alice"}, forgejo.New(fj.URL, "t"), nil, nil),
		Passwords: store,
	}
	got := callPassword(t, p, "alice", `{"username":"gaowenrong"}`)
	if got["password"] != "kept" {
		t.Fatalf("ensure %v", got)
	}
	if patches != 0 {
		t.Fatalf("patched existing password %d", patches)
	}
	got = callPassword(t, p, "alice", `{"username":"gaowenrong","password":"next"}`)
	if got["password"] != "next" {
		t.Fatalf("reset %v", got)
	}
	if patches != 1 {
		t.Fatalf("reset patches %d", patches)
	}
	if store.Get("gaowenrong") != "next" {
		t.Fatal("stored")
	}
}

func TestPasswordEnsureInitsWhenMissing(t *testing.T) {
	var body map[string]any
	fj := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/admin/users":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"login": "gaowenrong", "login_name": "gaowenrong", "source_id": 0}})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/admin/users/gaowenrong":
			_ = json.NewDecoder(r.Body).Decode(&body)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(fj.Close)
	store, err := passwd.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p := &Pages{
		Auth:      auth.New([]byte("test"), "alice"),
		Cat: catalog.New(config.Config{AdminUser: "alice"}, forgejo.New(fj.URL, "t"), nil, nil),
		Passwords: store,
	}
	got := callPassword(t, p, "alice", `{"username":"gaowenrong"}`)
	pw, _ := got["password"].(string)
	if len(pw) != 24 {
		t.Fatalf("init %v", got)
	}
	if body["password"] != pw {
		t.Fatalf("forgejo %v", body)
	}
	if store.Get("gaowenrong") != pw {
		t.Fatal("remember")
	}
	body = nil
	again := callPassword(t, p, "alice", `{"username":"gaowenrong"}`)
	if again["password"] != pw || body != nil {
		t.Fatalf("second ensure reset %v patch=%v", again, body)
	}
}

func TestPasswordEnsureInitsWhenVaultEmpty(t *testing.T) {
	var body map[string]any
	fj := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/admin/users":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"login": "zhuyu", "login_name": "zhuyu", "source_id": 0}})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/admin/users/zhuyu":
			_ = json.NewDecoder(r.Body).Decode(&body)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(fj.Close)
	store, err := passwd.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p := &Pages{
		Auth:      auth.New([]byte("test"), "alice"),
		Cat: catalog.New(config.Config{AdminUser: "alice"}, forgejo.New(fj.URL, "t"), nil, nil),
		Passwords: store,
		passwordSet: func(string) (bool, bool) {
			return true, true
		},
	}
	got := callPassword(t, p, "alice", `{"username":"zhuyu"}`)
	pw, _ := got["password"].(string)
	if len(pw) != 24 {
		t.Fatalf("init %v", got)
	}
	if body["password"] != pw {
		t.Fatalf("forgejo %v", body)
	}
	body = nil
	again := callPassword(t, p, "alice", `{"username":"zhuyu"}`)
	if again["password"] != pw || body != nil {
		t.Fatalf("second ensure reset %v patch=%v", again, body)
	}
}

func callPassword(t *testing.T, p *Pages, user, payload string) map[string]any {
	t.Helper()
	cookie := httptest.NewRecorder()
	p.SetSession(cookie, user)
	req := httptest.NewRequest(http.MethodPost, "/ui/password", strings.NewReader(payload))
	req.Header.Set("Cookie", cookie.Header().Get("Set-Cookie"))
	rr := httptest.NewRecorder()
	p.Password(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d %s", rr.Code, rr.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	return got
}

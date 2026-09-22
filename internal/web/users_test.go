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

func TestCreateUserGeneratesPasswordAndAuthor(t *testing.T) {
	var created map[string]any
	fj := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/admin/users":
			if err := json.NewDecoder(r.Body).Decode(&created); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"login":     created["username"],
				"full_name": created["full_name"],
			})
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/membership/"):
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(fj.Close)
	store, err := passwd.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{Org: "saidc", AdminUser: "alice", Domain: "acahti.s-aidc.com"}
	p := &Pages{
		Cfg:       cfg,
		Auth:      auth.New([]byte("test"), "alice"),
		Cat:       catalog.New(cfg, forgejo.New(fj.URL, "t"), nil, nil),
		Passwords: store,
	}
	cookie := httptest.NewRecorder()
	p.SetSession(cookie, "alice")
	req := httptest.NewRequest(http.MethodPost, "/ui/users", strings.NewReader(`{"username":"linshiyuee","author":"林诗月"}`))
	req.Header.Set("Cookie", cookie.Header().Get("Set-Cookie"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	p.Users(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d %s", rr.Code, rr.Body.String())
	}
	var out struct {
		User forgejo.User `json:"user"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.User.Login != "linshiyuee" || out.User.FullName != "林诗月" || len(out.User.Password) != 24 {
		t.Fatalf("%+v", out.User)
	}
	if created["full_name"] != "林诗月" || created["password"] != out.User.Password {
		t.Fatalf("forgejo %v", created)
	}
	if store.Get("linshiyuee") != out.User.Password {
		t.Fatal("remember")
	}
}

func TestCreateUserRequiresUsername(t *testing.T) {
	p := &Pages{Auth: auth.New([]byte("test"), "alice")}
	cookie := httptest.NewRecorder()
	p.SetSession(cookie, "alice")
	req := httptest.NewRequest(http.MethodPost, "/ui/users", strings.NewReader(`{"author":"林诗月"}`))
	req.Header.Set("Cookie", cookie.Header().Get("Set-Cookie"))
	rr := httptest.NewRecorder()
	p.Users(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status %d %s", rr.Code, rr.Body.String())
	}
}

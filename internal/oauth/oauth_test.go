package oauth

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"acahti/internal/auth"
)

func TestPKCERoundTrip(t *testing.T) {
	a := auth.New([]byte("secret"), "acahti")
	s, err := Open(t.TempDir(), "http://acahti.example", a)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	s.Register(rr, httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(`{"redirect_uris":["http://127.0.0.1:9/cb"],"client_name":"t"}`)))
	if rr.Code != http.StatusCreated {
		t.Fatalf("register %d %s", rr.Code, rr.Body.String())
	}
	var reg map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &reg); err != nil {
		t.Fatal(err)
	}
	cid, _ := reg["client_id"].(string)
	if cid == "" {
		t.Fatal("client_id")
	}

	verifier := "abcdefghijklmnopqrstuvwxyz012345"
	challenge := s256(verifier)
	approve := httptest.NewRequest(http.MethodPost, "/ui/oauth/approve", strings.NewReader(`{"client_id":"`+cid+`","redirect_uri":"http://127.0.0.1:9/cb","state":"s1","code_challenge":"`+challenge+`"}`))
	rr = httptest.NewRecorder()
	s.Approve(rr, approve, "alice")
	if rr.Code != http.StatusOK {
		t.Fatalf("approve %d %s", rr.Code, rr.Body.String())
	}
	var out struct {
		Redirect string `json:"redirect"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(out.Redirect)
	if err != nil {
		t.Fatal(err)
	}
	code := u.Query().Get("code")
	if code == "" || u.Query().Get("state") != "s1" {
		t.Fatalf("redirect %s", out.Redirect)
	}

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {"http://127.0.0.1:9/cb"},
		"code_verifier": {verifier},
		"client_id":     {cid},
	}
	tokReq := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	tokReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr = httptest.NewRecorder()
	s.Token(rr, tokReq)
	if rr.Code != http.StatusOK {
		t.Fatalf("token %d %s", rr.Code, rr.Body.String())
	}
	var tok map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &tok); err != nil {
		t.Fatal(err)
	}
	at, _ := tok["access_token"].(string)
	user, ok := a.Parse(at)
	if !ok || user != "alice" {
		t.Fatalf("access_token %q %v", at, ok)
	}

	rr = httptest.NewRecorder()
	Challenge(rr, "http://acahti.example/.well-known/oauth-protected-resource")
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("challenge %d", rr.Code)
	}
	b, _ := io.ReadAll(rr.Body)
	if !strings.Contains(rr.Header().Get("WWW-Authenticate"), "resource_metadata") || !strings.Contains(string(b), "unauthorized") {
		t.Fatalf("challenge headers %s body %s", rr.Header().Get("WWW-Authenticate"), b)
	}
}

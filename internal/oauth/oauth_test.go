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
	a := auth.New([]byte("secret"), "acahti_bot")
	s, err := Open(t.TempDir(), "http://acahti.example", "", a)
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
	if int(tok["expires_in"].(float64)) != AdvertisedExpiresIn {
		t.Fatalf("expires_in %v", tok["expires_in"])
	}
	if int(tok["refresh_token_expires_in"].(float64)) != AdvertisedExpiresIn {
		t.Fatalf("refresh_token_expires_in %v", tok["refresh_token_expires_in"])
	}
	rt, _ := tok["refresh_token"].(string)
	if rt == "" {
		t.Fatal("refresh_token")
	}

	form = url.Values{"grant_type": {"refresh_token"}, "refresh_token": {rt}}
	tokReq = httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	tokReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr = httptest.NewRecorder()
	s.Token(rr, tokReq)
	if rr.Code != http.StatusOK {
		t.Fatalf("refresh %d %s", rr.Code, rr.Body.String())
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &tok); err != nil {
		t.Fatal(err)
	}
	if tok["refresh_token"] != rt {
		t.Fatalf("refresh rotated: %v", tok["refresh_token"])
	}
	at, _ = tok["access_token"].(string)
	if user, ok = a.Parse(at); !ok || user != "alice" {
		t.Fatalf("refreshed access_token %q %v", at, ok)
	}

	jsonBody := `{"grant_type":"refresh_token","refresh_token":"` + rt + `"}`
	tokReq = httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(jsonBody))
	tokReq.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	s.Token(rr, tokReq)
	if rr.Code != http.StatusOK {
		t.Fatalf("json refresh %d %s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	s.Metadata(rr, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"logo_uri":"http://acahti.example/acahti.png"`) {
		t.Fatalf("as metadata %d %s", rr.Code, rr.Body.String())
	}
	rr = httptest.NewRecorder()
	s.Resource(rr, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource/mcp", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"logo_uri":"http://acahti.example/acahti.png"`) {
		t.Fatalf("resource metadata %d %s", rr.Code, rr.Body.String())
	}

	meta := ResourceMetadataURL("http://acahti.example")
	rr = httptest.NewRecorder()
	Challenge(rr, meta, false)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("challenge %d", rr.Code)
	}
	b, _ := io.ReadAll(rr.Body)
	if !strings.Contains(rr.Header().Get("WWW-Authenticate"), meta) || strings.Contains(rr.Header().Get("WWW-Authenticate"), "invalid_token") || !strings.Contains(string(b), "unauthorized") {
		t.Fatalf("challenge headers %s body %s", rr.Header().Get("WWW-Authenticate"), b)
	}
	rr = httptest.NewRecorder()
	Challenge(rr, meta, true)
	b, _ = io.ReadAll(rr.Body)
	if !strings.Contains(rr.Header().Get("WWW-Authenticate"), `error="invalid_token"`) || !strings.Contains(string(b), "invalid_token") {
		t.Fatalf("invalid challenge headers %s body %s", rr.Header().Get("WWW-Authenticate"), b)
	}
}

func TestHostAwareMetadata(t *testing.T) {
	a := auth.New([]byte("secret"), "acahti_bot")
	s, err := Open(t.TempDir(), "https://acahti.s-aidc.com", "acahti.s-aidc.com", a)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource/mcp", nil)
	req.Host = "acahti.saidc.ai"
	rr := httptest.NewRecorder()
	s.Resource(rr, req)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"resource":"https://acahti.saidc.ai/mcp"`) {
		t.Fatalf("resource %d %s", rr.Code, rr.Body.String())
	}
}

func TestPublicRoot(t *testing.T) {
	root := "https://acahti.s-aidc.com"
	if PublicRoot(root, "acahti.s-aidc.com", "acahti.saidc.ai") != "https://acahti.saidc.ai" {
		t.Fatalf("alias %s", PublicRoot(root, "acahti.s-aidc.com", "acahti.saidc.ai"))
	}
	if PublicRoot(root, "acahti.s-aidc.com", "evil.example") != root {
		t.Fatalf("unknown host %s", PublicRoot(root, "acahti.s-aidc.com", "evil.example"))
	}
	if PublicRoot("http://acahti.example", "", "acahti.example:8080") != "http://acahti.example" {
		t.Fatalf("port %s", PublicRoot("http://acahti.example", "", "acahti.example:8080"))
	}
}


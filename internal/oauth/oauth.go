package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"acahti/internal/auth"
)

type client struct {
	ID           string   `json:"client_id"`
	Name         string   `json:"client_name"`
	RedirectURIs []string `json:"redirect_uris"`
}

type codeRec struct {
	User      string
	ClientID  string
	Redirect  string
	Challenge string
	Expires   int64
}

type refreshRec struct {
	User     string `json:"user"`
	ClientID string `json:"client_id"`
	Expires  int64  `json:"exp"`
}

type disk struct {
	Clients []client              `json:"clients"`
	Refresh map[string]refreshRec `json:"refresh"`
}

type Server struct {
	Auth    *auth.Service
	RootURL string
	path    string
	mu      sync.Mutex
	clients map[string]client
	codes   map[string]codeRec
	refresh map[string]refreshRec
}

func Open(dir, rootURL string, a *auth.Service) (*Server, error) {
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Server{
		Auth:    a,
		RootURL: strings.TrimRight(rootURL, "/"),
		path:    filepath.Join(dir, "oauth.json"),
		clients: map[string]client{},
		codes:   map[string]codeRec{},
		refresh: map[string]refreshRec{},
	}
	b, err := os.ReadFile(s.path)
	if err == nil {
		var d disk
		if json.Unmarshal(b, &d) == nil {
			for _, c := range d.Clients {
				s.clients[c.ID] = c
			}
			if d.Refresh != nil {
				s.refresh = d.Refresh
			}
		}
	}
	return s, nil
}

func (s *Server) persist() {
	d := disk{Refresh: s.refresh}
	for _, c := range s.clients {
		d.Clients = append(d.Clients, c)
	}
	b, _ := json.MarshalIndent(d, "", "  ")
	_ = os.WriteFile(s.path, b, 0o600)
}

func nonce(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func loopback(u string) bool {
	p, err := url.Parse(u)
	if err != nil {
		return false
	}
	h := strings.ToLower(p.Hostname())
	return h == "127.0.0.1" || h == "localhost" || h == "::1" ||
		strings.HasSuffix(h, ".cursor.com") || h == "cursor.com" ||
		strings.HasSuffix(h, ".anthropic.com")
}

func (s *Server) allowRedirect(c client, uri string) bool {
	if loopback(uri) {
		return true
	}
	for _, r := range c.RedirectURIs {
		if r == uri {
			return true
		}
	}
	return false
}

func (s *Server) Metadata(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                s.RootURL,
		"authorization_endpoint":                s.RootURL + "/oauth/authorize",
		"token_endpoint":                        s.RootURL + "/oauth/token",
		"registration_endpoint":                 s.RootURL + "/oauth/register",
		"code_challenge_methods_supported":      []string{"S256"},
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"scopes_supported":                      []string{"mcp"},
	})
}

func (s *Server) Resource(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"resource":                 s.RootURL + "/mcp",
		"authorization_servers":    []string{s.RootURL},
		"bearer_methods_supported": []string{"header"},
	})
}

func (s *Server) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		RedirectURIs []string `json:"redirect_uris"`
		ClientName   string   `json:"client_name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	id := nonce(16)
	c := client{ID: id, Name: strings.TrimSpace(body.ClientName), RedirectURIs: body.RedirectURIs}
	s.mu.Lock()
	s.clients[id] = c
	s.persist()
	s.mu.Unlock()
	writeJSON(w, http.StatusCreated, map[string]any{
		"client_id":                  id,
		"client_name":                c.Name,
		"redirect_uris":              c.RedirectURIs,
		"token_endpoint_auth_method": "none",
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
	})
}

func (s *Server) Authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("response_type") != "code" {
		http.Error(w, "response_type", http.StatusBadRequest)
		return
	}
	user := s.Auth.CookieUser(r)
	if user == "" {
		next := "/oauth/authorize?" + r.URL.RawQuery
		http.Redirect(w, r, "/login?next="+url.QueryEscape(next), http.StatusFound)
		return
	}
	http.Redirect(w, r, "/oauth/consent?"+r.URL.RawQuery, http.StatusFound)
}

func (s *Server) Approve(w http.ResponseWriter, r *http.Request, user string) {
	var body struct {
		ClientID      string `json:"client_id"`
		RedirectURI   string `json:"redirect_uri"`
		State         string `json:"state"`
		CodeChallenge string `json:"code_challenge"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	s.mu.Lock()
	c, ok := s.clients[body.ClientID]
	s.mu.Unlock()
	if !ok || !s.allowRedirect(c, body.RedirectURI) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid client"})
		return
	}
	code := nonce(20)
	s.mu.Lock()
	s.codes[code] = codeRec{
		User: user, ClientID: body.ClientID, Redirect: body.RedirectURI,
		Challenge: body.CodeChallenge, Expires: time.Now().Add(5 * time.Minute).Unix(),
	}
	s.mu.Unlock()
	u, _ := url.Parse(body.RedirectURI)
	q := u.Query()
	q.Set("code", code)
	if body.State != "" {
		q.Set("state", body.State)
	}
	u.RawQuery = q.Encode()
	writeJSON(w, http.StatusOK, map[string]string{"redirect": u.String()})
}

func (s *Server) Token(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	_ = r.ParseForm()
	switch r.Form.Get("grant_type") {
	case "authorization_code":
		s.tokenCode(w, r)
	case "refresh_token":
		s.tokenRefresh(w, r)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_grant_type"})
	}
}

func (s *Server) tokenCode(w http.ResponseWriter, r *http.Request) {
	code := r.Form.Get("code")
	redir := r.Form.Get("redirect_uri")
	verifier := r.Form.Get("code_verifier")
	s.mu.Lock()
	rec, ok := s.codes[code]
	if ok {
		delete(s.codes, code)
	}
	s.mu.Unlock()
	if !ok || rec.Expires < time.Now().Unix() || rec.Redirect != redir {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
		return
	}
	if rec.Challenge != "" && s256(verifier) != rec.Challenge {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
		return
	}
	s.issue(w, rec.User, rec.ClientID)
}

func (s *Server) tokenRefresh(w http.ResponseWriter, r *http.Request) {
	rt := r.Form.Get("refresh_token")
	s.mu.Lock()
	rec, ok := s.refresh[rt]
	s.mu.Unlock()
	if !ok || rec.Expires < time.Now().Unix() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
		return
	}
	s.issue(w, rec.User, rec.ClientID)
}

func (s *Server) issue(w http.ResponseWriter, user, clientID string) {
	at := s.Auth.Issue(user)
	rt := nonce(24)
	s.mu.Lock()
	s.refresh[rt] = refreshRec{User: user, ClientID: clientID, Expires: time.Now().Add(90 * 24 * time.Hour).Unix()}
	s.persist()
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  at,
		"token_type":    "Bearer",
		"expires_in":    int(s.Auth.TTL.Seconds()),
		"refresh_token": rt,
		"scope":         "mcp",
	})
}

func s256(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func Challenge(w http.ResponseWriter, resourceMeta string) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="acahti", resource_metadata="`+resourceMeta+`"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = io.WriteString(w, `{"error":"unauthorized"}`+"\n")
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

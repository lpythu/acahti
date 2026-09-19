package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"acahti/internal/auth"
	"acahti/internal/brand"
	"acahti/internal/identity"
)

// Cursor MCP often fails to apply refresh tokens and re-prompts instead.
// Access tokens are therefore long-lived; refresh tokens are not rotated.
//
// AdvertisedExpiresIn is the wire expires_in / refresh_token_expires_in.
// Cursor has treated those RFC 6749 second counts as milliseconds, so a
// real 90d access TTL became ~2h and a 400d refresh TTL became ~10h.
// MaxInt32 is ~68y as seconds and ~25d as milliseconds, and does not
// overflow int32. Server-side HMAC / refresh records still use AccessTTL
// and RefreshTTL.
const (
	AccessTTL           = 90 * 24 * time.Hour
	RefreshTTL          = 400 * 24 * time.Hour
	AdvertisedExpiresIn = math.MaxInt32
)

func ResourceMetadataURL(root string) string {
	return strings.TrimRight(root, "/") + "/.well-known/oauth-protected-resource/mcp"
}

func PublicRoot(rootURL, domain, host string) string {
	root := strings.TrimRight(strings.TrimSpace(rootURL), "/")
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return root
	}
	for _, allowed := range identity.Hosts(rootURL, domain) {
		if host == allowed {
			scheme := "https"
			if u, err := url.Parse(root); err == nil && u.Scheme != "" {
				scheme = u.Scheme
			}
			return scheme + "://" + host
		}
	}
	return root
}

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
	Domain  string
	path    string
	mu      sync.Mutex
	clients map[string]client
	codes   map[string]codeRec
	refresh map[string]refreshRec
}

func Open(dir, rootURL, domain string, a *auth.Service) (*Server, error) {
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Server{
		Auth:    a,
		RootURL: strings.TrimRight(rootURL, "/"),
		Domain:  strings.TrimSpace(domain),
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
	now := time.Now().Unix()
	for k, rec := range s.refresh {
		if rec.Expires < now {
			delete(s.refresh, k)
		}
	}
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

func (s *Server) publicRoot(r *http.Request) string {
	host := ""
	if r != nil {
		host = r.Host
	}
	return PublicRoot(s.RootURL, s.Domain, host)
}

func (s *Server) Metadata(w http.ResponseWriter, r *http.Request) {
	root := s.publicRoot(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                root,
		"authorization_endpoint":                root + "/oauth/authorize",
		"token_endpoint":                        root + "/oauth/token",
		"registration_endpoint":                 root + "/oauth/register",
		"code_challenge_methods_supported":      []string{"S256"},
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"scopes_supported":                      []string{"mcp"},
		"logo_uri":                              brand.PNGURL(root),
	})
}

func (s *Server) Resource(w http.ResponseWriter, r *http.Request) {
	root := s.publicRoot(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"resource":                 root + "/mcp",
		"authorization_servers":    []string{root},
		"bearer_methods_supported": []string{"header"},
		"resource_name":            "Acahti",
		"logo_uri":                 brand.PNGURL(root),
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

func parseTokenForm(r *http.Request) {
	ct := r.Header.Get("Content-Type")
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	if strings.EqualFold(ct, "application/json") {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		r.Form = url.Values{}
		for k, v := range body {
			if s, ok := v.(string); ok {
				r.Form.Set(k, s)
			}
		}
		return
	}
	_ = r.ParseForm()
}

func (s *Server) Token(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	parseTokenForm(r)
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
	s.issue(w, rec.User, rec.ClientID, "")
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
	s.issue(w, rec.User, rec.ClientID, rt)
}

func (s *Server) issue(w http.ResponseWriter, user, clientID, rt string) {
	at := s.Auth.IssueFor(user, AccessTTL)
	if rt == "" {
		rt = nonce(24)
	}
	s.mu.Lock()
	s.refresh[rt] = refreshRec{User: user, ClientID: clientID, Expires: time.Now().Add(RefreshTTL).Unix()}
	s.persist()
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":             at,
		"token_type":               "Bearer",
		"expires_in":               AdvertisedExpiresIn,
		"refresh_token":            rt,
		"refresh_token_expires_in": AdvertisedExpiresIn,
		"scope":                    "mcp",
	})
}

func s256(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func Challenge(w http.ResponseWriter, resourceMeta string, invalidToken bool) {
	authn := `Bearer realm="acahti", resource_metadata="` + resourceMeta + `"`
	if invalidToken {
		authn = `Bearer realm="acahti", error="invalid_token", resource_metadata="` + resourceMeta + `"`
	}
	w.Header().Set("WWW-Authenticate", authn)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	if invalidToken {
		_, _ = io.WriteString(w, `{"error":"invalid_token"}`+"\n")
		return
	}
	_, _ = io.WriteString(w, `{"error":"unauthorized"}`+"\n")
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

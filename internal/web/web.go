package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"acahti/internal/catalog"
	"acahti/internal/config"
	"acahti/internal/events"
	"acahti/internal/forgejo"
	"acahti/internal/woodpecker"
)

type Pages struct {
	Cfg    config.Config
	Cat    *catalog.Catalog
	FJ     *forgejo.Client
	WP     *woodpecker.Client
	Hub    *events.Hub
	Secret []byte
	files  fs.FS
}

func New(cfg config.Config, fj *forgejo.Client, wp *woodpecker.Client, secret []byte, hub *events.Hub) *Pages {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		sub = distFS
	}
	return &Pages{
		Cfg:    cfg,
		Cat:    catalog.New(cfg, fj, wp),
		FJ:     fj,
		WP:     wp,
		Hub:    hub,
		Secret: secret,
		files:  sub,
	}
}

func (p *Pages) SessionUser(r *http.Request) (string, bool) {
	c, err := r.Cookie("acahti")
	if err != nil || c.Value == "" {
		return "", false
	}
	raw, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil {
		return "", false
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 {
		return "", false
	}
	user, exp, sig := parts[0], parts[1], parts[2]
	mac := hmac.New(sha256.New, p.Secret)
	mac.Write([]byte(user + "|" + exp))
	if !hmac.Equal([]byte(sig), []byte(hex.EncodeToString(mac.Sum(nil)))) {
		return "", false
	}
	until, _ := strconv.ParseInt(exp, 10, 64)
	if time.Now().Unix() > until {
		return "", false
	}
	return user, user == p.Cfg.AdminUser
}

func (p *Pages) SetSession(w http.ResponseWriter, user string) {
	exp := strconv.FormatInt(time.Now().Add(30*24*time.Hour).Unix(), 10)
	mac := hmac.New(sha256.New, p.Secret)
	mac.Write([]byte(user + "|" + exp))
	val := base64.RawURLEncoding.EncodeToString([]byte(user + "|" + exp + "|" + hex.EncodeToString(mac.Sum(nil))))
	http.SetCookie(w, &http.Cookie{Name: "acahti", Value: val, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 30 * 24 * 3600})
}

func (p *Pages) ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "acahti", Path: "/", MaxAge: -1})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (p *Pages) requireJSON(w http.ResponseWriter, r *http.Request) (string, bool, bool) {
	user, admin := p.SessionUser(r)
	if user == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return "", false, false
	}
	return user, admin, true
}

func (p *Pages) Me(w http.ResponseWriter, r *http.Request) {
	user, admin, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "admin": admin, "root_url": p.Cfg.RootURL, "org": p.Cfg.Org})
}

func (p *Pages) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if _, err := p.FJ.BasicUser(strings.TrimSpace(body.Username), body.Password); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid username or password"})
		return
	}
	p.SetSession(w, strings.TrimSpace(body.Username))
	writeJSON(w, http.StatusOK, map[string]any{
		"user":     strings.TrimSpace(body.Username),
		"admin":    strings.TrimSpace(body.Username) == p.Cfg.AdminUser,
		"root_url": p.Cfg.RootURL,
		"org":      p.Cfg.Org,
	})
}

func (p *Pages) Logout(w http.ResponseWriter, r *http.Request) {
	p.ClearSession(w)
	if r.Header.Get("Accept") == "application/json" || strings.HasPrefix(r.URL.Path, "/ui/") {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (p *Pages) Board(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := p.requireJSON(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, p.Cat.Inbox())
}

func (p *Pages) Users(w http.ResponseWriter, r *http.Request) {
	user, admin, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	if !admin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin only"})
		return
	}
	_ = user
	if r.Method == http.MethodPost {
		var body struct {
			Username string `json:"username"`
			Email    string `json:"email"`
			Password string `json:"password"`
			Admin    bool   `json:"admin"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		login := strings.TrimSpace(body.Username)
		pw := body.Password
		if login == "" || pw == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password required"})
			return
		}
		email := strings.TrimSpace(body.Email)
		if email == "" {
			email = login + "@users.local"
		}
		u, err := p.FJ.CreateUser(login, email, pw, body.Admin)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		}
		_ = p.FJ.AddOrgMember(p.Cfg.Org, u.Login)
		writeJSON(w, http.StatusOK, map[string]any{"user": u})
		return
	}
	users, _ := p.FJ.ListUsers()
	if users == nil {
		users = []forgejo.User{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": users})
}

func (p *Pages) Keys(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodPost {
		var body struct {
			Title string `json:"title"`
			Key   string `json:"key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		if err := p.FJ.AddKey(user, body.Title, strings.TrimSpace(body.Key)); err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	keys, _ := p.FJ.ListKeys(user)
	if keys == nil {
		keys = []forgejo.PublicKey{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"keys": keys})
}

func (p *Pages) Token(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	tok, err := p.FJ.CreateToken(user, "mcp-"+strconv.FormatInt(time.Now().Unix(), 10))
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": tok})
}

func (p *Pages) Password(w http.ResponseWriter, r *http.Request) {
	user, admin, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	target := strings.TrimSpace(body.Username)
	if target == "" {
		target = user
	}
	if target != user && !admin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin only"})
		return
	}
	if body.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password required"})
		return
	}
	if err := p.FJ.SetPassword(target, body.Password); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (p *Pages) Approve(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := p.requireJSON(w, r); !ok {
		return
	}
	var body struct {
		Repo   string `json:"repo"`
		Number int64  `json:"number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if err := p.WP.Approve(body.Repo, body.Number); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (p *Pages) Stack(w http.ResponseWriter, r *http.Request) {
	_, admin, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	if !admin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin only"})
		return
	}
	agents := []woodpecker.Agent{}
	if p.WP.Ready() {
		if a, err := p.WP.Agents(); err == nil && a != nil {
			agents = a
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"version":           p.Cfg.Version,
		"forgejo":           p.Cfg.ForgejoVersion,
		"woodpecker":        p.Cfg.WoodpeckerVersion,
		"postgres":          p.Cfg.PostgresVersion,
		"agents":            agents,
		"upgrade_hint":      "Ask a coding agent to follow skills/acahti-install: replace the tree, bash scripts/up.sh, then ROLE=both agent.sh on buildof.",
		"pinned_woodpecker": p.Cfg.WoodpeckerVersion,
	})
}

func (p *Pages) Files() fs.FS {
	return p.files
}

func (p *Pages) Index(w http.ResponseWriter, r *http.Request) {
	b, err := fs.ReadFile(p.files, "index.html")
	if err != nil {
		http.Error(w, "ui not built", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(b)
}

func (p *Pages) PublicFile(w http.ResponseWriter, r *http.Request) {
	http.FileServer(http.FS(p.files)).ServeHTTP(w, r)
}

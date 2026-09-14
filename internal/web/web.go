package web

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"acahti/internal/auth"
	"acahti/internal/catalog"
	"acahti/internal/config"
	"acahti/internal/events"
	"acahti/internal/forgejo"
	"acahti/internal/identity"
	"acahti/internal/invite"
	"acahti/internal/oauth"
	"acahti/internal/woodpecker"
	"acahti/skills"
)

type Pages struct {
	Cfg         config.Config
	Cat         *catalog.Catalog
	FJ          *forgejo.Client
	WP          *woodpecker.Client
	Hub         *events.Hub
	Auth        *auth.Service
	InviteStore *invite.Store
	OAuth       *oauth.Server
	files       fs.FS
}

func New(cfg config.Config, fj *forgejo.Client, wp *woodpecker.Client, a *auth.Service, inv *invite.Store, oa *oauth.Server, hub *events.Hub) *Pages {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		sub = distFS
	}
	return &Pages{
		Cfg: cfg, Cat: catalog.New(cfg, fj, wp), FJ: fj, WP: wp, Hub: hub,
		Auth: a, InviteStore: inv, OAuth: oa, files: sub,
	}
}

func (p *Pages) SessionUser(r *http.Request) (string, bool) {
	user := p.Auth.CookieUser(r)
	return user, p.Auth.IsAdmin(user)
}

func (p *Pages) SetSession(w http.ResponseWriter, user string) {
	p.Auth.SetCookie(w, user)
}

func (p *Pages) ClearSession(w http.ResponseWriter) {
	p.Auth.ClearCookie(w)
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
	writeJSON(w, http.StatusOK, identity.Session(user, admin, p.Cfg.RootURL, p.Cfg.Domain, p.Cfg.Org))
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
	login := strings.TrimSpace(body.Username)
	p.SetSession(w, login)
	writeJSON(w, http.StatusOK, identity.Session(login, login == p.Cfg.AdminUser, p.Cfg.RootURL, p.Cfg.Domain, p.Cfg.Org))
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
		email := identity.Email(login, identity.Domain(p.Cfg.RootURL, p.Cfg.Domain))
		if email == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "DOMAIN or ROOT_URL required"})
			return
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

func (p *Pages) Skill(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	_, _ = w.Write([]byte(skills.Acahti(p.Cfg.RootURL, p.Cfg.Org, identity.Domain(p.Cfg.RootURL, p.Cfg.Domain))))
}

func (p *Pages) Public(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"root_url": p.Cfg.RootURL,
		"org":      p.Cfg.Org,
		"skill":    p.Cfg.RootURL + "/skill.md",
		"join":     p.Cfg.RootURL + "/join",
		"mcp":      p.Cfg.RootURL + "/mcp",
	})
}

func (p *Pages) Join(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	login := strings.TrimSpace(body.Username)
	if login == "" || body.Password == "" || strings.TrimSpace(body.Code) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username, password, and invite required"})
		return
	}
	if p.InviteStore == nil || !p.InviteStore.Valid(strings.TrimSpace(body.Code)) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid invite"})
		return
	}
	email := identity.Email(login, identity.Domain(p.Cfg.RootURL, p.Cfg.Domain))
	u, err := p.FJ.CreateUser(login, email, body.Password, false)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	_ = p.FJ.AddOrgMember(p.Cfg.Org, u.Login)
	p.SetSession(w, u.Login)
	writeJSON(w, http.StatusOK, identity.Session(u.Login, false, p.Cfg.RootURL, p.Cfg.Domain, p.Cfg.Org))
}

func (p *Pages) Invites(w http.ResponseWriter, r *http.Request) {
	_, admin, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	if !admin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin only"})
		return
	}
	if p.InviteStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "invites unavailable"})
		return
	}
	switch r.Method {
	case http.MethodPost:
		c, err := p.InviteStore.Create()
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"code": c.Code, "url": p.Cfg.RootURL + "/join?code=" + c.Code})
	case http.MethodDelete:
		_ = p.InviteStore.Delete(r.PathValue("code"))
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		writeJSON(w, http.StatusOK, map[string]any{"invites": p.InviteStore.List(), "join": p.Cfg.RootURL + "/join"})
	}
}

func (p *Pages) OAuthApprove(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	if p.OAuth == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "oauth unavailable"})
		return
	}
	p.OAuth.Approve(w, r, user)
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

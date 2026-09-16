package web

import (
	"context"
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
	"acahti/internal/page"
	"acahti/internal/passwd"
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
	Passwords   *passwd.Store
	files       fs.FS
	passwordSet func(login string) (has, ok bool)
}

func New(cfg config.Config, cat *catalog.Catalog, fj *forgejo.Client, wp *woodpecker.Client, a *auth.Service, inv *invite.Store, oa *oauth.Server, hub *events.Hub) *Pages {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		sub = distFS
	}
	return &Pages{
		Cfg: cfg, Cat: cat, FJ: fj, WP: wp, Hub: hub,
		Auth: a, InviteStore: inv, OAuth: oa, files: sub,
	}
}

func (p *Pages) isAdmin(user string) bool {
	if user == "" {
		return false
	}
	if p.Cat != nil {
		return p.Cat.IsOrgAdmin(user)
	}
	return p.Auth.IsAdmin(user)
}

func (p *Pages) SessionUser(r *http.Request) (string, bool) {
	user := p.Auth.CookieUser(r)
	return user, p.isAdmin(user)
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

func (p *Pages) requireAdmin(w http.ResponseWriter, r *http.Request) (string, bool) {
	user, admin, ok := p.requireJSON(w, r)
	if !ok {
		return "", false
	}
	if !admin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin only"})
		return "", false
	}
	return user, true
}

func (p *Pages) Me(w http.ResponseWriter, r *http.Request) {
	user, admin, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	author := ""
	if p.FJ != nil {
		if u, err := p.FJ.UserSudo(user); err == nil {
			author = u.FullName
		}
	}
	writeJSON(w, http.StatusOK, identity.Session(user, author, admin, p.Cfg.RootURL, p.Cfg.Domain, p.Cfg.Org))
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
	u, err := p.FJ.BasicUser(strings.TrimSpace(body.Username), body.Password)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid username or password"})
		return
	}
	p.SetSession(w, u.Login)
	writeJSON(w, http.StatusOK, identity.Session(u.Login, u.FullName, p.isAdmin(u.Login), p.Cfg.RootURL, p.Cfg.Domain, p.Cfg.Org))
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
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	q := page.Parse(r)
	switch r.URL.Query().Get("section") {
	case "prs":
		out, err := p.Cat.BoardPRs(user, q)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, out)
	default:
		out, err := p.Cat.BoardPipes(user, q)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func (p *Pages) Users(w http.ResponseWriter, r *http.Request) {
	if _, ok := p.requireAdmin(w, r); !ok {
		return
	}
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
		u.FullName = identity.Name(u.Login, u.FullName)
		_ = p.rememberPassword(u.Login, pw)
		u.Password = pw
		u.HasPassword = true
		writeJSON(w, http.StatusOK, map[string]any{"user": u})
		return
	}
	out, err := p.FJ.ListUsers(page.Parse(r))
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	byLogin, err := p.Cat.TeamsByLogin()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	logins := make([]string, 0, len(out.Items))
	for _, u := range out.Items {
		logins = append(logins, u.Login)
	}
	byRepo, err := p.Cat.UserRepoAccessMany(logins)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	flags, flagErr := forgejo.UserPasswordFlags(r.Context(), p.Cfg.DatabaseURL)
	for i := range out.Items {
		out.Items[i].FullName = identity.Name(out.Items[i].Login, out.Items[i].FullName)
		names := byLogin[out.Items[i].Login]
		if names == nil {
			names = []string{}
		}
		out.Items[i].Teams = names
		repos := byRepo[out.Items[i].Login]
		if repos == nil {
			repos = []catalog.RepoPerm{}
		}
		dst := make([]forgejo.UserRepo, 0, len(repos))
		for _, row := range repos {
			dst = append(dst, forgejo.UserRepo{
				Repo: row.Repo, Permission: row.Permission, Team: row.Team, Direct: row.Direct,
			})
		}
		out.Items[i].Repos = dst
		pw := p.Passwords.Get(out.Items[i].Login)
		out.Items[i].Password = pw
		has := pw != ""
		if flagErr == nil {
			has = has || flags[out.Items[i].Login]
		}
		out.Items[i].HasPassword = has
	}
	writeJSON(w, http.StatusOK, out)
}

func (p *Pages) PatchUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := p.requireAdmin(w, r); !ok {
		return
	}
	login := strings.TrimSpace(r.PathValue("login"))
	if login == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username required"})
		return
	}
	var body struct {
		GitName string `json:"git_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	name := identity.Name(login, body.GitName)
	if err := p.FJ.EditUser(login, map[string]any{"full_name": name}); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	p.Cat.ForgetAuthors()
	writeJSON(w, http.StatusOK, identity.View(login, name, p.Cfg.RootURL, p.Cfg.Domain, p.Cfg.Org))
}

func (p *Pages) UserRepos(w http.ResponseWriter, r *http.Request) {
	if _, ok := p.requireAdmin(w, r); !ok {
		return
	}
	login := strings.TrimSpace(r.PathValue("login"))
	if login == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username required"})
		return
	}
	out, err := p.Cat.UserRepoAccess(login)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"repos": out})
}

func (p *Pages) RepoTeamGrant(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	var body struct {
		Granted bool `json:"granted"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := p.Cat.SetRepoTeamGrant(user, r.PathValue("owner"), r.PathValue("name"), body.Granted); err != nil {
		writeCatErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
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
	code := strings.TrimSpace(body.Code)
	if login == "" || body.Password == "" || code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username, password, and invite required"})
		return
	}
	if p.InviteStore == nil || !p.InviteStore.Valid(code) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid invite"})
		return
	}
	email := identity.Email(login, identity.Domain(p.Cfg.RootURL, p.Cfg.Domain))
	u, err := p.FJ.CreateUser(login, email, body.Password, false)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	_ = p.InviteStore.Delete(code)
	_ = p.FJ.AddOrgMember(p.Cfg.Org, u.Login)
	_ = p.rememberPassword(u.Login, body.Password)
	p.SetSession(w, u.Login)
	writeJSON(w, http.StatusOK, identity.Session(u.Login, u.FullName, false, p.Cfg.RootURL, p.Cfg.Domain, p.Cfg.Org))
}

func (p *Pages) Invites(w http.ResponseWriter, r *http.Request) {
	if _, ok := p.requireAdmin(w, r); !ok {
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
		if !admin {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password required"})
			return
		}
		pw, err := p.ensurePassword(target)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		}
		has := pw != ""
		if !has {
			if set, ok := p.forgejoPasswordSet(target); ok {
				has = set
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "password": pw, "has_password": has})
		return
	}
	if err := p.FJ.SetPassword(target, body.Password); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	if err := p.rememberPassword(target, body.Password); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "password": body.Password})
}

func (p *Pages) rememberPassword(login, password string) error {
	if p.Passwords == nil {
		return nil
	}
	return p.Passwords.Set(login, password)
}

func (p *Pages) forgejoPasswordSet(login string) (has, ok bool) {
	if p.passwordSet != nil {
		return p.passwordSet(login)
	}
	flags, err := forgejo.UserPasswordFlags(context.Background(), p.Cfg.DatabaseURL)
	if err != nil {
		return false, false
	}
	return flags[login], true
}

func (p *Pages) ensurePassword(login string) (string, error) {
	if pw := p.Passwords.Get(login); pw != "" {
		return pw, nil
	}
	return p.initPassword(login)
}

func (p *Pages) initPassword(login string) (string, error) {
	pw, err := passwd.Random()
	if err != nil {
		return "", err
	}
	if err := p.FJ.SetPassword(login, pw); err != nil {
		return "", err
	}
	if err := p.rememberPassword(login, pw); err != nil {
		return "", err
	}
	return pw, nil
}

func (p *Pages) Stack(w http.ResponseWriter, r *http.Request) {
	if _, ok := p.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"version":      p.Cfg.Version,
		"forgejo":      p.Cfg.ForgejoVersion,
		"ci":           p.Cfg.WoodpeckerVersion,
		"postgres":     p.Cfg.PostgresVersion,
		"upgrade_hint": "Ask a coding agent to follow skills/acahti-install: replace the tree, bash scripts/up.sh, then ROLE=both agent.sh on the Runner host (ACAHTI_BUILD).",
	})
}

func (p *Pages) Agents(w http.ResponseWriter, r *http.Request) {
	if _, ok := p.requireAdmin(w, r); !ok {
		return
	}
	out, err := p.Cat.ListAgents(page.Parse(r))
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, out)
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

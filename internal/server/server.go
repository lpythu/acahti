package server

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"acahti/internal/api"
	"acahti/internal/auth"
	"acahti/internal/config"
	"acahti/internal/events"
	"acahti/internal/forgejo"
	"acahti/internal/invite"
	"acahti/internal/mcp"
	"acahti/internal/oauth"
	"acahti/internal/web"
	"acahti/internal/woodpecker"
)

func New(cfg config.Config, fj *forgejo.Client, wp *woodpecker.Client, hub *events.Hub) http.Handler {
	a := auth.New([]byte(cfg.SessionSecret), cfg.AdminUser)
	inv, _ := invite.Open(cfg.DataDir)
	oa, _ := oauth.Open(cfg.DataDir, cfg.RootURL, a)
	pages := web.New(cfg, fj, wp, a, inv, oa, hub)
	mc := mcp.New(cfg, a, fj, wp)
	rest := &api.API{Cfg: cfg, Auth: a, FJ: fj, WP: wp, Hub: hub, MCP: mc}
	fjProxy := reverse(cfg.ForgejoURL)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("GET /skill.md", pages.Skill)
	mux.HandleFunc("GET /.well-known/oauth-authorization-server", oa.Metadata)
	mux.HandleFunc("GET /.well-known/oauth-protected-resource", oa.Resource)
	mux.HandleFunc("POST /oauth/register", oa.Register)
	mux.HandleFunc("GET /oauth/authorize", oa.Authorize)
	mux.HandleFunc("POST /oauth/token", oa.Token)
	mux.Handle("GET /mcp", mc)
	mux.Handle("POST /mcp", mc)
	mux.Handle("/acahti/v1/", rest)
	mux.HandleFunc("GET /ui/public", pages.Public)
	mux.HandleFunc("GET /ui/me", pages.Me)
	mux.HandleFunc("POST /ui/login", pages.Login)
	mux.HandleFunc("POST /ui/join", pages.Join)
	mux.HandleFunc("POST /ui/logout", pages.Logout)
	mux.HandleFunc("POST /ui/oauth/approve", pages.OAuthApprove)
	mux.HandleFunc("GET /ui/board", pages.Board)
	mux.HandleFunc("GET /ui/events", pages.Events)
	mux.HandleFunc("GET /ui/repos", pages.Repos)
	mux.HandleFunc("GET /ui/groups/{group}", pages.Groups)
	mux.HandleFunc("POST /ui/groups", pages.Groups)
	mux.HandleFunc("DELETE /ui/groups/{group}", pages.Groups)
	mux.HandleFunc("PUT /ui/groups/{group}/members/{login}", pages.GroupMember)
	mux.HandleFunc("DELETE /ui/groups/{group}/members/{login}", pages.GroupMember)
	mux.HandleFunc("PUT /ui/groups/{group}/repos/{repo}", pages.GroupRepo)
	mux.HandleFunc("DELETE /ui/groups/{group}/repos/{repo}", pages.GroupRepo)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}", pages.Repo)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}/access", pages.RepoAccess)
	mux.HandleFunc("PUT /ui/repos/{owner}/{name}/collaborators/{login}", pages.RepoCollaborator)
	mux.HandleFunc("DELETE /ui/repos/{owner}/{name}/collaborators/{login}", pages.RepoCollaborator)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}/contents", pages.RepoContents)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}/commits", pages.RepoCommits)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}/commits/{sha}", pages.Commit)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}/branches", pages.RepoBranches)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}/pulls", pages.RepoPulls)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}/pulls/{n}", pages.Pull)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}/pulls/{n}/comments", pages.PullComments)
	mux.HandleFunc("POST /ui/repos/{owner}/{name}/pulls/{n}/merge", pages.Pull)
	mux.HandleFunc("GET /ui/pipelines", pages.Pipelines)
	mux.HandleFunc("GET /ui/pipelines/{owner}/{name}/{n}", pages.Pipeline)
	mux.HandleFunc("GET /ui/pipelines/{owner}/{name}/{n}/log", pages.Pipeline)
	mux.HandleFunc("POST /ui/pipelines/{owner}/{name}/{n}/rerun", pages.Pipeline)
	mux.HandleFunc("POST /ui/pipelines/{owner}/{name}/{n}/approve", pages.Pipeline)
	mux.HandleFunc("GET /ui/packages", pages.Packages)
	mux.HandleFunc("GET /ui/packages/{kind}/{name...}", pages.Packages)
	mux.HandleFunc("GET /ui/stack", pages.Stack)
	mux.HandleFunc("GET /ui/agents", pages.Agents)
	mux.HandleFunc("POST /ui/pipelines/{owner}/{name}/trigger", pages.TriggerPipeline)
	mux.HandleFunc("GET /ui/users", pages.Users)
	mux.HandleFunc("POST /ui/users", pages.Users)
	mux.HandleFunc("GET /ui/invites", pages.Invites)
	mux.HandleFunc("POST /ui/invites", pages.Invites)
	mux.HandleFunc("DELETE /ui/invites/{code}", pages.Invites)
	mux.HandleFunc("GET /ui/keys", pages.Keys)
	mux.HandleFunc("POST /ui/keys", pages.Keys)
	mux.HandleFunc("POST /ui/password", pages.Password)
	mux.HandleFunc("POST /ui/approve", pages.Approve)
	mux.HandleFunc("GET /logout", pages.Logout)
	mux.Handle("GET /assets/", http.FileServer(http.FS(pages.Files())))
	mux.HandleFunc("GET /acahti.svg", pages.PublicFile)
	mux.HandleFunc("GET /acahti-mark.png", pages.PublicFile)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}
		pages.Index(w, r)
	})
	mux.HandleFunc("POST /hooks/forgejo", func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		kind, _ := payload["action"].(string)
		if kind == "" {
			kind, _ = payload["ref"].(string)
			if kind != "" {
				kind = "push"
			}
		}
		hub.Publish("forgejo", payload)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /hooks/woodpecker", func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		hub.Publish("woodpecker", payload)
		w.WriteHeader(http.StatusNoContent)
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case p == "/ci" || p == "/ci/":
			if r.Method == http.MethodGet || r.Method == http.MethodHead {
				http.Redirect(w, r, "/pipelines", http.StatusFound)
				return
			}
			writeClosed(w)
		case closedKernel(p):
			writeClosed(w)
		case strings.HasPrefix(p, "/api/packages/"):
			fjProxy.ServeHTTP(w, r)
		case gitHTTP(p):
			gitAs(a, cfg.AdminToken, fjProxy, w, r)
		default:
			mux.ServeHTTP(w, r)
		}
	})
}

func gitAs(a *auth.Service, admin string, p *httputil.ReverseProxy, w http.ResponseWriter, r *http.Request) {
	login := ""
	if u, pass, ok := r.BasicAuth(); ok {
		if user, valid := a.Parse(pass); valid && (u == user || u == "git") {
			login = user
		}
	}
	if login == "" {
		if user, ok := a.Parse(auth.Bearer(r)); ok {
			login = user
		}
	}
	if login == "" {
		w.Header().Set("WWW-Authenticate", `Basic realm="acahti"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	r.Header.Set("Authorization", "token "+admin)
	r.Header.Set("Sudo", login)
	r.Header.Set("X-WebAuth-User", login)
	p.ServeHTTP(w, r)
}

func closedKernel(p string) bool {
	if strings.HasPrefix(p, "/ci/") {
		return true
	}
	if strings.HasPrefix(p, "/login/oauth") {
		return true
	}
	if p == "/user/login" || strings.HasPrefix(p, "/user/login/") {
		return true
	}
	if p == "/api/v1" || strings.HasPrefix(p, "/api/v1/") {
		return true
	}
	return false
}

func writeClosed(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
}

func gitHTTP(p string) bool {
	if strings.Contains(p, ".git/") || strings.HasSuffix(p, ".git") {
		return true
	}
	if strings.HasSuffix(p, "/info/refs") || strings.HasSuffix(p, "/git-upload-pack") || strings.HasSuffix(p, "/git-receive-pack") {
		return true
	}
	return false
}

func reverse(target string) *httputil.ReverseProxy {
	u, err := url.Parse(target)
	if err != nil {
		u, _ = url.Parse("http://127.0.0.1")
	}
	p := httputil.NewSingleHostReverseProxy(u)
	orig := p.Director
	p.Director = func(r *http.Request) {
		fwd := r.Host
		orig(r)
		r.Host = u.Host
		if fwd != "" {
			r.Header.Set("X-Forwarded-Host", fwd)
		}
	}
	return p
}

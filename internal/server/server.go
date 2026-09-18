package server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"acahti/internal/api"
	"acahti/internal/auth"
	"acahti/internal/brand"
	"acahti/internal/catalog"
	"acahti/internal/config"
	"acahti/internal/events"
	"acahti/internal/identity"
	"acahti/internal/invite"
	"acahti/internal/mcp"
	"acahti/internal/oauth"
	"acahti/internal/passwd"
	"acahti/internal/pipeline"
	"acahti/internal/web"
)

func New(cfg config.Config, cat *catalog.Catalog, hub *events.Hub) http.Handler {
	a := auth.New([]byte(cfg.SessionSecret), cfg.AdminUser)
	inv, err := invite.Open(cfg.DataDir)
	if err != nil {
		log.Fatalf("invite store: %v", err)
	}
	oa, _ := oauth.Open(cfg.DataDir, cfg.RootURL, a)
	passwords, err := passwd.Open(cfg.DataDir)
	if err != nil {
		log.Fatalf("password store: %v", err)
	}
	cat.Notify = func(kind string, data any) { hub.Publish(kind, data) }
	go func() {
		cat.BackfillOrg()
		cat.BackfillHeatmaps()
		cat.Backfill()
		cat.WatchPipelines()
	}()
	pages := web.New(cfg, cat, a, inv, oa, hub)
	pages.Passwords = passwords
	mc := mcp.New(cfg, a, cat)
	rest := &api.API{Cfg: cfg, Auth: a, Hub: hub, Cat: cat, MCP: mc}
	fjProxy := reverse(cfg.ForgejoURL)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("GET /skill.md", pages.Skill)
	mux.HandleFunc("GET /.well-known/oauth-authorization-server", oa.Metadata)
	mux.HandleFunc("GET /.well-known/oauth-protected-resource", oa.Resource)
	mux.HandleFunc("GET /.well-known/oauth-protected-resource/mcp", oa.Resource)
	mux.HandleFunc("POST /oauth/register", oa.Register)
	mux.HandleFunc("GET /oauth/authorize", oa.Authorize)
	mux.HandleFunc("POST /oauth/token", oa.Token)
	mux.Handle("GET /mcp", mc)
	mux.Handle("POST /mcp", mc)
	mux.Handle("/acahti/v1/", rest)
	mux.HandleFunc("GET /ui/public", pages.Public)
	mux.HandleFunc("GET /ui/me", pages.Me)
	mux.HandleFunc("GET /ui/orgs", pages.Orgs)
	mux.HandleFunc("POST /ui/orgs", pages.Orgs)
	mux.HandleFunc("PUT /ui/orgs", pages.Orgs)
	mux.HandleFunc("POST /ui/login", pages.Login)
	mux.HandleFunc("POST /ui/join", pages.Join)
	mux.HandleFunc("POST /ui/logout", pages.Logout)
	mux.HandleFunc("POST /ui/oauth/approve", pages.OAuthApprove)
	mux.HandleFunc("GET /ui/board", pages.Board)
	mux.HandleFunc("GET /ui/board/heatmap", pages.BoardHeatmap)
	mux.HandleFunc("GET /ui/events", pages.Events)
	mux.HandleFunc("GET /ui/nav/tree", pages.NavTree)
	mux.HandleFunc("GET /ui/repos", pages.Repos)
	mux.HandleFunc("GET /ui/teams/{team}", pages.Teams)
	mux.HandleFunc("POST /ui/teams", pages.Teams)
	mux.HandleFunc("DELETE /ui/teams/{team}", pages.Teams)
	mux.HandleFunc("PUT /ui/teams/{team}/members/{login}", pages.TeamMember)
	mux.HandleFunc("DELETE /ui/teams/{team}/members/{login}", pages.TeamMember)
	mux.HandleFunc("PUT /ui/teams/{team}/repos/{repo}", pages.TeamRepo)
	mux.HandleFunc("DELETE /ui/teams/{team}/repos/{repo}", pages.TeamRepo)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}", pages.Repo)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}/access", pages.RepoAccess)
	mux.HandleFunc("PUT /ui/repos/{owner}/{name}/grant", pages.RepoTeamGrant)
	mux.HandleFunc("GET /ui/secrets", pages.OrgSecrets)
	mux.HandleFunc("PUT /ui/secrets/{name}", pages.OrgSecrets)
	mux.HandleFunc("DELETE /ui/secrets/{name}", pages.OrgSecrets)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}/secrets", pages.RepoSecrets)
	mux.HandleFunc("PUT /ui/repos/{owner}/{name}/secrets/{secret}", pages.RepoSecrets)
	mux.HandleFunc("DELETE /ui/repos/{owner}/{name}/secrets/{secret}", pages.RepoSecrets)
	mux.HandleFunc("PUT /ui/repos/{owner}/{name}/collaborators/{login}", pages.RepoCollaborator)
	mux.HandleFunc("DELETE /ui/repos/{owner}/{name}/collaborators/{login}", pages.RepoCollaborator)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}/contents", pages.RepoContents)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}/commits", pages.RepoCommits)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}/commits/{sha}", pages.Commit)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}/branches", pages.RepoBranches)
	mux.HandleFunc("PATCH /ui/repos/{owner}/{name}/branches/{branch}", pages.RepoBranches)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}/tags", pages.RepoTags)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}/pulls", pages.RepoPulls)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}/pulls/{n}", pages.Pull)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}/pulls/{n}/comments", pages.PullComments)
	mux.HandleFunc("POST /ui/repos/{owner}/{name}/pulls/{n}/merge", pages.Pull)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}/pipelines", pages.RepoPipelines)
	mux.HandleFunc("GET /ui/pipelines", pages.Pipelines)
	mux.HandleFunc("GET /ui/pipelines/{owner}/{name}/{n}", pages.Pipeline)
	mux.HandleFunc("GET /ui/pipelines/{owner}/{name}/{n}/log", pages.Pipeline)
	mux.HandleFunc("POST /ui/pipelines/{owner}/{name}/{n}/rerun", pages.Pipeline)
	mux.HandleFunc("POST /ui/pipelines/{owner}/{name}/{n}/cancel", pages.Pipeline)
	mux.HandleFunc("DELETE /ui/pipelines/{owner}/{name}/{n}", pages.Pipeline)
	mux.HandleFunc("POST /ui/pipelines/{owner}/{name}/{n}/approve", pages.Pipeline)
	mux.HandleFunc("GET /ui/packages", pages.Packages)
	mux.HandleFunc("GET /ui/packages/{kind}/{name...}", pages.Packages)
	mux.HandleFunc("GET /ui/stack", pages.Stack)
	mux.HandleFunc("GET /ui/agents", pages.Agents)
	mux.HandleFunc("GET /ui/queue", pages.Queue)
	mux.HandleFunc("POST /ui/pipelines/{owner}/{name}/trigger", pages.TriggerPipeline)
	mux.HandleFunc("GET /ui/users", pages.Users)
	mux.HandleFunc("POST /ui/users", pages.Users)
	mux.HandleFunc("GET /ui/users/{login}/repos", pages.UserRepos)
	mux.HandleFunc("PATCH /ui/users/{login}", pages.PatchUser)
	mux.HandleFunc("GET /ui/invites", pages.Invites)
	mux.HandleFunc("POST /ui/invites", pages.Invites)
	mux.HandleFunc("DELETE /ui/invites/{code}", pages.Invites)
	mux.HandleFunc("POST /ui/password", pages.Password)
	mux.HandleFunc("GET /logout", pages.Logout)
	mux.Handle("GET /assets/", http.FileServer(http.FS(pages.Files())))
	mux.HandleFunc("GET /acahti.svg", brand.ServeSVG)
	mux.HandleFunc("GET /acahti.png", brand.ServePNG)
	mux.HandleFunc("GET /favicon.ico", brand.ServePNG)
	mux.HandleFunc("GET /apple-touch-icon.png", brand.ServePNG)
	mux.HandleFunc("GET /apple-touch-icon-precomposed.png", brand.ServePNG)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}
		pages.Index(w, r)
	})
	mux.HandleFunc("POST /hooks/pipeline-config", pipeline.HandleConfig(cfg.ConfigToken, func(author, repo, sha string) pipeline.Ident {
		login := cat.JobLogin(repo, sha, author, cfg.AdminUser, identity.Domain(cfg.RootURL, cfg.Domain))
		if login == "" {
			return pipeline.Ident{}
		}
		return pipeline.Ident{User: login, Token: a.Issue(login), RootURL: cfg.RootURL, Org: cfg.Org}
	}))
	mux.HandleFunc("POST /hooks/woodpecker", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if p, ok := cat.IngestWoodpecker(raw); ok {
			hub.Publish("pipeline.updated", p)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /hooks/forgejo", func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if p, ok := cat.IngestForgejo(payload); ok {
			hub.Publish("pipeline.updated", p)
		} else {
			hub.Publish("forgejo", map[string]any{"ok": true})
		}
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
			gitAs(a, cat, cfg.AdminUser, fjProxy, w, r)
		case gitHTTP(p):
			gitAs(a, cat, cfg.AdminUser, fjProxy, w, r)
		default:
			mux.ServeHTTP(w, r)
		}
	})
}

func gitAs(a *auth.Service, cat *catalog.Catalog, adminUser string, p *httputil.ReverseProxy, w http.ResponseWriter, r *http.Request) {
	login := gitLogin(a, cat, r)
	if login == "" {
		w.Header().Set("WWW-Authenticate", `Basic realm="acahti"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	// Admin may clone/fetch (CI runners authenticate as the forge admin oauth user).
	// Push/receive stays forbidden so the bootstrap admin is not a shared write credential.
	if adminUser != "" && login == adminUser && gitWrite(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Header.Del("Authorization")
	r.Header.Set("X-WebAuth-User", login)
	p.ServeHTTP(w, r)
}

func gitWrite(r *http.Request) bool {
	p := r.URL.Path
	if strings.HasSuffix(p, "/git-receive-pack") {
		return true
	}
	if strings.HasSuffix(p, "/info/refs") && r.URL.Query().Get("service") == "git-receive-pack" {
		return true
	}
	return false
}

// gitLogin: Acahti login + password, or login + access_token (MCP / job identity).
func gitLogin(a *auth.Service, cat *catalog.Catalog, r *http.Request) string {
	if u, pass, ok := r.BasicAuth(); ok {
		if user, valid := a.Parse(pass); valid && u == user {
			return user
		}
		if cat != nil {
			if fu, err := cat.Authenticate(u, pass); err == nil && fu.Login == u {
				return fu.Login
			}
		}
	}
	if user, ok := a.Parse(auth.Bearer(r)); ok {
		return user
	}
	return ""
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

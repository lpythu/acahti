package server

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"acahti/internal/api"
	"acahti/internal/config"
	"acahti/internal/events"
	"acahti/internal/forgejo"
	"acahti/internal/mcp"
	"acahti/internal/web"
	"acahti/internal/woodpecker"
)

func New(cfg config.Config, fj *forgejo.Client, wp *woodpecker.Client, hub *events.Hub) http.Handler {
	pages := web.New(cfg, fj, wp, []byte(cfg.SessionSecret), hub)
	mc := mcp.New(cfg, fj, wp)
	rest := &api.API{Cfg: cfg, FJ: fj, WP: wp, Hub: hub, MCP: mc}
	fjProxy := reverse(cfg.ForgejoURL)
	wpProxy := reverse(strings.TrimSuffix(cfg.WoodpeckerURL, "/ci"))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.Handle("GET /mcp", mc)
	mux.Handle("POST /mcp", mc)
	mux.Handle("/acahti/v1/", rest)
	mux.HandleFunc("GET /ui/me", pages.Me)
	mux.HandleFunc("POST /ui/login", pages.Login)
	mux.HandleFunc("POST /ui/logout", pages.Logout)
	mux.HandleFunc("GET /ui/board", pages.Board)
	mux.HandleFunc("GET /ui/events", pages.Events)
	mux.HandleFunc("GET /ui/repos", pages.Repos)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}", pages.Repo)
	mux.HandleFunc("GET /ui/repos/{owner}/{name}/pulls/{n}", pages.Pull)
	mux.HandleFunc("POST /ui/repos/{owner}/{name}/pulls/{n}/merge", pages.Pull)
	mux.HandleFunc("GET /ui/pipelines", pages.Pipelines)
	mux.HandleFunc("GET /ui/pipelines/{owner}/{name}/{n}", pages.Pipeline)
	mux.HandleFunc("GET /ui/pipelines/{owner}/{name}/{n}/log", pages.Pipeline)
	mux.HandleFunc("POST /ui/pipelines/{owner}/{name}/{n}/rerun", pages.Pipeline)
	mux.HandleFunc("POST /ui/pipelines/{owner}/{name}/{n}/approve", pages.Pipeline)
	mux.HandleFunc("GET /ui/packages", pages.Packages)
	mux.HandleFunc("GET /ui/packages/{kind}/{name}", pages.Packages)
	mux.HandleFunc("GET /ui/stack", pages.Stack)
	mux.HandleFunc("POST /ui/pipelines/{owner}/{name}/trigger", pages.TriggerPipeline)
	mux.HandleFunc("GET /ui/users", pages.Users)
	mux.HandleFunc("POST /ui/users", pages.Users)
	mux.HandleFunc("GET /ui/keys", pages.Keys)
	mux.HandleFunc("POST /ui/keys", pages.Keys)
	mux.HandleFunc("GET /ui/tokens", pages.Tokens)
	mux.HandleFunc("DELETE /ui/tokens/{id}", pages.Tokens)
	mux.HandleFunc("POST /ui/password", pages.Password)
	mux.HandleFunc("POST /ui/token", pages.Token)
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
		case strings.HasPrefix(p, "/ci/") || p == "/ci":
			wpProxy.ServeHTTP(w, r)
		case strings.HasPrefix(p, "/login/oauth"):
			fjProxy.ServeHTTP(w, r)
		case strings.HasPrefix(p, "/api/packages/"):
			fjProxy.ServeHTTP(w, r)
		case gitHTTP(p):
			fjProxy.ServeHTTP(w, r)
		default:
			mux.ServeHTTP(w, r)
		}
	})
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

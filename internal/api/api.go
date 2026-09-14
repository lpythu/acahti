package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"acahti/internal/auth"
	"acahti/internal/config"
	"acahti/internal/events"
	"acahti/internal/httperr"
	"acahti/internal/mcp"
	"acahti/internal/oauth"
)

type API struct {
	Cfg  config.Config
	Auth *auth.Service
	Hub  *events.Hub
	MCP  *mcp.Server
}

func (a *API) token(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	if strings.HasPrefix(strings.ToLower(h), "token ") {
		return strings.TrimSpace(h[6:])
	}
	return ""
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/acahti/v1")
	if path == "/events" && r.Method == http.MethodGet {
		if _, ok := a.Auth.Parse(a.token(r)); !ok {
			oauth.Challenge(w, a.Cfg.RootURL+"/.well-known/oauth-protected-resource")
			return
		}
		a.Hub.SSE(w, r)
		return
	}
	tok := a.token(r)
	login, ok := a.Auth.Parse(tok)
	if !ok {
		oauth.Challenge(w, a.Cfg.RootURL+"/.well-known/oauth-protected-resource")
		return
	}
	args := map[string]any{}
	if r.Method != http.MethodGet && r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&args)
	}
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			args[k] = v[0]
		}
	}
	// path /repos/{owner}/{name}/...
	tool, extra := route(path, r.Method, args)
	if tool == "" {
		httperr.Write(w, http.StatusNotFound, "not_found", path)
		return
	}
	for k, v := range extra {
		args[k] = v
	}
	out, err := a.MCP.CallForAPI(login, tool, args)
	if err != nil {
		httperr.Write(w, http.StatusUnprocessableEntity, "failed_precondition", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func route(path, method string, args map[string]any) (string, map[string]any) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	extra := map[string]any{}
	if len(parts) == 1 && parts[0] == "me" && method == http.MethodGet {
		return "whoami", extra
	}
	if len(parts) == 1 && parts[0] == "repos" && method == http.MethodGet {
		return "repo_list", extra
	}
	if len(parts) == 1 && parts[0] == "repos" && method == http.MethodPost {
		return "repo_create", extra
	}
	if len(parts) == 1 && parts[0] == "packages" {
		return "pkg_list", extra
	}
	if len(parts) == 1 && parts[0] == "agents" {
		return "agent_status", extra
	}
	if len(parts) >= 3 && parts[0] == "repos" {
		extra["owner"], extra["name"] = parts[1], parts[2]
		rest := parts[3:]
		if len(rest) == 0 && method == http.MethodGet {
			return "repo_get", extra
		}
		if len(rest) == 1 && rest[0] == "branches" {
			return "branch_list", extra
		}
		if len(rest) >= 2 && rest[0] == "refs" {
			extra["ref"] = strings.Join(rest[1:], "/")
			return "ref_delete", extra
		}
		if len(rest) == 1 && rest[0] == "pulls" && method == http.MethodGet {
			return "pr_list", extra
		}
		if len(rest) == 1 && rest[0] == "pulls" && method == http.MethodPost {
			return "pr_create", extra
		}
		if len(rest) >= 2 && rest[0] == "pulls" {
			n, _ := strconv.Atoi(rest[1])
			extra["number"] = n
			if len(rest) == 2 && method == http.MethodGet {
				return "pr_get", extra
			}
			if len(rest) == 3 && rest[2] == "comments" && method == http.MethodGet {
				return "pr_comments", extra
			}
			if len(rest) == 3 && rest[2] == "comments" {
				return "pr_comment", extra
			}
			if len(rest) == 3 && rest[2] == "merge" {
				return "pr_merge", extra
			}
			if len(rest) == 2 && method == http.MethodDelete {
				return "pr_close", extra
			}
		}
		if len(rest) == 2 && rest[0] == "checks" {
			extra["sha"] = rest[1]
			return "checks_wait", extra
		}
		if len(rest) >= 1 && rest[0] == "pipelines" {
			extra["repo"] = extra["owner"].(string) + "/" + extra["name"].(string)
			if len(rest) == 1 && method == http.MethodGet {
				return "pipeline_list", extra
			}
			if len(rest) == 1 && method == http.MethodPost {
				return "pipeline_trigger", extra
			}
			if len(rest) >= 2 {
				n, _ := strconv.ParseInt(rest[1], 10, 64)
				extra["number"] = n
				if len(rest) == 2 && method == http.MethodGet {
					return "pipeline_get", extra
				}
				if len(rest) == 3 && rest[2] == "log" {
					return "pipeline_log", extra
				}
				if len(rest) == 3 && rest[2] == "rerun" {
					return "pipeline_rerun", extra
				}
				if len(rest) == 3 && rest[2] == "cancel" {
					return "pipeline_cancel", extra
				}
				if len(rest) == 3 && rest[2] == "approve" {
					return "deploy_approve", extra
				}
			}
		}
	}
	if len(parts) == 1 && parts[0] == "inbox" && method == http.MethodGet {
		return "inbox", extra
	}
	if len(parts) == 1 && parts[0] == "publish" {
		return "pkg_publish", extra
	}
	return "", nil
}

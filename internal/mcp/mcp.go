package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"acahti/internal/auth"
	"acahti/internal/brand"
	"acahti/internal/catalog"
	"acahti/internal/config"
	"acahti/internal/forgejo"
	"acahti/internal/identity"
	"acahti/internal/oauth"
	"acahti/internal/page"
	"acahti/internal/woodpecker"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type Server struct {
	Cfg     ConfigView
	Auth    *auth.Service
	FJ      *forgejo.Client
	WP      *woodpecker.Client
	Cat     *catalog.Catalog
	handler http.Handler
}

type ConfigView struct {
	Org     string
	RootURL string
	Domain  string
	Version string
}

func New(cfg config.Config, a *auth.Service, fj *forgejo.Client, wp *woodpecker.Client, cat *catalog.Catalog) *Server {
	s := &Server{
		Cfg:  ConfigView{Org: cfg.Org, RootURL: cfg.RootURL, Domain: cfg.Domain, Version: cfg.Version},
		Auth: a,
		FJ:   fj,
		WP:   wp,
		Cat:  cat,
	}
	s.handler = s.newHandler()
	return s
}

type toolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func tools() []toolSpec {
	obj := func(props map[string]any, req ...string) map[string]any {
		if props == nil {
			props = map[string]any{}
		}
		m := map[string]any{"type": "object", "properties": props}
		if len(req) > 0 {
			m["required"] = req
		}
		return m
	}
	str := map[string]any{"type": "string"}
	num := map[string]any{"type": "number"}
	pg := map[string]any{"page": num, "page_size": num}
	return []toolSpec{
		{Name: "repo_list", Description: "List repositories the token can see", InputSchema: obj(pg)},
		{Name: "repo_get", Description: "Get one repository", InputSchema: obj(map[string]any{"owner": str, "name": str}, "owner", "name")},
		{Name: "repo_create", Description: "Create a private org repo. Optional team attaches access. Does not set branch protection", InputSchema: obj(map[string]any{"name": str, "team": str}, "name")},
		{Name: "branch_list", Description: "List branches with default and protected flags from repo settings", InputSchema: obj(map[string]any{"owner": str, "name": str, "page": num, "page_size": num}, "owner", "name")},
		{Name: "ref_delete", Description: "Delete a git ref. ref is dev, heads/dev, or refs/heads/dev", InputSchema: obj(map[string]any{"owner": str, "name": str, "ref": str}, "owner", "name", "ref")},
		{Name: "pr_create", Description: "Open a pull request. base defaults to the repo default_branch", InputSchema: obj(map[string]any{"owner": str, "name": str, "title": str, "head": str, "base": str, "body": str}, "owner", "name", "title", "head")},
		{Name: "pr_list", Description: "List pull requests", InputSchema: obj(map[string]any{"owner": str, "name": str, "state": str, "page": num, "page_size": num}, "owner", "name")},
		{Name: "pr_get", Description: "Get a pull request with latest commit checks per context", InputSchema: obj(map[string]any{"owner": str, "name": str, "number": num}, "owner", "name", "number")},
		{Name: "pr_comment", Description: "Comment on a pull request", InputSchema: obj(map[string]any{"owner": str, "name": str, "number": num, "body": str}, "owner", "name", "number", "body")},
		{Name: "pr_comments", Description: "List pull request comments", InputSchema: obj(map[string]any{"owner": str, "name": str, "number": num, "page": num, "page_size": num}, "owner", "name", "number")},
		{Name: "pr_merge", Description: "Merge a PR when the latest pipeline round on the head SHA is green", InputSchema: obj(map[string]any{"owner": str, "name": str, "number": num}, "owner", "name", "number")},
		{Name: "pr_close", Description: "Close a pull request", InputSchema: obj(map[string]any{"owner": str, "name": str, "number": num}, "owner", "name", "number")},
		{Name: "checks_wait", Description: "Snapshot of commit checks for the latest pipeline round. Poll this tool; it does not block", InputSchema: obj(map[string]any{"owner": str, "name": str, "sha": str}, "owner", "name", "sha")},
		{Name: "pipeline_list", Description: "List pipelines for a repo with wait (queue, deps, concurrency) on in-flight runs. sha is a commit prefix", InputSchema: obj(map[string]any{"repo": str, "sha": str, "branch": str, "status": str, "page": num, "page_size": num}, "repo")},
		{Name: "pipeline_get", Description: "Get one pipeline with jobs, steps, and wait (queue, deps, concurrency)", InputSchema: obj(map[string]any{"repo": str, "number": num}, "repo", "number")},
		{Name: "pipeline_log", Description: "Fetch pipeline logs. Omit step for failed steps only", InputSchema: obj(map[string]any{"repo": str, "number": num, "step": num, "tail_lines": num}, "repo", "number")},
		{Name: "pipeline_rerun", Description: "Rerun a pipeline", InputSchema: obj(map[string]any{"repo": str, "number": num}, "repo", "number")},
		{Name: "pipeline_trigger", Description: "Manual run on a ref. Official release is still git push", InputSchema: obj(map[string]any{"repo": str, "ref": str}, "repo")},
		{Name: "pipeline_cancel", Description: "Cancel a running or queued pipeline", InputSchema: obj(map[string]any{"repo": str, "number": num}, "repo", "number")},
		{Name: "pipeline_delete", Description: "Delete a finished pipeline run", InputSchema: obj(map[string]any{"repo": str, "number": num}, "repo", "number")},
		{Name: "inbox", Description: "Island inbox: open PRs or pipelines needing attention (blocked/failed latest per repo)", InputSchema: obj(map[string]any{"section": str, "page": num, "page_size": num})},
		{Name: "pkg_publish", Description: "Publish a language package (pypi wheel URL or npm tarball URL)", InputSchema: obj(map[string]any{"kind": str, "url": str, "filename": str}, "kind", "url")},
		{Name: "pkg_list", Description: "List language packages", InputSchema: obj(map[string]any{"owner": str, "kind": str, "page": num, "page_size": num})},
		{Name: "pkg_delete", Description: "Delete a language package version. Org admin. Omit version to delete every version of that name", InputSchema: obj(map[string]any{"owner": str, "kind": str, "name": str, "version": str}, "kind", "name")},
		{Name: "whoami", Description: "Acahti git identity: git_name, git_email, clone_url_template, skill_url, skill_sha, apply_when_remote_host, setup_local, org_admin", InputSchema: obj(map[string]any{})},
		{Name: "agent_status", Description: "Host runners plus Woodpecker queue stats (pending, waiting_on_deps, running)", InputSchema: obj(pg)},
		{Name: "deploy_approve", Description: "Approve a gated deploy pipeline", InputSchema: obj(map[string]any{"repo": str, "number": num}, "repo", "number")},
		{Name: "secret_list", Description: "List pipeline secret names (never values). scope org is org admins. scope repo is the effective set (org inherited plus repo) for repo admins. repo or owner+name for repo scope", InputSchema: obj(map[string]any{"scope": str, "repo": str, "owner": str, "name": str, "page": num, "page_size": num})},
		{Name: "secret_put", Description: "Create or replace a pipeline secret. Does not echo value. org scope needs org admin. repo scope needs repo admin. name is the secret", InputSchema: obj(map[string]any{"scope": str, "name": str, "secret": str, "value": str, "events": map[string]any{"type": "array", "items": str}, "repo": str, "owner": str}, "value")},
		{Name: "secret_delete", Description: "Delete a pipeline secret. org scope needs org admin. repo scope needs repo admin and only deletes the repo override", InputSchema: obj(map[string]any{"scope": str, "name": str, "secret": str, "repo": str, "owner": str})},
	}
}

func (s *Server) version() string {
	if s != nil && strings.TrimSpace(s.Cfg.Version) != "" {
		return s.Cfg.Version
	}
	return "dev"
}

func ToolNames() []string {
	var out []string
	for _, t := range tools() {
		out = append(out, t.Name)
	}
	return out
}

type loginKey struct{}

func (s *Server) newHandler() http.Handler {
	server := sdk.NewServer(&sdk.Implementation{
		Name: "acahti", Title: "Acahti", Version: s.version(),
		WebsiteURL: brand.Root(s.Cfg.RootURL),
		Icons:      []sdk.Icon{{Source: brand.PNGURL(s.Cfg.RootURL), MIMEType: "image/png"}},
	}, &sdk.ServerOptions{Instructions: identity.Instructions(s.Cfg.RootURL, s.Cfg.Domain)})
	for _, tool := range tools() {
		sdk.AddTool(server, &sdk.Tool{
			Name: tool.Name, Description: tool.Description, InputSchema: tool.InputSchema,
		}, func(ctx context.Context, _ *sdk.CallToolRequest, args map[string]any) (*sdk.CallToolResult, any, error) {
			login, ok := ctx.Value(loginKey{}).(string)
			if !ok || login == "" {
				return nil, nil, fmt.Errorf("authentication required")
			}
			out, err := s.call(login, tool.Name, args)
			return nil, out, err
		})
	}
	transport := sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return server },
		&sdk.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	return http.NewCrossOriginProtection().Handler(transport)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Link", brand.Link(s.Cfg.RootURL))
	login, ok := s.Auth.Parse(auth.Bearer(r))
	if !ok {
		oauth.Challenge(w, s.Cfg.RootURL+"/.well-known/oauth-protected-resource")
		return
	}
	s.handler.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), loginKey{}, login)))
}

func (s *Server) CallForAPI(token, name string, a map[string]any) (any, error) {
	return s.call(token, name, a)
}

func (s *Server) call(token, name string, a map[string]any) (any, error) {
	str := func(k string) string {
		v, _ := a[k].(string)
		return v
	}
	num := func(k string) int64 {
		return asInt64(a[k])
	}
	org := s.Cfg.Org
	pq := page.FromInts(int(num("page")), int(num("page_size")))
	switch name {
	case "whoami":
		u, err := s.FJ.UserSudo(token)
		var v map[string]any
		if err != nil {
			v = identity.View(token, "", s.Cfg.RootURL, s.Cfg.Domain, org)
		} else {
			v = identity.View(u.Login, u.FullName, s.Cfg.RootURL, s.Cfg.Domain, org)
		}
		v["org_admin"] = s.Cat != nil && s.Cat.IsOrgAdmin(token)
		return v, nil
	case "repo_list":
		return s.Cat.ListRepos(token, "", pq)
	case "repo_get":
		repo, err := s.FJ.GetRepo(str("owner"), str("name"), token)
		if err != nil {
			return nil, err
		}
		if s.Cat != nil {
			repo = s.Cat.PublicRepo(repo)
			repo.CanManageSecrets = s.Cat.IsOrgAdmin(token) || repo.Permissions.Admin
		}
		return repo, nil
	case "repo_create":
		repo, err := s.FJ.CreateOrgRepo(org, str("name"), true)
		if err != nil {
			return nil, err
		}
		s.Cat.RememberRepo(repo)
		if team := str("team"); team != "" {
			if err := s.Cat.AttachRepo(team, repo.Name); err != nil {
				return nil, err
			}
			repo.Team = team
		}
		if s.WP.Ready() {
			_ = s.WP.Activate(org+"/"+repo.Name, strconv.FormatInt(repo.ID, 10))
		}
		return s.Cat.PublicRepo(repo), nil
	case "branch_list":
		return s.Cat.ListBranches(token, str("owner"), str("name"), pq)
	case "ref_delete":
		return map[string]any{"ok": true}, s.FJ.DeleteRef(str("owner"), str("name"), str("ref"))
	case "pr_create":
		base := str("base")
		if base == "" {
			repo, err := s.FJ.GetRepo(str("owner"), str("name"), token)
			if err != nil {
				return nil, err
			}
			base = repo.DefaultBranch
		}
		return s.FJ.CreatePR(str("owner"), str("name"), str("title"), str("head"), base, str("body"), token)
	case "pr_list":
		return s.FJ.ListPRs(str("owner"), str("name"), str("state"), pq)
	case "pr_get":
		return s.Cat.PRDetail(token, str("owner"), str("name"), int(num("number")))
	case "pr_comment":
		return map[string]any{"ok": true}, s.FJ.CommentPR(str("owner"), str("name"), int(num("number")), str("body"), token)
	case "pr_comments":
		return s.Cat.ListComments(token, str("owner"), str("name"), int(num("number")), pq)
	case "pr_merge":
		return s.Cat.MergePR(token, str("owner"), str("name"), int(num("number")))
	case "pr_close":
		return s.Cat.ClosePR(token, str("owner"), str("name"), int(num("number")))
	case "checks_wait":
		return s.waitChecks(str("owner"), str("name"), str("sha"))
	case "pipeline_list":
		return s.listPipes(token, repoArg(str), str("sha"), str("branch"), str("status"), pq)
	case "pipeline_get":
		return s.Cat.PipelineDetail(token, repoArg(str), num("number"))
	case "pipeline_log":
		return s.Cat.PipelineLogs(token, repoArg(str), num("number"), num("step"), num("tail_lines"))
	case "pipeline_rerun":
		pipe, err := s.WP.Rerun(repoArg(str), num("number"))
		if err != nil {
			return nil, err
		}
		return s.Cat.Remember(pipe), nil
	case "pipeline_trigger":
		repo := repoArg(str)
		if err := s.seeRepo(token, repo); err != nil {
			return nil, err
		}
		ref := str("ref")
		if ref == "" {
			ref = "dev"
		}
		pipe, err := s.WP.Trigger(repo, ref)
		if err != nil {
			return nil, err
		}
		return s.Cat.Remember(pipe), nil
	case "pipeline_cancel":
		pipe, err := s.Cat.CancelPipeline(token, repoArg(str), num("number"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"ok": true, "pipeline": pipe}, nil
	case "pipeline_delete":
		if err := s.Cat.DeletePipeline(token, repoArg(str), num("number")); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true}, nil
	case "inbox":
		section := str("section")
		if section == "prs" {
			return s.Cat.BoardPRs(token, pq)
		}
		return s.Cat.BoardPipes(token, pq)
	case "pkg_list":
		owner := str("owner")
		if owner == "" {
			owner = org
		}
		return s.FJ.ListPackages(owner, str("kind"), str("q"), pq, token)
	case "pkg_delete":
		return s.deletePackage(token, str)
	case "pkg_publish":
		return s.publish(token, str("kind"), str("url"), str("filename"))
	case "agent_status":
		return s.Cat.AgentStatus(pq)
	case "deploy_approve":
		repo := str("repo")
		if err := s.WP.Approve(repo, num("number")); err != nil {
			return nil, err
		}
		if pipe, err := s.Cat.Refresh(repo, num("number")); err == nil {
			return map[string]any{"ok": true, "pipeline": pipe}, nil
		}
		return map[string]any{"ok": true}, nil
	case "secret_list":
		return s.listSecrets(token, str, pq)
	case "secret_put":
		return s.putSecret(token, str, a)
	case "secret_delete":
		return s.deleteSecret(token, str)
	default:
		return nil, fmt.Errorf("unknown tool %s", name)
	}
}

func asInt64(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case json.Number:
		n, _ := x.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(x, 10, 64)
		return n
	case int:
		return int64(x)
	case int32:
		return int64(x)
	case int64:
		return x
	}
	return 0
}

func repoArg(str func(string) string) string {
	if r := str("repo"); r != "" {
		return r
	}
	o, n := str("owner"), str("name")
	if o != "" && n != "" {
		return o + "/" + n
	}
	return ""
}

func asStrings(v any) []string {
	switch x := v.(type) {
	case []string:
		return x
	case []any:
		out := make([]string, 0, len(x))
		for _, e := range x {
			s, _ := e.(string)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if strings.TrimSpace(x) == "" {
			return nil
		}
		parts := strings.FieldsFunc(x, func(r rune) bool { return r == ',' || r == ' ' })
		return parts
	default:
		return nil
	}
}

func (s *Server) secretScope(str func(string) string) (scope, repo, secret string) {
	scope = strings.ToLower(str("scope"))
	repo = repoArg(str)
	secret = str("secret")
	if secret == "" && (scope == "org" || (scope == "" && repo == "")) {
		secret = str("name")
	}
	if scope == "" {
		if repo != "" {
			scope = "repo"
		} else {
			scope = "org"
		}
	}
	return scope, repo, secret
}

func (s *Server) listSecrets(token string, str func(string) string, q page.Query) (any, error) {
	scope, repo, _ := s.secretScope(str)
	if s.Cat == nil {
		return nil, catalog.ErrNotFound
	}
	if scope == "repo" {
		owner, name, _ := strings.Cut(repo, "/")
		if owner == "" || name == "" {
			return nil, fmt.Errorf("%w: repo required", catalog.ErrInvalid)
		}
		return s.Cat.ListRepoSecrets(token, owner, name, q)
	}
	return s.Cat.ListOrgSecrets(token, q)
}

func (s *Server) putSecret(token string, str func(string) string, a map[string]any) (any, error) {
	scope, repo, secret := s.secretScope(str)
	if s.Cat == nil {
		return nil, catalog.ErrNotFound
	}
	events := asStrings(a["events"])
	if scope == "repo" {
		owner, name, _ := strings.Cut(repo, "/")
		if owner == "" || name == "" {
			return nil, fmt.Errorf("%w: repo required", catalog.ErrInvalid)
		}
		if secret == "" {
			secret = str("name")
		}
		return s.Cat.PutRepoSecret(token, owner, name, secret, str("value"), events)
	}
	return s.Cat.PutOrgSecret(token, secret, str("value"), events)
}

func (s *Server) deletePackage(token string, str func(string) string) (any, error) {
	if s.Cat == nil || !s.Cat.IsOrgAdmin(token) {
		return nil, catalog.ErrNotFound
	}
	kind := strings.ToLower(strings.TrimSpace(str("kind")))
	name := strings.TrimSpace(str("name"))
	version := strings.TrimSpace(str("version"))
	if kind == "" || name == "" {
		return nil, fmt.Errorf("%w: kind and name required", catalog.ErrInvalid)
	}
	owner := str("owner")
	if owner == "" {
		owner = s.Cfg.Org
	}
	if version != "" {
		if err := s.FJ.DeletePackage(owner, kind, name, version, token); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true, "deleted": []string{name + "@" + version}}, nil
	}
	all, err := page.Walk(func(q page.Query) (page.Result[forgejo.Package], error) {
		return s.FJ.ListPackages(owner, kind, name, q, token)
	})
	if err != nil {
		return nil, err
	}
	deleted := make([]string, 0)
	for _, p := range all {
		if p.Name != name || !strings.EqualFold(p.Type, kind) {
			continue
		}
		if err := s.FJ.DeletePackage(owner, p.Type, p.Name, p.Version, token); err != nil {
			return nil, err
		}
		deleted = append(deleted, p.Name+"@"+p.Version)
	}
	if len(deleted) == 0 {
		return nil, catalog.ErrNotFound
	}
	return map[string]any{"ok": true, "deleted": deleted}, nil
}

func (s *Server) deleteSecret(token string, str func(string) string) (any, error) {
	scope, repo, secret := s.secretScope(str)
	if s.Cat == nil {
		return nil, catalog.ErrNotFound
	}
	if scope == "repo" {
		owner, name, _ := strings.Cut(repo, "/")
		if owner == "" || name == "" {
			return nil, fmt.Errorf("%w: repo required", catalog.ErrInvalid)
		}
		if secret == "" {
			secret = str("name")
		}
		if err := s.Cat.DeleteRepoSecret(token, owner, name, secret); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true}, nil
	}
	if err := s.Cat.DeleteOrgSecret(token, secret); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

func (s *Server) seeRepo(token, repo string) error {
	owner, name, _ := strings.Cut(repo, "/")
	if owner == "" || name == "" {
		return fmt.Errorf("repo must be owner/name")
	}
	_, err := s.Cat.RepoHeader(token, owner, name, "")
	return err
}

func (s *Server) waitChecks(owner, name, sha string) (any, error) {
	var (
		ok  bool
		st  []forgejo.Status
		err error
	)
	if s.Cat != nil {
		ok, st, err = s.Cat.CommitChecks(owner, name, sha)
	} else {
		ok, st, err = s.FJ.ChecksGreen(owner, name, sha)
	}
	if err != nil {
		return nil, err
	}
	pending := false
	failed := false
	for _, x := range st {
		switch strings.ToLower(x.Status) {
		case "pending":
			pending = true
		case "failure", "error":
			failed = true
		}
	}
	done := !pending && (ok || failed)
	return map[string]any{"ok": ok && done, "done": done, "statuses": st}, nil
}

func (s *Server) listPipes(token, repo, sha, branch, status string, q page.Query) (any, error) {
	if repo == "" {
		return nil, fmt.Errorf("repo required")
	}
	return s.Cat.ListRepoPipelines(token, repo, sha, branch, status, q)
}

func (s *Server) publish(token, kind, fileURL, filename string) (any, error) {
	if fileURL == "" {
		return nil, fmt.Errorf("url required")
	}
	resp, err := http.Get(fileURL) //nolint:gosec // caller-supplied artifact URL
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if filename == "" {
		filename = fileURL[strings.LastIndex(fileURL, "/")+1:]
	}
	var path string
	switch kind {
	case "pypi":
		path = "/api/packages/" + s.Cfg.Org + "/pypi?filename=" + filename
	case "npm":
		path = "/api/packages/" + s.Cfg.Org + "/npm/" + filename
	default:
		return nil, fmt.Errorf("kind must be pypi or npm")
	}
	code, raw, err := s.FJ.PutBytes(path, token, "application/octet-stream", b)
	if err != nil {
		return nil, err
	}
	if code >= 400 {
		return nil, fmt.Errorf("publish %d: %s", code, strings.TrimSpace(string(raw)))
	}
	return map[string]any{"ok": true, "filename": filename}, nil
}

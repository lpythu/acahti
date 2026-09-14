package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"acahti/internal/auth"
	"acahti/internal/brand"
	"acahti/internal/catalog"
	"acahti/internal/config"
	"acahti/internal/forgejo"
	"acahti/internal/httperr"
	"acahti/internal/identity"
	"acahti/internal/oauth"
	"acahti/internal/page"
	"acahti/internal/woodpecker"
)

type Server struct {
	Cfg  ConfigView
	Auth *auth.Service
	FJ   *forgejo.Client
	WP   *woodpecker.Client
	Cat  *catalog.Catalog
}

type ConfigView struct {
	Org     string
	RootURL string
	Domain  string
}

func New(cfg config.Config, a *auth.Service, fj *forgejo.Client, wp *woodpecker.Client) *Server {
	return &Server{
		Cfg:  ConfigView{Org: cfg.Org, RootURL: cfg.RootURL, Domain: cfg.Domain},
		Auth: a,
		FJ:   fj,
		WP:   wp,
		Cat:  catalog.New(cfg, fj, wp),
	}
}

type rpcReq struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcRes struct {
	JSONRPC string  `json:"jsonrpc"`
	ID      any     `json:"id"`
	Result  any     `json:"result,omitempty"`
	Error   *rpcErr `json:"error,omitempty"`
}

type rpcErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
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
		{Name: "repo_create", Description: "Create a private org repo and protect dev and test. Optional team attaches access", InputSchema: obj(map[string]any{"name": str, "team": str}, "name")},
		{Name: "branch_list", Description: "List branches", InputSchema: obj(map[string]any{"owner": str, "name": str, "page": num, "page_size": num}, "owner", "name")},
		{Name: "ref_delete", Description: "Delete a git ref", InputSchema: obj(map[string]any{"owner": str, "name": str, "ref": str}, "owner", "name", "ref")},
		{Name: "pr_create", Description: "Open a pull request", InputSchema: obj(map[string]any{"owner": str, "name": str, "title": str, "head": str, "base": str, "body": str}, "owner", "name", "title", "head")},
		{Name: "pr_list", Description: "List pull requests", InputSchema: obj(map[string]any{"owner": str, "name": str, "state": str, "page": num, "page_size": num}, "owner", "name")},
		{Name: "pr_get", Description: "Get a pull request with commit checks", InputSchema: obj(map[string]any{"owner": str, "name": str, "number": num}, "owner", "name", "number")},
		{Name: "pr_comment", Description: "Comment on a pull request", InputSchema: obj(map[string]any{"owner": str, "name": str, "number": num, "body": str}, "owner", "name", "number", "body")},
		{Name: "pr_comments", Description: "List pull request comments", InputSchema: obj(map[string]any{"owner": str, "name": str, "number": num, "page": num, "page_size": num}, "owner", "name", "number")},
		{Name: "pr_merge", Description: "Merge a PR only when commit checks are green", InputSchema: obj(map[string]any{"owner": str, "name": str, "number": num}, "owner", "name", "number")},
		{Name: "checks_wait", Description: "Wait until commit checks finish or timeout. timeout_sec=0 is a snapshot", InputSchema: obj(map[string]any{"owner": str, "name": str, "sha": str, "timeout_sec": num}, "owner", "name", "sha")},
		{Name: "pipeline_list", Description: "List pipelines for a repo", InputSchema: obj(map[string]any{"repo": str, "sha": str, "branch": str, "status": str, "page": num, "page_size": num}, "repo")},
		{Name: "pipeline_get", Description: "Get one pipeline and its steps", InputSchema: obj(map[string]any{"repo": str, "number": num}, "repo", "number")},
		{Name: "pipeline_log", Description: "Fetch pipeline logs. Omit step for failed steps only", InputSchema: obj(map[string]any{"repo": str, "number": num, "step": num, "tail_lines": num}, "repo", "number")},
		{Name: "pipeline_rerun", Description: "Rerun a pipeline", InputSchema: obj(map[string]any{"repo": str, "number": num}, "repo", "number")},
		{Name: "pipeline_trigger", Description: "Manual run on a ref. Official release is still git push", InputSchema: obj(map[string]any{"repo": str, "ref": str}, "repo")},
		{Name: "pipeline_cancel", Description: "Cancel a running pipeline", InputSchema: obj(map[string]any{"repo": str, "number": num}, "repo", "number")},
		{Name: "inbox", Description: "Island inbox: open PRs, blocked deploys, or failed pipelines", InputSchema: obj(map[string]any{"section": str, "page": num, "page_size": num})},
		{Name: "pkg_publish", Description: "Publish a language package (pypi wheel URL or npm tarball URL)", InputSchema: obj(map[string]any{"kind": str, "url": str, "filename": str}, "kind", "url")},
		{Name: "pkg_list", Description: "List language packages", InputSchema: obj(map[string]any{"owner": str, "kind": str, "page": num, "page_size": num})},
		{Name: "whoami", Description: "Acahti git identity for this token: git_name, git_email, apply_when_remote_host, setup_local", InputSchema: obj(nil)},
		{Name: "agent_status", Description: "Host agent last contact", InputSchema: obj(pg)},
		{Name: "deploy_approve", Description: "Approve a gated deploy pipeline", InputSchema: obj(map[string]any{"repo": str, "number": num}, "repo", "number")},
	}
}

func ToolNames() []string {
	var out []string
	for _, t := range tools() {
		out = append(out, t.Name)
	}
	return out
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Link", brand.Link(s.Cfg.RootURL))
	login, ok := s.Auth.Parse(auth.Bearer(r))
	if !ok {
		oauth.Challenge(w, s.Cfg.RootURL+"/.well-known/oauth-protected-resource")
		return
	}
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		httperr.Write(w, http.StatusMethodNotAllowed, "bad_request", "POST JSON-RPC to /mcp")
		return
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		httperr.Write(w, http.StatusBadRequest, "bad_request", "read body")
		return
	}
	var req rpcReq
	if err := json.Unmarshal(raw, &req); err != nil {
		writeRPC(w, nil, nil, &rpcErr{Code: -32700, Message: "parse error"})
		return
	}
	res, rerr := s.dispatch(login, req)
	writeRPC(w, req.ID, res, rerr)
}

func (s *Server) dispatch(token string, req rpcReq) (any, *rpcErr) {
	switch req.Method {
	case "initialize":
		return map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo": map[string]any{
				"name":       "acahti",
				"title":      "Acahti",
				"version":    "1",
				"websiteUrl": brand.Root(s.Cfg.RootURL),
				"icons":      brand.Icons(s.Cfg.RootURL),
			},
			"instructions":    identity.Instructions(s.Cfg.RootURL, s.Cfg.Domain),
		}, nil
	case "notifications/initialized", "ping":
		return map[string]any{}, nil
	case "tools/list":
		return map[string]any{"tools": tools()}, nil
	case "tools/call":
		var p struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return nil, fail("bad_request", "invalid tools/call")
		}
		out, err := s.call(token, p.Name, p.Arguments)
		if err != nil {
			return nil, fail("tool_error", err.Error())
		}
		b, _ := json.Marshal(out)
		return map[string]any{
			"content": []map[string]any{{"type": "text", "text": string(b)}},
			"isError": false,
		}, nil
	default:
		return nil, &rpcErr{Code: -32601, Message: "method not found", Data: httperr.Info{Code: "not_found", Message: req.Method}}
	}
}

func fail(code, msg string) *rpcErr {
	return &rpcErr{Code: -32000, Message: msg, Data: httperr.Info{Code: code, Message: msg}}
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
		if err != nil {
			return identity.View(token, token, s.Cfg.RootURL, s.Cfg.Domain, org), nil
		}
		return identity.View(u.Login, u.FullName, s.Cfg.RootURL, s.Cfg.Domain, org), nil
	case "repo_list":
		return s.Cat.ListRepos(token, "", pq)
	case "repo_get":
		repo, err := s.FJ.GetRepo(str("owner"), str("name"), token)
		if err != nil {
			return nil, err
		}
		return s.Cat.PublicRepo(repo), nil
	case "repo_create":
		repo, err := s.FJ.CreateOrgRepo(org, str("name"), true)
		if err != nil {
			return nil, err
		}
		if team := str("team"); team != "" {
			if err := s.Cat.AttachRepo(team, repo.Name); err != nil {
				return nil, err
			}
			repo.Team = team
		}
		_ = s.FJ.ProtectTrains(org, repo.Name)
		if s.WP.Ready() {
			_ = s.WP.Activate(org + "/" + repo.Name)
		}
		return s.Cat.PublicRepo(repo), nil
	case "branch_list":
		return s.FJ.ListBranches(str("owner"), str("name"), pq)
	case "ref_delete":
		return map[string]any{"ok": true}, s.FJ.DeleteRef(str("owner"), str("name"), str("ref"))
	case "pr_create":
		base := str("base")
		if base == "" {
			base = "dev"
		}
		return s.FJ.CreatePR(str("owner"), str("name"), str("title"), str("head"), base, str("body"))
	case "pr_list":
		return s.FJ.ListPRs(str("owner"), str("name"), str("state"), pq)
	case "pr_get":
		return s.Cat.PRDetail(token, str("owner"), str("name"), int(num("number")))
	case "pr_comment":
		return map[string]any{"ok": true}, s.FJ.CommentPR(str("owner"), str("name"), int(num("number")), str("body"))
	case "pr_comments":
		return s.Cat.ListComments(token, str("owner"), str("name"), int(num("number")), pq)
	case "pr_merge":
		return s.Cat.MergePR(token, str("owner"), str("name"), int(num("number")))
	case "checks_wait":
		return s.waitChecks(str("owner"), str("name"), str("sha"), checkTimeout(a))
	case "pipeline_list":
		return s.listPipes(token, repoArg(str), str("sha"), str("branch"), str("status"), pq)
	case "pipeline_get":
		return s.Cat.PipelineDetail(token, repoArg(str), num("number"))
	case "pipeline_log":
		return s.Cat.PipelineLogs(token, repoArg(str), num("number"), num("step"), num("tail_lines"))
	case "pipeline_rerun":
		return s.WP.Rerun(repoArg(str), num("number"))
	case "pipeline_trigger":
		repo := repoArg(str)
		if err := s.seeRepo(token, repo); err != nil {
			return nil, err
		}
		ref := str("ref")
		if ref == "" {
			ref = "dev"
		}
		return s.WP.Trigger(repo, ref)
	case "pipeline_cancel":
		repo := repoArg(str)
		if err := s.seeRepo(token, repo); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true}, s.WP.Cancel(repo, num("number"))
	case "inbox":
		section := str("section")
		if section == "" {
			section = "failed"
		}
		if section == "prs" {
			return s.Cat.BoardPRs(token, pq)
		}
		return s.Cat.BoardPipes(token, section, pq)
	case "pkg_list":
		owner := str("owner")
		if owner == "" {
			owner = org
		}
		return s.FJ.ListPackages(owner, str("kind"), str("q"), pq)
	case "pkg_publish":
		return s.publish(token, str("kind"), str("url"), str("filename"))
	case "agent_status":
		return s.Cat.ListAgents(pq)
	case "deploy_approve":
		return map[string]any{"ok": true}, s.WP.Approve(str("repo"), num("number"))
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

func checkTimeout(a map[string]any) int64 {
	v, ok := a["timeout_sec"]
	if !ok {
		return 600
	}
	n := int64(0)
	switch x := v.(type) {
	case float64:
		n = int64(x)
	case json.Number:
		n, _ = x.Int64()
	case string:
		n, _ = strconv.ParseInt(x, 10, 64)
	case int:
		n = int64(x)
	case int64:
		n = x
	}
	if n < 0 {
		return 600
	}
	return n
}

func (s *Server) seeRepo(token, repo string) error {
	owner, name, _ := strings.Cut(repo, "/")
	if owner == "" || name == "" {
		return fmt.Errorf("repo must be owner/name")
	}
	_, err := s.Cat.RepoHeader(token, owner, name, "")
	return err
}

func (s *Server) waitChecks(owner, name, sha string, timeout int64) (any, error) {
	read := func() (map[string]any, bool, error) {
		ok, st, err := s.FJ.ChecksGreen(owner, name, sha)
		if err != nil {
			return nil, false, err
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
		if failed {
			return map[string]any{"ok": false, "statuses": st}, true, nil
		}
		if ok && !pending && len(st) > 0 {
			return map[string]any{"ok": true, "statuses": st}, true, nil
		}
		return map[string]any{"ok": false, "statuses": st}, false, nil
	}
	if timeout == 0 {
		out, _, err := read()
		return out, err
	}
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	var last any
	for time.Now().Before(deadline) {
		out, done, err := read()
		if err != nil {
			return nil, err
		}
		last = out
		if done {
			return out, nil
		}
		time.Sleep(3 * time.Second)
	}
	if last == nil {
		out, _, err := read()
		if err != nil {
			return nil, err
		}
		last = out
	}
	m, _ := last.(map[string]any)
	if m == nil {
		m = map[string]any{"ok": false}
	}
	m["timeout"] = true
	return m, nil
}

func (s *Server) listPipes(token, repo, sha, branch, status string, q page.Query) (any, error) {
	if repo == "" {
		return nil, fmt.Errorf("repo required")
	}
	if sha == "" && branch == "" && status == "" {
		return s.Cat.ListPipelines(token, repo, "", q)
	}
	var matched []woodpecker.Pipeline
	need := q.Norm().Page*q.Norm().Size + 1
	pq := page.Query{Page: 1, Size: page.MaxSize}
	for len(matched) < need && pq.Page <= page.MaxWalk {
		res, err := s.Cat.ListPipelines(token, repo, "", pq)
		if err != nil {
			return nil, err
		}
		matched = append(matched, filterPipes(res.Items, sha, branch, status)...)
		if !res.HasMore {
			break
		}
		pq.Page++
	}
	return page.Take(matched, q), nil
}

func filterPipes(items []woodpecker.Pipeline, sha, branch, status string) []woodpecker.Pipeline {
	sha = strings.ToLower(sha)
	status = strings.ToLower(status)
	var out []woodpecker.Pipeline
	for _, p := range items {
		if sha != "" && !strings.HasPrefix(strings.ToLower(p.Commit), sha) {
			continue
		}
		if branch != "" && p.Branch != branch {
			continue
		}
		if status != "" && strings.ToLower(p.Status) != status {
			continue
		}
		out = append(out, p)
	}
	return out
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

func writeRPC(w http.ResponseWriter, id, result any, err *rpcErr) {
	w.Header().Set("Content-Type", "application/json")
	res := rpcRes{JSONRPC: "2.0", ID: id, Result: result, Error: err}
	_ = json.NewEncoder(w).Encode(res)
}

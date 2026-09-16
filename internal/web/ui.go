package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"acahti/internal/page"
)

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func (p *Pages) publishPipe(pipe any) {
	if p.Hub == nil {
		return
	}
	p.Hub.Publish("pipeline.updated", pipe)
}

func (p *Pages) NavTree(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	out, err := p.Cat.NavTree(user)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (p *Pages) Events(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := p.requireJSON(w, r); !ok {
		return
	}
	if p.Hub == nil {
		writeErr(w, http.StatusServiceUnavailable, "events unavailable")
		return
	}
	p.Hub.SSE(w, r)
}

func (p *Pages) Repos(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	q := page.Parse(r)
	if r.URL.Query().Get("teams") == "1" {
		out, err := p.Cat.ListRepoTeams(user, q)
		if err != nil {
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, out)
		return
	}
	out, err := p.Cat.ListRepos(user, r.URL.Query().Get("team"), q)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (p *Pages) Repo(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	out, err := p.Cat.RepoHeader(user, r.PathValue("owner"), r.PathValue("name"), r.URL.Query().Get("ref"))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (p *Pages) RepoContents(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	out, err := p.Cat.RepoContents(user, r.PathValue("owner"), r.PathValue("name"), q.Get("ref"), q.Get("path"), page.Parse(r))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (p *Pages) RepoCommits(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	out, err := p.Cat.ListCommits(user, r.PathValue("owner"), r.PathValue("name"), r.URL.Query().Get("ref"), page.Parse(r))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (p *Pages) RepoBranches(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	owner, name := r.PathValue("owner"), r.PathValue("name")
	branch := r.PathValue("branch")
	if r.Method == http.MethodPatch && branch != "" {
		var body struct {
			Default   *bool `json:"default"`
			Protected *bool `json:"protected"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		if body.Default == nil && body.Protected == nil {
			writeErr(w, http.StatusBadRequest, "default or protected required")
			return
		}
		if body.Default != nil {
			if !*body.Default {
				writeErr(w, http.StatusBadRequest, "cannot unset default; set another branch")
				return
			}
			if err := p.Cat.SetDefaultBranch(user, owner, name, branch); err != nil {
				writeCatErr(w, err)
				return
			}
		}
		if body.Protected != nil {
			if err := p.Cat.SetBranchProtected(user, owner, name, branch, *body.Protected); err != nil {
				writeCatErr(w, err)
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	out, err := p.Cat.ListBranches(user, owner, name, page.Parse(r))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (p *Pages) RepoTags(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	out, err := p.Cat.ListTags(user, r.PathValue("owner"), r.PathValue("name"), page.Parse(r))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (p *Pages) RepoPulls(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	out, err := p.Cat.ListPulls(user, r.PathValue("owner"), r.PathValue("name"), r.URL.Query().Get("state"), page.Parse(r))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (p *Pages) PullComments(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil || n <= 0 {
		writeErr(w, http.StatusBadRequest, "invalid pull number")
		return
	}
	out, err := p.Cat.ListComments(user, r.PathValue("owner"), r.PathValue("name"), n, page.Parse(r))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (p *Pages) Commit(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	sha := r.PathValue("sha")
	if sha == "" {
		writeErr(w, http.StatusBadRequest, "missing commit")
		return
	}
	out, err := p.Cat.CommitDetail(user, r.PathValue("owner"), r.PathValue("name"), sha, page.Parse(r))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (p *Pages) Pull(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil || n <= 0 {
		writeErr(w, http.StatusBadRequest, "invalid pull number")
		return
	}
	if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/merge") {
		out, err := p.Cat.MergePR(user, r.PathValue("owner"), r.PathValue("name"), n)
		if err != nil {
			writeErr(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, out)
		return
	}
	out, err := p.Cat.PRDetail(user, r.PathValue("owner"), r.PathValue("name"), n)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (p *Pages) Pipelines(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	out, err := p.Cat.ListPipelines(user, q.Get("repo"), q.Get("team"), page.Parse(r))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (p *Pages) RepoPipelines(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	repo := r.PathValue("owner") + "/" + r.PathValue("name")
	q := r.URL.Query()
	out, err := p.Cat.ListRepoPipelines(user, repo, q.Get("sha"), q.Get("branch"), q.Get("status"), page.Parse(r))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (p *Pages) TriggerPipeline(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	if _, err := p.Cat.RepoHeader(user, r.PathValue("owner"), r.PathValue("name"), ""); err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	repo := r.PathValue("owner") + "/" + r.PathValue("name")
	branch := r.URL.Query().Get("ref")
	if branch == "" {
		var body struct {
			Ref string `json:"ref"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		branch = body.Ref
	}
	if branch == "" {
		branch = "dev"
	}
	pipe, err := p.WP.Trigger(repo, branch)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	pipe = p.Cat.Remember(pipe)
	p.publishPipe(pipe)
	writeJSON(w, http.StatusOK, pipe)
}

func (p *Pages) Pipeline(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	owner, name := r.PathValue("owner"), r.PathValue("name")
	n, err := strconv.ParseInt(r.PathValue("n"), 10, 64)
	if err != nil || n <= 0 {
		writeErr(w, http.StatusBadRequest, "invalid pipeline number")
		return
	}
	repo := owner + "/" + name
	switch {
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/log"):
		step, _ := strconv.ParseInt(r.URL.Query().Get("step"), 10, 64)
		text, err := p.Cat.StepLog(user, repo, n, step)
		if err != nil {
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"log": text})
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/rerun"):
		if _, err := p.Cat.RepoHeader(user, owner, name, ""); err != nil {
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		pipe, err := p.WP.Rerun(repo, n)
		if err != nil {
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		pipe = p.Cat.Remember(pipe)
		p.publishPipe(pipe)
		writeJSON(w, http.StatusOK, pipe)
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/cancel"):
		pipe, err := p.Cat.CancelPipeline(user, repo, n)
		if err != nil {
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		p.publishPipe(pipe)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "pipeline": pipe})
	case r.Method == http.MethodDelete:
		if err := p.Cat.DeletePipeline(user, repo, n); err != nil {
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/approve"):
		if _, err := p.Cat.RepoHeader(user, owner, name, ""); err != nil {
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		if err := p.WP.Approve(repo, n); err != nil {
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		if pipe, err := p.Cat.Refresh(repo, n); err == nil {
			p.publishPipe(pipe)
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		out, err := p.Cat.PipelineDetail(user, repo, n)
		if err != nil {
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func (p *Pages) Packages(w http.ResponseWriter, r *http.Request) {
	user, _, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	kind, name := r.PathValue("kind"), r.PathValue("name")
	q := page.Parse(r)
	if kind != "" && name != "" {
		out, err := p.Cat.ListPackageVersions(user, kind, name, q)
		if err != nil {
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, out)
		return
	}
	out, err := p.Cat.ListPackageRows(user, r.URL.Query().Get("kind"), q)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

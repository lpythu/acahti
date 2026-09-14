package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
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
	if _, _, ok := p.requireJSON(w, r); !ok {
		return
	}
	repos, err := p.Cat.ListRepos()
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"repos": repos})
}

func (p *Pages) Repo(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := p.requireJSON(w, r); !ok {
		return
	}
	q := r.URL.Query()
	out, err := p.Cat.RepoOverview(r.PathValue("owner"), r.PathValue("name"), q.Get("ref"), q.Get("path"))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (p *Pages) Pull(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := p.requireJSON(w, r); !ok {
		return
	}
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil || n <= 0 {
		writeErr(w, http.StatusBadRequest, "invalid pull number")
		return
	}
	if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/merge") {
		out, err := p.Cat.MergePR(r.PathValue("owner"), r.PathValue("name"), n)
		if err != nil {
			writeErr(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, out)
		return
	}
	out, err := p.Cat.PRDetail(r.PathValue("owner"), r.PathValue("name"), n)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (p *Pages) Pipelines(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := p.requireJSON(w, r); !ok {
		return
	}
	pipes, err := p.Cat.ListPipelines(r.URL.Query().Get("repo"))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"pipes": pipes})
}

func (p *Pages) TriggerPipeline(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := p.requireJSON(w, r); !ok {
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
	writeJSON(w, http.StatusOK, pipe)
}

func (p *Pages) Pipeline(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := p.requireJSON(w, r); !ok {
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
		text, err := p.Cat.StepLog(repo, n, step)
		if err != nil {
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"log": text})
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/rerun"):
		pipe, err := p.WP.Rerun(repo, n)
		if err != nil {
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, pipe)
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/approve"):
		if err := p.WP.Approve(repo, n); err != nil {
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		out, err := p.Cat.PipelineDetail(repo, n)
		if err != nil {
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func (p *Pages) Packages(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := p.requireJSON(w, r); !ok {
		return
	}
	kind, name := r.PathValue("kind"), r.PathValue("name")
	if kind != "" && name != "" {
		g, err := p.Cat.PackageGroup(kind, name)
		if err != nil {
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, g)
		return
	}
	pkgs, err := p.Cat.ListPackageRows(r.URL.Query().Get("kind"))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"packages": pkgs})
}


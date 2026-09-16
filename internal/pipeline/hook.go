package pipeline

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type fileMeta struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type configRequest struct {
	Configuration []fileMeta `json:"configuration"`
	Pipeline      struct {
		Author string `json:"author"`
		Commit string `json:"commit"`
		Event  string `json:"event"`
		Branch string `json:"branch"`
		Ref    string `json:"ref"`
	} `json:"pipeline"`
	Repo struct {
		FullName string `json:"full_name"`
		Owner    string `json:"owner"`
		Name     string `json:"name"`
	} `json:"repo"`
}

type configResponse struct {
	Configs []fileMeta `json:"configs"`
}

func repoFullName(req configRequest) string {
	if n := strings.TrimSpace(req.Repo.FullName); n != "" {
		return n
	}
	owner := strings.TrimSpace(req.Repo.Owner)
	name := strings.TrimSpace(req.Repo.Name)
	if owner != "" && name != "" {
		return owner + "/" + name
	}
	return ""
}

// HandleConfig expands pipe: in pipeline YAML. Auth is basic user "pipe" or query token=.
// issue mints an Acahti token for the user who triggered the run (job identity).
func HandleConfig(token string, issue func(author, repo, sha string) Ident) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		q := r.URL.Query().Get("token")
		authed := (ok && user == "pipe" && pass == token) || q == token
		if token == "" || !authed {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		var req configRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		var id Ident
		repo := repoFullName(req)
		if issue != nil {
			id = issue(strings.TrimSpace(req.Pipeline.Author), repo, strings.TrimSpace(req.Pipeline.Commit))
		}
		id.Repo = repo
		files := foldFinishJobs(req.Configuration, req.Pipeline.Event, req.Pipeline.Branch, req.Pipeline.Ref)
		out := make([]fileMeta, 0, len(files))
		for _, cfg := range files {
			data, err := ExpandFile(cfg.Name, []byte(cfg.Data), id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			out = append(out, fileMeta{Name: cfg.Name, Data: string(data)})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(configResponse{Configs: out})
	}
}

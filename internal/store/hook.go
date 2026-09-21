package store

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"acahti/internal/woodpecker"
)

type hookPipe struct {
	Repo     string
	Number   int64
	Pipeline woodpecker.Pipeline
}

func ParseWoodpecker(raw []byte) (hookPipe, bool) {
	var body struct {
		Repo struct {
			FullName string `json:"full_name"`
			Name     string `json:"name"`
			Owner    string `json:"owner"`
		} `json:"repo"`
		Pipeline json.RawMessage `json:"pipeline"`
	}
	if json.Unmarshal(raw, &body) != nil {
		return hookPipe{}, false
	}
	repo := strings.TrimSpace(body.Repo.FullName)
	if repo == "" && body.Repo.Owner != "" && body.Repo.Name != "" {
		repo = body.Repo.Owner + "/" + body.Repo.Name
	}
	if repo == "" || len(body.Pipeline) == 0 {
		return hookPipe{}, false
	}
	var p woodpecker.Pipeline
	p, err := woodpecker.DecodeKernel(body.Pipeline)
	if err != nil || p.Number == 0 {
		return hookPipe{}, false
	}
	p.Repo = repo
	p.HydrateJobs()
	return hookPipe{Repo: repo, Number: p.Number, Pipeline: p}, true
}

func ForgejoStatusURL(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	if s, _ := payload["target_url"].(string); strings.TrimSpace(s) != "" {
		return s
	}
	if commit, _ := payload["commit"].(map[string]any); commit != nil {
		if s, _ := commit["target_url"].(string); strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func pipelineNumberFromURL(raw string) int64 {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	// /repos/{id}/{n}/pipelines/{wf}
	if len(parts) >= 4 && parts[0] == "repos" && isDigits(parts[1]) && isDigits(parts[2]) && parts[3] == "pipelines" {
		return parsePos(parts[2])
	}
	// /repos/{owner}/{name}/pipelines/{n}
	if len(parts) >= 5 && parts[0] == "repos" && parts[3] == "pipelines" {
		return parsePos(parts[4])
	}
	// /ci/repos/{owner}/{name}/pipeline/{n}
	if len(parts) >= 6 && parts[0] == "ci" && parts[1] == "repos" && parts[4] == "pipeline" {
		return parsePos(parts[5])
	}
	return 0
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func parsePos(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func ParseForgejoStatus(payload map[string]any) (repo string, number int64, ok bool) {
	if payload == nil {
		return "", 0, false
	}
	repo = repoFrom(payload["repository"])
	if repo == "" {
		return "", 0, false
	}
	n := pipelineNumberFromURL(ForgejoStatusURL(payload))
	if n <= 0 {
		return repo, 0, false
	}
	return repo, n, true
}

func ParseForgejoRepoEvent(payload map[string]any) (action string, repo OrgRepo, ok bool) {
	if payload == nil {
		return "", OrgRepo{}, false
	}
	raw := payload["repository"]
	full := repoFrom(raw)
	if full == "" {
		return "", OrgRepo{}, false
	}
	action, _ = payload["action"].(string)
	if action == "" {
		refType, _ := payload["ref_type"].(string)
		switch {
		case refType == "repository":
			action = "created"
		case forgejoRepoTouch(payload) || refType == "branch" || refType == "tag":
			action = "push"
		default:
			return "", OrgRepo{}, false
		}
	}
	switch action {
	case "created", "deleted", "edited", "transferred", "privatized", "unarchived", "archived", "push":
	default:
		return "", OrgRepo{}, false
	}
	repo = OrgRepo{FullName: full}
	if m, _ := raw.(map[string]any); m != nil {
		repo.DefaultBranch, _ = m["default_branch"].(string)
		repo.Description, _ = m["description"].(string)
		repo.Updated = unixFrom(m["updated_unix"])
		repo.Archived, _ = m["archived"].(bool)
	}
	if action == "archived" {
		repo.Archived = true
	}
	if action == "unarchived" {
		repo.Archived = false
	}
	return action, repo, true
}

func forgejoRepoTouch(payload map[string]any) bool {
	if ref, _ := payload["ref"].(string); strings.HasPrefix(ref, "refs/") {
		return true
	}
	_, ok := payload["commits"]
	return ok
}

func unixFrom(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0
		}
		return i
	default:
		return 0
	}
}

func repoFrom(v any) string {
	m, _ := v.(map[string]any)
	if m == nil {
		return ""
	}
	if s, _ := m["full_name"].(string); s != "" {
		return s
	}
	return ""
}

func ForgejoActor(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	for _, key := range []string{"sender", "pusher", "user"} {
		m, _ := payload[key].(map[string]any)
		if m == nil {
			continue
		}
		if s, _ := m["login"].(string); strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
		if s, _ := m["username"].(string); strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

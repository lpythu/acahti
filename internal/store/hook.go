package store

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"acahti/internal/woodpecker"
)

var pipelineNum = regexp.MustCompile(`/pipeline/(\d+)`)

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
	if json.Unmarshal(body.Pipeline, &p) != nil || p.Number == 0 {
		return hookPipe{}, false
	}
	p.Repo = repo
	p.HydrateJobs()
	return hookPipe{Repo: repo, Number: p.Number, Pipeline: p}, true
}

func ParseForgejoStatus(payload map[string]any) (repo string, number int64, ok bool) {
	if payload == nil {
		return "", 0, false
	}
	repo = repoFrom(payload["repository"])
	if repo == "" {
		return "", 0, false
	}
	target, _ := payload["target_url"].(string)
	if target == "" {
		if commit, _ := payload["commit"].(map[string]any); commit != nil {
			target, _ = commit["target_url"].(string)
		}
	}
	m := pipelineNum.FindStringSubmatch(target)
	if len(m) < 2 {
		return "", 0, false
	}
	n, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil || n <= 0 {
		return "", 0, false
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

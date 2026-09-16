package pipeline

import (
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

// foldFinishJobs drops standalone ci.yaml when a cd.* / pkg job matches this
// event, so one workflow occupies the Runner slot until the train finishes.
func foldFinishJobs(in []fileMeta, event, branch, ref string) []fileMeta {
	if strings.TrimSpace(event) == "" {
		return in
	}
	finish := false
	for _, f := range in {
		if isFinishJob(f.Name) && whenMatches(f.Data, event, branch, ref) {
			finish = true
			break
		}
	}
	if !finish {
		return in
	}
	out := make([]fileMeta, 0, len(in))
	for _, f := range in {
		if isCIJob(f.Name) {
			continue
		}
		f.Data = stripDependsOn(f.Data, "ci")
		out = append(out, f)
	}
	return out
}

func jobBase(name string) string {
	base := path.Base(strings.TrimSpace(name))
	base = strings.ToLower(base)
	base = strings.TrimSuffix(base, ".yaml")
	base = strings.TrimSuffix(base, ".yml")
	return base
}

func isCIJob(name string) bool {
	return jobBase(name) == "ci"
}

func isFinishJob(name string) bool {
	return concurrencyKind(name) != ""
}

func whenMatches(data, event, branch, ref string) bool {
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(data), &doc); err != nil || doc == nil {
		return false
	}
	raw, ok := doc["when"]
	if !ok || raw == nil {
		return true
	}
	for _, clause := range asMaps(raw) {
		if clauseMatches(clause, event, branch, ref) {
			return true
		}
	}
	return false
}

func clauseMatches(clause map[string]any, event, branch, ref string) bool {
	for k := range clause {
		switch strings.ToLower(strings.TrimSpace(k)) {
		case "event", "branch":
		default:
			return false
		}
	}
	events := stringList(clause["event"])
	if len(events) > 0 && !containsFold(events, event) {
		return false
	}
	branches := stringList(clause["branch"])
	if len(branches) == 0 {
		return true
	}
	if containsFold(branches, branch) {
		return true
	}
	for _, b := range branches {
		if strings.EqualFold(ref, "refs/heads/"+b) || strings.EqualFold(ref, "refs/tags/"+b) {
			return true
		}
	}
	return false
}

func stripDependsOn(data, job string) string {
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(data), &doc); err != nil || doc == nil {
		return data
	}
	raw, ok := doc["depends_on"]
	if !ok {
		return data
	}
	kept := make([]any, 0)
	for _, item := range asList(raw) {
		if dependsOnName(item) == job {
			continue
		}
		kept = append(kept, item)
	}
	if len(kept) == 0 {
		delete(doc, "depends_on")
	} else {
		doc["depends_on"] = kept
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return data
	}
	return string(out)
}

func dependsOnName(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case map[string]any:
		return strings.TrimSpace(stringify(t["name"]))
	case map[any]any:
		if n, ok := t["name"]; ok {
			return strings.TrimSpace(stringify(n))
		}
	}
	return strings.TrimSpace(stringify(v))
}

func asMaps(v any) []map[string]any {
	switch t := v.(type) {
	case map[string]any:
		return []map[string]any{t}
	case []any:
		out := make([]map[string]any, 0, len(t))
		for _, item := range t {
			m, ok := item.(map[string]any)
			if ok {
				out = append(out, m)
			}
		}
		return out
	default:
		return nil
	}
}

func asList(v any) []any {
	switch t := v.(type) {
	case nil:
		return nil
	case []any:
		return t
	default:
		return []any{t}
	}
}

func stringList(v any) []string {
	out := make([]string, 0)
	for _, item := range asList(v) {
		s := strings.TrimSpace(stringify(item))
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func containsFold(list []string, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return false
	}
	for _, s := range list {
		if strings.EqualFold(s, want) {
			return true
		}
	}
	return false
}

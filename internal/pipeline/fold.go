package pipeline

import (
	"path"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const ciStepPrefix = "ci-"

// foldFinishJobs inlines ci.yaml steps into a matching cd.* / pkg job so one
// workflow occupies the Runner slot until the train finishes. CI still runs
// first (prefixed step names). Standalone ci.yaml is omitted only after it is
// copied in. PR / push-test keep standalone ci.
func foldFinishJobs(in []fileMeta, event, branch, ref string) []fileMeta {
	if strings.TrimSpace(event) == "" {
		return in
	}
	var ci *fileMeta
	finish := false
	for i, f := range in {
		if isCIJob(f.Name) {
			ci = &in[i]
		}
		if isFinishJob(f.Name) && whenMatches(f.Data, event, branch, ref) {
			finish = true
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
		if ci != nil && isFinishJob(f.Name) && whenMatches(f.Data, event, branch, ref) {
			f.Data = inlineCISteps(f.Data, ci.Data)
		} else {
			f.Data = stripDependsOn(f.Data, "ci")
		}
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

func inlineCISteps(finishData, ciData string) string {
	var finish, ci map[string]any
	if err := yaml.Unmarshal([]byte(finishData), &finish); err != nil || finish == nil {
		return stripDependsOn(finishData, "ci")
	}
	if err := yaml.Unmarshal([]byte(ciData), &ci); err != nil || ci == nil {
		return stripDependsOn(finishData, "ci")
	}
	ciSteps := yamlSteps(ci["steps"])
	if len(ciSteps) == 0 {
		return stripDependsOn(finishData, "ci")
	}
	prefixed := make(map[string]map[string]any, len(ciSteps))
	depended := map[string]struct{}{}
	for name, body := range ciSteps {
		step := cloneStep(body)
		deps := stepDependsOn(body)
		if len(deps) > 0 {
			rewritten := make([]any, 0, len(deps))
			for _, d := range deps {
				rewritten = append(rewritten, ciStepPrefix+d)
				depended[d] = struct{}{}
			}
			step["depends_on"] = rewritten
		}
		prefixed[ciStepPrefix+name] = step
	}
	leaves := make([]string, 0, len(ciSteps))
	for name := range ciSteps {
		if _, ok := depended[name]; !ok {
			leaves = append(leaves, ciStepPrefix+name)
		}
	}
	sort.Strings(leaves)
	if len(leaves) == 0 {
		for name := range prefixed {
			leaves = append(leaves, name)
		}
		sort.Strings(leaves)
	}
	merged := make(map[string]any, len(prefixed)+8)
	for name, step := range prefixed {
		merged[name] = step
	}
	for name, body := range yamlSteps(finish["steps"]) {
		step := cloneStep(body)
		if len(stepDependsOn(body)) == 0 && len(leaves) > 0 {
			deps := make([]any, 0, len(leaves))
			for _, leaf := range leaves {
				deps = append(deps, leaf)
			}
			step["depends_on"] = deps
		}
		merged[name] = step
	}
	finish["steps"] = merged
	out, err := yaml.Marshal(finish)
	if err != nil {
		return stripDependsOn(finishData, "ci")
	}
	return stripDependsOn(string(out), "ci")
}

func yamlSteps(raw any) map[string]map[string]any {
	out := map[string]map[string]any{}
	switch t := raw.(type) {
	case map[string]any:
		for name, body := range t {
			m, ok := body.(map[string]any)
			if !ok || strings.TrimSpace(name) == "" {
				continue
			}
			out[name] = m
		}
	case []any:
		for i, item := range t {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			name := strings.TrimSpace(stringify(m["name"]))
			if name == "" {
				name = stringify(i)
			}
			out[name] = m
		}
	}
	return out
}

func cloneStep(body map[string]any) map[string]any {
	out := make(map[string]any, len(body)+1)
	for k, v := range body {
		out[k] = v
	}
	return out
}

func stepDependsOn(body map[string]any) []string {
	if body == nil {
		return nil
	}
	return stringList(body["depends_on"])
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

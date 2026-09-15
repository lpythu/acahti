package woodpecker

import (
	"path"
	"strings"
)

func (e PipeError) File() string {
	m, ok := e.Data.(map[string]any)
	if !ok {
		return ""
	}
	s, _ := m["file"].(string)
	return s
}

func jobNameFromFile(file string) string {
	file = strings.TrimSpace(file)
	if file == "" || file == "." {
		return ""
	}
	base := path.Base(file)
	if base == "." || base == "/" || base == "" {
		return ""
	}
	base = strings.TrimSuffix(base, ".yaml")
	base = strings.TrimSuffix(base, ".yml")
	if base == "" || base == "." {
		return ""
	}
	return base
}

func failedStatus(state string) bool {
	switch strings.ToLower(state) {
	case "failure", "error", "failed", "killed", "declined":
		return true
	}
	return false
}

func InFlight(state string) bool {
	switch strings.ToLower(state) {
	case "running", "pending", "blocked":
		return true
	}
	return false
}

// DeleteAllowed mirrors Woodpecker: finished pipelines only.
func DeleteAllowed(state string) bool {
	return !InFlight(state)
}

func (p *Pipeline) flattenError() {
	if strings.TrimSpace(p.Error) != "" {
		return
	}
	for _, e := range p.Errors {
		if e.IsWarning || strings.TrimSpace(e.Message) == "" {
			continue
		}
		p.Error = e.Message
		return
	}
}

func (p *Pipeline) HydrateJobs() {
	p.flattenError()
	if len(p.Jobs) > 0 {
		return
	}
	seen := map[string]bool{}
	for _, e := range p.Errors {
		if e.IsWarning {
			continue
		}
		name := jobNameFromFile(e.File())
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		p.Jobs = append(p.Jobs, syntheticJob(name, p.Status, e.Message))
	}
}

func syntheticJob(name, state, err string) Job {
	return Job{
		Name:  name,
		State: state,
		Steps: []Step{{
			PID:   1,
			Name:  name,
			State: state,
			Error: err,
		}},
	}
}

func MergeDeclaredJobs(p Pipeline, declared []string) Pipeline {
	p.HydrateJobs()
	if len(declared) == 0 {
		return p
	}
	byName := make(map[string]Job, len(p.Jobs))
	order := make([]string, 0, len(declared)+len(p.Jobs))
	seen := map[string]bool{}
	for _, name := range declared {
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		order = append(order, name)
	}
	real := 0
	for _, j := range p.Jobs {
		if j.Name == "" || j.Name == "pipeline" || j.Name == "." {
			continue
		}
		real++
		byName[j.Name] = j
		if seen[j.Name] {
			continue
		}
		seen[j.Name] = true
		order = append(order, j.Name)
	}
	failedFiles := map[string]string{}
	for _, e := range p.Errors {
		if e.IsWarning {
			continue
		}
		if name := jobNameFromFile(e.File()); name != "" {
			failedFiles[name] = e.Message
		}
	}
	skip := "skipped"
	if InFlight(p.Status) {
		skip = "pending"
	}
	failName := ""
	if real == 0 && failedStatus(p.Status) {
		failName = primaryJob(declared)
	}
	out := make([]Job, 0, len(order))
	for _, name := range order {
		if j, ok := byName[name]; ok {
			out = append(out, j)
			continue
		}
		state := skip
		err := ""
		if msg, ok := failedFiles[name]; ok {
			state = p.Status
			err = msg
		} else if name == failName {
			state = p.Status
			err = p.Error
		}
		out = append(out, syntheticJob(name, state, err))
	}
	p.Jobs = out
	return p
}

func primaryJob(names []string) string {
	best := ""
	bestRank := 99
	for _, n := range names {
		if n == "" || n == "pipeline" {
			continue
		}
		r := declaredRank(n)
		if best == "" || r < bestRank || (r == bestRank && n < best) {
			best = n
			bestRank = r
		}
	}
	return best
}

func declaredRank(name string) int {
	n := strings.ToLower(name)
	switch {
	case strings.HasPrefix(n, "ci"), strings.HasPrefix(n, "build"), strings.HasPrefix(n, "test"), strings.HasPrefix(n, "lint"), strings.HasPrefix(n, "check"):
		return 0
	case strings.HasPrefix(n, "cd"), strings.HasPrefix(n, "deploy"), strings.HasPrefix(n, "release"):
		return 2
	default:
		return 1
	}
}

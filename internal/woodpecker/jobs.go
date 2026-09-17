package woodpecker

import (
	"path"
	"sort"
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

func InFlight(state string) bool {
	switch strings.ToLower(state) {
	case "running", "started", "pending", "created", "blocked":
		return true
	}
	return false
}

func jobFailed(state string) bool {
	switch strings.ToLower(state) {
	case "failure", "error", "killed", "declined":
		return true
	}
	return false
}

// PendingAfterFailure is Woodpecker leaving a depends_on child in the
// concurrency queue after its parent already failed. The pipeline stays
// running (or already failed) until a worker later skips that child.
func PendingAfterFailure(p Pipeline) bool {
	failed, pending, running := false, false, false
	for _, j := range p.Jobs {
		switch {
		case jobFailed(j.State):
			failed = true
		case strings.EqualFold(j.State, "running"), strings.EqualFold(j.State, "started"):
			running = true
		case strings.EqualFold(j.State, "pending"), strings.EqualFold(j.State, "created"):
			pending = true
		}
	}
	return failed && pending && !running
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

func (p *Pipeline) SortJobs() {
	sort.SliceStable(p.Jobs, func(i, j int) bool {
		if p.Jobs[i].PID != p.Jobs[j].PID {
			return p.Jobs[i].PID < p.Jobs[j].PID
		}
		return p.Jobs[i].Name < p.Jobs[j].Name
	})
}

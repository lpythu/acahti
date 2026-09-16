package woodpecker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const (
	WaitQueue       = "queue"
	WaitDeps        = "deps"
	WaitConcurrency = "concurrency"
)

type QueueStats struct {
	WorkerCount        int `json:"worker_count"`
	PendingCount       int `json:"pending_count"`
	WaitingOnDepsCount int `json:"waiting_on_deps_count"`
	RunningCount       int `json:"running_count"`
}

type QueueTask struct {
	ID               string `json:"id,omitempty"`
	Name             string `json:"name"`
	Repo             string `json:"repo,omitempty"`
	Number           int64  `json:"pipeline_number,omitempty"`
	PID              int64  `json:"pid,omitempty"`
	RepoID           int64  `json:"-"`
	PipelineID       int64  `json:"-"`
	AgentID          int64  `json:"-"`
	Agent            string `json:"agent,omitempty"`
	Wait             string `json:"wait,omitempty"`
	QueuePosition    int    `json:"queue_position,omitempty"`
	ConcurrencyGroup string `json:"concurrency_group,omitempty"`
	ConcurrencyLimit int    `json:"concurrency_limit,omitempty"`
}

type QueueInfo struct {
	Paused        bool        `json:"paused"`
	Stats         QueueStats  `json:"stats"`
	Pending       []QueueTask `json:"pending"`
	WaitingOnDeps []QueueTask `json:"waiting_on_deps"`
	Running       []QueueTask `json:"running"`
}

type kernelTask struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	PID              int64  `json:"pid"`
	RepoID           int64  `json:"repo_id"`
	PipelineID       int64  `json:"pipeline_id"`
	PipelineNumber   int64  `json:"pipeline_number"`
	AgentID          int64  `json:"agent_id"`
	AgentName        string `json:"agent_name"`
	ConcurrencyGroup string `json:"concurrency_group"`
	ConcurrencyLimit int    `json:"concurrency_limit"`
}

func StripWait(p Pipeline) Pipeline {
	p.Wait = ""
	p.QueuePosition = 0
	p.Agent = ""
	for i := range p.Jobs {
		p.Jobs[i].Wait = ""
		p.Jobs[i].QueuePosition = 0
		p.Jobs[i].Agent = ""
	}
	return p
}

func WaitFingerprint(p Pipeline) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s|%s|%d|%s", p.Status, p.Wait, p.QueuePosition, p.Agent)
	for _, j := range p.Jobs {
		fmt.Fprintf(&b, "|%s:%s:%s:%d:%s", j.Name, j.State, j.Wait, j.QueuePosition, j.Agent)
	}
	return b.String()
}

func Annotate(p Pipeline, q QueueInfo) Pipeline {
	p = StripWait(p)
	if !InFlight(p.Status) {
		return p
	}
	for i := range p.Jobs {
		annotateJob(&p.Jobs[i], p, q)
	}
	p.Wait, p.QueuePosition, p.Agent = pipelineWait(p)
	return p
}

func annotateJob(j *Job, p Pipeline, q QueueInfo) {
	if t, ok := findTask(q.Running, p, j.Name); ok {
		j.Wait = ""
		j.Agent = t.Agent
		if strings.EqualFold(j.State, "pending") {
			j.State = "running"
		}
		return
	}
	if t, ok := findTask(q.WaitingOnDeps, p, j.Name); ok {
		j.Wait = WaitDeps
		j.Agent = t.Agent
		return
	}
	if t, ok := findTask(q.Pending, p, j.Name); ok {
		j.Wait = t.Wait
		if j.Wait == "" {
			j.Wait = WaitQueue
		}
		j.QueuePosition = t.QueuePosition
		j.Agent = t.Agent
		return
	}
	if strings.EqualFold(j.State, "pending") {
		j.Wait = inferWait(p, *j)
	}
}

func findTask(tasks []QueueTask, p Pipeline, job string) (QueueTask, bool) {
	for _, t := range tasks {
		if !taskMatchesPipeline(t, p) {
			continue
		}
		if t.Name == "" || t.Name == job {
			return t, true
		}
	}
	return QueueTask{}, false
}

func taskMatchesPipeline(t QueueTask, p Pipeline) bool {
	if t.Number > 0 && t.Number == p.Number {
		return t.Repo == "" || t.Repo == p.Repo
	}
	if t.PipelineID > 0 && t.PipelineID == p.ID {
		return true
	}
	return false
}

func inferWait(p Pipeline, job Job) string {
	rank := declaredRank(job.Name)
	for _, other := range p.Jobs {
		if other.Name == job.Name {
			continue
		}
		if declaredRank(other.Name) >= rank {
			continue
		}
		st := strings.ToLower(other.State)
		if InFlight(st) || st == "" {
			return WaitDeps
		}
		if st != "success" && st != "skipped" {
			return WaitDeps
		}
	}
	return WaitQueue
}

func pipelineWait(p Pipeline) (wait string, pos int, agent string) {
	for _, j := range p.Jobs {
		if strings.EqualFold(j.State, "running") {
			return "", 0, j.Agent
		}
	}
	rank := map[string]int{WaitQueue: 3, WaitConcurrency: 2, WaitDeps: 1}
	for _, j := range p.Jobs {
		if rank[j.Wait] > rank[wait] {
			wait = j.Wait
			pos = j.QueuePosition
			agent = j.Agent
			continue
		}
		if j.Wait == wait && wait == WaitQueue && j.QueuePosition > 0 && (pos == 0 || j.QueuePosition < pos) {
			pos = j.QueuePosition
		}
	}
	if wait == "" && strings.EqualFold(p.Status, "pending") {
		wait = WaitQueue
	}
	return wait, pos, agent
}

func PaintAgents(agents []Agent, q QueueInfo) []Agent {
	byName := map[string]int{}
	byID := map[int64]int{}
	for _, t := range q.Running {
		if t.Agent != "" {
			byName[t.Agent]++
			continue
		}
		if t.AgentID != 0 {
			byID[t.AgentID]++
		}
	}
	out := append([]Agent(nil), agents...)
	for i, a := range out {
		if n, ok := byName[a.Name]; ok {
			out[i].Running = n
			continue
		}
		if n, ok := byID[a.ID]; ok {
			out[i].Running = n
		}
	}
	return out
}

func (c *Client) QueueInfo() (QueueInfo, error) {
	b, _, err := c.do(http.MethodGet, "/api/queue/info", nil)
	if err != nil {
		return QueueInfo{}, err
	}
	var raw struct {
		Paused        bool         `json:"paused"`
		Pending       []kernelTask `json:"pending"`
		WaitingOnDeps []kernelTask `json:"waiting_on_deps"`
		Running       []kernelTask `json:"running"`
		Stats         QueueStats   `json:"stats"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return QueueInfo{}, err
	}
	q := QueueInfo{Paused: raw.Paused, Stats: raw.Stats}
	q.Pending = c.publicTasks(raw.Pending, WaitQueue)
	q.WaitingOnDeps = c.publicTasks(raw.WaitingOnDeps, WaitDeps)
	q.Running = c.publicTasks(raw.Running, "")
	if q.Stats.PendingCount == 0 {
		q.Stats.PendingCount = len(q.Pending)
	}
	if q.Stats.WaitingOnDepsCount == 0 {
		q.Stats.WaitingOnDepsCount = len(q.WaitingOnDeps)
	}
	if q.Stats.RunningCount == 0 {
		q.Stats.RunningCount = len(q.Running)
	}
	return q, nil
}

func (c *Client) publicTasks(in []kernelTask, wait string) []QueueTask {
	out := make([]QueueTask, 0, len(in))
	for i, t := range in {
		task := QueueTask{
			ID:               t.ID,
			Name:             t.Name,
			PID:              t.PID,
			RepoID:           t.RepoID,
			PipelineID:       t.PipelineID,
			Number:           t.PipelineNumber,
			AgentID:          t.AgentID,
			Agent:            t.AgentName,
			ConcurrencyGroup: t.ConcurrencyGroup,
			ConcurrencyLimit: t.ConcurrencyLimit,
			Wait:             wait,
		}
		if wait == WaitQueue {
			task.QueuePosition = i + 1
			if t.ConcurrencyLimit > 0 {
				task.Wait = WaitConcurrency
			}
		}
		if name, err := c.repoName(t.RepoID); err == nil {
			task.Repo = name
		}
		out = append(out, task)
	}
	return out
}

func (c *Client) repoName(id int64) (string, error) {
	if id == 0 {
		return "", fmt.Errorf("empty repo id")
	}
	c.mu.Lock()
	name, ok := c.names[id]
	c.mu.Unlock()
	if ok {
		return name, nil
	}
	b, _, err := c.do(http.MethodGet, "/api/repos/"+strconv.FormatInt(id, 10), nil)
	if err != nil {
		return "", err
	}
	var repo Repo
	if err := json.Unmarshal(b, &repo); err != nil {
		return "", err
	}
	if repo.FullName == "" {
		return "", fmt.Errorf("woodpecker repo %d has no name", id)
	}
	c.rememberID(repo.FullName, repo.ID)
	if repo.ID != 0 && repo.ID != id {
		c.rememberID(repo.FullName, id)
	}
	return repo.FullName, nil
}

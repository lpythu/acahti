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

// QueueStats counts pipelines (one run), not jobs. A pipeline with a running
// ci and a cd still waiting on deps counts as running.
type QueueStats struct {
	WorkerCount  int `json:"worker_count"`
	PendingCount int `json:"pending_count"`
	RunningCount int `json:"running_count"`
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
	Fetched       bool        `json:"-"`
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
	for i := range p.Jobs {
		annotateJob(&p.Jobs[i], p, q)
	}
	if BlockedOnFailed(p) {
		p.Status = "failure"
		p.Wait = ""
		p.QueuePosition = 0
		p.Agent = ""
		return p
	}
	if !InFlight(p.Status) {
		return p
	}
	if q.Fetched && !InQueue(p, q) {
		p.Wait = ""
		p.QueuePosition = 0
		p.Agent = ""
		p.Status = settleFromJobs(p)
		return p
	}
	p.Wait, p.QueuePosition, p.Agent = pipelineWait(p)
	return p
}

func settleFromJobs(p Pipeline) string {
	if len(p.Jobs) == 0 {
		if InFlight(p.Status) {
			return "pending"
		}
		return p.Status
	}
	run, fail, pending := false, false, false
	for _, j := range p.Jobs {
		st := strings.ToLower(j.State)
		if st == "running" {
			run = true
		}
		if failedStatus(st) {
			fail = true
		}
		if st == "pending" || st == "blocked" {
			pending = true
		}
	}
	if run {
		return "running"
	}
	if fail {
		return "failure"
	}
	if pending {
		return "pending"
	}
	if !InFlight(p.Status) {
		return p.Status
	}
	return "success"
}

func InQueue(p Pipeline, q QueueInfo) bool {
	for _, t := range q.Running {
		if taskMatchesPipeline(t, p) {
			return true
		}
	}
	for _, t := range q.Pending {
		if taskMatchesPipeline(t, p) {
			return true
		}
	}
	for _, t := range q.WaitingOnDeps {
		if taskMatchesPipeline(t, p) {
			return true
		}
	}
	return false
}

func BlockedOnFailed(p Pipeline) bool {
	fail, run, leftover := false, false, false
	for _, j := range p.Jobs {
		st := strings.ToLower(j.State)
		if failedStatus(st) {
			fail = true
			continue
		}
		if st == "running" {
			run = true
			continue
		}
		if st == "pending" || st == "skipped" {
			leftover = true
		}
	}
	return fail && !run && leftover
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
	if q.Fetched && strings.EqualFold(j.State, "running") {
		j.State = "pending"
	}
	if upstreamFailed(p, *j) && !strings.EqualFold(j.State, "running") && !strings.EqualFold(j.State, "success") {
		skipOrphan(j)
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
	if strings.EqualFold(j.State, "pending") && (!q.Fetched || InQueue(p, q)) {
		j.Wait = inferWait(p, *j)
	}
}

func skipOrphan(j *Job) {
	j.State = "skipped"
	j.Wait = ""
	j.QueuePosition = 0
	j.Agent = ""
	for i := range j.Steps {
		if strings.EqualFold(j.Steps[i].State, "success") {
			continue
		}
		j.Steps[i].State = "skipped"
		j.Steps[i].Error = ""
	}
}

func upstreamFailed(p Pipeline, job Job) bool {
	rank := declaredRank(job.Name)
	for _, other := range p.Jobs {
		if other.Name == job.Name {
			continue
		}
		if declaredRank(other.Name) >= rank {
			continue
		}
		if failedStatus(other.State) {
			return true
		}
	}
	return false
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
	q := QueueInfo{Paused: raw.Paused, Stats: QueueStats{WorkerCount: raw.Stats.WorkerCount}, Fetched: true}
	q.Pending = c.publicTasks(raw.Pending, WaitQueue)
	q.WaitingOnDeps = c.publicTasks(raw.WaitingOnDeps, WaitDeps)
	q.Running = c.publicTasks(raw.Running, "")
	return q, nil
}

func PipelineStripStats(q QueueInfo, get func(QueueTask) (Pipeline, bool)) QueueStats {
	seen := map[string]bool{}
	var s QueueStats
	consider := func(tasks []QueueTask, fallback string) {
		for _, t := range tasks {
			k := pipelineKey(t)
			if k == "" || seen[k] {
				continue
			}
			seen[k] = true
			p, ok := get(t)
			if !ok {
				if fallback == "" {
					continue
				}
				p = Pipeline{Repo: t.Repo, Number: t.Number, ID: t.PipelineID, Status: fallback}
			}
			switch stripBucket(Annotate(p, q)) {
			case "running":
				s.RunningCount++
			case "queued":
				s.PendingCount++
			}
		}
	}
	consider(q.Running, "running")
	consider(q.Pending, "pending")
	consider(q.WaitingOnDeps, "")
	return s
}

func stripBucket(p Pipeline) string {
	if strings.EqualFold(p.Status, "running") && p.Wait == "" {
		return "running"
	}
	if p.Wait == WaitQueue || p.Wait == WaitConcurrency {
		return "queued"
	}
	return ""
}

func pipelineKey(t QueueTask) string {
	if t.Repo != "" && t.Number > 0 {
		return t.Repo + "#" + strconv.FormatInt(t.Number, 10)
	}
	if t.PipelineID > 0 {
		return "id:" + strconv.FormatInt(t.PipelineID, 10)
	}
	if t.Number > 0 {
		return "n:" + strconv.FormatInt(t.Number, 10)
	}
	return ""
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

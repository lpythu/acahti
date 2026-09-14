package woodpecker

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"acahti/internal/page"
)

const (
	reposTTL  = 60 * time.Second
	latestTTL = 20 * time.Second
	fanout    = 16
)

type latestEnt struct {
	at time.Time
	p  Pipeline
}

type latestCall struct {
	done chan struct{}
	p    Pipeline
	err  error
}

type Client struct {
	base      string
	token     string
	http      *http.Client
	mu        sync.Mutex
	loadMu    sync.Mutex
	ids       map[string]int64
	repos     []Repo
	reposAt   time.Time
	latest    map[string]latestEnt
	flight    sync.Map
	loadErr   error
	loadErrAt time.Time
}

func New(base, token string) *Client {
	return &Client{
		base:   strings.TrimRight(base, "/"),
		token:  token,
		http:   &http.Client{Timeout: 45 * time.Second},
		ids:    map[string]int64{},
		latest: map[string]latestEnt{},
	}
}

func (c *Client) Ready() bool {
	return c != nil && c.token != ""
}

type Repo struct {
	ID              int64  `json:"id"`
	ForgeRemoteID   string `json:"forge_remote_id"`
	FullName        string `json:"full_name"`
	Name            string `json:"name"`
	ConfigFile      string `json:"config_file"`
	IsActive        bool   `json:"active"`
	RequireApproval string `json:"require_approval"`
}

type Pipeline struct {
	ID       int64  `json:"id"`
	Number   int64  `json:"number"`
	Status   string `json:"status"`
	Event    string `json:"event"`
	Branch   string `json:"branch"`
	Ref      string `json:"ref"`
	Title    string `json:"title"`
	Message  string `json:"message"`
	Author   string `json:"author"`
	Avatar   string `json:"avatar"`
	Commit   string      `json:"commit"`
	Error    string      `json:"error"`
	Errors   []PipeError `json:"errors,omitempty"`
	Created  int64       `json:"created"`
	Started  int64       `json:"started"`
	Finished int64       `json:"finished"`
	Jobs     []Job       `json:"jobs,omitempty"`
	Repo     string      `json:"repo,omitempty"`
}

type PipeError struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	IsWarning bool   `json:"is_warning"`
	Data      any    `json:"data,omitempty"`
}

type Job struct {
	ID       int64  `json:"id"`
	PID      int64  `json:"pid"`
	Name     string `json:"name"`
	State    string `json:"state"`
	Children []Step `json:"children"`
}

type Step struct {
	ID    int64  `json:"id"`
	PID   int64  `json:"pid"`
	PPID  int64  `json:"ppid"`
	Name  string `json:"name"`
	State string `json:"state"`
	Error string `json:"error"`
	Type  string `json:"type"`
}

func (p *Pipeline) UnmarshalJSON(data []byte) error {
	type alias Pipeline
	aux := struct {
		alias
		Workflows []Job `json:"workflows"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*p = Pipeline(aux.alias)
	if len(p.Jobs) == 0 {
		p.Jobs = aux.Workflows
	}
	p.flattenError()
	return nil
}

func (p Pipeline) Steps() []Step {
	var out []Step
	for _, job := range p.Jobs {
		if len(job.Children) == 0 && job.PID > 0 {
			out = append(out, Step{PID: job.PID, Name: job.Name, State: job.State})
			continue
		}
		out = append(out, job.Children...)
	}
	if len(out) == 0 {
		out = []Step{{PID: 1, Name: "", State: p.Status, Error: p.Error}}
	}
	return out
}

func FormatLog(raw string) string {
	var lines []struct {
		Out  string `json:"out"`
		Data string `json:"data"`
	}
	if json.Unmarshal([]byte(raw), &lines) == nil && len(lines) > 0 {
		var b strings.Builder
		for _, l := range lines {
			if l.Out != "" {
				b.WriteString(l.Out)
				continue
			}
			if l.Data == "" {
				continue
			}
			dec, err := base64.StdEncoding.DecodeString(l.Data)
			if err != nil {
				b.WriteString(l.Data)
				continue
			}
			b.Write(dec)
		}
		return b.String()
	}
	return raw
}

type Agent struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Version  string `json:"version"`
	LastWork int64  `json:"last_work"`
	LastSeen int64  `json:"last_contact"`
	Labels   any    `json:"labels"`
}

func (c *Client) csrf() string {
	req, err := http.NewRequest(http.MethodGet, c.base+"/web-config.js", nil)
	if err != nil {
		return ""
	}
	if c.token != "" {
		req.Header.Set("Cookie", "user_sess="+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	const p = `WOODPECKER_CSRF = "`
	s := string(b)
	i := strings.Index(s, p)
	if i < 0 {
		return ""
	}
	s = s[i+len(p):]
	j := strings.Index(s, `"`)
	if j < 0 {
		return ""
	}
	return s[:j]
}

func (c *Client) do(method, path string, body any) ([]byte, int, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, rdr)
	if err != nil {
		return nil, 0, err
	}
	timeout := 8 * time.Second
	if strings.Contains(path, "/logs/") {
		timeout = 45 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req = req.WithContext(ctx)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Cookie", "user_sess="+c.token)
		if method != http.MethodGet && method != http.MethodHead {
			if csrf := c.csrf(); csrf != "" {
				req.Header.Set("X-CSRF-TOKEN", csrf)
			}
		}
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return b, resp.StatusCode, fmt.Errorf("woodpecker %s %s: %d %s", method, path, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return b, resp.StatusCode, nil
}

func (c *Client) ListRepos(q page.Query) (page.Result[Repo], error) {
	q = q.Norm()
	path := fmt.Sprintf("/api/user/repos?page=%d&perPage=%d", q.Page, q.LimitPlus())
	b, _, err := c.do(http.MethodGet, path, nil)
	if err != nil {
		return page.Result[Repo]{}, err
	}
	var out []Repo
	if err := json.Unmarshal(b, &out); err != nil {
		return page.Result[Repo]{}, err
	}
	return page.Clip(out, q), nil
}

func (c *Client) Activate(fullName, forgeRemoteID string) error {
	if forgeRemoteID == "" {
		return fmt.Errorf("empty forge_remote_id")
	}
	b, status, err := c.do(http.MethodPost, "/api/repos?forge_remote_id="+url.QueryEscape(forgeRemoteID), nil)
	id := int64(0)
	if err == nil {
		var repo Repo
		if json.Unmarshal(b, &repo) == nil {
			id = repo.ID
		}
	} else if status != http.StatusConflict {
		return err
	}
	c.forgetIDs()
	if id == 0 {
		looked, lerr := c.lookup(fullName)
		if lerr != nil {
			return lerr
		}
		id = looked.ID
	}
	if id == 0 {
		return fmt.Errorf("woodpecker repo %s has no id", fullName)
	}
	_, _, err = c.do(http.MethodPatch, "/api/repos/"+strconv.FormatInt(id, 10), map[string]any{
		"config_file": ".acahti/pipelines/",
	})
	return err
}

func (c *Client) lookup(fullName string) (Repo, error) {
	owner, name, ok := strings.Cut(fullName, "/")
	if !ok || owner == "" || name == "" {
		return Repo{}, fmt.Errorf("repo %s", fullName)
	}
	b, _, err := c.do(http.MethodGet, "/api/repos/lookup/"+url.PathEscape(owner)+"/"+url.PathEscape(name), nil)
	if err != nil {
		return Repo{}, err
	}
	var repo Repo
	if err := json.Unmarshal(b, &repo); err != nil {
		return Repo{}, err
	}
	return repo, nil
}

func (c *Client) forgetIDs() {
	c.mu.Lock()
	c.ids = map[string]int64{}
	c.repos = nil
	c.reposAt = time.Time{}
	c.latest = map[string]latestEnt{}
	c.loadErr = nil
	c.loadErrAt = time.Time{}
	c.mu.Unlock()
}

func (c *Client) lookupID(fullName string) (int64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.reposAt.IsZero() || time.Since(c.reposAt) > reposTTL {
		return 0, false
	}
	id, ok := c.ids[fullName]
	return id, ok
}

func (c *Client) loadIDs() error {
	repos, err := page.Walk(func(q page.Query) (page.Result[Repo], error) {
		return c.ListRepos(q)
	})
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ids = make(map[string]int64, len(repos)*2)
	c.repos = append([]Repo{}, repos...)
	c.reposAt = time.Now()
	for _, r := range repos {
		if r.FullName != "" {
			c.ids[r.FullName] = r.ID
		}
		if r.Name != "" {
			c.ids[r.Name] = r.ID
		}
	}
	return nil
}

func (c *Client) reposFresh() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return !c.reposAt.IsZero() && time.Since(c.reposAt) <= reposTTL
}

func (c *Client) ensureRepos() error {
	if c.reposFresh() {
		return nil
	}
	c.loadMu.Lock()
	defer c.loadMu.Unlock()
	if c.reposFresh() {
		return nil
	}
	c.mu.Lock()
	if c.loadErr != nil && time.Since(c.loadErrAt) < 5*time.Second {
		err := c.loadErr
		c.mu.Unlock()
		return err
	}
	c.mu.Unlock()
	err := c.loadIDs()
	c.mu.Lock()
	c.loadErr = err
	c.loadErrAt = time.Now()
	c.mu.Unlock()
	return err
}

func (c *Client) CachedRepos() ([]Repo, error) {
	if err := c.ensureRepos(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]Repo{}, c.repos...), nil
}

func (c *Client) repoKey(fullName string) (string, error) {
	if fullName == "" {
		return "", fmt.Errorf("empty repo")
	}
	if _, err := strconv.ParseInt(fullName, 10, 64); err == nil {
		return fullName, nil
	}
	if err := c.ensureRepos(); err != nil {
		return "", err
	}
	if id, ok := c.lookupID(fullName); ok {
		return strconv.FormatInt(id, 10), nil
	}
	return "", fmt.Errorf("woodpecker repo %s not found", fullName)
}

func (c *Client) cachedLatest(fullName string) (Pipeline, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.latest[fullName]
	if !ok || time.Since(e.at) > latestTTL {
		return Pipeline{}, false
	}
	return e.p, true
}

func (c *Client) storeLatest(fullName string, p Pipeline) {
	c.mu.Lock()
	c.latest[fullName] = latestEnt{at: time.Now(), p: p}
	c.mu.Unlock()
}

func (c *Client) forgetLatest(fullName string) {
	c.mu.Lock()
	delete(c.latest, fullName)
	c.mu.Unlock()
}

func (c *Client) latestListed(fullName string) (Pipeline, error) {
	if p, ok := c.cachedLatest(fullName); ok {
		return p, nil
	}
	call := &latestCall{done: make(chan struct{})}
	if actual, loaded := c.flight.LoadOrStore(fullName, call); loaded {
		wait := actual.(*latestCall)
		<-wait.done
		return wait.p, wait.err
	}
	defer func() {
		close(call.done)
		c.flight.Delete(fullName)
	}()
	res, err := c.ListPipelines(fullName, page.Query{Page: 1, Size: 1})
	if err != nil {
		call.err = err
		return Pipeline{}, err
	}
	if len(res.Items) == 0 {
		call.err = fmt.Errorf("no pipelines")
		return Pipeline{}, call.err
	}
	p := res.Items[0]
	c.storeLatest(fullName, p)
	call.p = p
	return p, nil
}

func (c *Client) latestPipe(fullName string, jobs bool) (Pipeline, error) {
	p, err := c.latestListed(fullName)
	if err != nil {
		return Pipeline{}, err
	}
	if !jobs || len(p.Jobs) > 0 {
		return p, nil
	}
	detail, err := c.GetPipeline(fullName, p.Number)
	if err != nil {
		return p, nil
	}
	detail.Repo = fullName
	c.storeLatest(fullName, detail)
	return detail, nil
}

func (c *Client) EnrichJobs(fullName string, pipes []Pipeline) []Pipeline {
	out := append([]Pipeline(nil), pipes...)
	sem := make(chan struct{}, fanout)
	var wg sync.WaitGroup
	for i := range out {
		if len(out[i].Jobs) > 0 || out[i].Number == 0 {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			d, err := c.GetPipeline(fullName, out[i].Number)
			if err != nil {
				return
			}
			d.Repo = fullName
			out[i] = d
		}(i)
	}
	wg.Wait()
	return out
}

func (c *Client) LatestPipelines(names []string, jobs bool) []Pipeline {
	got := make([]Pipeline, len(names))
	ok := make([]bool, len(names))
	sem := make(chan struct{}, fanout)
	var wg sync.WaitGroup
	for i, name := range names {
		if name == "" {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, name string) {
			defer wg.Done()
			defer func() { <-sem }()
			p, err := c.latestPipe(name, jobs)
			if err != nil {
				return
			}
			got[i] = p
			ok[i] = true
		}(i, name)
	}
	wg.Wait()
	out := make([]Pipeline, 0, len(names))
	for i, p := range got {
		if ok[i] {
			out = append(out, p)
		}
	}
	return out
}

func (c *Client) ListPipelines(fullName string, q page.Query) (page.Result[Pipeline], error) {
	q = q.Norm()
	key, err := c.repoKey(fullName)
	if err != nil {
		return page.Result[Pipeline]{}, err
	}
	b, _, err := c.do(http.MethodGet, fmt.Sprintf("/api/repos/%s/pipelines?page=%d&perPage=%d", key, q.Page, q.LimitPlus()), nil)
	if err != nil {
		return page.Result[Pipeline]{}, err
	}
	var out []Pipeline
	if err := json.Unmarshal(b, &out); err != nil {
		return page.Result[Pipeline]{}, err
	}
	for i := range out {
		out[i].Repo = fullName
	}
	return page.Clip(out, q), nil
}

func (c *Client) GetPipeline(fullName string, number int64) (Pipeline, error) {
	key, err := c.repoKey(fullName)
	if err != nil {
		return Pipeline{}, err
	}
	b, _, err := c.do(http.MethodGet, fmt.Sprintf("/api/repos/%s/pipelines/%d", key, number), nil)
	if err != nil {
		return Pipeline{}, err
	}
	var p Pipeline
	if err := json.Unmarshal(b, &p); err != nil {
		return Pipeline{}, err
	}
	p.Repo = fullName
	p.HydrateJobs()
	return p, nil
}

func (c *Client) PipelineLog(fullName string, number, step int64) (string, error) {
	key, err := c.repoKey(fullName)
	if err != nil {
		return "", err
	}
	b, _, err := c.do(http.MethodGet, fmt.Sprintf("/api/repos/%s/logs/%d/%d", key, number, step), nil)
	if err != nil {
		id, resolveErr := c.stepID(fullName, number, step)
		if resolveErr != nil || id == step {
			return "", err
		}
		b, _, err = c.do(http.MethodGet, fmt.Sprintf("/api/repos/%s/logs/%d/%d", key, number, id), nil)
		if err != nil {
			return "", err
		}
	}
	return string(b), nil
}

func (c *Client) stepID(fullName string, number, pid int64) (int64, error) {
	p, err := c.GetPipeline(fullName, number)
	if err != nil {
		return 0, err
	}
	for _, s := range p.Steps() {
		if s.ID == pid || s.PID == pid {
			if s.ID > 0 {
				return s.ID, nil
			}
			return s.PID, nil
		}
	}
	return 0, fmt.Errorf("step %d not found", pid)
}

func (c *Client) Trigger(fullName, branch string) (Pipeline, error) {
	key, err := c.repoKey(fullName)
	if err != nil {
		return Pipeline{}, err
	}
	body := map[string]any{}
	if branch != "" {
		body["branch"] = branch
	}
	b, _, err := c.do(http.MethodPost, "/api/repos/"+key+"/pipelines", body)
	if err != nil {
		return Pipeline{}, err
	}
	c.forgetLatest(fullName)
	var p Pipeline
	p.Repo = fullName
	return p, json.Unmarshal(b, &p)
}

func (c *Client) Rerun(fullName string, number int64) (Pipeline, error) {
	key, err := c.repoKey(fullName)
	if err != nil {
		return Pipeline{}, err
	}
	b, _, err := c.do(http.MethodPost, fmt.Sprintf("/api/repos/%s/pipelines/%d", key, number), nil)
	if err != nil {
		return Pipeline{}, err
	}
	c.forgetLatest(fullName)
	var p Pipeline
	p.Repo = fullName
	return p, json.Unmarshal(b, &p)
}

func (c *Client) Approve(fullName string, number int64) error {
	key, err := c.repoKey(fullName)
	if err != nil {
		return err
	}
	_, _, err = c.do(http.MethodPost, fmt.Sprintf("/api/repos/%s/pipelines/%d/approve", key, number), nil)
	if err == nil {
		c.forgetLatest(fullName)
	}
	return err
}

func (c *Client) Cancel(fullName string, number int64) error {
	key, err := c.repoKey(fullName)
	if err != nil {
		return err
	}
	_, _, err = c.do(http.MethodPost, fmt.Sprintf("/api/repos/%s/pipelines/%d/cancel", key, number), nil)
	if err == nil {
		c.forgetLatest(fullName)
	}
	return err
}

func (c *Client) Agents() ([]Agent, error) {
	b, _, err := c.do(http.MethodGet, "/api/agents", nil)
	if err != nil {
		return nil, err
	}
	var out []Agent
	return out, json.Unmarshal(b, &out)
}

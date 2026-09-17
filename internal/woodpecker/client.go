package woodpecker

import (
	"bytes"
	"context"
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

type Client struct {
	base  string
	token string
	http  *http.Client
	mu    sync.Mutex
	ids   map[string]int64
	names map[int64]string
}

func New(base, token string) *Client {
	return &Client{
		base:  strings.TrimRight(base, "/"),
		token: token,
		http:  &http.Client{Timeout: 45 * time.Second},
		ids:   map[string]int64{},
		names: map[int64]string{},
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
	ID            int64       `json:"id"`
	Number        int64       `json:"number"`
	Status        string      `json:"status"`
	Event         string      `json:"event"`
	Branch        string      `json:"branch"`
	Ref           string      `json:"ref"`
	Title         string      `json:"title"`
	Message       string      `json:"message"`
	Author        string      `json:"author"`
	Avatar        string      `json:"avatar"`
	Commit        string      `json:"commit"`
	Error         string      `json:"error"`
	Errors        []PipeError `json:"errors,omitempty"`
	Created       int64       `json:"created"`
	Started       int64       `json:"started"`
	Finished      int64       `json:"finished"`
	Jobs          []Job       `json:"jobs,omitempty"`
	Repo          string      `json:"repo,omitempty"`
	Wait          string      `json:"wait,omitempty"`
	QueuePosition int         `json:"queue_position,omitempty"`
	Agent         string      `json:"agent,omitempty"`
}

type PipeError struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	IsWarning bool   `json:"is_warning"`
	Data      any    `json:"data,omitempty"`
}

// Job is one `.acahti/pipelines/*.yaml` file in a run.
type Job struct {
	ID            int64    `json:"id,omitempty"`
	PID           int64    `json:"pid,omitempty"`
	Name          string   `json:"name"`
	State         string   `json:"state"`
	DependsOn     []string `json:"depends_on,omitempty"`
	Wait          string   `json:"wait,omitempty"`
	QueuePosition int      `json:"queue_position,omitempty"`
	Agent         string   `json:"agent,omitempty"`
	ArgosSID      string   `json:"argos_sid,omitempty"`
	ArgosURL      string   `json:"argos_url,omitempty"`
	Steps         []Step   `json:"steps,omitempty"`
}

type Step struct {
	ID      int64  `json:"id"`
	PID     int64  `json:"pid"`
	PPID    int64  `json:"ppid"`
	Name    string `json:"name"`
	State   string `json:"state"`
	Error   string `json:"error"`
	Type    string `json:"type"`
	LogTail string `json:"log_tail,omitempty"`
}

func (p Pipeline) Steps() []Step {
	var out []Step
	for _, j := range p.Jobs {
		out = append(out, j.Steps...)
	}
	return out
}

func logText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	if s == "" {
		return ""
	}
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	return s
}

func FormatLog(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" || s == "null" {
		return ""
	}
	var lines []struct {
		Out  string `json:"out"`
		Data []byte `json:"data"`
	}
	if err := json.Unmarshal([]byte(s), &lines); err != nil {
		return logText(raw)
	}
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	for _, l := range lines {
		chunk := l.Out
		if chunk == "" {
			chunk = string(l.Data)
		}
		b.WriteString(logText(chunk))
	}
	return b.String()
}

type Agent struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Backend  string `json:"backend,omitempty"`
	Version  string `json:"version"`
	Capacity int    `json:"capacity"`
	Running  int    `json:"running,omitempty"`
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
	if id > 0 {
		c.rememberID(fullName, id)
	}
	if id == 0 {
		looked, lerr := c.lookup(fullName)
		if lerr != nil {
			return lerr
		}
		id = looked.ID
		c.rememberID(fullName, id)
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

func (c *Client) rememberID(fullName string, id int64) {
	if fullName == "" || id == 0 {
		return
	}
	c.mu.Lock()
	c.ids[fullName] = id
	c.names[id] = fullName
	c.mu.Unlock()
}

func (c *Client) repoKey(fullName string) (string, error) {
	if fullName == "" {
		return "", fmt.Errorf("empty repo")
	}
	if _, err := strconv.ParseInt(fullName, 10, 64); err == nil {
		return fullName, nil
	}
	c.mu.Lock()
	id, ok := c.ids[fullName]
	c.mu.Unlock()
	if ok {
		return strconv.FormatInt(id, 10), nil
	}
	repo, err := c.lookup(fullName)
	if err != nil {
		return "", err
	}
	if repo.ID == 0 {
		return "", fmt.Errorf("woodpecker repo %s not found", fullName)
	}
	c.rememberID(fullName, repo.ID)
	if repo.FullName != "" {
		c.rememberID(repo.FullName, repo.ID)
	}
	return strconv.FormatInt(repo.ID, 10), nil
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
	out, err := DecodeKernelList(b)
	if err != nil {
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
	p, err := DecodeKernel(b)
	if err != nil {
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
	p, err := DecodeKernel(b)
	if err != nil {
		return Pipeline{}, err
	}
	p.Repo = fullName
	return p, nil
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
	p, err := DecodeKernel(b)
	if err != nil {
		return Pipeline{}, err
	}
	p.Repo = fullName
	return p, nil
}

func (c *Client) Approve(fullName string, number int64) error {
	key, err := c.repoKey(fullName)
	if err != nil {
		return err
	}
	_, _, err = c.do(http.MethodPost, fmt.Sprintf("/api/repos/%s/pipelines/%d/approve", key, number), nil)
	return err
}

func (c *Client) Cancel(fullName string, number int64) error {
	key, err := c.repoKey(fullName)
	if err != nil {
		return err
	}
	_, _, err = c.do(http.MethodPost, fmt.Sprintf("/api/repos/%s/pipelines/%d/cancel", key, number), nil)
	return err
}

func (c *Client) Delete(fullName string, number int64) error {
	key, err := c.repoKey(fullName)
	if err != nil {
		return err
	}
	_, _, err = c.do(http.MethodDelete, fmt.Sprintf("/api/repos/%s/pipelines/%d", key, number), nil)
	return err
}

func (c *Client) Agents() ([]Agent, error) {
	b, _, err := c.do(http.MethodGet, "/api/agents", nil)
	if err != nil {
		return nil, err
	}
	var raw []struct {
		Agent
		CustomLabels any `json:"custom_labels"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	out := make([]Agent, 0, len(raw))
	for _, a := range raw {
		got := a.Agent
		if got.Labels == nil {
			got.Labels = a.CustomLabels
		}
		out = append(out, got)
	}
	return out, nil
}

package woodpecker

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	base  string
	token string
	http  *http.Client
}

func New(base, token string) *Client {
	return &Client{
		base:  strings.TrimRight(base, "/"),
		token: token,
		http:  &http.Client{Timeout: 45 * time.Second},
	}
}

func (c *Client) Ready() bool {
	return c != nil && c.token != ""
}

type Repo struct {
	ID              int64  `json:"id"`
	FullName        string `json:"full_name"`
	Name            string `json:"name"`
	IsActive        bool   `json:"active"`
	RequireApproval string `json:"require_approval"`
}

type Pipeline struct {
	ID        int64      `json:"id"`
	Number    int64      `json:"number"`
	Status    string     `json:"status"`
	Event     string     `json:"event"`
	Branch    string     `json:"branch"`
	Ref       string     `json:"ref"`
	Title     string     `json:"title"`
	Message   string     `json:"message"`
	Author    string     `json:"author"`
	Avatar    string     `json:"avatar"`
	Commit    string     `json:"commit"`
	Error     string     `json:"error"`
	Created   int64      `json:"created"`
	Started   int64      `json:"started"`
	Finished  int64      `json:"finished"`
	Workflows []Workflow `json:"workflows"`
	Repo      string     `json:"repo,omitempty"`
}

type Workflow struct {
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

func (p Pipeline) Steps() []Step {
	var out []Step
	for _, wf := range p.Workflows {
		if len(wf.Children) == 0 && wf.PID > 0 {
			out = append(out, Step{PID: wf.PID, Name: wf.Name, State: wf.State})
			continue
		}
		out = append(out, wf.Children...)
	}
	if len(out) == 0 {
		out = []Step{{PID: 1, Name: p.Title, State: p.Status, Error: p.Error}}
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

func (c *Client) ListRepos() ([]Repo, error) {
	b, _, err := c.do(http.MethodGet, "/api/user/repos?all=true", nil)
	if err != nil {
		return nil, err
	}
	var out []Repo
	return out, json.Unmarshal(b, &out)
}

func (c *Client) Activate(fullName string) error {
	enc := url.PathEscape(fullName)
	_, _, err := c.do(http.MethodPost, "/api/repos/"+enc, nil)
	if err != nil {
		_, _, err = c.do(http.MethodPost, "/api/repos?forge_remote_id="+url.QueryEscape(fullName), nil)
	}
	return err
}

func (c *Client) repoKey(fullName string) (string, error) {
	if fullName == "" {
		return "", fmt.Errorf("empty repo")
	}
	if _, err := strconv.ParseInt(fullName, 10, 64); err == nil {
		return fullName, nil
	}
	repos, err := c.ListRepos()
	if err != nil {
		return "", err
	}
	for _, r := range repos {
		if r.FullName == fullName || r.Name == fullName {
			return strconv.FormatInt(r.ID, 10), nil
		}
	}
	return "", fmt.Errorf("woodpecker repo %s not found", fullName)
}

func (c *Client) ListPipelines(fullName string, page int) ([]Pipeline, error) {
	if page <= 0 {
		page = 1
	}
	key, err := c.repoKey(fullName)
	if err != nil {
		return nil, err
	}
	b, _, err := c.do(http.MethodGet, fmt.Sprintf("/api/repos/%s/pipelines?page=%d", key, page), nil)
	if err != nil {
		return nil, err
	}
	var out []Pipeline
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Repo = fullName
	}
	return out, nil
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
	p.Repo = fullName
	return p, json.Unmarshal(b, &p)
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

func (c *Client) RecentPipelines() ([]Pipeline, error) {
	return c.ListRecentPipelines(3)
}

func (c *Client) ListRecentPipelines(perRepo int) ([]Pipeline, error) {
	if perRepo <= 0 {
		perRepo = 20
	}
	repos, err := c.ListRepos()
	if err != nil {
		return nil, err
	}
	var all []Pipeline
	for _, r := range repos {
		if !r.IsActive {
			continue
		}
		ps, err := c.ListPipelines(r.FullName, 1)
		if err != nil {
			continue
		}
		if len(ps) > perRepo {
			ps = ps[:perRepo]
		}
		all = append(all, ps...)
	}
	return all, nil
}

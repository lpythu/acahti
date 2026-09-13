package forgejo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	base  string
	admin string
	http  *http.Client
}

func New(base, adminToken string) *Client {
	return &Client{
		base:  strings.TrimRight(base, "/"),
		admin: adminToken,
		http:  &http.Client{Timeout: 45 * time.Second},
	}
}

func (c *Client) Ready() bool {
	return c != nil && c.admin != ""
}

type User struct {
	ID       int64  `json:"id"`
	Login    string `json:"login"`
	Email    string `json:"email"`
	IsAdmin  bool   `json:"is_admin"`
	FullName string `json:"full_name"`
}

type Repo struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
	CloneURL      string `json:"clone_url"`
	SSHURL        string `json:"ssh_url"`
	HTMLURL       string `json:"html_url"`
	Repo          string `json:"repo,omitempty"`
}

type Branch struct {
	Name   string `json:"name"`
	Commit struct {
		ID string `json:"id"`
	} `json:"commit"`
}

type PR struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Body   string `json:"body"`
	User   User   `json:"user"`
	Head   struct {
		Ref string `json:"ref"`
		SHA string `json:"sha"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
	Mergeable bool   `json:"mergeable"`
	Merged    bool   `json:"merged"`
	URL       string `json:"url"`
	HTMLURL   string `json:"html_url"`
	Repo      string `json:"repo,omitempty"`
}

type Status struct {
	Status      string `json:"status"`
	Context     string `json:"context"`
	Description string `json:"description"`
	TargetURL   string `json:"target_url"`
}

type Package struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"`
}

type PublicKey struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Key   string `json:"key"`
}

type Comment struct {
	ID      int64  `json:"id"`
	Body    string `json:"body"`
	User    User   `json:"user"`
	Created string `json:"created_at"`
}

type AccessToken struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	TokenLastEight string `json:"token_last_eight"`
}

type BranchProtection struct {
	RuleName string `json:"rule_name"`
}

func (c *Client) do(method, path, token, sudo string, body any) ([]byte, int, error) {
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
	if token == "" {
		token = c.admin
	}
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	if sudo != "" {
		req.Header.Set("Sudo", sudo)
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
		return b, resp.StatusCode, fmt.Errorf("forgejo %s %s: %d %s", method, path, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return b, resp.StatusCode, nil
}

func (c *Client) User(token string) (User, error) {
	b, _, err := c.do(http.MethodGet, "/api/v1/user", token, "", nil)
	if err != nil {
		return User{}, err
	}
	var u User
	return u, json.Unmarshal(b, &u)
}

func (c *Client) BasicUser(user, pass string) (User, error) {
	req, err := http.NewRequest(http.MethodGet, c.base+"/api/v1/user", nil)
	if err != nil {
		return User{}, err
	}
	req.SetBasicAuth(user, pass)
	resp, err := c.http.Do(req)
	if err != nil {
		return User{}, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return User{}, fmt.Errorf("login failed")
	}
	var u User
	return u, json.Unmarshal(b, &u)
}

func (c *Client) ListUsers() ([]User, error) {
	b, _, err := c.do(http.MethodGet, "/api/v1/admin/users", "", "", nil)
	if err != nil {
		return nil, err
	}
	var out []User
	return out, json.Unmarshal(b, &out)
}

func (c *Client) SetPassword(login, password string) error {
	_, _, err := c.do(http.MethodPatch, "/api/v1/admin/users/"+url.PathEscape(login), "", "", map[string]any{
		"login_name": login,
		"password":   password,
	})
	return err
}

func (c *Client) CreateUser(login, email, password string, admin bool) (User, error) {
	b, _, err := c.do(http.MethodPost, "/api/v1/admin/users", "", "", map[string]any{
		"username":                  login,
		"email":                     email,
		"password":                  password,
		"must_change_password":      false,
		"restricted":                false,
		"visibility":                "private",
		"created_repo_limit":        -1,
		"max_repo_creation":         -1,
		"allow_create_organization": false,
		"admin":                     admin,
	})
	if err != nil {
		return User{}, err
	}
	var u User
	return u, json.Unmarshal(b, &u)
}

func (c *Client) AddOrgMember(org, user string) error {
	_, _, err := c.do(http.MethodPut, "/api/v1/orgs/"+url.PathEscape(org)+"/membership/"+url.PathEscape(user), "", "", map[string]any{
		"role": "member",
	})
	return err
}

func (c *Client) CreateTeam(org, name, perm string) (int64, error) {
	b, _, err := c.do(http.MethodPost, "/api/v1/orgs/"+url.PathEscape(org)+"/teams", "", "", map[string]any{
		"name":                      name,
		"permission":                perm,
		"can_create_org_repo":       false,
		"includes_all_repositories": false,
		"units":                     []string{"repo.code", "repo.issues", "repo.pulls", "repo.releases", "repo.ext_wiki", "repo.wiki", "repo.packages"},
	})
	if err != nil {
		return 0, err
	}
	var t struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(b, &t); err != nil {
		return 0, err
	}
	return t.ID, nil
}

func (c *Client) AddTeamMember(teamID int64, user string) error {
	_, _, err := c.do(http.MethodPut, fmt.Sprintf("/api/v1/teams/%d/members/%s", teamID, url.PathEscape(user)), "", "", nil)
	return err
}

func (c *Client) AddTeamRepo(teamID int64, org, repo string) error {
	_, _, err := c.do(http.MethodPut, fmt.Sprintf("/api/v1/teams/%d/repos/%s/%s", teamID, url.PathEscape(org), url.PathEscape(repo)), "", "", nil)
	return err
}

func (c *Client) CreateOrgRepo(org, name string, private bool) (Repo, error) {
	b, _, err := c.do(http.MethodPost, "/api/v1/orgs/"+url.PathEscape(org)+"/repos", "", "", map[string]any{
		"name":           name,
		"private":        private,
		"auto_init":      false,
		"default_branch": "dev",
	})
	if err != nil {
		return Repo{}, err
	}
	var r Repo
	return r, json.Unmarshal(b, &r)
}

func (c *Client) GetRepo(owner, name string, token string) (Repo, error) {
	b, _, err := c.do(http.MethodGet, "/api/v1/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name), token, "", nil)
	if err != nil {
		return Repo{}, err
	}
	var r Repo
	return r, json.Unmarshal(b, &r)
}

func (c *Client) ListRepos(token string) ([]Repo, error) {
	b, _, err := c.do(http.MethodGet, "/api/v1/user/repos?limit=50", token, "", nil)
	if err != nil {
		return nil, err
	}
	var out []Repo
	return out, json.Unmarshal(b, &out)
}

func (c *Client) ListOrgRepos(org string) ([]Repo, error) {
	b, _, err := c.do(http.MethodGet, "/api/v1/orgs/"+url.PathEscape(org)+"/repos?limit=50", "", "", nil)
	if err != nil {
		return nil, err
	}
	var out []Repo
	return out, json.Unmarshal(b, &out)
}

type Commit struct {
	SHA     string `json:"sha"`
	HTMLURL string `json:"html_url"`
	Commit  struct {
		Message string `json:"message"`
		Author  struct {
			Name string `json:"name"`
			Date string `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

type ContentEntry struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Type     string `json:"type"`
	SHA      string `json:"sha"`
	Size     int64  `json:"size"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

func (c *Client) ListCommits(owner, name, sha string) ([]Commit, error) {
	q := "?limit=20"
	if sha != "" {
		q += "&sha=" + url.QueryEscape(sha)
	}
	b, _, err := c.do(http.MethodGet, "/api/v1/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name)+"/commits"+q, "", "", nil)
	if err != nil {
		return nil, err
	}
	var out []Commit
	return out, json.Unmarshal(b, &out)
}

func (c *Client) ListContents(owner, name, ref, path string) ([]ContentEntry, error) {
	p := "/api/v1/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(name) + "/contents"
	if path != "" && path != "." {
		for _, part := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
			if part == "" {
				continue
			}
			p += "/" + url.PathEscape(part)
		}
	}
	if ref != "" {
		p += "?ref=" + url.QueryEscape(ref)
	}
	b, _, err := c.do(http.MethodGet, p, "", "", nil)
	if err != nil {
		return nil, err
	}
	var many []ContentEntry
	if json.Unmarshal(b, &many) == nil && (len(many) > 0 || string(b) == "[]") {
		return many, nil
	}
	var one ContentEntry
	if err := json.Unmarshal(b, &one); err != nil {
		return nil, err
	}
	if one.Name == "" {
		return []ContentEntry{}, nil
	}
	return []ContentEntry{one}, nil
}

func (c *Client) GetFile(owner, name, ref, path string) (ContentEntry, error) {
	ents, err := c.ListContents(owner, name, ref, path)
	if err != nil {
		return ContentEntry{}, err
	}
	if len(ents) == 0 {
		return ContentEntry{}, fmt.Errorf("not a file")
	}
	return ents[0], nil
}

func (c *Client) ListBranches(owner, name string) ([]Branch, error) {
	b, _, err := c.do(http.MethodGet, "/api/v1/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name)+"/branches", "", "", nil)
	if err != nil {
		return nil, err
	}
	var out []Branch
	return out, json.Unmarshal(b, &out)
}

func (c *Client) DeleteRef(owner, name, ref string) error {
	_, _, err := c.do(http.MethodDelete, "/api/v1/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name)+"/git/refs/"+strings.TrimPrefix(ref, "refs/"), "", "", nil)
	return err
}

func (c *Client) ProtectTrains(owner, name string) error {
	for _, b := range []string{"dev", "test"} {
		if err := c.ProtectDefault(owner, name, b); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) ProtectDefault(owner, name, branch string) error {
	if branch == "" {
		branch = "dev"
	}
	_, code, err := c.do(http.MethodPost, "/api/v1/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name)+"/branch_protections", "", "", map[string]any{
		"rule_name":                         branch,
		"enable_push":                       false,
		"enable_force_push":                 false,
		"enable_merge_whitelist":            false,
		"enable_status_check":               true,
		"status_check_contexts":             []string{},
		"block_on_rejected_reviews":         false,
		"block_on_official_review_requests": false,
		"block_on_outdated_branch":          false,
		"dismiss_stale_approvals":           false,
		"require_signed_commits":            false,
		"protected_file_patterns":           "",
		"unprotected_file_patterns":         "",
	})
	if err != nil && code != http.StatusConflict && code != http.StatusUnprocessableEntity {
		return err
	}
	return nil
}

func (c *Client) CreatePR(owner, name, title, head, base, body string) (PR, error) {
	b, _, err := c.do(http.MethodPost, "/api/v1/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name)+"/pulls", "", "", map[string]any{
		"title": title,
		"head":  head,
		"base":  base,
		"body":  body,
	})
	if err != nil {
		return PR{}, err
	}
	var p PR
	return p, json.Unmarshal(b, &p)
}

func (c *Client) ListPRs(owner, name, state string) ([]PR, error) {
	if state == "" {
		state = "open"
	}
	b, _, err := c.do(http.MethodGet, "/api/v1/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name)+"/pulls?state="+url.QueryEscape(state), "", "", nil)
	if err != nil {
		return nil, err
	}
	var out []PR
	return out, json.Unmarshal(b, &out)
}

func (c *Client) SearchPRs(org, state string) ([]PR, error) {
	if state == "" {
		state = "open"
	}
	repos, err := c.ListOrgRepos(org)
	if err != nil {
		return nil, err
	}
	var all []PR
	for _, r := range repos {
		parts := strings.SplitN(r.FullName, "/", 2)
		if len(parts) != 2 {
			continue
		}
		prs, err := c.ListPRs(parts[0], parts[1], state)
		if err != nil {
			continue
		}
		for i := range prs {
			prs[i].Repo = r.FullName
		}
		all = append(all, prs...)
	}
	return all, nil
}

func (c *Client) GetPR(owner, name string, number int) (PR, error) {
	b, _, err := c.do(http.MethodGet, fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d", url.PathEscape(owner), url.PathEscape(name), number), "", "", nil)
	if err != nil {
		return PR{}, err
	}
	var p PR
	return p, json.Unmarshal(b, &p)
}

func (c *Client) CommentPR(owner, name string, number int, body string) error {
	_, _, err := c.do(http.MethodPost, fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/comments", url.PathEscape(owner), url.PathEscape(name), number), "", "", map[string]any{
		"body": body,
	})
	return err
}

func (c *Client) MergePR(owner, name string, number int) error {
	_, _, err := c.do(http.MethodPost, fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d/merge", url.PathEscape(owner), url.PathEscape(name), number), "", "", map[string]any{
		"Do": "merge",
	})
	return err
}

func (c *Client) CommitStatuses(owner, name, sha string) ([]Status, error) {
	b, _, err := c.do(http.MethodGet, "/api/v1/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name)+"/commits/"+url.PathEscape(sha)+"/statuses", "", "", nil)
	if err != nil {
		return nil, err
	}
	var out []Status
	return out, json.Unmarshal(b, &out)
}

func (c *Client) ChecksGreen(owner, name, sha string) (bool, []Status, error) {
	st, err := c.CommitStatuses(owner, name, sha)
	if err != nil {
		return false, nil, err
	}
	if len(st) == 0 {
		return false, st, nil
	}
	ok := true
	for _, s := range st {
		switch strings.ToLower(s.Status) {
		case "success":
		case "pending", "warning":
			ok = false
		default:
			ok = false
		}
	}
	return ok, st, nil
}

func (c *Client) ListPackages(owner, typ string) ([]Package, error) {
	q := "/api/v1/packages/" + url.PathEscape(owner)
	if typ != "" {
		q += "?type=" + url.QueryEscape(typ)
	}
	b, _, err := c.do(http.MethodGet, q, "", "", nil)
	if err != nil {
		return nil, err
	}
	var out []Package
	return out, json.Unmarshal(b, &out)
}

func (c *Client) CreateToken(user, name string) (string, error) {
	b, _, err := c.do(http.MethodPost, "/api/v1/users/"+url.PathEscape(user)+"/tokens", "", user, map[string]any{
		"name":   name,
		"scopes": []string{"all"},
	})
	if err != nil {
		// fallback without sudo: admin token create for that user via CLI-equivalent API
		b, _, err = c.do(http.MethodPost, "/api/v1/users/"+url.PathEscape(user)+"/tokens", "", "", map[string]any{
			"name":   name,
			"scopes": []string{"all"},
		})
		if err != nil {
			return "", err
		}
	}
	var t struct {
		Sha1  string `json:"sha1"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(b, &t); err != nil {
		return "", err
	}
	if t.Sha1 != "" {
		return t.Sha1, nil
	}
	return t.Token, nil
}

func (c *Client) ListKeys(user string) ([]PublicKey, error) {
	b, _, err := c.do(http.MethodGet, "/api/v1/users/"+url.PathEscape(user)+"/keys", "", user, nil)
	if err != nil {
		return nil, err
	}
	var out []PublicKey
	return out, json.Unmarshal(b, &out)
}

func (c *Client) AddKey(user, title, key string) error {
	_, _, err := c.do(http.MethodPost, "/api/v1/user/keys", "", user, map[string]any{
		"title": title,
		"key":   key,
	})
	return err
}

func (c *Client) ListComments(owner, name string, number int) ([]Comment, error) {
	b, _, err := c.do(http.MethodGet, fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/comments", url.PathEscape(owner), url.PathEscape(name), number), "", "", nil)
	if err != nil {
		return nil, err
	}
	var out []Comment
	return out, json.Unmarshal(b, &out)
}

func (c *Client) ListBranchProtections(owner, name string) ([]BranchProtection, error) {
	b, _, err := c.do(http.MethodGet, "/api/v1/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name)+"/branch_protections", "", "", nil)
	if err != nil {
		return nil, err
	}
	var out []BranchProtection
	return out, json.Unmarshal(b, &out)
}

func (c *Client) ListTokens(user string) ([]AccessToken, error) {
	b, _, err := c.do(http.MethodGet, "/api/v1/users/"+url.PathEscape(user)+"/tokens", "", user, nil)
	if err != nil {
		return nil, err
	}
	var out []AccessToken
	return out, json.Unmarshal(b, &out)
}

func (c *Client) DeleteToken(user string, id int64) error {
	_, _, err := c.do(http.MethodDelete, fmt.Sprintf("/api/v1/users/%s/tokens/%d", url.PathEscape(user), id), "", user, nil)
	return err
}

func (c *Client) PutBytes(path, token, contentType string, body []byte) (int, []byte, error) {
	req, err := http.NewRequest(http.MethodPut, c.base+path, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	if token == "" {
		token = c.admin
	}
	req.Header.Set("Authorization", "token "+token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b, nil
}

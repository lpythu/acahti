package forgejo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"acahti/internal/identity"
	"acahti/internal/page"
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
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	LoginName string `json:"login_name"`
	SourceID  int64  `json:"source_id"`
	Email     string `json:"email"`
	IsAdmin   bool   `json:"is_admin"`
	FullName  string `json:"full_name"`
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
	Group         string `json:"group,omitempty"`
}

type Team struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Permission string `json:"permission"`
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
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
}

type PackageFile struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Size int64  `json:"Size"`
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
		return b, resp.StatusCode, apiError(b)
	}
	return b, resp.StatusCode, nil
}

func apiError(b []byte) error {
	var m struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(b, &m) == nil {
		if msg := strings.TrimSpace(m.Message); msg != "" {
			return fmt.Errorf("%s", msg)
		}
	}
	if s := strings.TrimSpace(string(b)); s != "" {
		return fmt.Errorf("%s", s)
	}
	return fmt.Errorf("request failed")
}

func listPage[T any](c *Client, path string, q page.Query, extra url.Values, sudo string) (page.Result[T], error) {
	u, err := url.Parse(path)
	if err != nil {
		return page.Result[T]{}, err
	}
	vals := u.Query()
	for k, vs := range extra {
		for _, v := range vs {
			vals.Set(k, v)
		}
	}
	q = q.Norm()
	vals.Set("page", strconv.Itoa(q.Page))
	vals.Set("limit", strconv.Itoa(q.LimitPlus()))
	u.RawQuery = vals.Encode()
	b, _, err := c.do(http.MethodGet, u.String(), "", sudo, nil)
	if err != nil {
		return page.Result[T]{}, err
	}
	var items []T
	if err := json.Unmarshal(b, &items); err != nil {
		return page.Result[T]{}, err
	}
	return page.Clip(items, q), nil
}

func (c *Client) User(token string) (User, error) {
	return c.user("", token)
}

func (c *Client) UserSudo(login string) (User, error) {
	return c.user(login, "")
}

func (c *Client) user(sudo, token string) (User, error) {
	b, _, err := c.do(http.MethodGet, "/api/v1/user", token, sudo, nil)
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

func (c *Client) ListUsers(q page.Query) (page.Result[User], error) {
	return listPage[User](c, "/api/v1/admin/users", q, nil, "")
}

func (c *Client) AllUsers() ([]User, error) {
	q := page.Query{Page: 1, Size: page.MaxSize}
	var users []User
	for q.Page <= page.MaxWalk {
		res, err := c.ListUsers(q)
		if err != nil {
			return nil, err
		}
		users = append(users, res.Items...)
		if !res.HasMore {
			break
		}
		q.Page++
	}
	if users == nil {
		users = []User{}
	}
	return users, nil
}

func (c *Client) AdminUser(login string) (User, error) {
	b, _, err := c.do(http.MethodGet, "/api/v1/admin/users/"+url.PathEscape(login), "", "", nil)
	if err != nil {
		return User{}, err
	}
	var u User
	return u, json.Unmarshal(b, &u)
}

func (c *Client) SetPassword(login, password string) error {
	return c.EditUser(login, map[string]any{
		"password":             password,
		"must_change_password": false,
	})
}

func (c *Client) EditUser(login string, fields map[string]any) error {
	u, err := c.AdminUser(login)
	if err != nil {
		return err
	}
	return c.patchAdminUser(u, fields)
}

func (c *Client) patchAdminUser(u User, fields map[string]any) error {
	login := strings.TrimSpace(u.Login)
	if login == "" {
		return fmt.Errorf("username required")
	}
	body := map[string]any{}
	for k, v := range fields {
		body[k] = v
	}
	loginName := strings.TrimSpace(u.LoginName)
	if loginName == "" {
		loginName = login
	}
	body["source_id"] = u.SourceID
	body["login_name"] = loginName
	_, _, err := c.do(http.MethodPatch, "/api/v1/admin/users/"+url.PathEscape(login), "", "", body)
	return err
}

func (c *Client) EnsureNoreply(rootURL, domain string) error {
	if !c.Ready() {
		return nil
	}
	d := identity.Domain(rootURL, domain)
	if d == "" {
		return nil
	}
	users, err := c.AllUsers()
	if err != nil {
		return err
	}
	for _, u := range users {
		if u.Login == "" {
			continue
		}
		fields := map[string]any{}
		if want := identity.Email(u.Login, d); u.Email != want {
			fields["email"] = want
		}
		if want := identity.Name(u.Login, u.FullName); u.FullName != want {
			fields["full_name"] = want
		}
		if len(fields) == 0 {
			continue
		}
		if err := c.patchAdminUser(u, fields); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) CreateUser(login, email, password string, admin bool) (User, error) {
	b, _, err := c.do(http.MethodPost, "/api/v1/admin/users", "", "", map[string]any{
		"source_id":                 0,
		"login_name":                login,
		"username":                  login,
		"full_name":                 login,
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

func (c *Client) RemoveTeamMember(teamID int64, user string) error {
	_, _, err := c.do(http.MethodDelete, fmt.Sprintf("/api/v1/teams/%d/members/%s", teamID, url.PathEscape(user)), "", "", nil)
	return err
}

func (c *Client) ListOrgTeams(org string, q page.Query) (page.Result[Team], error) {
	return listPage[Team](c, "/api/v1/orgs/"+url.PathEscape(org)+"/teams", q, nil, "")
}

func (c *Client) ListTeamRepos(teamID int64, q page.Query) (page.Result[Repo], error) {
	return listPage[Repo](c, fmt.Sprintf("/api/v1/teams/%d/repos", teamID), q, nil, "")
}

func (c *Client) ListTeamMembers(teamID int64, q page.Query) (page.Result[User], error) {
	return listPage[User](c, fmt.Sprintf("/api/v1/teams/%d/members", teamID), q, nil, "")
}

func (c *Client) ListRepoTeams(owner, name string, q page.Query) (page.Result[Team], error) {
	return listPage[Team](c, "/api/v1/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name)+"/teams", q, nil, "")
}

func (c *Client) FindOrgTeam(org, name string) (Team, error) {
	teams, err := page.Walk(func(q page.Query) (page.Result[Team], error) {
		return c.ListOrgTeams(org, q)
	})
	if err != nil {
		return Team{}, err
	}
	for _, t := range teams {
		if t.Name == name {
			return t, nil
		}
	}
	return Team{}, fmt.Errorf("team %s not found", name)
}

func (c *Client) AddCollaborator(owner, repo, user, perm string) error {
	_, _, err := c.do(http.MethodPut, fmt.Sprintf("/api/v1/repos/%s/%s/collaborators/%s", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(user)), "", "", map[string]any{
		"permission": perm,
	})
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

func (c *Client) GetRepo(owner, name string, sudo string) (Repo, error) {
	b, _, err := c.do(http.MethodGet, "/api/v1/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name), "", sudo, nil)
	if err != nil {
		return Repo{}, err
	}
	var r Repo
	return r, json.Unmarshal(b, &r)
}

func (c *Client) ListRepos(sudo string, q page.Query) (page.Result[Repo], error) {
	return listPage[Repo](c, "/api/v1/user/repos", q, nil, sudo)
}

func (c *Client) ListOrgRepos(org string, q page.Query) (page.Result[Repo], error) {
	return listPage[Repo](c, "/api/v1/orgs/"+url.PathEscape(org)+"/repos", q, nil, "")
}

type CommitPerson struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Date  string `json:"date"`
}

type CommitUser struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
}

type CommitFile struct {
	Filename         string `json:"filename"`
	Status           string `json:"status"`
	Additions        int    `json:"additions"`
	Deletions        int    `json:"deletions"`
	Changes          int    `json:"changes"`
	PreviousFilename string `json:"previous_filename,omitempty"`
	Patch            string `json:"patch"`
}

type CommitStats struct {
	Total     int `json:"total"`
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
}

type Commit struct {
	SHA     string `json:"sha"`
	HTMLURL string `json:"html_url"`
	Commit  struct {
		Message   string       `json:"message"`
		Author    CommitPerson `json:"author"`
		Committer CommitPerson `json:"committer"`
	} `json:"commit"`
	Author    *CommitUser `json:"author"`
	Committer *CommitUser `json:"committer"`
	Parents   []struct {
		SHA string `json:"sha"`
	} `json:"parents"`
	Files []CommitFile `json:"files"`
	Stats *CommitStats `json:"stats"`
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

func (c *Client) ListCommits(owner, name, sha string, q page.Query) (page.Result[Commit], error) {
	extra := url.Values{}
	if sha != "" {
		extra.Set("sha", sha)
	}
	return listPage[Commit](c, "/api/v1/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name)+"/commits", q, extra, "")
}

func (c *Client) GetCommit(owner, name, sha string) (Commit, error) {
	b, _, err := c.do(http.MethodGet, "/api/v1/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name)+"/git/commits/"+url.PathEscape(sha), "", "", nil)
	if err != nil {
		return Commit{}, err
	}
	var out Commit
	return out, json.Unmarshal(b, &out)
}

func (c *Client) GetCommitDiff(owner, name, sha string) (string, error) {
	b, _, err := c.do(http.MethodGet, "/api/v1/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name)+"/git/commits/"+url.PathEscape(sha)+".diff", "", "", nil)
	if err != nil {
		return "", err
	}
	return string(b), nil
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

func (c *Client) ListBranches(owner, name string, q page.Query) (page.Result[Branch], error) {
	return listPage[Branch](c, "/api/v1/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name)+"/branches", q, nil, "")
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

func (c *Client) ListPRs(owner, name, state string, q page.Query) (page.Result[PR], error) {
	if state == "" {
		state = "open"
	}
	return listPage[PR](c, "/api/v1/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name)+"/pulls", q, url.Values{"state": {state}}, "")
}

func (c *Client) SearchPRs(org, state string, q page.Query) (page.Result[PR], error) {
	if state == "" {
		state = "open"
	}
	extra := url.Values{"type": {"pulls"}, "state": {state}}
	if org != "" {
		extra.Set("owner", org)
	}
	res, err := listPage[issueHit](c, "/api/v1/repos/issues/search", q, extra, "")
	if err != nil {
		return page.Result[PR]{}, err
	}
	out := make([]PR, 0, len(res.Items))
	for _, h := range res.Items {
		p := h.PR
		if p.Repo == "" {
			p.Repo = h.Repository.FullName
		}
		if p.Repo == "" && p.HTMLURL != "" {
			parts := strings.Split(strings.TrimPrefix(p.HTMLURL, "https://"), "/")
			if len(parts) >= 3 {
				p.Repo = parts[1] + "/" + parts[2]
			}
		}
		out = append(out, p)
	}
	return page.Of(out, q, res.HasMore), nil
}

type issueHit struct {
	PR
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
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

func (c *Client) ListPackageFiles(owner, typ, name, version string) ([]PackageFile, error) {
	p := "/api/v1/packages/" + url.PathEscape(owner) + "/" + url.PathEscape(typ) + "/" + url.PathEscape(name) + "/" + url.PathEscape(version) + "/files"
	b, _, err := c.do(http.MethodGet, p, "", "", nil)
	if err != nil {
		return nil, err
	}
	var out []PackageFile
	return out, json.Unmarshal(b, &out)
}

func (c *Client) ListPackages(owner, typ string, q page.Query) (page.Result[Package], error) {
	extra := url.Values{}
	if typ != "" {
		extra.Set("type", typ)
	}
	return listPage[Package](c, "/api/v1/packages/"+url.PathEscape(owner), q, extra, "")
}

func (c *Client) ListKeys(user string, q page.Query) (page.Result[PublicKey], error) {
	return listPage[PublicKey](c, "/api/v1/users/"+url.PathEscape(user)+"/keys", q, nil, user)
}

func (c *Client) AddKey(user, title, key string) error {
	_, _, err := c.do(http.MethodPost, "/api/v1/user/keys", "", user, map[string]any{
		"title": title,
		"key":   key,
	})
	return err
}

func (c *Client) ListComments(owner, name string, number int, q page.Query) (page.Result[Comment], error) {
	return listPage[Comment](c, fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/comments", url.PathEscape(owner), url.PathEscape(name), number), q, nil, "")
}

func (c *Client) ListBranchProtections(owner, name string) ([]BranchProtection, error) {
	b, _, err := c.do(http.MethodGet, "/api/v1/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name)+"/branch_protections", "", "", nil)
	if err != nil {
		return nil, err
	}
	var out []BranchProtection
	return out, json.Unmarshal(b, &out)
}

func (c *Client) PutBytes(path, sudo, contentType string, body []byte) (int, []byte, error) {
	req, err := http.NewRequest(http.MethodPut, c.base+path, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "token "+c.admin)
	if sudo != "" {
		req.Header.Set("Sudo", sudo)
	}
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

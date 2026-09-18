package catalog

import (
	"fmt"
	"strconv"
	"strings"

	"acahti/internal/forgejo"
	"acahti/internal/page"
	"acahti/internal/woodpecker"
)

// --- identity / users (kernel) ---

func (c *Catalog) User(login string) (forgejo.User, error) {
	if c == nil || c.fj == nil {
		return forgejo.User{}, fmt.Errorf("git unavailable")
	}
	return c.fj.UserSudo(login)
}

func (c *Catalog) Authenticate(user, pass string) (forgejo.User, error) {
	if c == nil || c.fj == nil {
		return forgejo.User{}, fmt.Errorf("login failed")
	}
	return c.fj.BasicUser(user, pass)
}

func (c *Catalog) TokenUser(token string) (forgejo.User, error) {
	if c == nil || c.fj == nil {
		return forgejo.User{}, fmt.Errorf("login failed")
	}
	return c.fj.TokenUser(token)
}

func (c *Catalog) ListUsers(q page.Query) (page.Result[forgejo.User], error) {
	if c == nil || c.fj == nil || !c.fj.Ready() {
		return page.Result[forgejo.User]{}, fmt.Errorf("git unavailable")
	}
	return c.fj.ListUsers(q)
}

func (c *Catalog) CreateUser(login, email, password string, admin bool) (forgejo.User, error) {
	if c == nil || c.fj == nil || !c.fj.Ready() {
		return forgejo.User{}, fmt.Errorf("git unavailable")
	}
	u, err := c.fj.CreateUser(login, email, password, admin)
	if err != nil {
		return forgejo.User{}, err
	}
	_ = c.fj.AddOrgMember(c.Cfg.Org, u.Login)
	return u, nil
}

func (c *Catalog) EditUser(login string, fields map[string]any) error {
	if c == nil || c.fj == nil || !c.fj.Ready() {
		return fmt.Errorf("git unavailable")
	}
	if err := c.fj.EditUser(login, fields); err != nil {
		return err
	}
	c.ForgetAuthors()
	return nil
}

func (c *Catalog) SetPassword(login, password string) error {
	if c == nil || c.fj == nil || !c.fj.Ready() {
		return fmt.Errorf("git unavailable")
	}
	return c.fj.SetPassword(login, password)
}

// --- orgs ---

func (c *Catalog) ListAdminOrgs(q page.Query) (page.Result[forgejo.Org], error) {
	if c == nil || c.fj == nil || !c.fj.Ready() {
		return page.Result[forgejo.Org]{}, fmt.Errorf("git unavailable")
	}
	return c.fj.ListAdminOrgs(q)
}

func (c *Catalog) ListUserOrgs(user string, q page.Query) (page.Result[forgejo.Org], error) {
	if c == nil || c.fj == nil || !c.fj.Ready() {
		return page.Result[forgejo.Org]{}, fmt.Errorf("git unavailable")
	}
	return c.fj.ListUserOrgs(user, q)
}

func (c *Catalog) CreateOrg(user, name, fullName string) (forgejo.Org, error) {
	if c == nil || c.fj == nil || !c.fj.Ready() {
		return forgejo.Org{}, fmt.Errorf("git unavailable")
	}
	return c.fj.CreateOrg(user, name, fullName)
}

// --- repos / git commands ---

func (c *Catalog) GetRepo(owner, name, sudo string) (forgejo.Repo, error) {
	if c == nil || c.fj == nil || !c.fj.Ready() {
		return forgejo.Repo{}, fmt.Errorf("git unavailable")
	}
	repo, err := c.fj.GetRepo(owner, name, sudo)
	if err != nil {
		return forgejo.Repo{}, err
	}
	return c.PublicRepo(repo), nil
}

func (c *Catalog) CreateRepo(name, team string) (forgejo.Repo, error) {
	if c == nil || c.fj == nil || !c.fj.Ready() {
		return forgejo.Repo{}, fmt.Errorf("git unavailable")
	}
	repo, err := c.fj.CreateOrgRepo(c.Cfg.Org, name, true)
	if err != nil {
		return forgejo.Repo{}, err
	}
	c.RememberRepo(repo)
	if team != "" {
		if err := c.AttachRepo(team, repo.Name); err != nil {
			return forgejo.Repo{}, err
		}
		repo.Team = team
	}
	if c.wp != nil && c.wp.Ready() {
		_ = c.wp.Activate(c.Cfg.Org+"/"+repo.Name, strconv.FormatInt(repo.ID, 10))
	}
	return c.PublicRepo(repo), nil
}

func (c *Catalog) DeleteRef(owner, name, ref string) error {
	if c == nil || c.fj == nil || !c.fj.Ready() {
		return fmt.Errorf("git unavailable")
	}
	return c.fj.DeleteRef(owner, name, ref)
}

func (c *Catalog) CreatePR(owner, name, title, head, base, body, sudo string) (forgejo.PR, error) {
	if c == nil || c.fj == nil || !c.fj.Ready() {
		return forgejo.PR{}, fmt.Errorf("git unavailable")
	}
	if base == "" {
		repo, err := c.fj.GetRepo(owner, name, sudo)
		if err != nil {
			return forgejo.PR{}, err
		}
		base = repo.DefaultBranch
	}
	return c.fj.CreatePR(owner, name, title, head, base, body, sudo)
}

func (c *Catalog) CommentPR(owner, name string, number int, body, sudo string) error {
	if c == nil || c.fj == nil || !c.fj.Ready() {
		return fmt.Errorf("git unavailable")
	}
	return c.fj.CommentPR(owner, name, number, body, sudo)
}

func (c *Catalog) SearchPackages(owner, kind, query string, q page.Query, sudo string) (page.Result[forgejo.Package], error) {
	if c == nil || c.fj == nil || !c.fj.Ready() {
		return page.Of([]forgejo.Package{}, q, false), nil
	}
	if owner == "" {
		owner = c.Cfg.Org
	}
	return c.fj.ListPackages(owner, kind, query, q, sudo)
}

func (c *Catalog) DeletePackage(owner, kind, name, version, sudo string) error {
	if c == nil || c.fj == nil || !c.fj.Ready() {
		return fmt.Errorf("git unavailable")
	}
	if owner == "" {
		owner = c.Cfg.Org
	}
	return c.fj.DeletePackage(owner, kind, name, version, sudo)
}

func (c *Catalog) DeletePackageVersions(owner, kind, name, sudo string) ([]string, error) {
	if kind == "" || name == "" {
		return nil, fmt.Errorf("%w: kind and name required", ErrInvalid)
	}
	if owner == "" {
		owner = c.Cfg.Org
	}
	all, err := page.Walk(func(q page.Query) (page.Result[forgejo.Package], error) {
		return c.SearchPackages(owner, kind, name, q, sudo)
	})
	if err != nil {
		return nil, err
	}
	deleted := make([]string, 0)
	for _, p := range all {
		if p.Name != name || !strings.EqualFold(p.Type, kind) {
			continue
		}
		if err := c.DeletePackage(owner, p.Type, p.Name, p.Version, sudo); err != nil {
			return nil, err
		}
		deleted = append(deleted, p.Name+"@"+p.Version)
	}
	if len(deleted) == 0 {
		return nil, ErrNotFound
	}
	return deleted, nil
}

func (c *Catalog) PutBytes(path, sudo, contentType string, body []byte) (int, []byte, error) {
	if c == nil || c.fj == nil || !c.fj.Ready() {
		return 0, nil, fmt.Errorf("git unavailable")
	}
	return c.fj.PutBytes(path, sudo, contentType, body)
}

// --- pipeline commands ---

func (c *Catalog) TriggerPipeline(user, repo, ref string) (woodpecker.Pipeline, error) {
	owner, name, _ := strings.Cut(repo, "/")
	if _, err := c.RepoHeader(user, owner, name, ""); err != nil {
		return woodpecker.Pipeline{}, err
	}
	if c.wp == nil || !c.wp.Ready() {
		return woodpecker.Pipeline{}, fmt.Errorf("ci unavailable")
	}
	if ref == "" {
		ref = "dev"
	}
	pipe, err := c.wp.Trigger(repo, ref)
	if err != nil {
		return woodpecker.Pipeline{}, err
	}
	return c.Remember(pipe), nil
}

func (c *Catalog) RerunPipeline(user, repo string, number int64) (woodpecker.Pipeline, error) {
	owner, name, _ := strings.Cut(repo, "/")
	if _, err := c.RepoHeader(user, owner, name, ""); err != nil {
		return woodpecker.Pipeline{}, err
	}
	if c.wp == nil || !c.wp.Ready() {
		return woodpecker.Pipeline{}, fmt.Errorf("ci unavailable")
	}
	pipe, err := c.wp.Rerun(repo, number)
	if err != nil {
		return woodpecker.Pipeline{}, err
	}
	return c.Remember(pipe), nil
}

func (c *Catalog) ApprovePipeline(user, repo string, number int64) (woodpecker.Pipeline, error) {
	owner, name, _ := strings.Cut(repo, "/")
	if _, err := c.RepoHeader(user, owner, name, ""); err != nil {
		return woodpecker.Pipeline{}, err
	}
	if c.wp == nil || !c.wp.Ready() {
		return woodpecker.Pipeline{}, fmt.Errorf("ci unavailable")
	}
	if err := c.wp.Approve(repo, number); err != nil {
		return woodpecker.Pipeline{}, err
	}
	return c.Refresh(repo, number)
}

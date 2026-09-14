package catalog

import (
	"fmt"
	"sort"
	"strings"

	"acahti/internal/forgejo"
	"acahti/internal/page"
)

type AccessPerson struct {
	Login      string `json:"login"`
	Permission string `json:"permission"`
}

type TeamAccess struct {
	Name      string         `json:"name"`
	CanManage bool           `json:"can_manage"`
	Members   []AccessPerson `json:"members"`
	Repos     []forgejo.Repo `json:"repos"`
}

type RepoAccess struct {
	Team      string         `json:"team"`
	CanManage bool           `json:"can_manage"`
	Inherited []AccessPerson `json:"inherited"`
	Direct    []AccessPerson `json:"direct"`
}

func (c *Catalog) requireAdmin(user string) error {
	if !c.IsOrgAdmin(user) {
		return ErrForbidden
	}
	return nil
}

func (c *Catalog) requireRepoAdmin(user, owner, name string) (forgejo.Repo, error) {
	repo, err := c.seeRepo(user, owner, name)
	if err != nil {
		return forgejo.Repo{}, err
	}
	if c.IsOrgAdmin(user) || repo.Permissions.Admin {
		return repo, nil
	}
	return forgejo.Repo{}, ErrForbidden
}

func (c *Catalog) findTeam(name string) (teamRoles, error) {
	if err := ValidTeamName(name); err != nil {
		return teamRoles{}, err
	}
	clusters, err := c.clusterTeams()
	if err != nil {
		return teamRoles{}, err
	}
	t, ok := clusters[name]
	if !ok {
		return teamRoles{}, fmt.Errorf("%w: team %s", ErrNotFound, name)
	}
	return t, nil
}

func (c *Catalog) ensureRoleTeam(team, perm string) (forgejo.Team, error) {
	perm = forgejo.NormalizePerm(perm)
	name := roleTeamName(team, perm)
	if t, err := c.FJ.FindOrgTeam(c.Cfg.Org, name); err == nil {
		return t, nil
	}
	t, err := c.FJ.CreateTeam(c.Cfg.Org, name, perm)
	if err != nil {
		return forgejo.Team{}, err
	}
	if perm == permWrite {
		return t, nil
	}
	write, err := c.FJ.FindOrgTeam(c.Cfg.Org, team)
	if err != nil {
		return t, nil
	}
	repos, err := page.Walk(func(q page.Query) (page.Result[forgejo.Repo], error) {
		return c.FJ.ListTeamRepos(write.ID, q)
	})
	if err != nil {
		return t, nil
	}
	for _, r := range repos {
		_ = c.FJ.AddTeamRepo(t.ID, c.Cfg.Org, r.Name)
	}
	return t, nil
}

func (c *Catalog) TeamsByLogin() (map[string][]string, error) {
	if c == nil || !c.FJ.Ready() {
		return map[string][]string{}, nil
	}
	clusters, err := c.clusterTeams()
	if err != nil {
		return nil, err
	}
	out := map[string][]string{}
	for name, t := range clusters {
		members, err := c.membersOf(t)
		if err != nil {
			return nil, err
		}
		for _, m := range members {
			out[m.Login] = append(out[m.Login], name)
		}
	}
	for login, names := range out {
		sort.Strings(names)
		out[login] = names
	}
	return out, nil
}

func (c *Catalog) membersOf(t teamRoles) ([]AccessPerson, error) {
	byLogin := map[string]string{}
	order := []string{}
	for perm, role := range t.roles {
		members, err := page.Walk(func(q page.Query) (page.Result[forgejo.User], error) {
			return c.FJ.ListTeamMembers(role.ID, q)
		})
		if err != nil {
			return nil, err
		}
		for _, u := range members {
			if u.Login == "" {
				continue
			}
			prev, ok := byLogin[u.Login]
			if !ok {
				order = append(order, u.Login)
			}
			byLogin[u.Login] = strongerPerm(prev, perm)
		}
	}
	sort.Strings(order)
	out := make([]AccessPerson, 0, len(order))
	for _, login := range order {
		out = append(out, AccessPerson{Login: login, Permission: byLogin[login]})
	}
	return out, nil
}

func strongerPerm(a, b string) string {
	rank := map[string]int{permRead: 1, permWrite: 2, permAdmin: 3}
	if rank[b] > rank[a] {
		return forgejo.NormalizePerm(b)
	}
	if a == "" {
		return forgejo.NormalizePerm(b)
	}
	return forgejo.NormalizePerm(a)
}

func (c *Catalog) TeamAccess(user, name string) (TeamAccess, error) {
	t, err := c.findTeam(name)
	if err != nil {
		return TeamAccess{}, err
	}
	admin := c.IsOrgAdmin(user)
	if !admin && !c.inTeam(user, t) {
		repos, err := c.teamRepos(t.ids())
		if err != nil {
			return TeamAccess{}, err
		}
		if len(markTeam(repos, name, visibleSetMust(c, user))) == 0 {
			return TeamAccess{}, ErrNotFound
		}
	}
	members, err := c.membersOf(t)
	if err != nil {
		return TeamAccess{}, err
	}
	repos, err := c.teamRepos(t.ids())
	if err != nil {
		return TeamAccess{}, err
	}
	var visible map[string]bool
	if !admin {
		visible = visibleSetMust(c, user)
	}
	return TeamAccess{
		Name:      name,
		CanManage: admin,
		Members:   members,
		Repos:     c.PublicRepos(markTeam(repos, name, visible)),
	}, nil
}

func visibleSetMust(c *Catalog, user string) map[string]bool {
	repos, err := c.userRepos(user)
	if err != nil {
		return map[string]bool{}
	}
	return visibleSet(repos)
}

func (c *Catalog) CreateTeam(user, name string) (TeamAccess, error) {
	if err := c.requireAdmin(user); err != nil {
		return TeamAccess{}, err
	}
	if err := ValidTeamName(name); err != nil {
		return TeamAccess{}, err
	}
	if _, err := c.FJ.FindOrgTeam(c.Cfg.Org, name); err == nil {
		return TeamAccess{}, fmt.Errorf("%w: team exists", ErrInvalid)
	}
	if _, err := c.FJ.CreateTeam(c.Cfg.Org, name, permWrite); err != nil {
		return TeamAccess{}, err
	}
	_ = c.SetTeamMember(user, name, user, permAdmin)
	c.forget()
	return c.TeamAccess(user, name)
}

func (c *Catalog) DeleteTeam(user, name string) error {
	if err := c.requireAdmin(user); err != nil {
		return err
	}
	t, err := c.findTeam(name)
	if err != nil {
		return err
	}
	for _, role := range t.roles {
		if err := c.FJ.DeleteTeam(role.ID); err != nil {
			return err
		}
	}
	c.forget()
	return nil
}

func (c *Catalog) SetTeamMember(user, team, login, perm string) error {
	if err := c.requireAdmin(user); err != nil {
		return err
	}
	t, err := c.findTeam(team)
	if err != nil {
		return err
	}
	login = strings.TrimSpace(login)
	if login == "" {
		return fmt.Errorf("%w login", ErrInvalid)
	}
	perm = forgejo.NormalizePerm(perm)
	want, err := c.ensureRoleTeam(team, perm)
	if err != nil {
		return err
	}
	if err := c.FJ.AddTeamMember(want.ID, login); err != nil {
		return err
	}
	for p, role := range t.roles {
		if p == perm || role.ID == want.ID {
			continue
		}
		_ = c.FJ.RemoveTeamMember(role.ID, login)
	}
	c.forget()
	return nil
}

func (c *Catalog) RemoveTeamMember(user, team, login string) error {
	if err := c.requireAdmin(user); err != nil {
		return err
	}
	t, err := c.findTeam(team)
	if err != nil {
		return err
	}
	for _, role := range t.roles {
		_ = c.FJ.RemoveTeamMember(role.ID, login)
	}
	c.forget()
	return nil
}

func (c *Catalog) AttachRepo(team, repo string) error {
	t, err := c.findTeam(team)
	if err != nil {
		return err
	}
	repo = strings.TrimSpace(repo)
	if i := strings.LastIndex(repo, "/"); i >= 0 {
		repo = repo[i+1:]
	}
	if repo == "" {
		return fmt.Errorf("%w repo", ErrInvalid)
	}
	if _, err := c.FJ.GetRepo(c.Cfg.Org, repo, ""); err != nil {
		return fmt.Errorf("%w: %s", ErrNotFound, err)
	}
	if _, ok := t.roles[permWrite]; !ok {
		if _, err := c.ensureRoleTeam(team, permWrite); err != nil {
			return err
		}
		t, err = c.findTeam(team)
		if err != nil {
			return err
		}
	}
	for _, role := range t.roles {
		if err := c.FJ.AddTeamRepo(role.ID, c.Cfg.Org, repo); err != nil {
			return err
		}
	}
	c.forget()
	return nil
}

func (c *Catalog) AddTeamRepo(user, team, repo string) error {
	if err := c.requireAdmin(user); err != nil {
		return err
	}
	return c.AttachRepo(team, repo)
}

func (c *Catalog) RemoveTeamRepo(user, team, repo string) error {
	if err := c.requireAdmin(user); err != nil {
		return err
	}
	t, err := c.findTeam(team)
	if err != nil {
		return err
	}
	if i := strings.LastIndex(repo, "/"); i >= 0 {
		repo = repo[i+1:]
	}
	for _, role := range t.roles {
		_ = c.FJ.RemoveTeamRepo(role.ID, c.Cfg.Org, repo)
	}
	c.forget()
	return nil
}

func (c *Catalog) RepoAccess(user, owner, name string) (RepoAccess, error) {
	repo, err := c.seeRepo(user, owner, name)
	if err != nil {
		return RepoAccess{}, err
	}
	inherited := []AccessPerson{}
	if repo.Team != "" {
		if t, err := c.findTeam(repo.Team); err == nil {
			inherited, err = c.membersOf(t)
			if err != nil {
				return RepoAccess{}, err
			}
		}
	}
	inTeam := map[string]bool{}
	for _, p := range inherited {
		inTeam[p.Login] = true
	}
	cols, err := page.Walk(func(q page.Query) (page.Result[forgejo.User], error) {
		return c.FJ.ListCollaborators(owner, name, q)
	})
	if err != nil {
		cols = nil
	}
	direct := make([]AccessPerson, 0, len(cols))
	for _, u := range cols {
		if inTeam[u.Login] {
			continue
		}
		perm := u.Permissions.Level()
		if p, err := c.FJ.CollaboratorPerm(owner, name, u.Login); err == nil && p != "" {
			perm = p
		}
		direct = append(direct, AccessPerson{Login: u.Login, Permission: perm})
	}
	sort.Slice(direct, func(i, j int) bool { return direct[i].Login < direct[j].Login })
	return RepoAccess{
		Team:      repo.Team,
		CanManage: c.IsOrgAdmin(user) || repo.Permissions.Admin,
		Inherited: inherited,
		Direct:    direct,
	}, nil
}

func (c *Catalog) SetCollaborator(user, owner, name, login, perm string) error {
	if _, err := c.requireRepoAdmin(user, owner, name); err != nil {
		return err
	}
	login = strings.TrimSpace(login)
	if login == "" {
		return fmt.Errorf("%w login", ErrInvalid)
	}
	if err := c.FJ.AddCollaborator(owner, name, login, perm); err != nil {
		return err
	}
	c.forget()
	return nil
}

func (c *Catalog) RemoveCollaborator(user, owner, name, login string) error {
	if _, err := c.requireRepoAdmin(user, owner, name); err != nil {
		return err
	}
	if err := c.FJ.RemoveCollaborator(owner, name, login); err != nil {
		return err
	}
	c.forget()
	return nil
}

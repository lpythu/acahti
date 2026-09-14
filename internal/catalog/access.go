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

type GroupAccess struct {
	Name      string         `json:"name"`
	CanManage bool           `json:"can_manage"`
	Members   []AccessPerson `json:"members"`
	Repos     []forgejo.Repo `json:"repos"`
}

type RepoAccess struct {
	Group     string         `json:"group"`
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

func (c *Catalog) findGroup(name string) (groupTeams, error) {
	if err := ValidGroupName(name); err != nil {
		return groupTeams{}, err
	}
	clusters, err := c.clusterGroups()
	if err != nil {
		return groupTeams{}, err
	}
	g, ok := clusters[name]
	if !ok {
		return groupTeams{}, fmt.Errorf("%w: group %s", ErrNotFound, name)
	}
	return g, nil
}

func (c *Catalog) ensureRoleTeam(group, perm string) (forgejo.Team, error) {
	perm = forgejo.NormalizePerm(perm)
	name := roleTeamName(group, perm)
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
	write, err := c.FJ.FindOrgTeam(c.Cfg.Org, group)
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

func (c *Catalog) GroupsByLogin() (map[string][]string, error) {
	if c == nil || !c.FJ.Ready() {
		return map[string][]string{}, nil
	}
	clusters, err := c.clusterGroups()
	if err != nil {
		return nil, err
	}
	out := map[string][]string{}
	for name, g := range clusters {
		members, err := c.groupMembers(g)
		if err != nil {
			return nil, err
		}
		for _, m := range members {
			out[m.Login] = append(out[m.Login], name)
		}
	}
	for login, gs := range out {
		sort.Strings(gs)
		out[login] = gs
	}
	return out, nil
}

func (c *Catalog) groupMembers(g groupTeams) ([]AccessPerson, error) {
	byLogin := map[string]string{}
	order := []string{}
	for perm, t := range g.teams {
		members, err := page.Walk(func(q page.Query) (page.Result[forgejo.User], error) {
			return c.FJ.ListTeamMembers(t.ID, q)
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

func (c *Catalog) GroupAccess(user, name string) (GroupAccess, error) {
	g, err := c.findGroup(name)
	if err != nil {
		return GroupAccess{}, err
	}
	admin := c.IsOrgAdmin(user)
	if !admin && !c.inGroup(user, g) {
		repos, err := c.teamRepos(g.ids())
		if err != nil {
			return GroupAccess{}, err
		}
		if len(markGroup(repos, name, visibleSetMust(c, user))) == 0 {
			return GroupAccess{}, ErrNotFound
		}
	}
	members, err := c.groupMembers(g)
	if err != nil {
		return GroupAccess{}, err
	}
	repos, err := c.teamRepos(g.ids())
	if err != nil {
		return GroupAccess{}, err
	}
	var visible map[string]bool
	if !admin {
		visible = visibleSetMust(c, user)
	}
	return GroupAccess{
		Name:      name,
		CanManage: admin,
		Members:   members,
		Repos:     markGroup(repos, name, visible),
	}, nil
}

func visibleSetMust(c *Catalog, user string) map[string]bool {
	repos, err := c.userRepos(user)
	if err != nil {
		return map[string]bool{}
	}
	return visibleSet(repos)
}

func (c *Catalog) CreateGroup(user, name string) (GroupAccess, error) {
	if err := c.requireAdmin(user); err != nil {
		return GroupAccess{}, err
	}
	if err := ValidGroupName(name); err != nil {
		return GroupAccess{}, err
	}
	if _, err := c.FJ.FindOrgTeam(c.Cfg.Org, name); err == nil {
		return GroupAccess{}, fmt.Errorf("%w: group exists", ErrInvalid)
	}
	if _, err := c.FJ.CreateTeam(c.Cfg.Org, name, permWrite); err != nil {
		return GroupAccess{}, err
	}
	_ = c.SetGroupMember(user, name, user, permAdmin)
	c.forget()
	return c.GroupAccess(user, name)
}

func (c *Catalog) DeleteGroup(user, name string) error {
	if err := c.requireAdmin(user); err != nil {
		return err
	}
	g, err := c.findGroup(name)
	if err != nil {
		return err
	}
	for _, t := range g.teams {
		if err := c.FJ.DeleteTeam(t.ID); err != nil {
			return err
		}
	}
	c.forget()
	return nil
}

func (c *Catalog) SetGroupMember(user, group, login, perm string) error {
	if err := c.requireAdmin(user); err != nil {
		return err
	}
	g, err := c.findGroup(group)
	if err != nil {
		return err
	}
	login = strings.TrimSpace(login)
	if login == "" {
		return fmt.Errorf("%w login", ErrInvalid)
	}
	perm = forgejo.NormalizePerm(perm)
	want, err := c.ensureRoleTeam(group, perm)
	if err != nil {
		return err
	}
	if err := c.FJ.AddTeamMember(want.ID, login); err != nil {
		return err
	}
	for p, t := range g.teams {
		if p == perm || t.ID == want.ID {
			continue
		}
		_ = c.FJ.RemoveTeamMember(t.ID, login)
	}
	c.forget()
	return nil
}

func (c *Catalog) RemoveGroupMember(user, group, login string) error {
	if err := c.requireAdmin(user); err != nil {
		return err
	}
	g, err := c.findGroup(group)
	if err != nil {
		return err
	}
	for _, t := range g.teams {
		_ = c.FJ.RemoveTeamMember(t.ID, login)
	}
	c.forget()
	return nil
}

func (c *Catalog) AttachRepo(group, repo string) error {
	g, err := c.findGroup(group)
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
	if _, ok := g.teams[permWrite]; !ok {
		if _, err := c.ensureRoleTeam(group, permWrite); err != nil {
			return err
		}
		g, err = c.findGroup(group)
		if err != nil {
			return err
		}
	}
	for _, t := range g.teams {
		if err := c.FJ.AddTeamRepo(t.ID, c.Cfg.Org, repo); err != nil {
			return err
		}
	}
	c.forget()
	return nil
}

func (c *Catalog) AddGroupRepo(user, group, repo string) error {
	if err := c.requireAdmin(user); err != nil {
		return err
	}
	return c.AttachRepo(group, repo)
}

func (c *Catalog) RemoveGroupRepo(user, group, repo string) error {
	if err := c.requireAdmin(user); err != nil {
		return err
	}
	g, err := c.findGroup(group)
	if err != nil {
		return err
	}
	if i := strings.LastIndex(repo, "/"); i >= 0 {
		repo = repo[i+1:]
	}
	for _, t := range g.teams {
		_ = c.FJ.RemoveTeamRepo(t.ID, c.Cfg.Org, repo)
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
	if repo.Group != "" {
		if g, err := c.findGroup(repo.Group); err == nil {
			inherited, err = c.groupMembers(g)
			if err != nil {
				return RepoAccess{}, err
			}
		}
	}
	inGroup := map[string]bool{}
	for _, p := range inherited {
		inGroup[p.Login] = true
	}
	cols, err := page.Walk(func(q page.Query) (page.Result[forgejo.User], error) {
		return c.FJ.ListCollaborators(owner, name, q)
	})
	if err != nil {
		cols = nil
	}
	direct := make([]AccessPerson, 0, len(cols))
	for _, u := range cols {
		if inGroup[u.Login] {
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
		Group:     repo.Group,
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

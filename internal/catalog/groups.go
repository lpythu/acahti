package catalog

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"unicode"

	"acahti/internal/forgejo"
	"acahti/internal/page"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
	ErrInvalid   = errors.New("invalid")
)

const (
	permRead  = "read"
	permWrite = "write"
	permAdmin = "admin"
)

func ownersTeam(name string) bool {
	return strings.EqualFold(strings.TrimSpace(name), "Owners")
}

func ValidGroupName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || ownersTeam(name) {
		return fmt.Errorf("%w group name", ErrInvalid)
	}
	if len(name) > 64 {
		return fmt.Errorf("%w group name", ErrInvalid)
	}
	for i, r := range name {
		if i == 0 && !unicode.IsLetter(r) {
			return fmt.Errorf("%w group name", ErrInvalid)
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			continue
		}
		return fmt.Errorf("%w group name", ErrInvalid)
	}
	return nil
}

func roleTeamName(group, perm string) string {
	switch forgejo.NormalizePerm(perm) {
	case permRead:
		return group + ".read"
	case permAdmin:
		return group + ".admin"
	default:
		return group
	}
}

func parseRoleTeam(name string) (group, perm string, ok bool) {
	if ownersTeam(name) {
		return "", "", false
	}
	switch {
	case strings.HasSuffix(name, ".read"):
		g := strings.TrimSuffix(name, ".read")
		if ValidGroupName(g) != nil {
			return "", "", false
		}
		return g, permRead, true
	case strings.HasSuffix(name, ".admin"):
		g := strings.TrimSuffix(name, ".admin")
		if ValidGroupName(g) != nil {
			return "", "", false
		}
		return g, permAdmin, true
	default:
		if ValidGroupName(name) != nil {
			return "", "", false
		}
		return name, permWrite, true
	}
}

func visibleSet(repos []forgejo.Repo) map[string]bool {
	out := make(map[string]bool, len(repos))
	for _, r := range repos {
		if r.FullName != "" {
			out[r.FullName] = true
		}
	}
	return out
}

func markGroup(repos []forgejo.Repo, group string, visible map[string]bool) []forgejo.Repo {
	out := make([]forgejo.Repo, 0, len(repos))
	seen := map[string]bool{}
	for _, r := range repos {
		if r.FullName == "" || seen[r.FullName] {
			continue
		}
		if visible != nil && !visible[r.FullName] {
			continue
		}
		seen[r.FullName] = true
		r.Group = group
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].FullName < out[j].FullName })
	return out
}

func groupCounts(grouped map[string][]forgejo.Repo) []RepoGroup {
	out := make([]RepoGroup, 0, len(grouped))
	for g, repos := range grouped {
		out = append(out, RepoGroup{Group: g, Count: len(repos)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Group < out[j].Group })
	return out
}

func flattenGroups(grouped map[string][]forgejo.Repo, group string) []forgejo.Repo {
	if group != "" {
		repos := grouped[group]
		if repos == nil {
			return []forgejo.Repo{}
		}
		return repos
	}
	var out []forgejo.Repo
	for _, repos := range grouped {
		out = append(out, repos...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].FullName < out[j].FullName })
	if out == nil {
		out = []forgejo.Repo{}
	}
	return out
}

func (c *Catalog) IsOrgAdmin(user string) bool {
	if user == "" || !c.FJ.Ready() {
		return false
	}
	if c.mem != nil {
		if ok, hit := c.mem.adminOf(user); hit {
			return ok
		}
	}
	ok := false
	if user == c.Cfg.AdminUser {
		ok = true
	} else if u, err := c.FJ.UserSudo(user); err == nil && u.IsAdmin {
		ok = true
	} else if t, err := c.FJ.FindOrgTeam(c.Cfg.Org, "Owners"); err == nil {
		ok, _ = c.FJ.TeamHasMember(t.ID, user)
	}
	if c.mem != nil {
		c.mem.setAdmin(user, ok)
	}
	return ok
}

func (c *Catalog) orgTeams() ([]forgejo.Team, error) {
	return page.Walk(func(q page.Query) (page.Result[forgejo.Team], error) {
		return c.FJ.ListOrgTeams(c.Cfg.Org, q)
	})
}

func (c *Catalog) userRepos(user string) ([]forgejo.Repo, error) {
	if !c.FJ.Ready() {
		return []forgejo.Repo{}, nil
	}
	if c.mem != nil {
		if repos, ok := c.mem.reposOf(user); ok {
			return repos, nil
		}
	}
	var (
		repos []forgejo.Repo
		err   error
	)
	if c.IsOrgAdmin(user) {
		repos, err = page.Walk(func(q page.Query) (page.Result[forgejo.Repo], error) {
			return c.FJ.ListOrgRepos(c.Cfg.Org, q)
		})
	} else {
		repos, err = page.Walk(func(q page.Query) (page.Result[forgejo.Repo], error) {
			return c.FJ.ListRepos(user, q)
		})
	}
	if err != nil {
		return nil, err
	}
	if c.mem != nil {
		c.mem.setRepos(user, repos)
	}
	return repos, nil
}

func (c *Catalog) seeRepo(user, owner, name string) (forgejo.Repo, error) {
	repo, err := c.FJ.GetRepo(owner, name, user)
	if err != nil {
		return forgejo.Repo{}, fmt.Errorf("%w: %s", ErrNotFound, err)
	}
	repo.Group = c.repoGroup(owner, name)
	return repo, nil
}

func (c *Catalog) repoGroup(owner, name string) string {
	res, err := c.FJ.ListRepoTeams(owner, name, page.Query{Page: 1, Size: page.MaxSize})
	if err != nil {
		return ""
	}
	for _, t := range res.Items {
		if g, _, ok := parseRoleTeam(t.Name); ok {
			return g
		}
	}
	return ""
}

type groupTeams struct {
	name  string
	teams map[string]forgejo.Team
}

func (g groupTeams) ids() []int64 {
	var out []int64
	for _, t := range g.teams {
		out = append(out, t.ID)
	}
	return out
}

func (c *Catalog) clusterGroups() (map[string]groupTeams, error) {
	teams, err := c.orgTeams()
	if err != nil {
		return nil, err
	}
	out := map[string]groupTeams{}
	for _, t := range teams {
		g, perm, ok := parseRoleTeam(t.Name)
		if !ok {
			continue
		}
		cur := out[g]
		cur.name = g
		if cur.teams == nil {
			cur.teams = map[string]forgejo.Team{}
		}
		cur.teams[perm] = t
		out[g] = cur
	}
	return out, nil
}

func (c *Catalog) teamRepos(ids []int64) ([]forgejo.Repo, error) {
	parts := make([][]forgejo.Repo, len(ids))
	errs := make([]error, len(ids))
	var wg sync.WaitGroup
	for i, id := range ids {
		wg.Add(1)
		go func(i int, id int64) {
			defer wg.Done()
			repos, err := page.Walk(func(q page.Query) (page.Result[forgejo.Repo], error) {
				return c.FJ.ListTeamRepos(id, q)
			})
			parts[i], errs[i] = repos, err
		}(i, id)
	}
	wg.Wait()
	var all []forgejo.Repo
	for i, err := range errs {
		if err != nil {
			return nil, err
		}
		all = append(all, parts[i]...)
	}
	return all, nil
}

func (c *Catalog) inGroup(user string, g groupTeams) bool {
	for _, t := range g.teams {
		ok, err := c.FJ.TeamHasMember(t.ID, user)
		if err == nil && ok {
			return true
		}
	}
	return false
}

func (c *Catalog) grouped(user string) (map[string][]forgejo.Repo, error) {
	if c.mem != nil {
		if g, ok := c.mem.groupsOf(user); ok {
			return g, nil
		}
	}
	visibleRepos, err := c.userRepos(user)
	if err != nil {
		return nil, err
	}
	visible := visibleSet(visibleRepos)
	if !c.FJ.Ready() {
		return map[string][]forgejo.Repo{}, nil
	}
	admin := c.IsOrgAdmin(user)
	clusters, err := c.clusterGroups()
	if err != nil {
		return nil, err
	}
	type item struct {
		name string
		g    groupTeams
	}
	items := make([]item, 0, len(clusters))
	for name, g := range clusters {
		items = append(items, item{name: name, g: g})
	}
	type row struct {
		name string
		keep []forgejo.Repo
		skip bool
		err  error
	}
	rows := make([]row, len(items))
	var wg sync.WaitGroup
	for i, it := range items {
		wg.Add(1)
		go func(i int, it item) {
			defer wg.Done()
			repos, err := c.teamRepos(it.g.ids())
			if err != nil {
				rows[i].err = err
				return
			}
			if admin {
				rows[i] = row{name: it.name, keep: markGroup(repos, it.name, nil)}
				return
			}
			keep := markGroup(repos, it.name, visible)
			rows[i] = row{name: it.name, keep: keep, skip: len(keep) == 0 && !c.inGroup(user, it.g)}
		}(i, it)
	}
	wg.Wait()
	out := map[string][]forgejo.Repo{}
	for _, r := range rows {
		if r.err != nil {
			return nil, r.err
		}
		if r.skip {
			continue
		}
		out[r.name] = r.keep
	}
	if c.mem != nil {
		c.mem.setGroups(user, out)
	}
	return out, nil
}

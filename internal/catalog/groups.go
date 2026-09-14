package catalog

import (
	"errors"
	"fmt"
	"sort"
	"strings"
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
	if user == c.Cfg.AdminUser {
		return true
	}
	if u, err := c.FJ.UserSudo(user); err == nil && u.IsAdmin {
		return true
	}
	t, err := c.FJ.FindOrgTeam(c.Cfg.Org, "Owners")
	if err != nil {
		return false
	}
	ok, err := c.FJ.TeamHasMember(t.ID, user)
	return err == nil && ok
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
	if c.IsOrgAdmin(user) {
		return page.Walk(func(q page.Query) (page.Result[forgejo.Repo], error) {
			return c.FJ.ListOrgRepos(c.Cfg.Org, q)
		})
	}
	return page.Walk(func(q page.Query) (page.Result[forgejo.Repo], error) {
		return c.FJ.ListRepos(user, q)
	})
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
	var all []forgejo.Repo
	for _, id := range ids {
		repos, err := page.Walk(func(q page.Query) (page.Result[forgejo.Repo], error) {
			return c.FJ.ListTeamRepos(id, q)
		})
		if err != nil {
			return nil, err
		}
		all = append(all, repos...)
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
	out := map[string][]forgejo.Repo{}
	for name, g := range clusters {
		repos, err := c.teamRepos(g.ids())
		if err != nil {
			return nil, err
		}
		var keep []forgejo.Repo
		if admin {
			keep = markGroup(repos, name, nil)
		} else {
			keep = markGroup(repos, name, visible)
			if len(keep) == 0 && !c.inGroup(user, g) {
				continue
			}
		}
		out[name] = keep
	}
	return out, nil
}

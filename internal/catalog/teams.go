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

func ValidTeamName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || ownersTeam(name) {
		return fmt.Errorf("%w team name", ErrInvalid)
	}
	if len(name) > 64 {
		return fmt.Errorf("%w team name", ErrInvalid)
	}
	for i, r := range name {
		if i == 0 && !unicode.IsLetter(r) {
			return fmt.Errorf("%w team name", ErrInvalid)
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			continue
		}
		return fmt.Errorf("%w team name", ErrInvalid)
	}
	return nil
}

func roleTeamName(team, perm string) string {
	switch forgejo.NormalizePerm(perm) {
	case permRead:
		return team + ".read"
	case permAdmin:
		return team + ".admin"
	default:
		return team
	}
}

func parseRoleTeam(name string) (team, perm string, ok bool) {
	if ownersTeam(name) {
		return "", "", false
	}
	switch {
	case strings.HasSuffix(name, ".read"):
		t := strings.TrimSuffix(name, ".read")
		if ValidTeamName(t) != nil {
			return "", "", false
		}
		return t, permRead, true
	case strings.HasSuffix(name, ".admin"):
		t := strings.TrimSuffix(name, ".admin")
		if ValidTeamName(t) != nil {
			return "", "", false
		}
		return t, permAdmin, true
	default:
		if ValidTeamName(name) != nil {
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

func markTeam(repos []forgejo.Repo, team string, visible map[string]bool) []forgejo.Repo {
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
		r.Team = team
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].FullName < out[j].FullName })
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
	if c.mem != nil {
		if teams, ok := c.mem.teamsOf(); ok {
			return teams, nil
		}
	}
	teams, err := page.Walk(func(q page.Query) (page.Result[forgejo.Team], error) {
		return c.FJ.ListOrgTeams(c.Cfg.Org, q)
	})
	if err != nil {
		return nil, err
	}
	if c.mem != nil {
		c.mem.setTeams(teams)
	}
	return teams, nil
}

func (c *Catalog) seeOK(user, owner, name string) error {
	if _, err := c.FJ.GetRepo(owner, name, user); err != nil {
		return fmt.Errorf("%w: %s", ErrNotFound, err)
	}
	return nil
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
	repo.Team = c.repoTeam(owner, name)
	return repo, nil
}

func (c *Catalog) repoTeam(owner, name string) string {
	res, err := c.FJ.ListRepoTeams(owner, name, page.Query{Page: 1, Size: page.MaxSize})
	if err != nil {
		return ""
	}
	for _, t := range res.Items {
		if team, _, ok := parseRoleTeam(t.Name); ok {
			return team
		}
	}
	return ""
}

type teamRoles struct {
	name  string
	roles map[string]forgejo.Team
}

func (t teamRoles) ids() []int64 {
	var out []int64
	for _, r := range t.roles {
		out = append(out, r.ID)
	}
	return out
}

func (t teamRoles) writeID() int64 {
	if r, ok := t.roles[permWrite]; ok {
		return r.ID
	}
	for _, r := range t.roles {
		return r.ID
	}
	return 0
}

func (c *Catalog) clusterTeams() (map[string]teamRoles, error) {
	teams, err := c.orgTeams()
	if err != nil {
		return nil, err
	}
	out := map[string]teamRoles{}
	for _, t := range teams {
		name, perm, ok := parseRoleTeam(t.Name)
		if !ok {
			continue
		}
		cur := out[name]
		cur.name = name
		if cur.roles == nil {
			cur.roles = map[string]forgejo.Team{}
		}
		cur.roles[perm] = t
		out[name] = cur
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

func (c *Catalog) inTeam(user string, t teamRoles) bool {
	for _, r := range t.roles {
		ok, err := c.FJ.TeamHasMember(r.ID, user)
		if err == nil && ok {
			return true
		}
	}
	return false
}

package catalog

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"acahti/internal/forgejo"
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

func (c *Catalog) seeOK(user, owner, name string) error {
	if _, err := c.FJ.GetRepo(owner, name, user); err != nil {
		return fmt.Errorf("%w: %s", ErrNotFound, err)
	}
	return nil
}

func (c *Catalog) userRepos(user string) ([]forgejo.Repo, error) {
	if !c.indexed() {
		return []forgejo.Repo{}, nil
	}
	repos, err := c.Idx.VisibleRepos(user, c.IsOrgAdmin(user))
	if err != nil {
		return nil, err
	}
	items := c.asRepos(repos, "")
	c.paintPerms(user, items)
	return items, nil
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
	if !c.indexed() {
		return ""
	}
	team, ok, err := c.Idx.RepoTeam(owner + "/" + name)
	if err != nil || !ok {
		return ""
	}
	return team
}

type teamRoles struct {
	name  string
	roles map[string]forgejo.Team
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

func clusterFrom(teams []forgejo.Team) map[string]teamRoles {
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
	return out
}

func (c *Catalog) teamRepos(name string) ([]forgejo.Repo, error) {
	if !c.indexed() || name == "" {
		return nil, nil
	}
	rows, err := c.Idx.TeamRepos(name)
	if err != nil {
		return nil, err
	}
	return c.asRepos(rows, name), nil
}

func (c *Catalog) inTeam(user string, t teamRoles) bool {
	if user == "" || t.name == "" {
		return false
	}
	if !c.indexed() {
		return false
	}
	ok, err := c.Idx.HasMember(t.name, user)
	return err == nil && ok
}

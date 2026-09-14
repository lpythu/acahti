package catalog

import (
	"sort"
	"strings"

	"acahti/internal/forgejo"
	"acahti/internal/page"
)

func ownersTeam(name string) bool {
	return strings.EqualFold(strings.TrimSpace(name), "Owners")
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
	for _, r := range repos {
		if !visible[r.FullName] {
			continue
		}
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

func (c *Catalog) userRepos(user string) ([]forgejo.Repo, error) {
	if !c.FJ.Ready() {
		return []forgejo.Repo{}, nil
	}
	return page.Walk(func(q page.Query) (page.Result[forgejo.Repo], error) {
		return c.FJ.ListRepos(user, q)
	})
}

func (c *Catalog) seeRepo(user, owner, name string) (forgejo.Repo, error) {
	repo, err := c.FJ.GetRepo(owner, name, user)
	if err != nil {
		return forgejo.Repo{}, err
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
		if !ownersTeam(t.Name) {
			return t.Name
		}
	}
	return ""
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
	teams, err := page.Walk(func(q page.Query) (page.Result[forgejo.Team], error) {
		return c.FJ.ListOrgTeams(c.Cfg.Org, q)
	})
	if err != nil {
		return nil, err
	}
	out := map[string][]forgejo.Repo{}
	assigned := map[string]bool{}
	for _, t := range teams {
		if ownersTeam(t.Name) {
			continue
		}
		repos, err := page.Walk(func(q page.Query) (page.Result[forgejo.Repo], error) {
			return c.FJ.ListTeamRepos(t.ID, q)
		})
		if err != nil {
			return nil, err
		}
		keep := markGroup(repos, t.Name, visible)
		if len(keep) == 0 {
			continue
		}
		out[t.Name] = keep
		for _, r := range keep {
			assigned[r.FullName] = true
		}
	}
	var loose []forgejo.Repo
	for _, r := range visibleRepos {
		if assigned[r.FullName] {
			continue
		}
		r.Group = c.Cfg.Org
		loose = append(loose, r)
	}
	if len(out) == 0 && len(loose) > 0 {
		out[c.Cfg.Org] = markGroup(loose, c.Cfg.Org, visible)
		return out, nil
	}
	if len(loose) > 0 {
		out[c.Cfg.Org] = append(out[c.Cfg.Org], markGroup(loose, c.Cfg.Org, visible)...)
		sort.Slice(out[c.Cfg.Org], func(i, j int) bool { return out[c.Cfg.Org][i].FullName < out[c.Cfg.Org][j].FullName })
	}
	return out, nil
}

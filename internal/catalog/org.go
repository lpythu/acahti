package catalog

import (
	"log"
	"strings"
	"time"

	"acahti/internal/forgejo"
	"acahti/internal/page"
	"acahti/internal/store"
)

func (c *Catalog) publishCatalog() {
	if c != nil && c.Notify != nil {
		c.Notify("catalog.updated", map[string]any{"ok": true})
	}
}

func (c *Catalog) indexed() bool {
	return c != nil && c.Idx != nil && c.Idx.Ready()
}

type teamRepoHit struct {
	team, repo, name string
	roles            teamRoles
}

func folderLinkKey(team, repo string) string {
	return team + "\x00" + repo
}

func forgejoTeamReposToStrip(folder map[string]store.TeamRepoLink, fj []teamRepoHit) []teamRepoHit {
	var out []teamRepoHit
	for _, hit := range fj {
		if l, ok := folder[folderLinkKey(hit.team, hit.repo)]; ok && l.Granted {
			continue
		}
		out = append(out, hit)
	}
	return out
}

func grantedFolderLinks(folder map[string]store.TeamRepoLink) []store.TeamRepoLink {
	out := make([]store.TeamRepoLink, 0)
	for _, l := range folder {
		if l.Granted {
			out = append(out, l)
		}
	}
	return out
}

func (c *Catalog) asRepo(r store.OrgRepo, team string) forgejo.Repo {
	name := r.FullName
	if _, n, ok := strings.Cut(r.FullName, "/"); ok {
		name = n
	}
	return c.PublicRepo(forgejo.Repo{
		Name:          name,
		FullName:      r.FullName,
		DefaultBranch: r.DefaultBranch,
		Team:          team,
		Updated:       r.Updated,
		Archived:      r.Archived,
	})
}

func asOrgRepo(r forgejo.Repo) store.OrgRepo {
	return store.OrgRepo{
		FullName:      r.FullName,
		DefaultBranch: r.DefaultBranch,
		Updated:       orgUpdated(r),
		Archived:      r.Archived,
	}
}

func orgUpdated(r forgejo.Repo) int64 {
	if r.UpdatedUnix > 0 {
		return r.UpdatedUnix
	}
	if r.Updated > 0 {
		return r.Updated
	}
	return time.Now().Unix()
}

func (c *Catalog) asRepos(rows []store.OrgRepo, team string) []forgejo.Repo {
	out := make([]forgejo.Repo, 0, len(rows))
	for _, r := range rows {
		out = append(out, c.asRepo(r, team))
	}
	return out
}

func rolesFrom(t store.OrgTeam) teamRoles {
	out := teamRoles{name: t.Name, roles: map[string]forgejo.Team{}}
	if t.WriteID > 0 {
		out.roles[permWrite] = forgejo.Team{ID: t.WriteID, Name: t.Name, Permission: permWrite}
	}
	if t.ReadID > 0 {
		out.roles[permRead] = forgejo.Team{ID: t.ReadID, Name: roleTeamName(t.Name, permRead), Permission: permRead}
	}
	if t.AdminID > 0 {
		out.roles[permAdmin] = forgejo.Team{ID: t.AdminID, Name: roleTeamName(t.Name, permAdmin), Permission: permAdmin}
	}
	return out
}

func (c *Catalog) RememberRepo(r forgejo.Repo) {
	if !c.indexed() || r.FullName == "" {
		return
	}
	_ = c.Idx.UpsertRepo(asOrgRepo(r))
	c.publishCatalog()
}

func (c *Catalog) syncTeamIDs(name string) {
	if !c.indexed() || name == "" || c.FJ == nil || !c.FJ.Ready() {
		return
	}
	t := store.OrgTeam{Name: name}
	if w, err := c.FJ.FindOrgTeam(c.Cfg.Org, name); err == nil {
		t.WriteID = w.ID
	}
	if r, err := c.FJ.FindOrgTeam(c.Cfg.Org, roleTeamName(name, permRead)); err == nil {
		t.ReadID = r.ID
	}
	if a, err := c.FJ.FindOrgTeam(c.Cfg.Org, roleTeamName(name, permAdmin)); err == nil {
		t.AdminID = a.ID
	}
	_ = c.Idx.UpsertTeam(t)
}

func (c *Catalog) ApplyForgejoCatalog(payload map[string]any) bool {
	if !c.indexed() {
		return false
	}
	action, repo, ok := store.ParseForgejoRepoEvent(payload)
	if !ok {
		return false
	}
	switch action {
	case "deleted":
		_ = c.Idx.DeleteRepo(repo.FullName)
	default:
		_ = c.Idx.UpsertRepo(repo)
	}
	c.publishCatalog()
	return true
}

func (c *Catalog) BackfillOrg() {
	if !c.indexed() || c.FJ == nil || !c.FJ.Ready() {
		return
	}
	var (
		raw []forgejo.Team
		err error
	)
	for i := 0; i < 20; i++ {
		raw, err = page.Walk(func(q page.Query) (page.Result[forgejo.Team], error) {
			return c.FJ.ListOrgTeams(c.Cfg.Org, q)
		})
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Printf("org backfill: teams: %v", err)
		return
	}
	orgRepos, err := page.Walk(func(q page.Query) (page.Result[forgejo.Repo], error) {
		return c.FJ.ListOrgRepos(c.Cfg.Org, q)
	})
	if err != nil {
		log.Printf("org backfill: repos: %v", err)
		return
	}
	clusters := clusterFrom(raw)
	teams := make([]store.OrgTeam, 0, len(clusters))
	seenRepo := map[string]store.OrgRepo{}
	for _, r := range orgRepos {
		if r.FullName == "" {
			continue
		}
		seenRepo[r.FullName] = asOrgRepo(r)
	}
	var members []store.OrgMember
	var collabs []store.OrgCollaborator
	existing, err := c.Idx.ListTeamRepoLinks()
	if err != nil {
		log.Printf("org backfill: folder links: %v", err)
		existing = nil
	}
	folder := map[string]store.TeamRepoLink{}
	for _, l := range existing {
		if l.Team == "" || l.Repo == "" {
			continue
		}
		folder[folderLinkKey(l.Team, l.Repo)] = l
	}
	var fjRepos []teamRepoHit
	for name, t := range clusters {
		ot := store.OrgTeam{Name: name}
		if r, ok := t.roles[permWrite]; ok {
			ot.WriteID = r.ID
		}
		if r, ok := t.roles[permRead]; ok {
			ot.ReadID = r.ID
		}
		if r, ok := t.roles[permAdmin]; ok {
			ot.AdminID = r.ID
		}
		teams = append(teams, ot)
		byLogin := map[string]string{}
		for perm, role := range t.roles {
			users, err := page.Walk(func(q page.Query) (page.Result[forgejo.User], error) {
				return c.FJ.ListTeamMembers(role.ID, q)
			})
			if err != nil {
				log.Printf("org backfill: members %s: %v", role.Name, err)
				continue
			}
			for _, u := range users {
				if u.Login == "" {
					continue
				}
				byLogin[u.Login] = strongerPerm(byLogin[u.Login], perm)
			}
		}
		for login, role := range byLogin {
			members = append(members, store.OrgMember{Team: name, Login: login, Role: role})
		}
		id := t.writeID()
		if id == 0 {
			continue
		}
		owned, err := page.Walk(func(q page.Query) (page.Result[forgejo.Repo], error) {
			return c.FJ.ListTeamRepos(id, q)
		})
		if err != nil {
			log.Printf("org backfill: team repos %s: %v", name, err)
			continue
		}
		for _, r := range owned {
			if r.FullName == "" {
				continue
			}
			if _, ok := seenRepo[r.FullName]; !ok {
				seenRepo[r.FullName] = asOrgRepo(r)
			}
			short := r.Name
			if short == "" {
				_, short, _ = strings.Cut(r.FullName, "/")
			}
			fjRepos = append(fjRepos, teamRepoHit{team: name, repo: r.FullName, name: short, roles: t})
		}
	}
	repos := make([]store.OrgRepo, 0, len(seenRepo))
	for _, r := range seenRepo {
		repos = append(repos, r)
	}
	for _, r := range repos {
		owner, name, ok := strings.Cut(r.FullName, "/")
		if !ok || owner == "" || name == "" {
			continue
		}
		users, err := page.Walk(func(q page.Query) (page.Result[forgejo.User], error) {
			return c.FJ.ListCollaborators(owner, name, q)
		})
		if err != nil {
			log.Printf("org backfill: collaborators %s: %v", r.FullName, err)
			continue
		}
		for _, u := range users {
			if u.Login == "" {
				continue
			}
			perm := forgejo.NormalizePerm(u.Permissions.Level())
			if !u.Permissions.Admin && !u.Permissions.Push {
				if p, err := c.FJ.CollaboratorPerm(owner, name, u.Login); err == nil && p != "" {
					perm = p
				}
			}
			collabs = append(collabs, store.OrgCollaborator{Repo: r.FullName, Login: u.Login, Role: perm})
		}
	}
	for _, hit := range forgejoTeamReposToStrip(folder, fjRepos) {
		for _, role := range hit.roles.roles {
			_ = c.FJ.RemoveTeamRepo(role.ID, c.Cfg.Org, hit.name)
		}
	}
	for _, l := range grantedFolderLinks(folder) {
		t, ok := clusters[l.Team]
		if !ok {
			continue
		}
		_, short, _ := strings.Cut(l.Repo, "/")
		if short == "" {
			continue
		}
		for _, role := range t.roles {
			_ = c.FJ.AddTeamRepo(role.ID, c.Cfg.Org, short)
		}
	}
	links := make([]store.TeamRepoLink, 0, len(folder))
	for _, l := range folder {
		links = append(links, l)
	}
	if err := c.Idx.ReplaceOrg(teams, repos, links, members, collabs); err != nil {
		log.Printf("org backfill: replace: %v", err)
		return
	}
	c.publishCatalog()
}

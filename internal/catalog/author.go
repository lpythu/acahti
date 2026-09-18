package catalog

import (
	"strings"

	"acahti/internal/forgejo"
	"acahti/internal/identity"
	"acahti/internal/woodpecker"
)

func gitAuthor(cm forgejo.Commit) (name, login, avatar, email string) {
	name = strings.TrimSpace(cm.Commit.Author.Name)
	email = strings.TrimSpace(cm.Commit.Author.Email)
	if cm.Author != nil {
		login = strings.TrimSpace(cm.Author.Login)
		avatar = strings.TrimSpace(cm.Author.AvatarURL)
		if name == "" {
			name = identity.Name(login, cm.Author.FullName)
		}
	}
	return name, login, avatar, email
}

func (c *Catalog) applyCommitAuthor(p woodpecker.Pipeline) woodpecker.Pipeline {
	name, login, avatar, _ := c.commitWho(p.Repo, p.Commit)
	switch {
	case name != "":
		p.Author = name
	case login != "":
		p.Author = c.authorOf(login)
	default:
		p.Author = c.authorOf(p.Author)
	}
	if avatar != "" {
		p.Avatar = avatar
	}
	return p
}

func (c *Catalog) commitWho(repo, sha string) (name, login, avatar, email string) {
	repo, sha = strings.TrimSpace(repo), strings.TrimSpace(sha)
	if c == nil || c.fj == nil || repo == "" || sha == "" {
		return "", "", "", ""
	}
	if c.mem != nil {
		if w, ok := c.mem.commitWhoOf(repo, sha); ok {
			return w.Name, w.Login, w.Avatar, w.Email
		}
	}
	owner, repoName, ok := strings.Cut(repo, "/")
	if !ok {
		return "", "", "", ""
	}
	cm, err := c.fj.GetCommit(owner, repoName, sha)
	if err != nil {
		return "", "", "", ""
	}
	name, login, avatar, email = gitAuthor(cm)
	if c.mem != nil {
		c.mem.setCommitWho(repo, sha, commitWho{Name: name, Login: login, Avatar: avatar, Email: email})
	}
	return name, login, avatar, email
}

func JobUser(login, email, name, fallback, admin, domain string, names map[string]string) string {
	admin = strings.TrimSpace(admin)
	if u := identity.LoginFromNoreply(email, domain); u != "" && u != admin {
		return u
	}
	login = strings.TrimSpace(login)
	if login != "" && login != admin {
		return login
	}
	name = strings.TrimSpace(name)
	for u, display := range names {
		if strings.TrimSpace(display) == name && u != "" && u != admin {
			return u
		}
	}
	return strings.TrimSpace(fallback)
}

func (c *Catalog) JobLogin(repo, sha, fallback, admin, domain string) string {
	name, login, _, email := c.commitWho(repo, sha)
	return JobUser(login, email, name, fallback, admin, domain, c.authorNames())
}

func (c *Catalog) authorOf(login string) string {
	login = strings.TrimSpace(login)
	if login == "" {
		return ""
	}
	if n := c.authorNames()[login]; n != "" {
		return n
	}
	return login
}

func (c *Catalog) withAuthors(people []AccessPerson) []AccessPerson {
	names := c.authorNames()
	for i := range people {
		if n := names[people[i].Login]; n != "" {
			people[i].Author = n
			continue
		}
		people[i].Author = identity.Name(people[i].Login, people[i].Author)
	}
	return people
}

func (c *Catalog) authorNames() map[string]string {
	if c == nil || c.fj == nil || c.mem == nil {
		return map[string]string{}
	}
	if m, ok := c.mem.authorsOf(); ok {
		return m
	}
	users, err := c.fj.AllUsers()
	if err != nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(users))
	for _, u := range users {
		out[u.Login] = identity.Name(u.Login, u.FullName)
	}
	c.mem.setAuthors(out)
	return out
}

package catalog

import (
	"strings"

	"acahti/internal/identity"
)

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
	if c == nil || c.FJ == nil || c.mem == nil {
		return map[string]string{}
	}
	if m, ok := c.mem.authorsOf(); ok {
		return m
	}
	users, err := c.FJ.AllUsers()
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

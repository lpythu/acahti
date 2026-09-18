package catalog

import (
	"fmt"
	"strings"
	"unicode"

	"acahti/internal/store"
)

func NormOrg(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func ValidOrgName(name string) error {
	name = NormOrg(name)
	if name == "" || len(name) > 40 {
		return fmt.Errorf("%w org name", ErrInvalid)
	}
	for i, r := range name {
		if i == 0 && !(unicode.IsLetter(r) || unicode.IsDigit(r)) {
			return fmt.Errorf("%w org name", ErrInvalid)
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			continue
		}
		return fmt.Errorf("%w org name", ErrInvalid)
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return fmt.Errorf("%w org name", ErrInvalid)
	}
	return nil
}

func (c *Catalog) ForOrg(org string) *Catalog {
	org = NormOrg(org)
	if c == nil || org == "" || org == NormOrg(c.Cfg.Org) {
		return c
	}
	cp := *c
	cp.Cfg.Org = org
	return &cp
}

func (c *Catalog) inOrg(full string) bool {
	owner, _, ok := strings.Cut(full, "/")
	return ok && strings.EqualFold(owner, c.Cfg.Org)
}

func (c *Catalog) orgRepos(repos []store.OrgRepo) []store.OrgRepo {
	out := make([]store.OrgRepo, 0, len(repos))
	for _, r := range repos {
		if c.inOrg(r.FullName) {
			out = append(out, r)
		}
	}
	return out
}

func (c *Catalog) orgNames(names []string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		if c.inOrg(n) {
			out = append(out, n)
		}
	}
	return out
}

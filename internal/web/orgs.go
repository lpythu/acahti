package web

import (
	"encoding/json"
	"net/http"
	"strings"

	"acahti/internal/catalog"
	"acahti/internal/forgejo"
	"acahti/internal/identity"
	"acahti/internal/page"
)

const orgCookie = "acahti_org"

type Org struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

func (p *Pages) cat(r *http.Request) *catalog.Catalog {
	return p.Cat.ForOrg(p.activeOrg(r))
}

func (p *Pages) activeOrg(r *http.Request) string {
	if c, err := r.Cookie(orgCookie); err == nil {
		if name := catalog.NormOrg(c.Value); name != "" {
			return name
		}
	}
	return catalog.NormOrg(p.Cfg.Org)
}

func (p *Pages) setOrgCookie(w http.ResponseWriter, org string) {
	http.SetCookie(w, &http.Cookie{
		Name:     orgCookie,
		Value:    catalog.NormOrg(org),
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(p.Auth.TTL.Seconds()),
	})
}

func (p *Pages) clearOrgCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: orgCookie, Path: "/", MaxAge: -1})
}

func (p *Pages) listOrgs(user string) []Org {
	var (
		raw []forgejo.Org
		err error
	)
	if p.isAdmin(user) {
		raw, err = page.Walk(func(q page.Query) (page.Result[forgejo.Org], error) {
			return p.Cat.ListAdminOrgs(q)
		})
	}
	if err != nil || raw == nil {
		raw, _ = page.Walk(func(q page.Query) (page.Result[forgejo.Org], error) {
			return p.Cat.ListUserOrgs(user, q)
		})
	}
	seen := map[string]struct{}{}
	out := make([]Org, 0, len(raw)+1)
	add := func(name, full string) {
		name = catalog.NormOrg(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		if strings.TrimSpace(full) == "" {
			full = name
		}
		out = append(out, Org{Name: name, FullName: full})
	}
	for _, o := range raw {
		add(o.Slug(), o.FullName)
	}
	add(p.Cfg.Org, p.Cfg.Org)
	return out
}

func hasOrg(orgs []Org, name string) bool {
	name = catalog.NormOrg(name)
	for _, o := range orgs {
		if o.Name == name {
			return true
		}
	}
	return false
}

func (p *Pages) pickOrg(r *http.Request, orgs []Org) string {
	org := p.activeOrg(r)
	if hasOrg(orgs, org) {
		return org
	}
	home := catalog.NormOrg(p.Cfg.Org)
	if hasOrg(orgs, home) {
		return home
	}
	if len(orgs) > 0 {
		return orgs[0].Name
	}
	return home
}

func (p *Pages) writeSession(w http.ResponseWriter, r *http.Request, login, author string, admin bool) {
	orgs := p.listOrgs(login)
	org := p.pickOrg(r, orgs)
	v := identity.Session(login, author, admin, p.Cfg.RootURL, p.Cfg.Domain, org)
	v["orgs"] = orgs
	writeJSON(w, http.StatusOK, v)
}

func (p *Pages) Orgs(w http.ResponseWriter, r *http.Request) {
	user, admin, ok := p.requireJSON(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"items": p.listOrgs(user)})
	case http.MethodPost:
		if !admin {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin only"})
			return
		}
		var body struct {
			Name     string `json:"name"`
			FullName string `json:"full_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		name := catalog.NormOrg(body.Name)
		if err := catalog.ValidOrgName(name); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		org, err := p.Cat.CreateOrg(user, name, strings.TrimSpace(body.FullName))
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		}
		slug := catalog.NormOrg(org.Slug())
		if slug == "" {
			slug = name
		}
		full := strings.TrimSpace(org.FullName)
		if full == "" {
			full = slug
		}
		p.setOrgCookie(w, slug)
		writeJSON(w, http.StatusOK, Org{Name: slug, FullName: full})
	case http.MethodPut:
		var body struct {
			Org string `json:"org"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		name := catalog.NormOrg(body.Org)
		if !hasOrg(p.listOrgs(user), name) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "org not found"})
			return
		}
		p.setOrgCookie(w, name)
		author := ""
		if u, err := p.Cat.User(user); err == nil {
			author = u.FullName
		}
		req := r.Clone(r.Context())
		req.AddCookie(&http.Cookie{Name: orgCookie, Value: name})
		p.writeSession(w, req, user, author, admin)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
	}
}

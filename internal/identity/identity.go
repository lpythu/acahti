package identity

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func Domain(rootURL, domain string) string {
	if d := strings.TrimSpace(domain); d != "" {
		return strings.ToLower(d)
	}
	u, err := url.Parse(strings.TrimSpace(rootURL))
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

func Email(login, domain string) string {
	login = strings.TrimSpace(login)
	domain = Domain("", domain)
	if login == "" || domain == "" {
		return ""
	}
	return login + "@noreply." + domain
}

func Name(login, fullName string) string {
	if s := strings.TrimSpace(fullName); s != "" {
		return s
	}
	return strings.TrimSpace(login)
}

func Hosts(rootURL, domain string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(h string) {
		h = strings.ToLower(strings.TrimSpace(h))
		if h == "" {
			return
		}
		if _, ok := seen[h]; ok {
			return
		}
		seen[h] = struct{}{}
		out = append(out, h)
	}
	add(domain)
	if u, err := url.Parse(strings.TrimSpace(rootURL)); err == nil {
		add(u.Hostname())
	}
	return out
}

func SetupLocal(name, email string) []string {
	return []string{
		"git config --local user.name " + strconv.Quote(name),
		"git config --local user.email " + strconv.Quote(email),
	}
}

func View(login, fullName, rootURL, domain, org string) map[string]any {
	d := Domain(rootURL, domain)
	name := Name(login, fullName)
	email := Email(login, d)
	return map[string]any{
		"login":                  login,
		"git_name":               name,
		"git_email":              email,
		"root_url":               rootURL,
		"org":                    org,
		"apply_when_remote_host": Hosts(rootURL, domain),
		"setup_local":            SetupLocal(name, email),
	}
}

func Session(login string, admin bool, rootURL, domain, org string) map[string]any {
	v := View(login, login, rootURL, domain, org)
	v["user"] = login
	v["admin"] = admin
	return v
}

func Instructions(rootURL, domain string) string {
	hosts := Hosts(rootURL, domain)
	list := strings.Join(hosts, ", ")
	if list == "" {
		list = "the acahti host"
	}
	return fmt.Sprintf("Git author: before any commit, run git remote -v and call whoami. If a remote URL host is in apply_when_remote_host (%s), run setup_local in that repository (git config --local only). Do not use git config --global. Remotes on github.com, codeup.aliyun.com, or any other host keep the laptop identity. This applies to business repos hosted on this acahti (for example api-gateway), not the acahti product repo on GitHub.", list)
}

package identity

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"acahti/skills"
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

func LoginFromNoreply(email, domain string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	domain = strings.ToLower(strings.TrimSpace(domain))
	if email == "" || domain == "" {
		return ""
	}
	user, ok := strings.CutSuffix(email, "@noreply."+domain)
	if !ok || user == "" || strings.Contains(user, "@") {
		return ""
	}
	return user
}

func Name(login, author string) string {
	if s := strings.TrimSpace(author); s != "" {
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
	const legacy, current = "acahti.saidc.ai", "acahti.s-aidc.com"
	for _, h := range out {
		if h == legacy || h == current {
			add(legacy)
			add(current)
			break
		}
	}
	return out
}

func SetupLocal(name, email string) []string {
	return []string{
		"git config --local user.name " + strconv.Quote(name),
		"git config --local user.email " + strconv.Quote(email),
	}
}

func View(login, author, rootURL, domain, org string) map[string]any {
	d := Domain(rootURL, domain)
	name := Name(login, author)
	email := Email(login, d)
	root := strings.TrimRight(strings.TrimSpace(rootURL), "/")
	return map[string]any{
		"login":                  login,
		"git_name":               name,
		"git_email":              email,
		"root_url":               rootURL,
		"org":                    org,
		"clone_url_template":     root + "/" + org + "/<repo>.git",
		"skill_url":              root + "/skill.md",
		"skill_sha":              skills.SHA(rootURL, org, d),
		"apply_when_remote_host": Hosts(rootURL, domain),
		"setup_local":            SetupLocal(name, email),
	}
}

func Session(login, author string, admin bool, rootURL, domain, org string) map[string]any {
	v := View(login, author, rootURL, domain, org)
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
	root := strings.TrimRight(strings.TrimSpace(rootURL), "/")
	return fmt.Sprintf("GET %s/skill.md this turn and follow it. Discover MCP tools this session. Git author: resolve the actual intended push destination, including push URLs, before changing identity; never infer it from directory or repo name. Only for commits and pushes targeting a host in apply_when_remote_host (%s), call fresh MCP whoami and run setup_local (git config --local only) with whoami.git_name and whoami.git_email. Verify author and committer before committing and outgoing commits before pushing. Preserve original authorship for imported or replayed commits. Do not use git config --global. Leave non-Acahti destinations unchanged, even when another remote points to Acahti. Restore any temporary local identity changes after the Acahti operation. If the destination is ambiguous, clarify before changing identity.", root, list)
}

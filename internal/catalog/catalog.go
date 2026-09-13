package catalog

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"acahti/internal/config"
	"acahti/internal/forgejo"
	"acahti/internal/woodpecker"
)

type Catalog struct {
	Cfg config.Config
	FJ  *forgejo.Client
	WP  *woodpecker.Client
}

func New(cfg config.Config, fj *forgejo.Client, wp *woodpecker.Client) *Catalog {
	return &Catalog{Cfg: cfg, FJ: fj, WP: wp}
}

type BranchInfo struct {
	Name      string `json:"name"`
	SHA       string `json:"sha"`
	Default   bool   `json:"default"`
	Protected bool   `json:"protected"`
}

type FileBlob struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

type RepoOverview struct {
	Repo        forgejo.Repo           `json:"repo"`
	CloneHTTPS  string                 `json:"clone_https"`
	CloneSSH    string                 `json:"clone_ssh"`
	Ref         string                 `json:"ref"`
	Path        string                 `json:"path"`
	Branches    []BranchInfo           `json:"branches"`
	Protections []string               `json:"protections"`
	Commits     []forgejo.Commit       `json:"commits"`
	Entries     []forgejo.ContentEntry `json:"entries"`
	File        *FileBlob              `json:"file,omitempty"`
	Readme      string                 `json:"readme"`
	Pulls       []forgejo.PR           `json:"pulls"`
	Pipes       []woodpecker.Pipeline  `json:"pipes"`
}

type PackageGroup struct {
	Type     string            `json:"type"`
	Name     string            `json:"name"`
	Latest   string            `json:"latest"`
	Versions []forgejo.Package `json:"versions"`
}

type PRDetail struct {
	PR       forgejo.PR        `json:"pr"`
	Checks   []forgejo.Status  `json:"checks"`
	Green    bool              `json:"green"`
	Comments []forgejo.Comment `json:"comments"`
}

type PipelineDetail struct {
	Pipeline woodpecker.Pipeline `json:"pipeline"`
	Steps    []woodpecker.Step   `json:"steps"`
}

type Inbox struct {
	PRs     []forgejo.PR          `json:"prs"`
	Blocked []woodpecker.Pipeline `json:"blocked"`
	Failed  []woodpecker.Pipeline `json:"failed"`
	Agents  []woodpecker.Agent    `json:"agents"`
}

func (c *Catalog) CloneHTTPS(fullName string) string {
	return strings.TrimRight(c.Cfg.RootURL, "/") + "/" + fullName + ".git"
}

func (c *Catalog) CloneSSH(fullName string) string {
	host := strings.TrimSpace(c.Cfg.Domain)
	if host == "" {
		if u, err := url.Parse(c.Cfg.RootURL); err == nil {
			host = u.Hostname()
		}
	}
	if host == "" {
		host = "localhost"
	}
	port := strings.TrimSpace(c.Cfg.GitSSHPort)
	if port == "" || port == "22" {
		return fmt.Sprintf("ssh://git@%s/%s.git", host, fullName)
	}
	return fmt.Sprintf("ssh://git@%s:%s/%s.git", host, port, fullName)
}

func decodeContent(e forgejo.ContentEntry) string {
	raw := e.Content
	if e.Encoding == "base64" {
		if dec, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(raw, "\n", "")); err == nil {
			return string(dec)
		}
	}
	return raw
}

func (c *Catalog) ListRepos() ([]forgejo.Repo, error) {
	if !c.FJ.Ready() {
		return []forgejo.Repo{}, nil
	}
	repos, err := c.FJ.ListOrgRepos(c.Cfg.Org)
	if err != nil {
		return nil, err
	}
	if repos == nil {
		return []forgejo.Repo{}, nil
	}
	return repos, nil
}

func (c *Catalog) RepoOverview(owner, name, ref, path string) (RepoOverview, error) {
	repo, err := c.FJ.GetRepo(owner, name, "")
	if err != nil {
		return RepoOverview{}, err
	}
	if ref == "" {
		ref = repo.DefaultBranch
		if ref == "" {
			ref = "dev"
		}
	}
	if path == "." {
		path = ""
	}
	out := RepoOverview{
		Repo:        repo,
		CloneHTTPS:  c.CloneHTTPS(repo.FullName),
		CloneSSH:    c.CloneSSH(repo.FullName),
		Ref:         ref,
		Path:        path,
		Branches:    []BranchInfo{},
		Protections: []string{},
		Commits:     []forgejo.Commit{},
		Entries:     []forgejo.ContentEntry{},
		Pulls:       []forgejo.PR{},
		Pipes:       []woodpecker.Pipeline{},
	}
	protSet := map[string]bool{}
	if prot, err := c.FJ.ListBranchProtections(owner, name); err == nil {
		for _, p := range prot {
			if p.RuleName != "" {
				out.Protections = append(out.Protections, p.RuleName)
				protSet[p.RuleName] = true
			}
		}
	}
	if br, err := c.FJ.ListBranches(owner, name); err == nil {
		for _, b := range br {
			out.Branches = append(out.Branches, BranchInfo{
				Name:      b.Name,
				SHA:       b.Commit.ID,
				Default:   b.Name == repo.DefaultBranch,
				Protected: protSet[b.Name],
			})
		}
	}
	if commits, err := c.FJ.ListCommits(owner, name, ref); err == nil && commits != nil {
		out.Commits = commits
	}
	if ents, err := c.FJ.ListContents(owner, name, ref, path); err == nil && ents != nil {
		if len(ents) == 1 && ents[0].Type == "file" && path != "" {
			e := ents[0]
			out.File = &FileBlob{Name: e.Name, Path: e.Path, Content: decodeContent(e)}
		} else {
			for i := range ents {
				ents[i].Content = ""
			}
			out.Entries = ents
			readmePath := "README.md"
			if path != "" {
				readmePath = strings.TrimSuffix(path, "/") + "/README.md"
			}
			if file, err := c.FJ.GetFile(owner, name, ref, readmePath); err == nil && file.Type == "file" {
				out.Readme = decodeContent(file)
			}
		}
	}
	if pulls, err := c.FJ.ListPRs(owner, name, "open"); err == nil {
		for i := range pulls {
			pulls[i].Repo = repo.FullName
		}
		out.Pulls = pulls
	}
	if c.WP.Ready() {
		if pipes, err := c.WP.ListPipelines(repo.FullName, 1); err == nil {
			if len(pipes) > 8 {
				pipes = pipes[:8]
			}
			out.Pipes = pipes
		}
	}
	return out, nil
}

func (c *Catalog) PackageGroup(kind, name string) (PackageGroup, error) {
	pkgs, err := c.ListPackages(kind)
	if err != nil {
		return PackageGroup{}, err
	}
	out := PackageGroup{Type: kind, Name: name, Versions: []forgejo.Package{}}
	for _, p := range pkgs {
		if p.Type == kind && p.Name == name {
			out.Versions = append(out.Versions, p)
			if out.Latest == "" {
				out.Latest = p.Version
			}
		}
	}
	return out, nil
}

func (c *Catalog) PRDetail(owner, name string, number int) (PRDetail, error) {
	pr, err := c.FJ.GetPR(owner, name, number)
	if err != nil {
		return PRDetail{}, err
	}
	pr.Repo = owner + "/" + name
	out := PRDetail{PR: pr, Checks: []forgejo.Status{}, Comments: []forgejo.Comment{}}
	if pr.Head.SHA != "" {
		ok, st, err := c.FJ.ChecksGreen(owner, name, pr.Head.SHA)
		if err == nil {
			out.Green = ok
			if st != nil {
				out.Checks = st
			}
		}
	}
	if comments, err := c.FJ.ListComments(owner, name, number); err == nil && comments != nil {
		out.Comments = comments
	}
	return out, nil
}

func (c *Catalog) MergePR(owner, name string, number int) (map[string]any, error) {
	pr, err := c.FJ.GetPR(owner, name, number)
	if err != nil {
		return nil, err
	}
	ok, st, err := c.FJ.ChecksGreen(owner, name, pr.Head.SHA)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("checks not green")
	}
	if err := c.FJ.MergePR(owner, name, number); err != nil {
		return nil, err
	}
	return map[string]any{"merged": true, "statuses": st}, nil
}

func (c *Catalog) ListPipelines(repo string) ([]woodpecker.Pipeline, error) {
	if !c.WP.Ready() {
		return []woodpecker.Pipeline{}, nil
	}
	if repo != "" {
		ps, err := c.WP.ListPipelines(repo, 1)
		if err != nil {
			return nil, err
		}
		if ps == nil {
			return []woodpecker.Pipeline{}, nil
		}
		return ps, nil
	}
	ps, err := c.WP.ListRecentPipelines(20)
	if err != nil {
		return nil, err
	}
	if ps == nil {
		return []woodpecker.Pipeline{}, nil
	}
	return ps, nil
}

func (c *Catalog) PipelineDetail(repo string, number int64) (PipelineDetail, error) {
	p, err := c.WP.GetPipeline(repo, number)
	if err != nil {
		return PipelineDetail{}, err
	}
	p.Repo = repo
	return PipelineDetail{Pipeline: p, Steps: p.Steps()}, nil
}

func (c *Catalog) StepLog(repo string, number, step int64) (string, error) {
	if step <= 0 {
		step = 1
	}
	raw, err := c.WP.PipelineLog(repo, number, step)
	if err != nil {
		return "", err
	}
	return woodpecker.FormatLog(raw), nil
}

func (c *Catalog) ListPackages(kind string) ([]forgejo.Package, error) {
	if !c.FJ.Ready() {
		return []forgejo.Package{}, nil
	}
	pkgs, err := c.FJ.ListPackages(c.Cfg.Org, kind)
	if err != nil {
		return nil, err
	}
	if pkgs == nil {
		return []forgejo.Package{}, nil
	}
	return pkgs, nil
}

func (c *Catalog) Inbox() Inbox {
	out := Inbox{
		PRs:     []forgejo.PR{},
		Blocked: []woodpecker.Pipeline{},
		Failed:  []woodpecker.Pipeline{},
		Agents:  []woodpecker.Agent{},
	}
	if c.FJ.Ready() {
		if prs, err := c.FJ.SearchPRs(c.Cfg.Org, "open"); err == nil && prs != nil {
			for i := range prs {
				if prs[i].Repo == "" && prs[i].HTMLURL != "" {
					parts := strings.Split(strings.TrimPrefix(prs[i].HTMLURL, "https://"), "/")
					if len(parts) >= 3 {
						prs[i].Repo = parts[1] + "/" + parts[2]
					}
				}
			}
			out.PRs = prs
		}
	}
	if c.WP.Ready() {
		if pipes, err := c.WP.ListRecentPipelines(8); err == nil {
			for _, p := range pipes {
				switch strings.ToLower(p.Status) {
				case "blocked":
					out.Blocked = append(out.Blocked, p)
				case "failure", "error", "killed", "declined":
					out.Failed = append(out.Failed, p)
				}
			}
		}
	}
	return out
}

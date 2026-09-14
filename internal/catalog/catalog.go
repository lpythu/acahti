package catalog

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"

	"acahti/internal/config"
	"acahti/internal/forgejo"
	"acahti/internal/page"
	"acahti/internal/woodpecker"
)

type Catalog struct {
	Cfg config.Config
	FJ  *forgejo.Client
	WP  *woodpecker.Client
	mem *memo
}

func New(cfg config.Config, fj *forgejo.Client, wp *woodpecker.Client) *Catalog {
	return &Catalog{Cfg: cfg, FJ: fj, WP: wp, mem: newMemo()}
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

type RepoHeader struct {
	Repo       forgejo.Repo `json:"repo"`
	CloneHTTPS string       `json:"clone_https"`
	CloneSSH   string       `json:"clone_ssh"`
	Ref        string       `json:"ref"`
}

type RepoContents struct {
	page.Result[forgejo.ContentEntry]
	Ref    string    `json:"ref"`
	Path   string    `json:"path"`
	File   *FileBlob `json:"file,omitempty"`
	Readme string    `json:"readme"`
}

type RepoGroup struct {
	Group string `json:"group"`
	Count int    `json:"count"`
}

type PackageRow struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Latest    string `json:"latest"`
	UpdatedAt string `json:"updated_at"`
}

type PRDetail struct {
	PR     forgejo.PR       `json:"pr"`
	Checks []forgejo.Status `json:"checks"`
	Green  bool             `json:"green"`
}

type PipelineDetail struct {
	Pipeline woodpecker.Pipeline `json:"pipeline"`
	Steps    []woodpecker.Step   `json:"steps"`
}

type CommitDetail struct {
	page.Result[forgejo.CommitFile]
	Commit forgejo.Commit      `json:"commit"`
	Stats  forgejo.CommitStats `json:"stats"`
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

func (c *Catalog) ListRepoGroups(user string, q page.Query) (page.Result[RepoGroup], error) {
	clusters, err := c.clusterGroups()
	if err != nil {
		return page.Result[RepoGroup]{}, err
	}
	admin := c.IsOrgAdmin(user)
	type item struct {
		name string
		g    groupTeams
	}
	items := make([]item, 0, len(clusters))
	for name, g := range clusters {
		items = append(items, item{name: name, g: g})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].name < items[j].name })
	type row struct {
		g    RepoGroup
		skip bool
	}
	rows := make([]row, len(items))
	var wg sync.WaitGroup
	for i, it := range items {
		wg.Add(1)
		go func(i int, it item) {
			defer wg.Done()
			if !admin && !c.inGroup(user, it.g) {
				rows[i].skip = true
				return
			}
			n := 0
			if id := it.g.writeID(); id > 0 {
				n, _ = c.FJ.TeamRepoCount(id)
			}
			rows[i].g = RepoGroup{Group: it.name, Count: n}
		}(i, it)
	}
	wg.Wait()
	out := make([]RepoGroup, 0, len(rows))
	for _, r := range rows {
		if r.skip {
			continue
		}
		out = append(out, r.g)
	}
	return page.Take(out, q), nil
}

func (c *Catalog) ListRepos(user, group string, q page.Query) (page.Result[forgejo.Repo], error) {
	if !c.FJ.Ready() {
		return page.Of([]forgejo.Repo{}, q, false), nil
	}
	if group == "" {
		if c.IsOrgAdmin(user) {
			return c.FJ.ListOrgRepos(c.Cfg.Org, q)
		}
		return c.FJ.ListRepos(user, q)
	}
	g, err := c.findGroup(group)
	if err != nil {
		return page.Result[forgejo.Repo]{}, err
	}
	if !c.IsOrgAdmin(user) && !c.inGroup(user, g) {
		return page.Of([]forgejo.Repo{}, q, false), nil
	}
	id := g.writeID()
	if id == 0 {
		return page.Of([]forgejo.Repo{}, q, false), nil
	}
	res, err := c.FJ.ListTeamRepos(id, q)
	if err != nil {
		return page.Result[forgejo.Repo]{}, err
	}
	for i := range res.Items {
		res.Items[i].Group = group
	}
	return res, nil
}

func (c *Catalog) RepoHeader(user, owner, name, ref string) (RepoHeader, error) {
	repo, err := c.seeRepo(user, owner, name)
	if err != nil {
		return RepoHeader{}, err
	}
	if ref == "" {
		ref = repo.DefaultBranch
		if ref == "" {
			ref = "dev"
		}
	}
	return RepoHeader{
		Repo:       repo,
		CloneHTTPS: c.CloneHTTPS(repo.FullName),
		CloneSSH:   c.CloneSSH(repo.FullName),
		Ref:        ref,
	}, nil
}

func (c *Catalog) RepoContents(user, owner, name, ref, path string, q page.Query) (RepoContents, error) {
	head, err := c.RepoHeader(user, owner, name, ref)
	if err != nil {
		return RepoContents{}, err
	}
	ref = head.Ref
	if path == "." {
		path = ""
	}
	out := RepoContents{Ref: ref, Path: path, Result: page.Of([]forgejo.ContentEntry{}, q, false)}
	readmePath := "README.md"
	if path != "" {
		readmePath = strings.TrimSuffix(path, "/") + "/README.md"
	}
	var (
		ents   []forgejo.ContentEntry
		readme string
		wg     sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		ents, _ = c.FJ.ListContents(owner, name, ref, path)
	}()
	go func() {
		defer wg.Done()
		if file, err := c.FJ.GetFile(owner, name, ref, readmePath); err == nil && file.Type == "file" {
			readme = decodeContent(file)
		}
	}()
	wg.Wait()
	if ents == nil {
		return out, nil
	}
	if len(ents) == 1 && ents[0].Type == "file" && path != "" {
		e := ents[0]
		out.File = &FileBlob{Name: e.Name, Path: e.Path, Content: decodeContent(e)}
		return out, nil
	}
	for i := range ents {
		ents[i].Content = ""
	}
	out.Result = page.Take(ents, q)
	out.Readme = readme
	return out, nil
}

func (c *Catalog) ListBranches(user, owner, name string, q page.Query) (page.Result[BranchInfo], error) {
	repo, err := c.FJ.GetRepo(owner, name, user)
	if err != nil {
		return page.Result[BranchInfo]{}, fmt.Errorf("%w: %s", ErrNotFound, err)
	}
	var (
		prot []forgejo.BranchProtection
		res  page.Result[forgejo.Branch]
		wg   sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		prot, _ = c.FJ.ListBranchProtections(owner, name)
	}()
	go func() {
		defer wg.Done()
		res, err = c.FJ.ListBranches(owner, name, q)
	}()
	wg.Wait()
	if err != nil {
		return page.Result[BranchInfo]{}, err
	}
	protSet := map[string]bool{}
	for _, p := range prot {
		if p.RuleName != "" {
			protSet[p.RuleName] = true
		}
	}
	out := make([]BranchInfo, 0, len(res.Items))
	for _, b := range res.Items {
		out = append(out, BranchInfo{
			Name:      b.Name,
			SHA:       b.Commit.ID,
			Default:   b.Name == repo.DefaultBranch,
			Protected: protSet[b.Name],
		})
	}
	return page.Of(out, q, res.HasMore), nil
}

func (c *Catalog) ListCommits(user, owner, name, ref string, q page.Query) (page.Result[forgejo.Commit], error) {
	if ref == "" {
		repo, err := c.FJ.GetRepo(owner, name, user)
		if err != nil {
			return page.Result[forgejo.Commit]{}, fmt.Errorf("%w: %s", ErrNotFound, err)
		}
		ref = repo.DefaultBranch
		if ref == "" {
			ref = "dev"
		}
	} else if err := c.seeOK(user, owner, name); err != nil {
		return page.Result[forgejo.Commit]{}, err
	}
	return c.FJ.ListCommits(owner, name, ref, q)
}

func (c *Catalog) ListPulls(user, owner, name, state string, q page.Query) (page.Result[forgejo.PR], error) {
	if err := c.seeOK(user, owner, name); err != nil {
		return page.Result[forgejo.PR]{}, err
	}
	res, err := c.FJ.ListPRs(owner, name, state, q)
	if err != nil {
		return page.Result[forgejo.PR]{}, err
	}
	full := owner + "/" + name
	for i := range res.Items {
		res.Items[i].Repo = full
	}
	return res, nil
}

func (c *Catalog) ListComments(user, owner, name string, number int, q page.Query) (page.Result[forgejo.Comment], error) {
	if err := c.seeOK(user, owner, name); err != nil {
		return page.Result[forgejo.Comment]{}, err
	}
	return c.FJ.ListComments(owner, name, number, q)
}

func (c *Catalog) ListPackageVersions(kind, name string, q page.Query) (page.Result[forgejo.Package], error) {
	if !c.FJ.Ready() {
		return page.Of([]forgejo.Package{}, q, false), nil
	}
	res, err := c.FJ.ListPackages(c.Cfg.Org, kind, name, q)
	if err != nil {
		return page.Result[forgejo.Package]{}, err
	}
	items := res.Items[:0]
	for _, p := range res.Items {
		if p.Name == name {
			items = append(items, p)
		}
	}
	return page.Of(items, q, res.HasMore), nil
}

func (c *Catalog) ListPackageRows(kind string, q page.Query) (page.Result[PackageRow], error) {
	if !c.FJ.Ready() {
		return page.Of([]PackageRow{}, q, false), nil
	}
	res, err := c.FJ.ListPackages(c.Cfg.Org, kind, "", q)
	if err != nil {
		return page.Result[PackageRow]{}, err
	}
	type key struct{ typ, name string }
	seen := map[key]PackageRow{}
	order := []key{}
	for _, p := range res.Items {
		k := key{p.Type, p.Name}
		cur, ok := seen[k]
		if !ok {
			seen[k] = PackageRow{Type: p.Type, Name: p.Name, Latest: p.Version, UpdatedAt: p.CreatedAt}
			order = append(order, k)
			continue
		}
		if p.CreatedAt > cur.UpdatedAt {
			cur.UpdatedAt = p.CreatedAt
			cur.Latest = p.Version
			seen[k] = cur
		}
	}
	out := make([]PackageRow, 0, len(order))
	for _, k := range order {
		out = append(out, seen[k])
	}
	return page.Of(out, q, res.HasMore), nil
}

func (c *Catalog) PRDetail(user, owner, name string, number int) (PRDetail, error) {
	if _, err := c.seeRepo(user, owner, name); err != nil {
		return PRDetail{}, err
	}
	pr, err := c.FJ.GetPR(owner, name, number)
	if err != nil {
		return PRDetail{}, err
	}
	pr.Repo = owner + "/" + name
	out := PRDetail{PR: pr, Checks: []forgejo.Status{}}
	if pr.Head.SHA != "" {
		ok, st, err := c.FJ.ChecksGreen(owner, name, pr.Head.SHA)
		if err == nil {
			out.Green = ok
			if st != nil {
				c.rewriteChecks(st)
				out.Checks = st
			}
		}
	}
	return out, nil
}

func (c *Catalog) MergePR(user, owner, name string, number int) (map[string]any, error) {
	if _, err := c.seeRepo(user, owner, name); err != nil {
		return nil, err
	}
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
	c.rewriteChecks(st)
	return map[string]any{"merged": true, "statuses": st}, nil
}

func (c *Catalog) rewriteChecks(st []forgejo.Status) {
	for i := range st {
		st[i].TargetURL = c.acahtiCheckURL(st[i].TargetURL)
	}
}

func (c *Catalog) acahtiCheckURL(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	p := strings.TrimRight(u.Path, "/")
	root := strings.TrimRight(c.Cfg.RootURL, "/")
	if p == "/ci" {
		return root + "/pipelines"
	}
	rest, ok := strings.CutPrefix(p, "/ci/")
	if !ok {
		return raw
	}
	rest = strings.TrimPrefix(rest, "repos/")
	rest = strings.ReplaceAll(rest, "/pipeline/", "/")
	parts := strings.Split(rest, "/")
	if len(parts) >= 3 && parts[0] != "" && parts[1] != "" && parts[2] != "" {
		return root + "/pipelines/" + parts[0] + "/" + parts[1] + "/" + parts[2]
	}
	if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
		return root + "/pipelines?repo=" + parts[0] + "/" + parts[1]
	}
	return root + "/pipelines"
}

func (c *Catalog) pipelineNames(user, group string) ([]string, error) {
	repos, err := c.WP.CachedRepos()
	if err != nil {
		return nil, err
	}
	var want map[string]bool
	if group != "" {
		g, err := c.findGroup(group)
		if err != nil {
			return nil, err
		}
		id := g.writeID()
		if id == 0 {
			return []string{}, nil
		}
		team, err := page.Walk(func(pq page.Query) (page.Result[forgejo.Repo], error) {
			return c.FJ.ListTeamRepos(id, pq)
		})
		if err != nil {
			return nil, err
		}
		want = visibleSet(team)
	} else if !c.IsOrgAdmin(user) {
		visible, err := c.userRepos(user)
		if err != nil {
			return nil, err
		}
		want = visibleSet(visible)
	}
	var names []string
	for _, r := range repos {
		if !r.IsActive {
			continue
		}
		if want != nil && !want[r.FullName] {
			continue
		}
		names = append(names, r.FullName)
	}
	sort.Strings(names)
	return names, nil
}

func (c *Catalog) ListPipelines(user, repo, group string, q page.Query) (page.Result[woodpecker.Pipeline], error) {
	if !c.WP.Ready() {
		return page.Of([]woodpecker.Pipeline{}, q, false), nil
	}
	if repo != "" {
		owner, name, _ := strings.Cut(repo, "/")
		if err := c.seeOK(user, owner, name); err != nil {
			return page.Result[woodpecker.Pipeline]{}, err
		}
		res, err := c.WP.ListPipelines(repo, q)
		if err != nil {
			return page.Result[woodpecker.Pipeline]{}, err
		}
		res.Items = c.WP.EnrichJobs(repo, res.Items)
		return res, nil
	}
	if group != "" {
		g, err := c.findGroup(group)
		if err != nil {
			return page.Result[woodpecker.Pipeline]{}, err
		}
		id := g.writeID()
		if id == 0 {
			return page.Of([]woodpecker.Pipeline{}, q, false), nil
		}
		res, err := c.FJ.ListTeamRepos(id, q)
		if err != nil {
			return page.Result[woodpecker.Pipeline]{}, err
		}
		names := make([]string, len(res.Items))
		for i, r := range res.Items {
			names[i] = r.FullName
		}
		return page.Of(c.WP.LatestPipelines(names, true), q, res.HasMore), nil
	}
	names, err := c.pipelineNames(user, "")
	if err != nil {
		return page.Result[woodpecker.Pipeline]{}, err
	}
	slice := page.Take(names, q)
	return page.Of(c.WP.LatestPipelines(slice.Items, true), q, slice.HasMore), nil
}

func (c *Catalog) CommitDetail(user, owner, name, sha string, q page.Query) (CommitDetail, error) {
	if _, err := c.seeRepo(user, owner, name); err != nil {
		return CommitDetail{}, err
	}
	cm, err := c.FJ.GetCommit(owner, name, sha)
	if err != nil {
		return CommitDetail{}, err
	}
	files := cm.Files
	needPatch := len(files) == 0
	for _, f := range files {
		if f.Patch == "" && f.Filename != "" {
			needPatch = true
			break
		}
	}
	if needPatch {
		if raw, err := c.FJ.GetCommitDiff(owner, name, sha); err == nil {
			parsed := forgejo.ParseUnifiedDiff(raw)
			if len(files) == 0 {
				files = parsed
			} else {
				byName := map[string]forgejo.CommitFile{}
				for _, f := range parsed {
					byName[f.Filename] = f
				}
				for i, f := range files {
					if f.Patch != "" {
						continue
					}
					if p, ok := byName[f.Filename]; ok {
						files[i].Patch = p.Patch
						if files[i].Additions == 0 && files[i].Deletions == 0 {
							files[i].Additions = p.Additions
							files[i].Deletions = p.Deletions
							files[i].Changes = p.Changes
						}
						if files[i].Status == "" {
							files[i].Status = p.Status
						}
					}
				}
			}
		}
	}
	stats := forgejo.CommitStats{}
	if cm.Stats != nil {
		stats = *cm.Stats
	} else {
		for _, f := range files {
			stats.Additions += f.Additions
			stats.Deletions += f.Deletions
		}
		stats.Total = stats.Additions + stats.Deletions
	}
	cm.Files = nil
	cm.Stats = &stats
	return CommitDetail{Commit: cm, Stats: stats, Result: page.Take(files, q)}, nil
}

func (c *Catalog) PipelineDetail(user, repo string, number int64) (PipelineDetail, error) {
	owner, name, _ := strings.Cut(repo, "/")
	if _, err := c.seeRepo(user, owner, name); err != nil {
		return PipelineDetail{}, err
	}
	p, err := c.WP.GetPipeline(repo, number)
	if err != nil {
		return PipelineDetail{}, err
	}
	p.Repo = repo
	return PipelineDetail{Pipeline: p, Steps: p.Steps()}, nil
}

func (c *Catalog) StepLog(user, repo string, number, step int64) (string, error) {
	owner, name, _ := strings.Cut(repo, "/")
	if _, err := c.seeRepo(user, owner, name); err != nil {
		return "", err
	}
	if step <= 0 {
		step = 1
	}
	raw, err := c.WP.PipelineLog(repo, number, step)
	if err != nil {
		return "", err
	}
	return woodpecker.FormatLog(raw), nil
}

func stepFailed(state string) bool {
	switch strings.ToLower(state) {
	case "failure", "error", "failed", "killed", "declined":
		return true
	}
	return false
}

func tailLog(text string, n int64) string {
	if n <= 0 {
		return text
	}
	lines := strings.Split(text, "\n")
	if int64(len(lines)) <= n {
		return text
	}
	return strings.Join(lines[len(lines)-int(n):], "\n")
}

func (c *Catalog) PipelineLogs(user, repo string, number, step, tail int64) (map[string]any, error) {
	detail, err := c.PipelineDetail(user, repo, number)
	if err != nil {
		return nil, err
	}
	if step > 0 {
		text, err := c.StepLog(user, repo, number, step)
		if err != nil {
			return nil, err
		}
		return map[string]any{"log": tailLog(text, tail), "steps": detail.Steps}, nil
	}
	var failed []woodpecker.Step
	for _, s := range detail.Steps {
		if stepFailed(s.State) {
			failed = append(failed, s)
		}
	}
	if len(failed) == 0 {
		return map[string]any{"log": "", "steps": detail.Steps}, nil
	}
	var b strings.Builder
	for _, s := range failed {
		id := s.ID
		if id == 0 {
			id = s.PID
		}
		text, err := c.StepLog(user, repo, number, id)
		if err != nil {
			text = err.Error()
		}
		fmt.Fprintf(&b, "=== %s (step %d) %s ===\n%s\n", s.Name, id, s.State, tailLog(text, tail))
	}
	return map[string]any{"log": b.String(), "steps": failed}, nil
}

func (c *Catalog) BoardPRs(user string, q page.Query) (page.Result[forgejo.PR], error) {
	if !c.FJ.Ready() {
		return page.Of([]forgejo.PR{}, q, false), nil
	}
	if c.IsOrgAdmin(user) {
		return c.FJ.SearchPRs(c.Cfg.Org, "open", q)
	}
	visible, err := c.userRepos(user)
	if err != nil {
		return page.Result[forgejo.PR]{}, err
	}
	allow := visibleSet(visible)
	var matched []forgejo.PR
	need := q.Norm().Page*q.Norm().Size + 1
	pq := page.Query{Page: 1, Size: page.MaxSize}
	for len(matched) < need && pq.Page <= page.MaxWalk {
		res, err := c.FJ.SearchPRs(c.Cfg.Org, "open", pq)
		if err != nil {
			return page.Result[forgejo.PR]{}, err
		}
		for _, p := range res.Items {
			if allow[p.Repo] {
				matched = append(matched, p)
			}
		}
		if !res.HasMore {
			break
		}
		pq.Page++
	}
	return page.Take(matched, q), nil
}

func (c *Catalog) BoardPipes(user, kind string, q page.Query) (page.Result[woodpecker.Pipeline], error) {
	if !c.WP.Ready() {
		return page.Of([]woodpecker.Pipeline{}, q, false), nil
	}
	want := map[string]bool{}
	switch kind {
	case "blocked":
		want["blocked"] = true
	default:
		want["failure"] = true
		want["error"] = true
		want["killed"] = true
		want["declined"] = true
	}
	repos, err := c.WP.CachedRepos()
	if err != nil {
		return page.Result[woodpecker.Pipeline]{}, err
	}
	var allow map[string]bool
	if !c.IsOrgAdmin(user) {
		visible, err := c.userRepos(user)
		if err != nil {
			return page.Result[woodpecker.Pipeline]{}, err
		}
		allow = visibleSet(visible)
	}
	var names []string
	for _, r := range repos {
		if !r.IsActive {
			continue
		}
		if allow != nil && !allow[r.FullName] {
			continue
		}
		names = append(names, r.FullName)
	}
	need := q.Norm().Page*q.Norm().Size + 1
	var matched []woodpecker.Pipeline
	for i := 0; i < len(names) && len(matched) < need; i += 16 {
		end := i + 16
		if end > len(names) {
			end = len(names)
		}
		for _, p := range c.WP.LatestPipelines(names[i:end], false) {
			if want[strings.ToLower(p.Status)] {
				matched = append(matched, p)
			}
		}
	}
	return page.Take(matched, q), nil
}

func (c *Catalog) ListAgents(q page.Query) (page.Result[woodpecker.Agent], error) {
	if !c.WP.Ready() {
		return page.Of([]woodpecker.Agent{}, q, false), nil
	}
	agents, err := c.WP.Agents()
	if err != nil {
		return page.Result[woodpecker.Agent]{}, err
	}
	return page.Take(agents, q), nil
}

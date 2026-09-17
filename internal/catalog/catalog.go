package catalog

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/url"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"acahti/internal/config"
	"acahti/internal/forgejo"
	"acahti/internal/page"
	"acahti/internal/pipeline"
	"acahti/internal/store"
	"acahti/internal/woodpecker"
)

type Catalog struct {
	Cfg    config.Config
	FJ     *forgejo.Client
	WP     *woodpecker.Client
	Idx    *store.Store
	Notify func(kind string, data any)
	mem    *memo
}

func New(cfg config.Config, fj *forgejo.Client, wp *woodpecker.Client, idx *store.Store) *Catalog {
	return &Catalog{Cfg: cfg, FJ: fj, WP: wp, Idx: idx, mem: newMemo()}
}

type BranchInfo struct {
	Name      string `json:"name"`
	SHA       string `json:"sha"`
	Default   bool   `json:"default"`
	Protected bool   `json:"protected"`
}

type TagInfo struct {
	Name string `json:"name"`
	SHA  string `json:"sha"`
}

type FileBlob struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

type RepoHeader struct {
	Repo       forgejo.Repo `json:"repo"`
	CloneHTTPS string       `json:"clone_https"`
	Ref        string       `json:"ref"`
}

type RepoContents struct {
	page.Result[forgejo.ContentEntry]
	Ref    string    `json:"ref"`
	Path   string    `json:"path"`
	File   *FileBlob `json:"file,omitempty"`
	Readme string    `json:"readme"`
}

type RepoTeam struct {
	Team  string `json:"team"`
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
	Team     string              `json:"team"`
	Files    []FileBlob          `json:"files"`
}

type NavTeam struct {
	Team  string         `json:"team"`
	Repos []forgejo.Repo `json:"repos"`
}

type CommitDetail struct {
	page.Result[forgejo.CommitFile]
	Commit forgejo.Commit      `json:"commit"`
	Stats  forgejo.CommitStats `json:"stats"`
}

func (c *Catalog) CloneHTTPS(fullName string) string {
	return strings.TrimRight(c.Cfg.RootURL, "/") + "/" + fullName + ".git"
}

func (c *Catalog) PublicRepo(r forgejo.Repo) forgejo.Repo {
	name := r.FullName
	if name == "" && r.Name != "" {
		name = strings.TrimRight(c.Cfg.Org, "/") + "/" + r.Name
	}
	if name != "" {
		r.CloneURL = c.CloneHTTPS(name)
	}
	return r
}

func (c *Catalog) PublicRepos(items []forgejo.Repo) []forgejo.Repo {
	for i := range items {
		items[i] = c.PublicRepo(items[i])
	}
	return items
}

func (c *Catalog) exposeRepos(res page.Result[forgejo.Repo], err error) (page.Result[forgejo.Repo], error) {
	if err != nil {
		return res, err
	}
	res.Items = c.PublicRepos(res.Items)
	return res, nil
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

func (c *Catalog) ListRepoTeams(user string, q page.Query) (page.Result[RepoTeam], error) {
	if !c.indexed() {
		return page.Of([]RepoTeam{}, q, false), nil
	}
	rows, err := c.Idx.TeamCounts(user, c.IsOrgAdmin(user))
	if err != nil {
		return page.Result[RepoTeam]{}, err
	}
	out := make([]RepoTeam, 0, len(rows))
	for _, r := range rows {
		out = append(out, RepoTeam{Team: r.Team, Count: r.Count})
	}
	return page.Take(out, q), nil
}

func (c *Catalog) ListRepos(user, team string, q page.Query) (page.Result[forgejo.Repo], error) {
	if !c.indexed() {
		return page.Of([]forgejo.Repo{}, q, false), nil
	}
	admin := c.IsOrgAdmin(user)
	res, err := c.Idx.ListReposPage(user, team, admin, q)
	if err != nil {
		return page.Result[forgejo.Repo]{}, err
	}
	items := c.asRepos(res.Items, team)
	if team == "" {
		for i := range items {
			if items[i].Team == "" {
				items[i].Team, _, _ = c.Idx.RepoTeam(items[i].FullName)
			}
		}
	}
	c.paintPerms(user, items)
	return page.Result[forgejo.Repo]{Items: items, Page: res.Page, Size: res.Size, HasMore: res.HasMore}, nil
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
	repo = c.PublicRepo(repo)
	repo.CanManageSecrets = c.IsOrgAdmin(user) || repo.Permissions.Admin
	return RepoHeader{
		Repo:       repo,
		CloneHTTPS: c.CloneHTTPS(repo.FullName),
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

func (c *Catalog) SetDefaultBranch(user, owner, name, branch string) error {
	if _, err := c.requireRepoAdmin(user, owner, name); err != nil {
		return err
	}
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return fmt.Errorf("%w: branch", ErrInvalid)
	}
	repo, err := c.FJ.SetDefaultBranch(owner, name, branch)
	if err != nil {
		return err
	}
	c.RememberRepo(repo)
	return nil
}

func (c *Catalog) SetBranchProtected(user, owner, name, branch string, protected bool) error {
	if _, err := c.requireRepoAdmin(user, owner, name); err != nil {
		return err
	}
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return fmt.Errorf("%w: branch", ErrInvalid)
	}
	if protected {
		return c.FJ.ProtectBranch(owner, name, branch)
	}
	return c.FJ.UnprotectBranch(owner, name, branch)
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

func (c *Catalog) ListTags(user, owner, name string, q page.Query) (page.Result[TagInfo], error) {
	if err := c.seeOK(user, owner, name); err != nil {
		return page.Result[TagInfo]{}, err
	}
	res, err := c.FJ.ListTags(owner, name, q)
	if err != nil {
		return page.Result[TagInfo]{}, err
	}
	out := make([]TagInfo, 0, len(res.Items))
	for _, t := range res.Items {
		out = append(out, TagInfo{Name: t.Name, SHA: t.Commit.SHA})
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
	res, err := c.FJ.ListCommits(owner, name, ref, q)
	if err != nil {
		return page.Result[forgejo.Commit]{}, err
	}
	c.paintCommitChecks(owner, name, res.Items)
	return res, nil
}

func (c *Catalog) paintCommitChecks(owner, name string, items []forgejo.Commit) {
	if c.FJ == nil || !c.FJ.Ready() || len(items) == 0 {
		return
	}
	var wg sync.WaitGroup
	for i := range items {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			st, err := c.FJ.CommitStatuses(owner, name, items[i].SHA)
			if err != nil || len(st) == 0 {
				return
			}
			st = forgejo.LatestStatuses(forgejo.LatestRound(st))
			c.rewriteChecks(st)
			items[i].CheckStatus = forgejo.RollupStatus(st)
			for _, s := range st {
				if s.TargetURL == "" {
					continue
				}
				if u, err := url.Parse(s.TargetURL); err == nil && u.Path != "" {
					items[i].CheckURL = u.Path
					if u.RawQuery != "" {
						items[i].CheckURL += "?" + u.RawQuery
					}
				} else {
					items[i].CheckURL = s.TargetURL
				}
				break
			}
		}(i)
	}
	wg.Wait()
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

func (c *Catalog) ListPackageVersions(user, kind, name string, q page.Query) (page.Result[forgejo.Package], error) {
	if !c.FJ.Ready() {
		return page.Of([]forgejo.Package{}, q, false), nil
	}
	res, err := c.FJ.ListPackages(c.Cfg.Org, kind, name, q, user)
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

func (c *Catalog) ListPackageRows(user, kind string, q page.Query) (page.Result[PackageRow], error) {
	if !c.FJ.Ready() {
		return page.Of([]PackageRow{}, q, false), nil
	}
	res, err := c.FJ.ListPackages(c.Cfg.Org, kind, "", q, user)
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
		ok, st, err := c.CommitChecks(owner, name, pr.Head.SHA)
		if err == nil {
			out.Green = ok
			out.Checks = st
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
	ok, st, err := c.CommitChecks(owner, name, pr.Head.SHA)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("checks not green")
	}
	if err := c.FJ.MergePR(owner, name, number, user); err != nil {
		return nil, err
	}
	return map[string]any{"merged": true, "statuses": st}, nil
}

func (c *Catalog) ClosePR(user, owner, name string, number int) (map[string]any, error) {
	if _, err := c.seeRepo(user, owner, name); err != nil {
		return nil, err
	}
	if err := c.FJ.ClosePR(owner, name, number, user); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "state": "closed"}, nil
}

func (c *Catalog) CommitChecks(owner, name, sha string) (bool, []forgejo.Status, error) {
	ok, st, err := c.FJ.ChecksGreen(owner, name, sha)
	if err != nil {
		return false, nil, err
	}
	c.rewriteChecks(st)
	return ok, st, nil
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
		return root + "/repos/" + parts[0] + "/" + parts[1] + "/pipelines/" + parts[2]
	}
	if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
		return root + "/pipelines?repo=" + parts[0] + "/" + parts[1]
	}
	return root + "/pipelines"
}

func (c *Catalog) ListRepoPipelines(user, repo, sha, branch, status string, q page.Query) (page.Result[woodpecker.Pipeline], error) {
	if !c.canSeeIndexedRepo(user, repo) {
		return page.Of([]woodpecker.Pipeline{}, q, false), nil
	}
	f := store.Filter{Repos: []string{repo}, SHA: sha, Branch: branch}
	if status != "" {
		f.Status = []string{status}
	}
	res, err := c.listIndexed(f, q)
	if err != nil {
		return page.Result[woodpecker.Pipeline]{}, err
	}
	res.Items = c.paintPipes(res.Items)
	return res, nil
}

func expandPipeStatus(status string) []string {
	var out []string
	for _, part := range strings.Split(status, ",") {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "", "all":
			continue
		case "failed":
			out = append(out, "failure", "error", "killed", "declined")
		case "blocked":
			out = append(out, "blocked")
		case "running":
			out = append(out, "running", "pending")
		case "success":
			out = append(out, "success")
		default:
			out = append(out, strings.ToLower(strings.TrimSpace(part)))
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (c *Catalog) ListPipelines(user, repo, team, status string, q page.Query) (page.Result[woodpecker.Pipeline], error) {
	var names []string
	if repo != "" {
		if !c.canSeeIndexedRepo(user, repo) {
			return page.Of([]woodpecker.Pipeline{}, q, false), nil
		}
		names = []string{repo}
	} else {
		var err error
		names, err = c.visiblePipeRepos(user, team)
		if err != nil {
			return page.Result[woodpecker.Pipeline]{}, err
		}
	}
	if c.Idx == nil {
		return page.Of([]woodpecker.Pipeline{}, q, false), nil
	}
	want := expandPipeStatus(status)
	res, err := c.Idx.LatestByRepo(names, want, q)
	if err != nil {
		return page.Result[woodpecker.Pipeline]{}, err
	}
	res.Items = c.mergeQueueHeads(res.Items, names, want, q.Norm().Page)
	res.Items = c.paintPipes(res.Items)
	return res, nil
}

func (c *Catalog) visiblePipeRepos(user, team string) ([]string, error) {
	if !c.indexed() {
		return []string{}, nil
	}
	admin := c.IsOrgAdmin(user)
	if team != "" {
		repos, err := c.Idx.VisibleTeamRepos(user, team, admin)
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, len(repos))
		for _, r := range repos {
			out = append(out, r.FullName)
		}
		return out, nil
	}
	return c.Idx.VisibleRepoNames(user, admin)
}

func (c *Catalog) canSeeIndexedRepo(user, repo string) bool {
	if repo == "" {
		return false
	}
	if c.IsOrgAdmin(user) {
		return true
	}
	if !c.indexed() {
		return false
	}
	ok, err := c.Idx.HasVisibleRepo(user, repo)
	return err == nil && ok
}

func (c *Catalog) listIndexed(f store.Filter, q page.Query) (page.Result[woodpecker.Pipeline], error) {
	if c.Idx == nil {
		return page.Of([]woodpecker.Pipeline{}, q, false), nil
	}
	return c.Idx.List(f, q)
}

func permFromRole(role string) forgejo.Perm {
	switch role {
	case "admin":
		return forgejo.Perm{Admin: true, Push: true, Pull: true}
	case "write":
		return forgejo.Perm{Push: true, Pull: true}
	case "read":
		return forgejo.Perm{Pull: true}
	}
	return forgejo.Perm{}
}

func (c *Catalog) paintPerms(user string, items []forgejo.Repo) {
	if c.IsOrgAdmin(user) {
		for i := range items {
			items[i].Permissions = forgejo.Perm{Admin: true, Push: true, Pull: true}
		}
		return
	}
	if c.Idx == nil || user == "" {
		return
	}
	perms, err := c.Idx.UserRepoPerms(user)
	if err != nil {
		return
	}
	for i := range items {
		items[i].Permissions = permFromRole(perms[items[i].FullName])
	}
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

func statusAllowed(status string, want []string) bool {
	if len(want) == 0 {
		return true
	}
	status = strings.ToLower(strings.TrimSpace(status))
	for _, w := range want {
		if status == w {
			return true
		}
	}
	return false
}

func repoAllowed(repo string, repos []string) bool {
	if repos == nil {
		return true
	}
	for _, r := range repos {
		if r == repo {
			return true
		}
	}
	return false
}

func (c *Catalog) queueHeadNumbers(repos []string) map[string]int64 {
	best := map[string]int64{}
	q := c.queueInfo()
	add := func(tasks []woodpecker.QueueTask) {
		for _, t := range tasks {
			if t.Repo == "" || t.Number == 0 || !repoAllowed(t.Repo, repos) {
				continue
			}
			if t.Number > best[t.Repo] {
				best[t.Repo] = t.Number
			}
		}
	}
	add(q.Running)
	add(q.Pending)
	add(q.WaitingOnDeps)
	return best
}

// mergeQueueHeads replaces stale indexed heads with in-flight Woodpecker
// runs. The queue is live; the pipeline index often only sees a hook after
// a run finishes, so LatestByRepo can keep showing yesterday's success.
func (c *Catalog) mergeQueueHeads(items []woodpecker.Pipeline, repos, want []string, pageNo int) []woodpecker.Pipeline {
	heads := c.queueHeadNumbers(repos)
	if len(heads) == 0 {
		return items
	}
	out := append([]woodpecker.Pipeline(nil), items...)
	seen := map[string]int{}
	for i, p := range out {
		seen[p.Repo] = i
	}
	for repo, n := range heads {
		if i, ok := seen[repo]; ok && out[i].Number > n {
			continue
		}
		p, err := c.Refresh(repo, n)
		if err != nil || !statusAllowed(p.Status, want) {
			continue
		}
		if i, ok := seen[repo]; ok {
			if out[i].Number == p.Number && out[i].Status == p.Status {
				continue
			}
			out[i] = p
			continue
		}
		if pageNo <= 1 {
			seen[repo] = len(out)
			out = append(out, p)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := woodpecker.InFlight(out[i].Status), woodpecker.InFlight(out[j].Status)
		if a != b {
			return a
		}
		if out[i].Created != out[j].Created {
			return out[i].Created > out[j].Created
		}
		return out[i].Number > out[j].Number
	})
	return out
}

func (c *Catalog) paintPipes(pipes []woodpecker.Pipeline) []woodpecker.Pipeline {
	out := append([]woodpecker.Pipeline(nil), pipes...)
	q := c.queueInfo()
	var wg sync.WaitGroup
	for i := range out {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			out[i] = c.decoratePipe(out[i])
			out[i] = woodpecker.Annotate(out[i], q)
		}(i)
	}
	wg.Wait()
	return out
}

func (c *Catalog) Present(p woodpecker.Pipeline) woodpecker.Pipeline {
	p = c.decoratePipe(p)
	return woodpecker.Annotate(p, c.queueInfo())
}

func (c *Catalog) queueInfo() woodpecker.QueueInfo {
	if c.mem != nil {
		if q, ok := c.mem.queueOf(); ok {
			return q
		}
	}
	if c.WP == nil || !c.WP.Ready() {
		return woodpecker.QueueInfo{}
	}
	q, err := c.WP.QueueInfo()
	if err != nil {
		return woodpecker.QueueInfo{}
	}
	workers := q.Stats.WorkerCount
	q.Stats = woodpecker.PipelineStripStats(q)
	q.Stats.WorkerCount = workers
	if c.mem != nil {
		c.mem.setQueue(q)
	}
	return q
}

func (c *Catalog) jobDepends(p woodpecker.Pipeline) map[string][]string {
	out := map[string][]string{}
	for _, f := range c.pipelineFiles(p) {
		n := pipeline.JobName(f.Name)
		if n == "" {
			continue
		}
		if d := pipeline.DependsOn(f.Content); len(d) > 0 {
			out[n] = d
		}
	}
	return out
}

func (c *Catalog) decoratePipe(p woodpecker.Pipeline) woodpecker.Pipeline {
	p.HydrateJobs()
	p.AttachDepends(c.jobDepends(p))
	p.SortJobs()
	return c.applyCommitAuthor(p)
}

func (c *Catalog) pipelineFiles(p woodpecker.Pipeline) []FileBlob {
	ref := p.Commit
	if ref == "" {
		ref = p.Branch
	}
	if ref == "" || p.Repo == "" {
		return nil
	}
	if c.mem != nil {
		if files, ok := c.mem.filesOf(p.Repo, ref); ok {
			return files
		}
	}
	owner, name, ok := strings.Cut(p.Repo, "/")
	if !ok || c.FJ == nil {
		return nil
	}
	ents, err := c.FJ.ListContents(owner, name, ref, ".acahti/pipelines")
	if err != nil {
		return nil
	}
	var want []forgejo.ContentEntry
	for _, e := range ents {
		if e.Type != "file" && e.Type != "blob" && e.Type != "" {
			continue
		}
		ext := strings.ToLower(path.Ext(e.Name))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		want = append(want, e)
	}
	out := make([]FileBlob, len(want))
	var wg sync.WaitGroup
	for i, e := range want {
		wg.Add(1)
		go func(i int, e forgejo.ContentEntry) {
			defer wg.Done()
			filePath := e.Path
			if filePath == "" {
				filePath = ".acahti/pipelines/" + e.Name
			}
			content := decodeContent(e)
			if content == "" {
				if raw, err := c.FJ.GetFile(owner, name, ref, filePath); err == nil {
					content = decodeContent(raw)
				}
			}
			out[i] = FileBlob{Name: e.Name, Path: filePath, Content: content}
		}(i, e)
	}
	wg.Wait()
	if c.mem != nil {
		c.mem.setFiles(p.Repo, ref, out)
	}
	return out
}

func (c *Catalog) Remember(p woodpecker.Pipeline) woodpecker.Pipeline {
	if p.Repo == "" || p.Number == 0 {
		return p
	}
	p = c.decoratePipe(p)
	p = c.mergeLogTails(p)
	p = c.captureTerminalLogs(p)
	p = woodpecker.StripWait(p)
	if c.Idx != nil {
		_ = c.Idx.Upsert(p)
	}
	return c.Present(p)
}

func (c *Catalog) mergeLogTails(p woodpecker.Pipeline) woodpecker.Pipeline {
	if c.Idx == nil {
		return p
	}
	old, ok, err := c.Idx.Get(p.Repo, p.Number)
	if err != nil || !ok {
		return p
	}
	prev := map[int64]string{}
	for _, s := range old.Steps() {
		if s.LogTail != "" {
			prev[stepID(s)] = s.LogTail
		}
	}
	for i := range p.Jobs {
		for j := range p.Jobs[i].Steps {
			id := stepID(p.Jobs[i].Steps[j])
			if p.Jobs[i].Steps[j].LogTail == "" {
				p.Jobs[i].Steps[j].LogTail = prev[id]
			}
		}
	}
	return p
}

func (c *Catalog) captureTerminalLogs(p woodpecker.Pipeline) woodpecker.Pipeline {
	if c.WP == nil || !c.WP.Ready() {
		return p
	}
	for i := range p.Jobs {
		for j := range p.Jobs[i].Steps {
			s := &p.Jobs[i].Steps[j]
			if s.LogTail != "" || !stepFailed(s.State) {
				continue
			}
			id := stepID(*s)
			if id == 0 {
				continue
			}
			raw, err := c.WP.PipelineLog(p.Repo, p.Number, id)
			if err != nil {
				if s.Error != "" {
					s.LogTail = s.Error
				} else if p.Error != "" {
					s.LogTail = p.Error
				}
				continue
			}
			s.LogTail = woodpecker.FormatLog(raw)
			if s.LogTail == "" && s.Error != "" {
				s.LogTail = s.Error
			}
		}
	}
	return p
}

func (c *Catalog) CancelPipeline(user, repo string, number int64) (woodpecker.Pipeline, error) {
	if user != "" {
		owner, name, _ := strings.Cut(repo, "/")
		if _, err := c.seeRepo(user, owner, name); err != nil {
			return woodpecker.Pipeline{}, err
		}
	}
	if c.WP == nil || !c.WP.Ready() {
		return woodpecker.Pipeline{}, fmt.Errorf("ci unavailable")
	}
	raw, err := c.WP.GetPipeline(repo, number)
	if err != nil {
		return woodpecker.Pipeline{}, err
	}
	raw.Repo = repo
	raw.HydrateJobs()
	if !woodpecker.InFlight(raw.Status) && !woodpecker.PendingAfterFailure(raw) {
		return c.Remember(raw), nil
	}
	if err := c.WP.Cancel(repo, number); err != nil {
		again, rerr := c.WP.GetPipeline(repo, number)
		if rerr == nil {
			again.Repo = repo
			if !woodpecker.InFlight(again.Status) {
				return c.Remember(again), nil
			}
		}
		return woodpecker.Pipeline{}, err
	}
	return c.Refresh(repo, number)
}

func (c *Catalog) DeletePipeline(user, repo string, number int64) error {
	if user != "" {
		owner, name, _ := strings.Cut(repo, "/")
		if _, err := c.seeRepo(user, owner, name); err != nil {
			return err
		}
	}
	fresh, err := c.Refresh(repo, number)
	if err != nil {
		return err
	}
	if !woodpecker.DeleteAllowed(fresh.Status) {
		return fmt.Errorf("cannot delete pipeline with status %s", fresh.Status)
	}
	if err := c.WP.Delete(repo, number); err != nil {
		return err
	}
	if c.Idx != nil {
		_ = c.Idx.DeletePipeline(repo, number)
	}
	return nil
}

func (c *Catalog) Refresh(repo string, number int64) (woodpecker.Pipeline, error) {
	if c.WP == nil || !c.WP.Ready() {
		return woodpecker.Pipeline{}, fmt.Errorf("ci unavailable")
	}
	p, err := c.WP.GetPipeline(repo, number)
	if err != nil {
		return woodpecker.Pipeline{}, err
	}
	p.Repo = repo
	p.HydrateJobs()
	if woodpecker.PendingAfterFailure(p) {
		if err := c.WP.Cancel(repo, number); err == nil {
			if again, err := c.WP.GetPipeline(repo, number); err == nil {
				again.Repo = repo
				p = again
			}
		}
	}
	return c.Remember(p), nil
}

func (c *Catalog) IngestWoodpecker(raw []byte) (woodpecker.Pipeline, bool) {
	h, ok := store.ParseWoodpecker(raw)
	if !ok {
		return woodpecker.Pipeline{}, false
	}
	p := h.Pipeline
	if (len(p.Jobs) == 0 || woodpecker.PendingAfterFailure(p)) && c.WP != nil && c.WP.Ready() {
		if d, err := c.Refresh(h.Repo, h.Number); err == nil {
			return d, true
		}
	}
	return c.Remember(p), true
}

func (c *Catalog) IngestForgejo(payload map[string]any) (woodpecker.Pipeline, bool) {
	repo, n, ok := store.ParseForgejoStatus(payload)
	if ok {
		p, err := c.Refresh(repo, n)
		if err != nil {
			return woodpecker.Pipeline{}, false
		}
		return p, true
	}
	if !c.ApplyForgejoCatalog(payload) {
		c.forget()
	}
	return woodpecker.Pipeline{}, false
}

func (c *Catalog) Backfill() {
	if c.Idx == nil || c.WP == nil || !c.WP.Ready() {
		return
	}
	var repos []woodpecker.Repo
	var err error
	for i := 0; i < 20; i++ {
		repos, err = page.Walk(func(q page.Query) (page.Result[woodpecker.Repo], error) {
			return c.WP.ListRepos(q)
		})
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Printf("pipeline backfill: %v", err)
		return
	}
	for _, r := range repos {
		if !r.IsActive || r.FullName == "" {
			continue
		}
		res, err := c.WP.ListPipelines(r.FullName, page.Query{Page: 1, Size: page.MaxSize})
		if err != nil {
			continue
		}
		for _, p := range res.Items {
			if len(p.Jobs) == 0 {
				if d, err := c.WP.GetPipeline(r.FullName, p.Number); err == nil {
					p = d
				}
			}
			c.Remember(p)
		}
	}
}

func (c *Catalog) NavTree(user string) ([]NavTeam, error) {
	if !c.indexed() {
		return []NavTeam{}, nil
	}
	rows, err := c.Idx.NavTree(user, c.IsOrgAdmin(user))
	if err != nil {
		return nil, err
	}
	out := make([]NavTeam, 0, len(rows))
	for _, row := range rows {
		repos := c.asRepos(row.Repos, row.Team)
		c.paintPerms(user, repos)
		out = append(out, NavTeam{Team: row.Team, Repos: repos})
	}
	return out, nil
}

func (c *Catalog) PipelineDetail(user, repo string, number int64) (PipelineDetail, error) {
	owner, name, _ := strings.Cut(repo, "/")
	head, err := c.seeRepo(user, owner, name)
	if err != nil {
		return PipelineDetail{}, err
	}
	var p woodpecker.Pipeline
	if c.Idx != nil {
		if got, ok, err := c.Idx.Get(repo, number); err != nil {
			return PipelineDetail{}, err
		} else if ok {
			p = got
		}
	}
	if p.Number == 0 || woodpecker.InFlight(p.Status) {
		fresh, err := c.Refresh(repo, number)
		if err != nil {
			if p.Number == 0 {
				return PipelineDetail{}, err
			}
		} else {
			p = fresh
		}
	}
	p.Repo = repo
	p = c.Present(p)
	return PipelineDetail{Pipeline: p, Team: head.Team, Files: c.pipelineFiles(p)}, nil
}

func (c *Catalog) StepLog(user, repo string, number, step int64) (string, error) {
	owner, name, _ := strings.Cut(repo, "/")
	if _, err := c.seeRepo(user, owner, name); err != nil {
		return "", err
	}
	if step <= 0 {
		step = 1
	}
	if c.Idx != nil {
		if p, ok, err := c.Idx.Get(repo, number); err == nil && ok {
			for _, s := range p.Steps() {
				if stepID(s) == step && s.LogTail != "" {
					return s.LogTail, nil
				}
			}
		}
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
		return map[string]any{"log": tailLog(text, tail)}, nil
	}
	var failed []woodpecker.Step
	for _, s := range detail.Pipeline.Steps() {
		if stepFailed(s.State) {
			failed = append(failed, s)
		}
	}
	if len(failed) == 0 {
		return map[string]any{"log": detail.Pipeline.Error}, nil
	}
	var b strings.Builder
	for _, s := range failed {
		id := s.ID
		if id == 0 {
			id = s.PID
		}
		text, err := c.StepLog(user, repo, number, id)
		if err != nil {
			if s.Error != "" {
				text = s.Error
			} else if detail.Pipeline.Error != "" {
				text = detail.Pipeline.Error
			} else {
				text = err.Error()
			}
		}
		fmt.Fprintf(&b, "=== %s (step %d) %s ===\n%s\n", s.Name, id, s.State, tailLog(text, tail))
	}
	return map[string]any{"log": b.String()}, nil
}

func (c *Catalog) fillPRHeads(items []forgejo.PR) {
	for i := range items {
		p := &items[i]
		if p.Head.Ref != "" {
			continue
		}
		owner, name, ok := strings.Cut(p.Repo, "/")
		if !ok || p.Number == 0 {
			continue
		}
		full, err := c.FJ.GetPR(owner, name, p.Number)
		if err != nil {
			continue
		}
		p.Head = full.Head
		p.Base = full.Base
		if p.HTMLURL == "" {
			p.HTMLURL = full.HTMLURL
		}
	}
}

func (c *Catalog) BoardPRs(user string, q page.Query) (page.Result[forgejo.PR], error) {
	if !c.FJ.Ready() {
		return page.Of([]forgejo.PR{}, q, false), nil
	}
	var allow map[string]bool
	if !c.IsOrgAdmin(user) {
		visible, err := c.userRepos(user)
		if err != nil {
			return page.Result[forgejo.PR]{}, err
		}
		allow = visibleSet(visible)
	}
	var matched []forgejo.PR
	need := q.Norm().Page*q.Norm().Size + 1
	pq := page.Query{Page: 1, Size: page.MaxSize}
	for len(matched) < need && pq.Page <= page.MaxWalk {
		res, err := c.FJ.SearchPRs(c.Cfg.Org, "open", pq)
		if err != nil {
			return page.Result[forgejo.PR]{}, err
		}
		batch := res.Items
		if allow != nil {
			var vis []forgejo.PR
			for _, p := range batch {
				if allow[p.Repo] {
					vis = append(vis, p)
				}
			}
			batch = vis
		}
		c.fillPRHeads(batch)
		matched = append(matched, c.mergeReadyPRs(batch)...)
		if !res.HasMore {
			break
		}
		pq.Page++
	}
	return page.Take(matched, q), nil
}

func commitStatusKey(repo, sha string) string {
	return repo + "\n" + strings.ToLower(strings.TrimSpace(sha))
}

func (c *Catalog) mergeReadyPRs(prs []forgejo.PR) []forgejo.PR {
	if c.Idx == nil || len(prs) == 0 {
		return nil
	}
	repos := make([]string, 0, len(prs))
	shas := make([]string, 0, len(prs))
	for _, p := range prs {
		if p.Repo == "" || strings.TrimSpace(p.Head.SHA) == "" {
			continue
		}
		repos = append(repos, p.Repo)
		shas = append(shas, p.Head.SHA)
	}
	st, err := c.Idx.LatestStatusByCommit(repos, shas)
	if err != nil {
		return nil
	}
	var out []forgejo.PR
	for _, p := range prs {
		if !strings.EqualFold(st[commitStatusKey(p.Repo, p.Head.SHA)], "success") {
			continue
		}
		out = append(out, p)
	}
	return out
}

func (c *Catalog) ListAgents(q page.Query) (page.Result[woodpecker.Agent], error) {
	status, err := c.AgentStatus(q)
	if err != nil {
		return page.Result[woodpecker.Agent]{}, err
	}
	return page.Of(status.Items, page.Query{Page: status.Page, Size: status.Size}, status.HasMore), nil
}

type AgentStatus struct {
	Items   []woodpecker.Agent   `json:"items"`
	Page    int                  `json:"page"`
	Size    int                  `json:"page_size"`
	HasMore bool                 `json:"has_more"`
	Queue   woodpecker.QueueInfo `json:"queue"`
}

func (c *Catalog) AgentStatus(q page.Query) (AgentStatus, error) {
	if c.WP == nil || !c.WP.Ready() {
		return AgentStatus{Items: []woodpecker.Agent{}, Page: q.Norm().Page, Size: q.Norm().Size, Queue: woodpecker.QueueInfo{}}, nil
	}
	agents, err := c.WP.Agents()
	if err != nil {
		return AgentStatus{}, err
	}
	qi := c.queueInfo()
	agents = woodpecker.PaintAgents(liveAgents(agents, time.Now()), qi)
	res := page.Take(agents, q)
	return AgentStatus{Items: res.Items, Page: res.Page, Size: res.Size, HasMore: res.HasMore, Queue: qi}, nil
}

func (c *Catalog) Queue() woodpecker.QueueInfo {
	if c.WP == nil || !c.WP.Ready() {
		return woodpecker.QueueInfo{}
	}
	return c.queueInfo()
}

const agentAlive = 5 * time.Minute

func liveAgents(agents []woodpecker.Agent, now time.Time) []woodpecker.Agent {
	var out []woodpecker.Agent
	for _, a := range agents {
		if a.LastSeen > 0 && now.Sub(time.Unix(a.LastSeen, 0)) <= agentAlive {
			out = append(out, a)
		}
	}
	return out
}

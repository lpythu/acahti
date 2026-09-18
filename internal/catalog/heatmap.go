package catalog

import (
	"log"
	"strings"
	"time"

	"acahti/internal/forgejo"
	"acahti/internal/page"
)

const heatWeeks = 53
const heatCommitPages = 80

type HeatDay struct {
	Date  string `json:"date"`
	Value int64  `json:"value"`
}

type Heatmap struct {
	Total int64     `json:"total"`
	Days  []HeatDay `json:"days"`
}

func (c *Catalog) BoardHeatmap(user string) (Heatmap, error) {
	now := time.Now()
	start, end := heatRange(now)
	if c.Idx == nil {
		return fillHeatmap(nil, start, end, now), nil
	}
	counts, err := c.Idx.Heatmap(user, start, end)
	if err != nil {
		return Heatmap{}, err
	}
	return fillHeatmap(counts, start, end, now), nil
}

func (c *Catalog) BackfillHeatmaps() {
	c.heatMu.Lock()
	if c.heatBusy {
		c.heatAgain = true
		c.heatMu.Unlock()
		return
	}
	c.heatBusy = true
	c.heatMu.Unlock()

	wrote := c.syncCommitHeat()

	c.heatMu.Lock()
	again := c.heatAgain
	c.heatAgain = false
	c.heatBusy = false
	c.heatMu.Unlock()
	if again {
		c.BackfillHeatmaps()
		return
	}
	if wrote && c.Notify != nil {
		c.Notify("forgejo", map[string]any{"ok": true})
	}
}

func (c *Catalog) syncCommitHeat() bool {
	if !c.indexed() || c.fj == nil || !c.fj.Ready() {
		return false
	}
	users, err := c.fj.AllUsers()
	if err != nil {
		log.Printf("heatmap users: %v", err)
		return false
	}
	byEmail, byName := heatUserIndex(users)
	logins := map[string]struct{}{}
	for _, u := range users {
		if login := strings.TrimSpace(u.Login); login != "" {
			logins[login] = struct{}{}
		}
	}
	if extra, err := c.Idx.HeatmapLogins(); err == nil {
		for _, login := range extra {
			if login != "" {
				logins[login] = struct{}{}
			}
		}
	}
	repos, err := c.Idx.VisibleRepos("", true)
	if err != nil {
		log.Printf("heatmap repos: %v", err)
		return false
	}
	start, _ := heatRange(time.Now())
	startKey := start.Format("2006-01-02")
	counts := map[string]map[string]int64{}
	for _, repo := range repos {
		owner, name, ok := strings.Cut(repo.FullName, "/")
		if !ok || owner == "" || name == "" {
			continue
		}
		c.addRepoCommits(owner, name, startKey, byEmail, byName, counts)
	}
	for login := range logins {
		if err := c.Idx.ReplaceHeatmap(login, counts[login]); err != nil {
			log.Printf("heatmap %s: %v", login, err)
		}
	}
	return true
}

func (c *Catalog) addRepoCommits(owner, name, startKey string, byEmail, byName map[string]string, counts map[string]map[string]int64) {
	for n := 1; n <= heatCommitPages; n++ {
		res, err := c.fj.ListHeatCommits(owner, name, page.Query{Page: n, Size: page.MaxSize})
		if err != nil {
			log.Printf("heatmap %s/%s: %v", owner, name, err)
			return
		}
		older := false
		for _, cm := range res.Items {
			day, ok := commitHeatDay(cm.Commit.Author.Date)
			if !ok {
				continue
			}
			if day < startKey {
				older = true
				continue
			}
			login := commitHeatLogin(cm, byEmail, byName)
			if login == "" {
				continue
			}
			days := counts[login]
			if days == nil {
				days = map[string]int64{}
				counts[login] = days
			}
			days[day]++
		}
		if older || !res.HasMore {
			return
		}
	}
}

func heatUserIndex(users []forgejo.User) (byEmail, byName map[string]string) {
	byEmail = map[string]string{}
	names := map[string]string{}
	dup := map[string]bool{}
	for _, u := range users {
		login := strings.TrimSpace(u.Login)
		if login == "" {
			continue
		}
		if email := strings.ToLower(strings.TrimSpace(u.Email)); email != "" {
			byEmail[email] = login
		}
		for _, name := range []string{login, strings.TrimSpace(u.FullName)} {
			key := strings.ToLower(name)
			if key == "" {
				continue
			}
			if prev, ok := names[key]; ok && prev != login {
				dup[key] = true
				continue
			}
			names[key] = login
		}
	}
	byName = map[string]string{}
	for key, login := range names {
		if !dup[key] {
			byName[key] = login
		}
	}
	return byEmail, byName
}

func commitHeatLogin(cm forgejo.Commit, byEmail, byName map[string]string) string {
	if cm.Author != nil {
		if login := strings.TrimSpace(cm.Author.Login); login != "" {
			return login
		}
	}
	if email := strings.ToLower(strings.TrimSpace(cm.Commit.Author.Email)); email != "" {
		if login := byEmail[email]; login != "" {
			return login
		}
	}
	if name := strings.ToLower(strings.TrimSpace(cm.Commit.Author.Name)); name != "" {
		return byName[name]
	}
	return ""
}

func commitHeatDay(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if len(raw) >= 10 && raw[4] == '-' && raw[7] == '-' {
		return raw[:10], true
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return "", false
	}
	return t.UTC().Format("2006-01-02"), true
}

func heatRange(now time.Time) (start, end time.Time) {
	now = now.UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	start = today.AddDate(0, 0, -364)
	for start.Weekday() != time.Monday {
		start = start.AddDate(0, 0, -1)
	}
	return start, start.AddDate(0, 0, heatWeeks*7)
}

func fillHeatmap(counts map[string]int64, start, end, now time.Time) Heatmap {
	today := now.UTC()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	// Viewer local today can be one calendar day ahead of UTC.
	limit := today.AddDate(0, 0, 1)
	out := Heatmap{Days: make([]HeatDay, 0, heatWeeks*7)}
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		n := counts[key]
		out.Days = append(out.Days, HeatDay{Date: key, Value: n})
		if !d.After(limit) {
			out.Total += n
		}
	}
	return out
}

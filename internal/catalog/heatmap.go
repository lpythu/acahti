package catalog

import (
	"fmt"
	"log"
	"strings"
	"time"

	"acahti/internal/forgejo"
	"acahti/internal/page"
	"acahti/internal/store"
)

const heatWeeks = 53

var heatZone = time.FixedZone("UTC+8", 8*60*60)

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

func (c *Catalog) BoardActivities(user, date string, q page.Query) (page.Result[forgejo.Activity], error) {
	if _, err := heatDate(date, time.Now()); err != nil {
		return page.Result[forgejo.Activity]{}, err
	}
	if c.fj == nil || !c.fj.Ready() {
		return page.Of([]forgejo.Activity{}, q, false), nil
	}
	return c.fj.ListUserActivityFeeds(user, date, q)
}

func heatDate(date string, now time.Time) (time.Time, error) {
	d, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(date), heatZone)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: date", ErrInvalid)
	}
	start, end := heatRange(now)
	today := time.Date(now.In(heatZone).Year(), now.In(heatZone).Month(), now.In(heatZone).Day(), 0, 0, 0, 0, heatZone)
	if d.Before(start) || !d.Before(end) || d.After(today) {
		return time.Time{}, fmt.Errorf("%w: date", ErrInvalid)
	}
	return d, nil
}

func (c *Catalog) RememberHeat(payload map[string]any) {
	c.syncHeat(store.ForgejoActor(payload))
}

func (c *Catalog) BackfillHeatmaps() {
	if !c.indexed() || c.fj == nil || !c.fj.Ready() {
		return
	}
	seen := map[string]struct{}{}
	if logins, err := c.Idx.HeatmapLogins(); err == nil {
		for _, login := range logins {
			if login != "" {
				seen[login] = struct{}{}
			}
		}
	}
	if users, err := c.fj.AllUsers(); err == nil {
		for _, u := range users {
			if login := strings.TrimSpace(u.Login); login != "" {
				seen[login] = struct{}{}
			}
		}
	}
	for login := range seen {
		c.syncHeat(login)
	}
	if c.Notify != nil {
		c.Notify("forgejo", map[string]any{"ok": true})
	}
}

func (c *Catalog) syncHeat(login string) {
	if login == "" || !c.indexed() || c.fj == nil || !c.fj.Ready() {
		return
	}
	points, err := c.fj.UserHeatmap(login)
	if err != nil {
		log.Printf("heatmap %s: %v", login, err)
		return
	}
	if err := c.Idx.ReplaceHeatmap(login, heatCounts(points)); err != nil {
		log.Printf("heatmap %s: %v", login, err)
	}
}

func heatCounts(points []forgejo.HeatPoint) map[string]int64 {
	out := map[string]int64{}
	for _, p := range points {
		if p.Timestamp <= 0 || p.Contributions == 0 {
			continue
		}
		key := time.Unix(p.Timestamp, 0).In(heatZone).Format("2006-01-02")
		out[key] += p.Contributions
	}
	return out
}

func heatRange(now time.Time) (start, end time.Time) {
	now = now.In(heatZone)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, heatZone)
	start = today.AddDate(0, 0, -364)
	for start.Weekday() != time.Monday {
		start = start.AddDate(0, 0, -1)
	}
	return start, start.AddDate(0, 0, heatWeeks*7)
}

func fillHeatmap(counts map[string]int64, start, end, now time.Time) Heatmap {
	start = time.Date(start.In(heatZone).Year(), start.In(heatZone).Month(), start.In(heatZone).Day(), 0, 0, 0, 0, heatZone)
	end = time.Date(end.In(heatZone).Year(), end.In(heatZone).Month(), end.In(heatZone).Day(), 0, 0, 0, 0, heatZone)
	now = now.In(heatZone)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, heatZone)
	out := Heatmap{Days: make([]HeatDay, 0, heatWeeks*7)}
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		n := counts[key]
		out.Days = append(out.Days, HeatDay{Date: key, Value: n})
		if !d.After(today) {
			out.Total += n
		}
	}
	return out
}

package catalog

import (
	"fmt"
	"log"
	"strings"
	"time"

	"acahti/internal/page"
	"acahti/internal/store"
	"acahti/internal/woodpecker"
)

// helm --wait is 5m. Longer than that with no new log is a Woodpecker zombie.
const (
	reapEvery  = time.Minute
	staleAfter = 15 * time.Minute
)

type logWatch struct {
	fp    string
	since time.Time
}

func (c *Catalog) WatchPipelines() {
	watches := map[string]logWatch{}
	c.Reap(time.Now(), watches)
	t := time.NewTicker(reapEvery)
	defer t.Stop()
	for range t.C {
		c.Reap(time.Now(), watches)
	}
}

func (c *Catalog) Reap(now time.Time, watches map[string]logWatch) {
	if c.Idx == nil || c.WP == nil || !c.WP.Ready() {
		return
	}
	listed, err := page.Walk(func(q page.Query) (page.Result[woodpecker.Pipeline], error) {
		return c.Idx.List(store.Filter{Status: []string{"running"}}, q)
	})
	if err != nil {
		log.Printf("pipeline reap: list: %v", err)
		return
	}
	keep := map[string]logWatch{}
	for _, p := range listed {
		c.reapOne(p, now, watches, keep)
	}
	for k := range watches {
		delete(watches, k)
	}
	for k, w := range keep {
		watches[k] = w
	}
}

func (c *Catalog) reapOne(p woodpecker.Pipeline, now time.Time, prev, keep map[string]logWatch) {
	fresh, err := c.Refresh(p.Repo, p.Number)
	if err != nil {
		for _, s := range runningSteps(p) {
			key := watchKey(p.Repo, p.Number, stepID(s))
			if w, ok := prev[key]; ok {
				keep[key] = w
			}
		}
		return
	}
	if !strings.EqualFold(fresh.Status, "running") {
		c.emit(fresh)
		return
	}
	steps := runningSteps(fresh)
	if len(steps) == 0 {
		steps = []woodpecker.Step{{}}
	}
	cancel := false
	for _, s := range steps {
		id := stepID(s)
		key := watchKey(fresh.Repo, fresh.Number, id)
		text, err := c.WP.PipelineLog(fresh.Repo, fresh.Number, id)
		if err != nil {
			if w, ok := prev[key]; ok {
				keep[key] = w
			}
			continue
		}
		next, stale := bumpWatch(prev[key], logFP(woodpecker.FormatLog(text)), now, staleAfter)
		keep[key] = next
		if stale {
			cancel = true
		}
	}
	if !cancel {
		return
	}
	done, err := c.CancelPipeline("", fresh.Repo, fresh.Number)
	if err != nil {
		log.Printf("pipeline reap: cancel %s #%d: %v", fresh.Repo, fresh.Number, err)
		return
	}
	log.Printf("pipeline reap: cancel %s #%d (silent %s)", fresh.Repo, fresh.Number, staleAfter)
	c.emit(done)
}

func (c *Catalog) emit(p woodpecker.Pipeline) {
	if c.Notify != nil {
		c.Notify("pipeline.updated", p)
	}
}

func runningSteps(p woodpecker.Pipeline) []woodpecker.Step {
	var out []woodpecker.Step
	for _, s := range p.Steps() {
		if strings.EqualFold(s.State, "running") {
			out = append(out, s)
		}
	}
	return out
}

func stepID(s woodpecker.Step) int64 {
	if s.ID > 0 {
		return s.ID
	}
	return s.PID
}

func watchKey(repo string, number, step int64) string {
	return fmt.Sprintf("%s#%d/%d", repo, number, step)
}

func logFP(text string) string {
	n := len(text)
	tail := text
	if n > 128 {
		tail = text[n-128:]
	}
	return fmt.Sprintf("%d:%s", n, tail)
}

func bumpWatch(w logWatch, fp string, now time.Time, after time.Duration) (logWatch, bool) {
	if w.fp != fp || w.since.IsZero() {
		return logWatch{fp: fp, since: now}, false
	}
	return w, now.Sub(w.since) >= after
}

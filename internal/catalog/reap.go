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
	queueEvery = 3 * time.Second
	staleAfter = 15 * time.Minute
)

type logWatch struct {
	fp    string
	since time.Time
}

func (c *Catalog) WatchPipelines() {
	watches := map[string]logWatch{}
	queueFP := map[string]string{}
	c.Reap(time.Now(), watches)
	c.reapQueue(queueFP)
	logs := time.NewTicker(reapEvery)
	defer logs.Stop()
	queue := time.NewTicker(queueEvery)
	defer queue.Stop()
	for {
		select {
		case <-logs.C:
			c.Reap(time.Now(), watches)
		case <-queue.C:
			c.reapQueue(queueFP)
		}
	}
}

func (c *Catalog) Reap(now time.Time, watches map[string]logWatch) {
	if c.Idx == nil || c.wp == nil || !c.wp.Ready() {
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
	if c.wp == nil || !c.wp.Ready() {
		return
	}
	raw, err := c.wp.GetPipeline(p.Repo, p.Number)
	if err != nil {
		for _, s := range runningSteps(p) {
			key := watchKey(p.Repo, p.Number, stepID(s))
			if w, ok := prev[key]; ok {
				keep[key] = w
			}
		}
		return
	}
	raw.Repo = p.Repo
	raw.HydrateJobs()
	kernelRunning := strings.EqualFold(raw.Status, "running")
	steps := runningSteps(raw)
	fresh := c.Remember(raw)
	if !kernelRunning {
		c.emit(fresh)
		return
	}
	if len(steps) == 0 {
		steps = []woodpecker.Step{{}}
	}
	var staleSteps []woodpecker.Step
	for _, s := range steps {
		id := stepID(s)
		key := watchKey(fresh.Repo, fresh.Number, id)
		text, err := c.wp.PipelineLog(fresh.Repo, fresh.Number, id)
		if err != nil {
			if w, ok := prev[key]; ok {
				keep[key] = w
			}
			continue
		}
		next, stale := bumpWatch(prev[key], logFP(woodpecker.FormatLog(text)), now, staleAfter)
		keep[key] = next
		if stale {
			staleSteps = append(staleSteps, s)
		}
	}
	if len(staleSteps) == 0 {
		return
	}
	reason := silentCancelReason(staleSteps)
	done, err := c.cancelPipeline("", fresh.Repo, fresh.Number, reason)
	if err != nil {
		log.Printf("pipeline reap: cancel %s #%d: %v", fresh.Repo, fresh.Number, err)
		return
	}
	log.Printf("pipeline reap: cancel %s #%d (%s)", fresh.Repo, fresh.Number, reason)
	c.emit(done)
}

func silentCancelReason(steps []woodpecker.Step) string {
	var names []string
	seen := map[string]bool{}
	for _, s := range steps {
		n := strings.TrimSpace(s.Name)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		names = append(names, n)
	}
	msg := fmt.Sprintf("canceled: no new log for %dm", int(staleAfter.Minutes()))
	if len(names) > 0 {
		msg += " on step " + strings.Join(names, ", ")
	}
	return msg
}

func (c *Catalog) emit(p woodpecker.Pipeline) {
	p = c.Present(p)
	if c.Notify != nil {
		c.Notify("pipeline.updated", p)
	}
}

func (c *Catalog) reapQueue(fp map[string]string) {
	if c.wp == nil || !c.wp.Ready() {
		return
	}
	seen := map[string]woodpecker.Pipeline{}
	if c.Idx != nil {
		listed, err := page.Walk(func(q page.Query) (page.Result[woodpecker.Pipeline], error) {
			return c.Idx.List(store.Filter{Status: []string{"running", "pending"}}, q)
		})
		if err != nil {
			log.Printf("pipeline queue: list: %v", err)
			return
		}
		for _, p := range listed {
			seen[fmt.Sprintf("%s#%d", p.Repo, p.Number)] = p
		}
	}
	for repo, n := range c.queueHeadNumbers(nil) {
		key := fmt.Sprintf("%s#%d", repo, n)
		if _, ok := seen[key]; !ok {
			seen[key] = woodpecker.Pipeline{Repo: repo, Number: n, Status: "pending"}
		}
	}
	keep := map[string]string{}
	for _, p := range seen {
		painted := p
		fresh, err := c.Refresh(p.Repo, p.Number)
		if err != nil {
			painted = c.Present(p)
		} else {
			painted = fresh
		}
		key := fmt.Sprintf("%s#%d", painted.Repo, painted.Number)
		next := woodpecker.WaitFingerprint(painted)
		keep[key] = next
		if fp[key] != next {
			c.emit(painted)
		}
	}
	for k := range fp {
		delete(fp, k)
	}
	for k, v := range keep {
		fp[k] = v
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

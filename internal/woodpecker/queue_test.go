package woodpecker

import (
	"testing"
)

func TestAnnotateQueuePosition(t *testing.T) {
	p := Pipeline{
		Repo:   "saidc/tm-web",
		ID:     9,
		Number: 12,
		Status: "pending",
		Jobs: []Job{
			{Name: "ci", State: "pending"},
			{Name: "cd.office", State: "pending"},
		},
	}
	q := QueueInfo{
		Pending: []QueueTask{
			{Name: "ci", Repo: "saidc/other", Number: 1, Wait: WaitQueue, QueuePosition: 1},
			{Name: "ci", Repo: "saidc/tm-web", Number: 12, PipelineID: 9, Wait: WaitQueue, QueuePosition: 2},
		},
		WaitingOnDeps: []QueueTask{
			{Name: "cd.office", Repo: "saidc/tm-web", Number: 12, PipelineID: 9, Wait: WaitDeps},
		},
	}
	got := Annotate(p, q)
	if got.Wait != WaitQueue || got.QueuePosition != 2 {
		t.Fatalf("pipeline wait=%q pos=%d", got.Wait, got.QueuePosition)
	}
	if got.Jobs[0].Wait != WaitQueue || got.Jobs[0].QueuePosition != 2 {
		t.Fatalf("ci %+v", got.Jobs[0])
	}
	if got.Jobs[1].Wait != WaitDeps {
		t.Fatalf("cd %+v", got.Jobs[1])
	}
}

func TestAnnotateRunningClearsWait(t *testing.T) {
	p := Pipeline{
		Repo:   "saidc/tm-web",
		Number: 3,
		Status: "running",
		Jobs:   []Job{{Name: "ci", State: "pending"}},
	}
	q := QueueInfo{Running: []QueueTask{{Name: "ci", Repo: "saidc/tm-web", Number: 3, Agent: "buildof"}}}
	got := Annotate(p, q)
	if got.Wait != "" || got.Jobs[0].Wait != "" || got.Jobs[0].Agent != "buildof" || got.Jobs[0].State != "running" {
		t.Fatalf("%+v", got)
	}
}

func TestAnnotateConcurrency(t *testing.T) {
	p := Pipeline{
		Repo:   "saidc/tm-web",
		Number: 4,
		Status: "pending",
		Jobs:   []Job{{Name: "cd.office", State: "pending"}},
	}
	q := QueueInfo{Pending: []QueueTask{{
		Name: "cd.office", Repo: "saidc/tm-web", Number: 4,
		Wait: WaitConcurrency, QueuePosition: 1, ConcurrencyLimit: 1,
	}}}
	got := Annotate(p, q)
	if got.Wait != WaitConcurrency || got.Jobs[0].Wait != WaitConcurrency {
		t.Fatalf("%+v", got)
	}
}

func TestAnnotateDoesNotInventSkipOrWait(t *testing.T) {
	p := Pipeline{
		Repo:   "saidc/api-gateway",
		Number: 49,
		Status: "running",
		Jobs: []Job{
			{Name: "ci", State: "failure"},
			{Name: "cd.office", State: "pending"},
		},
	}
	got := Annotate(p, QueueInfo{})
	if got.Status != "running" {
		t.Fatalf("status %+v", got)
	}
	if got.Jobs[1].State != "pending" || got.Jobs[1].Wait != "" {
		t.Fatalf("cd %+v", got.Jobs[1])
	}
}

func TestStripWait(t *testing.T) {
	p := StripWait(Pipeline{Wait: WaitQueue, QueuePosition: 3, Jobs: []Job{{Wait: WaitQueue, QueuePosition: 3, Agent: "a"}}})
	if p.Wait != "" || p.QueuePosition != 0 || p.Jobs[0].Wait != "" || p.Jobs[0].Agent != "" {
		t.Fatalf("%+v", p)
	}
}

func TestPaintAgents(t *testing.T) {
	got := PaintAgents([]Agent{{ID: 1, Name: "buildof", Capacity: 4}}, QueueInfo{
		Running: []QueueTask{{Agent: "buildof"}, {Agent: "buildof"}},
	})
	if len(got) != 1 || got[0].Running != 2 {
		t.Fatalf("%+v", got)
	}
}

func TestWaitFingerprintChanges(t *testing.T) {
	a := Pipeline{Status: "pending", Wait: WaitQueue, QueuePosition: 2}
	b := a
	b.QueuePosition = 1
	if WaitFingerprint(a) == WaitFingerprint(b) {
		t.Fatal("position must change fingerprint")
	}
}

func TestAnnotateEmptyQueueDoesNotInferWait(t *testing.T) {
	p := Pipeline{
		Repo:   "saidc/voidgate",
		Number: 36,
		Status: "pending",
		Jobs:   []Job{{Name: "ci", State: "pending"}},
	}
	got := Annotate(p, QueueInfo{Fetched: true})
	if got.Status != "pending" || got.Wait != "" || got.Jobs[0].Wait != "" {
		t.Fatalf("%+v", got)
	}
}

func TestPipelineStripStatsRunningAndQueued(t *testing.T) {
	q := QueueInfo{
		Fetched: true,
		Running: []QueueTask{{Repo: "saidc/a", Number: 1, Name: "ci"}},
		Pending: []QueueTask{{Repo: "saidc/b", Number: 2, Name: "ci"}},
	}
	s := PipelineStripStats(q)
	if s.RunningCount != 1 || s.PendingCount != 1 {
		t.Fatalf("%+v", s)
	}
}

func TestPipelineStripStatsOnePipelineTwoJobs(t *testing.T) {
	q := QueueInfo{
		Running: []QueueTask{{Repo: "saidc/a", Number: 1, Name: "ci"}},
		Pending: []QueueTask{{Repo: "saidc/a", Number: 1, Name: "cd.office"}},
	}
	s := PipelineStripStats(q)
	if s.RunningCount != 1 || s.PendingCount != 0 {
		t.Fatalf("%+v", s)
	}
}

func TestInQueue(t *testing.T) {
	p := Pipeline{Repo: "saidc/tm-web", Number: 12, ID: 9}
	q := QueueInfo{Pending: []QueueTask{{Repo: "saidc/tm-web", Number: 12}}}
	if !InQueue(p, q) {
		t.Fatal("pending task must match")
	}
	if InQueue(p, QueueInfo{Fetched: true}) {
		t.Fatal("empty queue")
	}
}

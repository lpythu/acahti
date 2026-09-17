package woodpecker

import (
	"testing"
)

func TestInFlight(t *testing.T) {
	if !InFlight("running") || !InFlight("started") || !InFlight("pending") || !InFlight("created") || !InFlight("blocked") {
		t.Fatal("in flight")
	}
	if InFlight("success") || InFlight("failure") || InFlight("killed") || InFlight("canceled") || InFlight("skipped") {
		t.Fatal("settled")
	}
	if !DeleteAllowed("error") || !DeleteAllowed("failure") || DeleteAllowed("running") || DeleteAllowed("blocked") {
		t.Fatal("delete allowed")
	}
}

func TestPendingAfterFailure(t *testing.T) {
	if !PendingAfterFailure(Pipeline{Jobs: []Job{
		{Name: "ci", State: "failure"},
		{Name: "cd.office", State: "pending"},
	}}) {
		t.Fatal("failed parent with queued child")
	}
	if PendingAfterFailure(Pipeline{Jobs: []Job{
		{Name: "ci", State: "failure"},
		{Name: "notify", State: "running"},
	}}) {
		t.Fatal("a running job must keep the pipeline live")
	}
	if PendingAfterFailure(Pipeline{Jobs: []Job{
		{Name: "ci", State: "failure"},
		{Name: "cd.office", State: "skipped"},
	}}) {
		t.Fatal("already skipped")
	}
}

func TestJobNameFromFileEmptyIsNotDot(t *testing.T) {
	if got := jobNameFromFile(""); got != "" {
		t.Fatalf("%q", got)
	}
	if got := jobNameFromFile("."); got != "" {
		t.Fatalf("%q", got)
	}
	if got := jobNameFromFile(".acahti/pipelines/ci.yaml"); got != "ci" {
		t.Fatalf("%q", got)
	}
}

func TestHydrateJobsSkipsErrorWithoutFile(t *testing.T) {
	p := Pipeline{Status: "error", Errors: []PipeError{{Type: "generic", Message: "pipeline definition not found"}}}
	p.HydrateJobs()
	if len(p.Jobs) != 0 {
		t.Fatalf("jobs=%+v", p.Jobs)
	}
}

func TestPipelineErrorsBecomeJobs(t *testing.T) {
	p, err := DecodeKernel([]byte(`{"number":4,"status":"error","title":"","errors":[{"type":"linter","message":"invalid depends_on","data":{"file":".acahti/pipelines/ci.yaml"}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if p.Error != "invalid depends_on" {
		t.Fatalf("error=%q", p.Error)
	}
	p.HydrateJobs()
	if len(p.Jobs) != 1 || p.Jobs[0].Name != "ci" || p.Jobs[0].State != "error" {
		t.Fatalf("jobs=%+v", p.Jobs)
	}
	if p.Jobs[0].Steps[0].Error != "invalid depends_on" {
		t.Fatalf("step=%+v", p.Jobs[0].Steps)
	}
}

func TestSortJobsByDependsOn(t *testing.T) {
	p := Pipeline{Jobs: []Job{
		{PID: 1, Name: "cd.office", DependsOn: []string{"ci"}},
		{PID: 2, Name: "ci"},
		{PID: 3, Name: "e2e.office", DependsOn: []string{"cd.office"}},
	}}
	p.SortJobs()
	if p.Jobs[0].Name != "ci" || p.Jobs[1].Name != "cd.office" || p.Jobs[2].Name != "e2e.office" {
		t.Fatalf("%+v", p.Jobs)
	}
}

func TestAttachDependsFromYAMLThenSort(t *testing.T) {
	p := Pipeline{Jobs: []Job{
		{PID: 1, Name: "cd.office"},
		{PID: 2, Name: "ci"},
		{PID: 3, Name: "e2e.office"},
	}}
	p.AttachDepends(map[string][]string{
		"cd.office":  {"ci"},
		"e2e.office": {"cd.office"},
	})
	p.SortJobs()
	if p.Jobs[0].Name != "ci" || p.Jobs[1].Name != "cd.office" || p.Jobs[2].Name != "e2e.office" {
		t.Fatalf("%+v", p.Jobs)
	}
}

func TestSortJobsByPIDThenName(t *testing.T) {
	p := Pipeline{Jobs: []Job{
		{PID: 3, Name: "z"},
		{PID: 1, Name: "b"},
		{PID: 1, Name: "a"},
	}}
	p.SortJobs()
	if p.Jobs[0].Name != "a" || p.Jobs[1].Name != "b" || p.Jobs[2].Name != "z" {
		t.Fatalf("%+v", p.Jobs)
	}
}

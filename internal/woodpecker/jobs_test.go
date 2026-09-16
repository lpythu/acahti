package woodpecker

import (
	"testing"
)

func TestInFlight(t *testing.T) {
	if !InFlight("running") || !InFlight("pending") || !InFlight("blocked") {
		t.Fatal("in flight")
	}
	if InFlight("success") || InFlight("failure") || InFlight("killed") {
		t.Fatal("settled")
	}
	if !DeleteAllowed("error") || !DeleteAllowed("failure") || DeleteAllowed("running") || DeleteAllowed("blocked") {
		t.Fatal("delete allowed")
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

func TestSortJobsByName(t *testing.T) {
	p := Pipeline{Jobs: []Job{{Name: "cd.hk"}, {Name: "ci"}, {Name: "cd.office"}}}
	p.SortJobs()
	if len(p.Jobs) != 3 || p.Jobs[0].Name != "cd.hk" || p.Jobs[1].Name != "cd.office" || p.Jobs[2].Name != "ci" {
		t.Fatalf("%+v", p.Jobs)
	}
}

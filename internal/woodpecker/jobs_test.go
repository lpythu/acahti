package woodpecker

import (
	"encoding/json"
	"testing"
)

func TestInFlight(t *testing.T) {
	if !InFlight("running") || !InFlight("pending") || !InFlight("blocked") {
		t.Fatal("in flight")
	}
	if InFlight("success") || InFlight("failure") || InFlight("killed") {
		t.Fatal("settled")
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
	var p Pipeline
	raw := `{"number":4,"status":"error","title":"","errors":[{"type":"linter","message":"invalid depends_on","data":{"file":".acahti/pipelines/ci.yaml"}}]}`
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatal(err)
	}
	if p.Error != "invalid depends_on" {
		t.Fatalf("error=%q", p.Error)
	}
	p.HydrateJobs()
	if len(p.Jobs) != 1 || p.Jobs[0].Name != "ci" || p.Jobs[0].State != "error" {
		t.Fatalf("jobs=%+v", p.Jobs)
	}
	if p.Jobs[0].Children[0].Error != "invalid depends_on" {
		t.Fatalf("step=%+v", p.Jobs[0].Children)
	}
}

func TestMergeDeclaredKeepsFailureAndSkipsRest(t *testing.T) {
	p := Pipeline{
		Status: "error",
		Error:  "yaml: line 3: did not find expected key",
		Errors: []PipeError{{
			Type:    "compiler",
			Message: "yaml: line 3: did not find expected key",
			Data:    map[string]any{"file": ".acahti/pipelines/ci.yaml"},
		}},
	}
	p.HydrateJobs()
	got := MergeDeclaredJobs(p, []string{"ci", "cd.office", "cd.hk"})
	if len(got.Jobs) != 3 {
		t.Fatalf("jobs=%+v", got.Jobs)
	}
	if got.Jobs[0].Name != "ci" || got.Jobs[0].State != "error" {
		t.Fatalf("ci=%+v", got.Jobs[0])
	}
	if got.Jobs[1].Name != "cd.office" || got.Jobs[1].State != "skipped" {
		t.Fatalf("office=%+v", got.Jobs[1])
	}
	if got.Jobs[2].Name != "cd.hk" || got.Jobs[2].State != "skipped" {
		t.Fatalf("hk=%+v", got.Jobs[2])
	}
}

func TestMergeDeclaredIgnoresDotJob(t *testing.T) {
	p := Pipeline{
		Status: "error",
		Error:  "pipeline definition not found",
		Jobs:   []Job{{Name: ".", State: "error"}},
	}
	got := MergeDeclaredJobs(p, []string{"ci", "cd.office"})
	if len(got.Jobs) != 2 || got.Jobs[0].Name != "ci" || got.Jobs[0].State != "error" {
		t.Fatalf("%+v", got.Jobs)
	}
	if got.Jobs[1].Name != "cd.office" || got.Jobs[1].State != "skipped" {
		t.Fatalf("%+v", got.Jobs)
	}
}

func TestMergeDeclaredAssignsErrorToCiNotCd(t *testing.T) {
	p := Pipeline{Status: "error", Error: "could not parse config"}
	got := MergeDeclaredJobs(p, []string{"cd.hk", "cd.office", "ci"})
	if len(got.Jobs) != 3 {
		t.Fatalf("jobs=%+v", got.Jobs)
	}
	by := map[string]string{}
	for _, j := range got.Jobs {
		by[j.Name] = j.State
	}
	if by["ci"] != "error" || by["cd.office"] != "skipped" || by["cd.hk"] != "skipped" {
		t.Fatalf("%v", by)
	}
}

func TestMergeDeclaredAssignsGenericErrorToFirstJob(t *testing.T) {
	p := Pipeline{Status: "error", Error: "could not parse config"}
	p.HydrateJobs()
	if len(p.Jobs) != 0 {
		t.Fatalf("hydrate=%+v", p.Jobs)
	}
	got := MergeDeclaredJobs(p, []string{"ci", "cd.office"})
	if len(got.Jobs) != 2 {
		t.Fatalf("jobs=%+v", got.Jobs)
	}
	if got.Jobs[0].Name != "ci" || got.Jobs[0].State != "error" {
		t.Fatalf("ci=%+v", got.Jobs[0])
	}
	if got.Jobs[1].Name != "cd.office" || got.Jobs[1].State != "skipped" {
		t.Fatalf("cd=%+v", got.Jobs[1])
	}
}

func TestMergeDeclaredKeepsRuntimeFailure(t *testing.T) {
	p := Pipeline{
		Status: "failure",
		Jobs: []Job{{
			Name:  "ci",
			State: "failure",
			Children: []Step{{
				PID:   2,
				Name:  "check",
				State: "failure",
				Error: "exit 1",
			}},
		}},
	}
	got := MergeDeclaredJobs(p, []string{"ci", "cd.office"})
	if got.Jobs[0].State != "failure" || got.Jobs[0].Children[0].Name != "check" {
		t.Fatalf("ci=%+v", got.Jobs[0])
	}
	if got.Jobs[1].Name != "cd.office" || got.Jobs[1].State != "skipped" {
		t.Fatalf("cd=%+v", got.Jobs[1])
	}
}

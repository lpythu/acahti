package pipeline

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFoldDropsCIWhenOfficeMatches(t *testing.T) {
	in := []fileMeta{
		{Name: ".acahti/pipelines/ci.yaml", Data: "when:\n  - event: [push, pull_request]\nsteps:\n  x:\n    image: bash\n    commands: [true]\n"},
		{Name: ".acahti/pipelines/cd.office.yaml", Data: "when:\n  - event: [push, manual]\n    branch: [dev]\ndepends_on:\n  - ci\nsteps:\n  deploy:\n    image: bash\n    commands: [true]\n"},
	}
	got := foldFinishJobs(in, "push", "dev", "refs/heads/dev")
	if len(got) != 1 || !strings.Contains(got[0].Name, "cd.office") {
		t.Fatalf("got %+v", got)
	}
	if strings.Contains(got[0].Data, "depends_on") || strings.Contains(got[0].Data, "\nci\n") {
		t.Fatalf("leftover ci dep:\n%s", got[0].Data)
	}
}

func TestFoldKeepsCIOnPullRequest(t *testing.T) {
	in := []fileMeta{
		{Name: "ci.yaml", Data: "when:\n  - event: [push, pull_request]\nsteps:\n  x:\n    image: bash\n    commands: [true]\n"},
		{Name: "cd.office.yaml", Data: "when:\n  - event: [push]\n    branch: [dev]\ndepends_on: [ci]\nsteps:\n  x:\n    image: bash\n    commands: [true]\n"},
	}
	got := foldFinishJobs(in, "pull_request", "dev", "refs/heads/dev")
	if len(got) != 2 {
		t.Fatalf("pr should keep both: %+v", names(got))
	}
}

func TestFoldKeepsCIOnTestPush(t *testing.T) {
	in := []fileMeta{
		{Name: "ci.yaml", Data: "when:\n  - event: [push]\nsteps:\n  x:\n    image: bash\n    commands: [true]\n"},
		{Name: "cd.office.yaml", Data: "when:\n  - event: [push]\n    branch: [dev]\nsteps:\n  x:\n    image: bash\n    commands: [true]\n"},
	}
	got := foldFinishJobs(in, "push", "test", "refs/heads/test")
	if len(got) != 2 {
		t.Fatalf("test push should keep ci: %+v", names(got))
	}
}

func TestFoldDropsCIOnHkTag(t *testing.T) {
	in := []fileMeta{
		{Name: "ci.yaml", Data: "when:\n  - event: [tag]\nsteps:\n  x:\n    image: bash\n    commands: [true]\n"},
		{Name: "cd.hk.yaml", Data: "when:\n  - event: [tag, manual]\ndepends_on:\n  - ci\nsteps:\n  x:\n    image: bash\n    commands: [true]\n"},
	}
	got := foldFinishJobs(in, "tag", "", "refs/tags/v1.2.3")
	if len(got) != 1 || jobBase(got[0].Name) != "cd.hk" {
		t.Fatalf("got %+v", names(got))
	}
	if strings.Contains(got[0].Data, "depends_on") {
		t.Fatalf("leftover depends_on:\n%s", got[0].Data)
	}
}

func TestFoldSkipsEmptyEvent(t *testing.T) {
	in := []fileMeta{
		{Name: "ci.yaml", Data: "steps:\n  x:\n    image: bash\n    commands: [true]\n"},
		{Name: "cd.office.yaml", Data: "when:\n  - event: [push]\n    branch: [dev]\nsteps:\n  x:\n    image: bash\n    commands: [true]\n"},
	}
	got := foldFinishJobs(in, "", "dev", "refs/heads/dev")
	if len(got) != 2 {
		t.Fatalf("empty event must not fold: %+v", names(got))
	}
}

func TestHandleConfigFoldsOfficePush(t *testing.T) {
	h := HandleConfig("secret", nil)
	body, _ := json.Marshal(configRequest{
		Configuration: []fileMeta{
			{Name: "ci.yaml", Data: "when:\n  - event: [push]\nsteps:\n  x:\n    image: bash\n    commands: [true]\n"},
			{Name: "cd.office.yaml", Data: "when:\n  - event: [push]\n    branch: [dev]\ndepends_on: [ci]\nsteps:\n  x:\n    image: bash\n    commands: [true]\n"},
		},
	})
	var req configRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	req.Pipeline.Event = "push"
	req.Pipeline.Branch = "dev"
	req.Pipeline.Ref = "refs/heads/dev"
	body, _ = json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/hooks/pipeline-config", strings.NewReader(string(body)))
	httpReq.SetBasicAuth("pipe", "secret")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httpReq)
	if rr.Code != http.StatusOK {
		t.Fatalf("%d %s", rr.Code, rr.Body.String())
	}
	var resp configResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Configs) != 1 || jobBase(resp.Configs[0].Name) != "cd.office" {
		t.Fatalf("resp=%+v", resp)
	}
	if strings.Contains(resp.Configs[0].Data, "depends_on") {
		t.Fatalf("leftover depends_on:\n%s", resp.Configs[0].Data)
	}
}

func names(in []fileMeta) []string {
	out := make([]string, 0, len(in))
	for _, f := range in {
		out = append(out, f.Name)
	}
	return out
}

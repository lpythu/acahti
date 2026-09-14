package woodpecker

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPipelineJobsFromKernelWorkflows(t *testing.T) {
	var p Pipeline
	if err := json.Unmarshal([]byte(`{"number":2,"status":"success","workflows":[{"name":"ci","state":"success"}]}`), &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Jobs) != 1 || p.Jobs[0].Name != "ci" {
		t.Fatalf("jobs=%v", p.Jobs)
	}
	out, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"jobs"`) || strings.Contains(string(out), `"workflows"`) {
		t.Fatalf("public json=%s", out)
	}
}

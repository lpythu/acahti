package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	"acahti/internal/httperr"
)

func TestToolSet(t *testing.T) {
	need := []string{
		"whoami",
		"repo_list", "repo_get", "repo_create",
		"branch_list", "ref_delete",
		"pr_create", "pr_list", "pr_get", "pr_comment", "pr_merge",
		"checks_wait", "pipeline_log", "pipeline_rerun",
		"pkg_publish", "pkg_list", "agent_status", "deploy_approve",
	}
	have := map[string]bool{}
	for _, n := range ToolNames() {
		have[n] = true
	}
	for _, n := range need {
		if !have[n] {
			t.Fatalf("missing tool %s", n)
		}
	}
}

func TestStructuredError(t *testing.T) {
	err := fail("failed_precondition", "checks not green")
	b, _ := json.Marshal(err.Data)
	var info httperr.Info
	if json.Unmarshal(b, &info) != nil {
		t.Fatal("data must be httperr.Info")
	}
	if info.Code != "failed_precondition" {
		t.Fatalf("code=%s", info.Code)
	}
	if !strings.Contains(info.Message, "checks") {
		t.Fatalf("message=%s", info.Message)
	}
}

func TestCodeForStatus(t *testing.T) {
	if httperr.CodeForStatus(404) != "not_found" {
		t.Fatal(httperr.CodeForStatus(404))
	}
}

func TestInitializeInstructions(t *testing.T) {
	s := &Server{Cfg: ConfigView{RootURL: "https://acahti.saidc.ai", Domain: "acahti.saidc.ai", Org: "acme"}}
	res, err := s.dispatch("", rpcReq{Method: "initialize"})
	if err != nil {
		t.Fatal(err)
	}
	m, _ := res.(map[string]any)
	inst, _ := m["instructions"].(string)
	if !strings.Contains(inst, "apply_when_remote_host") || !strings.Contains(inst, "acahti.saidc.ai") {
		t.Fatalf("instructions=%s", inst)
	}
}

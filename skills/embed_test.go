package skills

import (
	"strings"
	"testing"
)

func TestAcahtiFillsIslandURLs(t *testing.T) {
	s := Acahti("https://acahti.example.com", "acme", "acahti.example.com")
	if strings.Contains(s, "https://https://") {
		t.Fatalf("double scheme:\n%s", s)
	}
	for _, need := range []string{
		"Install https://acahti.example.com/skill.md",
		"Join    https://acahti.example.com/join",
		"https://acahti.example.com/mcp",
		"https://acahti.example.com/acme/<repo>.git",
		"team (Platform, ModelCamp",
		"pipeline_list",
		"secret_put",
		"Pipeline secrets",
		"Git tags are `vX.Y.Z`",
		"${CI_COMMIT_TAG#v}",
		"checks_wait",
		"git fetch --all",
		"Protection is per repository",
		"GET `https://acahti.example.com/skill.md` now",
		"whoami.git_name",
	} {
		if !strings.Contains(s, need) {
			t.Fatalf("missing %q in\n%s", need, s)
		}
	}
	if strings.Contains(s, "SSH optional") || strings.Contains(s, "ssh://git@") {
		t.Fatalf("must not offer git-over-ssh:\n%s", s)
	}
	if len(SHA("https://acahti.example.com", "acme", "acahti.example.com")) != 12 {
		t.Fatal("sha")
	}
}

func TestDemoFillsIslandURLs(t *testing.T) {
	s := Demo("https://acahti.example.com", "acme", "acahti.example.com")
	for _, need := range []string{
		"Install https://acahti.example.com/skill.md",
		"Install https://acahti.example.com/demo.md",
		"acahti-demo ok",
		"checks_wait",
		"pr_merge",
		"demo-<yyyymmdd>",
	} {
		if !strings.Contains(s, need) {
			t.Fatalf("missing %q in\n%s", need, s)
		}
	}
	if len(DemoSHA("https://acahti.example.com", "acme", "acahti.example.com")) != 12 {
		t.Fatal("demo sha")
	}
}

func TestAgentPrompt(t *testing.T) {
	p := AgentPrompt("https://acahti.example.com/", "acme")
	for _, need := range []string{
		"Install https://acahti.example.com/skill.md",
		"Install https://acahti.example.com/demo.md",
		"under acme",
		"I will only watch the Board",
	} {
		if !strings.Contains(p, need) {
			t.Fatalf("missing %q in\n%s", need, p)
		}
	}
	if strings.Contains(p, "https://https://") {
		t.Fatalf("double scheme:\n%s", p)
	}
}

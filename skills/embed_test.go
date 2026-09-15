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
		"`git_name` is the commit author",
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

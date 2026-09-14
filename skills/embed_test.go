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
		"ssh://git@acahti.example.com:2222/acme/<repo>.git",
	} {
		if !strings.Contains(s, need) {
			t.Fatalf("missing %q in\n%s", need, s)
		}
	}
}

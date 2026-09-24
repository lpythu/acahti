package identity

import (
	"reflect"
	"strings"
	"testing"
)

func TestEmailAndHosts(t *testing.T) {
	if Email("alice", "acahti.saidc.ai") != "alice@noreply.acahti.saidc.ai" {
		t.Fatalf("email=%s", Email("alice", "acahti.saidc.ai"))
	}
	if LoginFromNoreply("lipeiyang@noreply.acahti.saidc.ai", "acahti.saidc.ai") != "lipeiyang" {
		t.Fatal("noreply login")
	}
	if LoginFromNoreply("other@example.com", "acahti.saidc.ai") != "" {
		t.Fatal("foreign email")
	}
	if Name("alice", "") != "alice" || Name("  alice  ", "  ") != "alice" || Name("alice", "Ada") != "Ada" {
		t.Fatal("name")
	}
	got := Hosts("https://acahti.saidc.ai", "acahti.saidc.ai")
	if !reflect.DeepEqual(got, []string{"acahti.saidc.ai", "acahti.s-aidc.com"}) {
		t.Fatalf("hosts=%v", got)
	}
	got = Hosts("https://acahti.s-aidc.com", "acahti.s-aidc.com")
	if !reflect.DeepEqual(got, []string{"acahti.s-aidc.com", "acahti.saidc.ai"}) {
		t.Fatalf("current hosts=%v", got)
	}
	if Domain("https://acahti.example.com", "") != "acahti.example.com" {
		t.Fatalf("domain from url")
	}
}

func TestView(t *testing.T) {
	v := View("alice", "", "https://acahti.saidc.ai", "acahti.saidc.ai", "acme")
	if v["git_email"] != "alice@noreply.acahti.saidc.ai" {
		t.Fatalf("git_email=%v", v["git_email"])
	}
	if v["git_name"] != "alice" {
		t.Fatalf("git_name=%v", v["git_name"])
	}
	if View("alice", "Ada", "https://acahti.saidc.ai", "acahti.saidc.ai", "acme")["git_name"] != "Ada" {
		t.Fatal("git_name from author")
	}
	if v["clone_url_template"] != "https://acahti.saidc.ai/acme/<repo>.git" {
		t.Fatalf("clone=%v", v["clone_url_template"])
	}
	if v["skill_url"] != "https://acahti.saidc.ai/skill.md" {
		t.Fatalf("skill_url=%v", v["skill_url"])
	}
	sha, _ := v["skill_sha"].(string)
	if len(sha) != 12 {
		t.Fatalf("skill_sha=%v", sha)
	}
	cmds, _ := v["setup_local"].([]string)
	if len(cmds) != 2 || !strings.Contains(cmds[0], `user.name "alice"`) || !strings.Contains(cmds[1], "user.email") {
		t.Fatalf("setup_local=%v", cmds)
	}
}

func TestInstructions(t *testing.T) {
	s := Instructions("https://acahti.saidc.ai", "acahti.saidc.ai")
	for _, need := range []string{"whoami", "git_name", "apply_when_remote_host", "acahti.saidc.ai", "--local", "--global", "directory or repo name", "/skill.md", "actual intended push destination", "Preserve original authorship", "Leave non-Acahti destinations unchanged"} {
		if !strings.Contains(s, need) {
			t.Fatalf("missing %q in %s", need, s)
		}
	}
	if strings.Contains(s, "ANY remote") {
		t.Fatal("an unrelated Acahti remote must not change another destination's identity")
	}
}

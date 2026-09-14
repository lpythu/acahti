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
	if Name("alice", "") != "alice" || Name("alice", "Alice Chen") != "Alice Chen" {
		t.Fatal("name")
	}
	got := Hosts("https://acahti.saidc.ai", "acahti.saidc.ai")
	if !reflect.DeepEqual(got, []string{"acahti.saidc.ai"}) {
		t.Fatalf("hosts=%v", got)
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
	cmds, _ := v["setup_local"].([]string)
	if len(cmds) != 2 || !strings.Contains(cmds[0], "user.name") || !strings.Contains(cmds[1], "user.email") {
		t.Fatalf("setup_local=%v", cmds)
	}
}

func TestInstructions(t *testing.T) {
	s := Instructions("https://acahti.saidc.ai", "acahti.saidc.ai")
	for _, need := range []string{"whoami", "apply_when_remote_host", "acahti.saidc.ai", "--local", "--global", "github.com", "api-gateway"} {
		if !strings.Contains(s, need) {
			t.Fatalf("missing %q in %s", need, s)
		}
	}
}

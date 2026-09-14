package catalog

import (
	"encoding/json"
	"strings"
	"testing"

	"acahti/internal/config"
	"acahti/internal/forgejo"
	"acahti/internal/store"
	"acahti/internal/woodpecker"
)

func TestOwnersTeam(t *testing.T) {
	if !ownersTeam("Owners") || !ownersTeam("owners") {
		t.Fatal("Owners is reserved")
	}
	if ownersTeam("Platform") {
		t.Fatal("Platform is a code team")
	}
}

func TestValidTeamName(t *testing.T) {
	if err := ValidTeamName("Platform"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", "Owners", "1bad", "foo.bar", "has space"} {
		if ValidTeamName(name) == nil {
			t.Fatalf("accepted %q", name)
		}
	}
}

func TestParseRoleTeam(t *testing.T) {
	name, perm, ok := parseRoleTeam("Platform")
	if !ok || name != "Platform" || perm != permWrite {
		t.Fatalf("%s %s %v", name, perm, ok)
	}
	name, perm, ok = parseRoleTeam("Platform.read")
	if !ok || name != "Platform" || perm != permRead {
		t.Fatalf("%s %s %v", name, perm, ok)
	}
	name, perm, ok = parseRoleTeam("Platform.admin")
	if !ok || name != "Platform" || perm != permAdmin {
		t.Fatalf("%s %s %v", name, perm, ok)
	}
	if _, _, ok = parseRoleTeam("Owners"); ok {
		t.Fatal("owners")
	}
	if roleTeamName("Express", "admin") != "Express.admin" || roleTeamName("Express", "write") != "Express" {
		t.Fatal("roleTeamName")
	}
}

func TestMarkTeamVisible(t *testing.T) {
	repos := []forgejo.Repo{
		{FullName: "saidc/api-gateway"},
		{FullName: "saidc/ejp"},
	}
	got := markTeam(repos, "Platform", map[string]bool{"saidc/api-gateway": true})
	if len(got) != 1 || got[0].FullName != "saidc/api-gateway" || got[0].Team != "Platform" {
		t.Fatalf("%+v", got)
	}
}

func TestWriteID(t *testing.T) {
	t0 := teamRoles{roles: map[string]forgejo.Team{permRead: {ID: 1}, permWrite: {ID: 2}}}
	if t0.writeID() != 2 {
		t.Fatal(t0.writeID())
	}
}

func TestDecoratePipeNeedsDeclaredNames(t *testing.T) {
	c := New(config.Config{}, nil, nil, nil)
	p := c.decoratePipe(woodpecker.Pipeline{Status: "error", Error: "bad yaml", Repo: "saidc/demo"})
	if p.Error != "bad yaml" {
		t.Fatalf("%+v", p)
	}
	if len(p.Jobs) != 0 {
		t.Fatalf("jobs=%+v", p.Jobs)
	}
}

func TestStepFailedAndTailLog(t *testing.T) {
	if !stepFailed("failure") || !stepFailed("killed") || stepFailed("success") {
		t.Fatal("stepFailed")
	}
	text := "a\nb\nc\nd"
	if got := tailLog(text, 2); got != "c\nd" {
		t.Fatalf("tail=%q", got)
	}
	if got := tailLog(text, 0); got != text {
		t.Fatalf("all=%q", got)
	}
}

func TestAsRepoUpdated(t *testing.T) {
	c := New(config.Config{RootURL: "https://acahti.example.com", Org: "saidc"}, nil, nil, nil)
	got := c.asRepo(store.OrgRepo{FullName: "saidc/argos-pack", DefaultBranch: "dev", Updated: 1710000000}, "")
	if got.Name != "argos-pack" || got.Updated != 1710000000 || got.Team != "" {
		t.Fatalf("%+v", got)
	}
}

func TestPublicRepoCloneHTTPS(t *testing.T) {
	c := New(config.Config{RootURL: "https://acahti.example.com", Org: "acme"}, nil, nil, nil)
	got := c.PublicRepo(forgejo.Repo{FullName: "acme/demo", CloneURL: "http://forgejo:3000/acme/demo.git"})
	if got.CloneURL != "https://acahti.example.com/acme/demo.git" {
		t.Fatalf("clone_url=%q", got.CloneURL)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "ssh_url") || strings.Contains(string(raw), "ssh://") {
		t.Fatalf("ssh leaked: %s", raw)
	}
}

func TestAcahtiCheckURL(t *testing.T) {
	c := New(config.Config{RootURL: "https://acahti.example.com"}, nil, nil, nil)
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"https://acahti.example.com/ci", "https://acahti.example.com/pipelines"},
		{"https://acahti.example.com/ci/", "https://acahti.example.com/pipelines"},
		{"https://acahti.example.com/ci/acme/demo/12", "https://acahti.example.com/pipelines/acme/demo/12"},
		{"https://acahti.example.com/ci/repos/acme/demo/pipeline/12", "https://acahti.example.com/pipelines/acme/demo/12"},
		{"https://acahti.example.com/repos/acme/demo", "https://acahti.example.com/repos/acme/demo"},
	}
	for _, tc := range cases {
		if got := c.acahtiCheckURL(tc.in); got != tc.want {
			t.Fatalf("acahtiCheckURL(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

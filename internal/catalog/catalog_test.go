package catalog

import (
	"testing"

	"acahti/internal/config"
	"acahti/internal/forgejo"
)

func TestOwnersTeam(t *testing.T) {
	if !ownersTeam("Owners") || !ownersTeam("owners") {
		t.Fatal("Owners is reserved")
	}
	if ownersTeam("Platform") {
		t.Fatal("Platform is a code group")
	}
}

func TestValidGroupName(t *testing.T) {
	if err := ValidGroupName("Platform"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", "Owners", "1bad", "foo.bar", "has space"} {
		if ValidGroupName(name) == nil {
			t.Fatalf("accepted %q", name)
		}
	}
}

func TestParseRoleTeam(t *testing.T) {
	g, perm, ok := parseRoleTeam("Platform")
	if !ok || g != "Platform" || perm != permWrite {
		t.Fatalf("%s %s %v", g, perm, ok)
	}
	g, perm, ok = parseRoleTeam("Platform.read")
	if !ok || g != "Platform" || perm != permRead {
		t.Fatalf("%s %s %v", g, perm, ok)
	}
	g, perm, ok = parseRoleTeam("Platform.admin")
	if !ok || g != "Platform" || perm != permAdmin {
		t.Fatalf("%s %s %v", g, perm, ok)
	}
	if _, _, ok = parseRoleTeam("Owners"); ok {
		t.Fatal("owners")
	}
	if roleTeamName("Express", "admin") != "Express.admin" || roleTeamName("Express", "write") != "Express" {
		t.Fatal("roleTeamName")
	}
}

func TestMarkGroupVisible(t *testing.T) {
	repos := []forgejo.Repo{
		{FullName: "saidc/api-gateway"},
		{FullName: "saidc/ejp"},
	}
	got := markGroup(repos, "Platform", map[string]bool{"saidc/api-gateway": true})
	if len(got) != 1 || got[0].FullName != "saidc/api-gateway" || got[0].Group != "Platform" {
		t.Fatalf("%+v", got)
	}
}

func TestWriteID(t *testing.T) {
	g := groupTeams{teams: map[string]forgejo.Team{permRead: {ID: 1}, permWrite: {ID: 2}}}
	if g.writeID() != 2 {
		t.Fatal(g.writeID())
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

func TestAcahtiCheckURL(t *testing.T) {
	c := New(config.Config{RootURL: "https://acahti.example.com"}, nil, nil)
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

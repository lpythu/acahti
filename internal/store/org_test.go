package store

import "testing"

func TestStrongerRole(t *testing.T) {
	if strongerRole("read", "write") != "write" || strongerRole("admin", "write") != "admin" || strongerRole("", "read") != "read" {
		t.Fatal("strongerRole")
	}
}

func TestAppendUnassigned(t *testing.T) {
	teams := []NavTeam{{Team: "Platform", Repos: []OrgRepo{{FullName: "saidc/ejp"}}}}
	if got := appendUnassigned(teams, nil); len(got) != 1 || got[0].Team != "Platform" {
		t.Fatalf("%+v", got)
	}
	got := appendUnassigned(teams, []OrgRepo{{FullName: "saidc/argos-pack"}})
	if len(got) != 2 || got[1].Team != "" || len(got[1].Repos) != 1 || got[1].Repos[0].FullName != "saidc/argos-pack" {
		t.Fatalf("%+v", got)
	}
}

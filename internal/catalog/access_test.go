package catalog

import (
	"testing"

	"acahti/internal/config"
	"acahti/internal/store"
)

func TestMergeRepoACLDirectAndGranted(t *testing.T) {
	got := mergeRepoACL([]store.UserRepoACL{
		{Login: "lan", Repo: "saidc/dock-shell", Role: "read", Team: "Platform", Direct: false},
		{Login: "lan", Repo: "saidc/tm-web", Role: "write", Team: "", Direct: true},
		{Login: "lan", Repo: "saidc/saidc-ui", Role: "read", Team: "Web", Direct: false},
		{Login: "lan", Repo: "saidc/saidc-ui", Role: "admin", Team: "Web", Direct: true},
	})
	if len(got) != 3 {
		t.Fatalf("%+v", got)
	}
	if got[0].Repo != "saidc/dock-shell" || got[0].Permission != "read" || got[0].Team != "Platform" || got[0].Direct {
		t.Fatalf("%+v", got[0])
	}
	if got[1].Repo != "saidc/saidc-ui" || got[1].Permission != "admin" || got[1].Team != "Web" || !got[1].Direct {
		t.Fatalf("%+v", got[1])
	}
	if got[2].Repo != "saidc/tm-web" || got[2].Permission != "write" || got[2].Team != "" || !got[2].Direct {
		t.Fatalf("%+v", got[2])
	}
}

func TestMergeRepoACLUngrantedTeamInvisible(t *testing.T) {
	got := mergeRepoACL(nil)
	if len(got) != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestUserRepoAccessUnindexed(t *testing.T) {
	c := New(config.Config{}, nil, nil, nil)
	got, err := c.UserRepoAccess("lanjiashuo")
	if err != nil || len(got) != 0 {
		t.Fatalf("%+v %v", got, err)
	}
	many, err := c.UserRepoAccessMany([]string{"a", "b"})
	if err != nil || len(many["a"]) != 0 || len(many["b"]) != 0 {
		t.Fatalf("%+v %v", many, err)
	}
}

package catalog

import (
	"testing"

	"acahti/internal/config"
)

func TestRepoPermListMergesAndSorts(t *testing.T) {
	got := repoPermList(map[string]string{
		"saidc/tm-web":     "write",
		"saidc/dock-shell": "read",
		"saidc/saidc-ui":   "admin",
	})
	if len(got) != 3 {
		t.Fatalf("%+v", got)
	}
	if got[0].Repo != "saidc/dock-shell" || got[0].Permission != "read" {
		t.Fatalf("%+v", got[0])
	}
	if got[1].Repo != "saidc/saidc-ui" || got[1].Permission != "admin" {
		t.Fatalf("%+v", got[1])
	}
	if got[2].Repo != "saidc/tm-web" || got[2].Permission != "write" {
		t.Fatalf("%+v", got[2])
	}
}

func TestUserRepoAccessUnindexed(t *testing.T) {
	c := New(config.Config{}, nil, nil, nil)
	got, err := c.UserRepoAccess("lanjiashuo")
	if err != nil || len(got) != 0 {
		t.Fatalf("%+v %v", got, err)
	}
}

package catalog

import (
	"testing"

	"acahti/internal/store"
)

func TestForgejoTeamReposToStripUngranted(t *testing.T) {
	folder := map[string]store.TeamRepoLink{
		folderLinkKey("Platform", "saidc/ejp"):    {Team: "Platform", Repo: "saidc/ejp", Granted: false},
		folderLinkKey("Platform", "saidc/tm-web"): {Team: "Platform", Repo: "saidc/tm-web", Granted: true},
	}
	fj := []teamRepoHit{
		{team: "Platform", repo: "saidc/ejp", name: "ejp"},
		{team: "Platform", repo: "saidc/tm-web", name: "tm-web"},
		{team: "Platform", repo: "saidc/legacy", name: "legacy"},
	}
	got := forgejoTeamReposToStrip(folder, fj)
	if len(got) != 2 {
		t.Fatalf("%+v", got)
	}
	if got[0].name != "ejp" || got[1].name != "legacy" {
		t.Fatalf("%+v", got)
	}
	keep := grantedFolderLinks(folder)
	if len(keep) != 1 || keep[0].Repo != "saidc/tm-web" {
		t.Fatalf("%+v", keep)
	}
}

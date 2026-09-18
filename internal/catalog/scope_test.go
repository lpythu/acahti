package catalog

import (
	"testing"

	"acahti/internal/config"
	"acahti/internal/store"
)

func TestValidOrgName(t *testing.T) {
	if err := ValidOrgName("saidc"); err != nil {
		t.Fatal(err)
	}
	if err := ValidOrgName("Acme-1"); err != nil {
		t.Fatal(err)
	}
	if ValidOrgName("") == nil || ValidOrgName("-acme") == nil || ValidOrgName("acme_") == nil {
		t.Fatal("expected invalid")
	}
}

func TestForOrg(t *testing.T) {
	c := New(config.Config{Org: "saidc"}, nil, nil, nil)
	if c.ForOrg("saidc") != c || c.ForOrg("") != c {
		t.Fatal("same org should reuse catalog")
	}
	got := c.ForOrg("acme")
	if got == c || got.Cfg.Org != "acme" || c.Cfg.Org != "saidc" {
		t.Fatalf("for org=%s home=%s", got.Cfg.Org, c.Cfg.Org)
	}
}

func TestOrgRepos(t *testing.T) {
	c := New(config.Config{Org: "saidc"}, nil, nil, nil)
	got := c.orgRepos([]store.OrgRepo{
		{FullName: "saidc/ops"},
		{FullName: "acme/demo"},
		{FullName: "SAIDC/web"},
	})
	if len(got) != 2 || got[0].FullName != "saidc/ops" || got[1].FullName != "SAIDC/web" {
		t.Fatalf("%v", got)
	}
}

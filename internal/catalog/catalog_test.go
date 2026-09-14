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

func TestFlattenGroups(t *testing.T) {
	grouped := map[string][]forgejo.Repo{
		"Express":  {{FullName: "saidc/ejp", Group: "Express"}},
		"Platform": {{FullName: "saidc/docs", Group: "Platform"}, {FullName: "saidc/ops", Group: "Platform"}},
	}
	if n := len(flattenGroups(grouped, "Express")); n != 1 {
		t.Fatalf("express=%d", n)
	}
	all := flattenGroups(grouped, "")
	if len(all) != 3 || all[0].FullName != "saidc/docs" {
		t.Fatalf("all=%+v", all)
	}
	counts := groupCounts(grouped)
	if len(counts) != 2 || counts[0].Group != "Express" || counts[0].Count != 1 || counts[1].Count != 2 {
		t.Fatalf("%+v", counts)
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

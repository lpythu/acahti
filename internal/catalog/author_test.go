package catalog

import (
	"testing"

	"acahti/internal/config"
	"acahti/internal/forgejo"
	"acahti/internal/woodpecker"
)

func TestAuthorOfFallsBackToLogin(t *testing.T) {
	c := New(config.Config{}, nil, nil, nil)
	if c.authorOf("ada") != "ada" || c.authorOf("  ") != "" {
		t.Fatal(c.authorOf("ada"))
	}
}

func TestWithAuthorsUsesFullNameThenLogin(t *testing.T) {
	c := New(config.Config{}, nil, nil, nil)
	got := c.withAuthors([]AccessPerson{
		{Login: "ada", Author: "Ada Lovelace"},
		{Login: "bob"},
	})
	if got[0].Author != "Ada Lovelace" || got[1].Author != "bob" {
		t.Fatalf("%+v", got)
	}
}

func TestGitAuthorPrefersCommitName(t *testing.T) {
	cm := forgejo.Commit{Author: &forgejo.CommitUser{Login: "acahti", FullName: "acahti"}}
	cm.Commit.Author.Name = "兰佳硕"
	name, login, _, _ := gitAuthor(cm)
	if name != "兰佳硕" || login != "acahti" {
		t.Fatal(name, login)
	}
}

func TestGitAuthorFallsBackToForgejoUser(t *testing.T) {
	cm := forgejo.Commit{Author: &forgejo.CommitUser{Login: "lan", FullName: "兰佳硕"}}
	name, login, _, _ := gitAuthor(cm)
	if name != "兰佳硕" || login != "lan" {
		t.Fatal(name, login)
	}
}

func TestApplyCommitAuthorKeepsWoodpeckerWhenNoCommit(t *testing.T) {
	c := New(config.Config{}, nil, nil, nil)
	p := c.applyCommitAuthor(woodpecker.Pipeline{Author: "acahti"})
	if p.Author != "acahti" {
		t.Fatal(p.Author)
	}
}

func TestJobUserPrefersNoreplyOverForgeLogin(t *testing.T) {
	got := JobUser("acahti", "lipeiyang@noreply.acahti.saidc.ai", "李沛阳", "acahti", "acahti", "acahti.saidc.ai", map[string]string{
		"lipeiyang": "李沛阳",
	})
	if got != "lipeiyang" {
		t.Fatal(got)
	}
}

func TestJobUserMapsDisplayName(t *testing.T) {
	got := JobUser("acahti", "", "兰佳硕", "acahti", "acahti", "acahti.saidc.ai", map[string]string{
		"lan": "兰佳硕",
	})
	if got != "lan" {
		t.Fatal(got)
	}
}

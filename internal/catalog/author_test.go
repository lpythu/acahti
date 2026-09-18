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
	cm := forgejo.Commit{Author: &forgejo.CommitUser{Login: "acahti_bot", FullName: "acahti_bot"}}
	cm.Commit.Author.Name = "兰佳硕"
	name, login, _, _ := gitAuthor(cm)
	if name != "兰佳硕" || login != "acahti_bot" {
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
	p := c.applyCommitAuthor(woodpecker.Pipeline{Author: "acahti_bot"})
	if p.Author != "acahti_bot" {
		t.Fatal(p.Author)
	}
}

func TestJobUserPrefersNoreplyOverForgeLogin(t *testing.T) {
	got := JobUser("acahti_bot", "lipeiyang@noreply.acahti.saidc.ai", "李沛阳", "acahti_bot", "acahti_bot", "acahti.saidc.ai", map[string]string{
		"lipeiyang": "李沛阳",
	})
	if got != "lipeiyang" {
		t.Fatal(got)
	}
}

func TestJobUserMapsDisplayName(t *testing.T) {
	got := JobUser("acahti_bot", "", "兰佳硕", "acahti_bot", "acahti_bot", "acahti.saidc.ai", map[string]string{
		"lan": "兰佳硕",
	})
	if got != "lan" {
		t.Fatal(got)
	}
}

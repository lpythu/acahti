package catalog

import (
	"testing"
	"time"

	"acahti/internal/forgejo"
)

func TestMemoDropAndTTL(t *testing.T) {
	m := newMemo()
	m.setAdmin("ada", true)
	m.setRepos("ada", []forgejo.Repo{{FullName: "saidc/ejp"}})
	if ok, hit := m.adminOf("ada"); !hit || !ok {
		t.Fatal("admin")
	}
	if repos, hit := m.reposOf("ada"); !hit || len(repos) != 1 {
		t.Fatal("repos")
	}
	m.drop()
	if _, hit := m.adminOf("ada"); hit {
		t.Fatal("dropped")
	}
	m.setAdmin("ada", true)
	m.admin["ada"] = stamp[bool]{at: time.Now().Add(-2 * memoTTL), v: true}
	if _, hit := m.adminOf("ada"); hit {
		t.Fatal("ttl")
	}
}

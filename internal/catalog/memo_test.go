package catalog

import (
	"testing"
	"time"
)

func TestMemoDropAndTTL(t *testing.T) {
	m := newMemo()
	m.setAdmin("ada", true)
	if ok, hit := m.adminOf("ada"); !hit || !ok {
		t.Fatal("admin")
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

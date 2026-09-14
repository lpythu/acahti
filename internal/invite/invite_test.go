package invite

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInviteLifecycle(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if s.Valid("nope") {
		t.Fatal("empty store")
	}
	c, err := s.Create()
	if err != nil || c.Code == "" {
		t.Fatalf("create %v %v", c, err)
	}
	if !s.Valid(c.Code) {
		t.Fatal("valid")
	}
	if len(s.List()) != 1 {
		t.Fatal("list")
	}
	if err := s.Delete(c.Code); err != nil {
		t.Fatal(err)
	}
	if s.Valid(c.Code) {
		t.Fatal("deleted")
	}
}

func TestOpenPersists(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "invites.json")); err != nil {
		t.Fatal(err)
	}
	c, err := s.Create()
	if err != nil {
		t.Fatal(err)
	}
	s2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !s2.Valid(c.Code) {
		t.Fatal("reload")
	}
}

func TestOpenRequiresWritable(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	if _, err := Open(dir); err == nil {
		t.Fatal("expected unwritable")
	}
}

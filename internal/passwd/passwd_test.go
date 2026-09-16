package passwd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if s.Get("gaowenrong") != "" {
		t.Fatal("empty")
	}
	if err := s.Set("gaowenrong", "secret"); err != nil {
		t.Fatal(err)
	}
	if s.Get("gaowenrong") != "secret" {
		t.Fatal("get")
	}
	s2, err := Open(filepath.Dir(s.path))
	if err != nil {
		t.Fatal(err)
	}
	if s2.Get(" gaowenrong ") != "secret" {
		t.Fatal("reload")
	}
}

func TestNilStore(t *testing.T) {
	var s *Store
	if s.Get("x") != "" {
		t.Fatal("nil get")
	}
	if err := s.Set("x", "y"); err != nil {
		t.Fatal(err)
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

func TestRandom(t *testing.T) {
	a, err := Random()
	if err != nil || len(a) != 24 {
		t.Fatalf("random %q %v", a, err)
	}
	b, err := Random()
	if err != nil || a == b {
		t.Fatal("unique")
	}
}

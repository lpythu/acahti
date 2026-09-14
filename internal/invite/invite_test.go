package invite

import "testing"

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

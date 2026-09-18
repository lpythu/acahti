package auth

import (
	"testing"
	"time"
)

func TestIssueParse(t *testing.T) {
	s := New([]byte("secret"), "acahti_bot")
	tok := s.Issue("alice")
	user, ok := s.Parse(tok)
	if !ok || user != "alice" {
		t.Fatalf("got %q %v", user, ok)
	}
	if _, ok := s.Parse("nope"); ok {
		t.Fatal("bad token")
	}
	if !s.IsAdmin("acahti_bot") || s.IsAdmin("alice") {
		t.Fatal("admin")
	}
	if _, ok := s.Parse(s.IssueFor("alice", -time.Minute)); ok {
		t.Fatal("expired token accepted")
	}
}

package forgejo

import "testing"

func TestDBURL(t *testing.T) {
	got := dbURL("postgres://acahti:secret@postgres:5432/acahti?sslmode=disable")
	want := "postgres://acahti:secret@postgres:5432/forgejo?sslmode=disable"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if dbURL("") != "" || dbURL("not a url") != "" {
		t.Fatal("empty")
	}
}

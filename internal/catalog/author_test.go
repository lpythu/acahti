package catalog

import (
	"testing"

	"acahti/internal/config"
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

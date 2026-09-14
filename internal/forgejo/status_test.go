package forgejo

import (
	"testing"
	"time"
)

func TestLatestStatusesPrefersNewest(t *testing.T) {
	old := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	neu := old.Add(time.Hour)
	got := LatestStatuses([]Status{
		{Context: "ci", Status: "pending", CreatedAt: old},
		{Context: "ci", Status: "success", CreatedAt: neu},
		{Context: "cd", Status: "success", CreatedAt: neu},
	})
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	by := map[string]string{}
	for _, s := range got {
		by[s.Context] = s.Status
	}
	if by["ci"] != "success" || by["cd"] != "success" {
		t.Fatalf("%v", by)
	}
}

func TestLatestStatusesNewestFirstWhenNoTime(t *testing.T) {
	got := LatestStatuses([]Status{
		{Context: "ci", Status: "success"},
		{Context: "ci", Status: "pending"},
	})
	if len(got) != 1 || got[0].Status != "success" {
		t.Fatalf("%+v", got)
	}
}

func TestNormalizeRef(t *testing.T) {
	for _, in := range []string{"dev", "heads/dev", "refs/heads/dev"} {
		got, err := NormalizeRef(in)
		if err != nil || got != "heads/dev" {
			t.Fatalf("%s -> %s %v", in, got, err)
		}
	}
	if _, err := NormalizeRef("  "); err == nil {
		t.Fatal("empty")
	}
}

package catalog

import (
	"testing"
	"time"

	"acahti/internal/forgejo"
)

func TestHeatRangeMondayAligned(t *testing.T) {
	now := time.Date(2026, 9, 18, 15, 4, 0, 0, time.UTC)
	start, end := heatRange(now)
	if start.Weekday() != time.Monday {
		t.Fatalf("start weekday %s", start.Weekday())
	}
	if end.Sub(start) != heatWeeks*7*24*time.Hour {
		t.Fatalf("span %s", end.Sub(start))
	}
	if start.After(time.Date(2025, 9, 18, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("start too late %s", start)
	}
}

func TestFillHeatmap(t *testing.T) {
	start := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 14)
	now := time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)
	got := fillHeatmap(map[string]int64{"2026-09-07": 2, "2026-09-18": 4, "2026-09-20": 9}, start, end, now)
	if len(got.Days) != 14 {
		t.Fatalf("days %d", len(got.Days))
	}
	if got.Days[0].Date != "2026-09-07" || got.Days[0].Value != 2 {
		t.Fatalf("first %+v", got.Days[0])
	}
	if got.Total != 6 {
		t.Fatalf("total %d includes future", got.Total)
	}
}

func TestCommitHeat(t *testing.T) {
	cm := forgejo.Commit{Author: &forgejo.CommitUser{Login: "ada"}}
	if commitHeatLogin(cm, nil, nil) != "ada" {
		t.Fatal("login")
	}
	cm = forgejo.Commit{}
	cm.Commit.Author.Email = "Ada@Ex.com"
	cm.Commit.Author.Name = "Ada Lovelace"
	if commitHeatLogin(cm, map[string]string{"ada@ex.com": "ada"}, nil) != "ada" {
		t.Fatal("email")
	}
	cm.Commit.Author.Email = ""
	if commitHeatLogin(cm, nil, map[string]string{"ada lovelace": "ada"}) != "ada" {
		t.Fatal("name")
	}
	day, ok := commitHeatDay("2026-09-18T01:30:00+08:00")
	if !ok || day != "2026-09-18" {
		t.Fatalf("day %s %v", day, ok)
	}
}

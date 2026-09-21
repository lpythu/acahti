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

func TestHeatDate(t *testing.T) {
	now := time.Date(2026, 9, 21, 15, 0, 0, 0, heatZone)
	if _, err := heatDate("2026-09-21", now); err != nil {
		t.Fatal(err)
	}
	for _, date := range []string{"", "21-09-2026", "2026-09-22", "2024-01-01"} {
		if _, err := heatDate(date, now); err == nil {
			t.Fatalf("accepted %q", date)
		}
	}
}

func TestHeatCounts(t *testing.T) {
	// 2026-09-17 16:30 UTC is 2026-09-18 00:30 in UTC+8.
	got := heatCounts([]forgejo.HeatPoint{
		{Timestamp: time.Date(2026, 9, 17, 16, 30, 0, 0, time.UTC).Unix(), Contributions: 3},
		{Timestamp: time.Date(2026, 9, 17, 16, 45, 0, 0, time.UTC).Unix(), Contributions: 1},
		{Timestamp: 0, Contributions: 9},
	})
	if got["2026-09-18"] != 4 || len(got) != 1 {
		t.Fatalf("%v", got)
	}
}

package forgejo

import (
	"net/http"
	"net/http/httptest"
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
	if len(got) != 1 || got[0].Status != "pending" {
		t.Fatalf("%+v", got)
	}
}

func TestLatestRoundKeepsMaxPipeline(t *testing.T) {
	got := LatestRound([]Status{
		{Context: "ci/woodpecker/push/ci", Status: "failure", TargetURL: "http://localhost:8000/ci/repos/acme/demo/pipeline/7"},
		{Context: "ci/woodpecker/pr/ci", Status: "success", TargetURL: "http://localhost:8000/ci/repos/acme/demo/pipeline/12"},
	})
	if len(got) != 1 || got[0].Context != "ci/woodpecker/pr/ci" {
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
	got, err := NormalizeRef("feat/cd-smoke")
	if err != nil || got != "heads/feat/cd-smoke" {
		t.Fatalf("nested -> %s %v", got, err)
	}
	if _, err := NormalizeRef("  "); err == nil {
		t.Fatal("empty")
	}
}

func TestChecksGreenEmptyIsGreen(t *testing.T) {
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(hs.Close)
	ok, st, err := New(hs.URL, "t").ChecksGreen("acme", "demo", "abc")
	if err != nil || !ok || len(st) != 0 {
		t.Fatalf("ok=%v st=%+v err=%v", ok, st, err)
	}
}

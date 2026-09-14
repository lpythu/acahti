package catalog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"acahti/internal/config"
	"acahti/internal/woodpecker"
)

func TestLogFPChangesWhenLogGrows(t *testing.T) {
	a := logFP("Error: UPGRADE FAILED")
	b := logFP("Error: UPGRADE FAILED\nmore")
	if a == b {
		t.Fatal("fingerprint must change when log grows")
	}
	if logFP("Error: UPGRADE FAILED") != a {
		t.Fatal("same log must fingerprint the same")
	}
}

func TestBumpWatch(t *testing.T) {
	now := time.Date(2026, 9, 14, 23, 0, 0, 0, time.UTC)
	w, stale := bumpWatch(logWatch{}, "1:a", now, time.Minute)
	if stale || w.fp != "1:a" || !w.since.Equal(now) {
		t.Fatalf("start %+v stale=%v", w, stale)
	}
	later := now.Add(30 * time.Second)
	w, stale = bumpWatch(w, "1:a", later, time.Minute)
	if stale {
		t.Fatal("not stale yet")
	}
	w, stale = bumpWatch(w, "1:a", now.Add(time.Minute), time.Minute)
	if !stale {
		t.Fatal("should be stale")
	}
	w, stale = bumpWatch(w, "2:ab", now.Add(2*time.Minute), time.Minute)
	if stale || w.fp != "2:ab" {
		t.Fatalf("growth resets %+v stale=%v", w, stale)
	}
}

func TestRunningSteps(t *testing.T) {
	p := woodpecker.Pipeline{
		Status: "running",
		Jobs: []woodpecker.Job{{
			Name:  "cd.hk",
			State: "running",
			Children: []woodpecker.Step{
				{PID: 3, Name: "clone", State: "success"},
				{ID: 466, PID: 4, Name: "cd", State: "running"},
			},
		}, {
			Name:  "ci",
			State: "success",
			Children: []woodpecker.Step{
				{PID: 6, Name: "ci", State: "success"},
			},
		}},
	}
	got := runningSteps(p)
	if len(got) != 1 || got[0].Name != "cd" || stepID(got[0]) != 466 {
		t.Fatalf("%+v", got)
	}
}

func TestWatchKey(t *testing.T) {
	if watchKey("saidc/exweb", 9, 466) != "saidc/exweb#9/466" {
		t.Fatal(watchKey("saidc/exweb", 9, 466))
	}
}

func TestReapOneCancelsSilentStep(t *testing.T) {
	var canceled bool
	body := `{"number":9,"status":"running","workflows":[{"name":"cd.hk","state":"running","children":[{"id":466,"pid":4,"name":"cd","state":"running","type":"commands"}]}]}`
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/lookup/"):
			_, _ = w.Write([]byte(`{"id":1,"full_name":"saidc/exweb"}`))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/pipelines/9"):
			_, _ = w.Write([]byte(body))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/logs/"):
			_ = json.NewEncoder(w).Encode([]map[string]string{{"out": "Error: UPGRADE FAILED"}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/cancel"):
			canceled = true
			w.WriteHeader(http.StatusNoContent)
		case strings.HasSuffix(r.URL.Path, "/web-config.js"):
			_, _ = w.Write([]byte(`WOODPECKER_CSRF = "tok";`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(s.Close)
	c := New(config.Config{}, nil, woodpecker.New(s.URL, "t"), nil)
	p := woodpecker.Pipeline{Repo: "saidc/exweb", Number: 9, Status: "running"}
	now := time.Date(2026, 9, 14, 23, 0, 0, 0, time.UTC)
	prev := map[string]logWatch{"saidc/exweb#9/466": {fp: logFP("Error: UPGRADE FAILED"), since: now.Add(-staleAfter)}}
	keep := map[string]logWatch{}
	c.reapOne(p, now, prev, keep)
	if !canceled {
		t.Fatal("expected cancel")
	}
}

func TestReapOneKeepsFreshLog(t *testing.T) {
	var canceled bool
	body := `{"number":9,"status":"running","workflows":[{"name":"cd.hk","state":"running","children":[{"id":466,"pid":4,"name":"cd","state":"running","type":"commands"}]}]}`
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/lookup/"):
			_, _ = w.Write([]byte(`{"id":1,"full_name":"saidc/exweb"}`))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/pipelines/9"):
			_, _ = w.Write([]byte(body))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/logs/"):
			_ = json.NewEncoder(w).Encode([]map[string]string{{"out": "==> helm"}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/cancel"):
			canceled = true
		case strings.HasSuffix(r.URL.Path, "/web-config.js"):
			_, _ = w.Write([]byte(`WOODPECKER_CSRF = "tok";`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(s.Close)
	c := New(config.Config{}, nil, woodpecker.New(s.URL, "t"), nil)
	p := woodpecker.Pipeline{Repo: "saidc/exweb", Number: 9, Status: "running"}
	now := time.Date(2026, 9, 14, 23, 0, 0, 0, time.UTC)
	keep := map[string]logWatch{}
	c.reapOne(p, now, map[string]logWatch{}, keep)
	if canceled {
		t.Fatal("must not cancel a live step")
	}
	if keep["saidc/exweb#9/466"].fp == "" {
		t.Fatalf("watch %+v", keep)
	}
}

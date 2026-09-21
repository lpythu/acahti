package forgejo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"acahti/internal/page"
)

func TestListUserActivityFeeds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/users/lipeiyang/activities/feeds" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.Header.Get("Sudo") != "lipeiyang" {
			t.Fatalf("sudo=%s", r.Header.Get("Sudo"))
		}
		q := r.URL.Query()
		if q.Get("date") != "2026-09-21" || q.Get("only-performed-by") != "true" {
			t.Fatalf("query %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode([]Activity{{
			ID:      9,
			OpType:  "push",
			Content: "abcdef1234567890",
			Repo:    &Repo{FullName: "saidc/tm-web"},
		}})
	}))
	t.Cleanup(srv.Close)
	got, err := New(srv.URL, "admin").ListUserActivityFeeds("lipeiyang", "2026-09-21", page.FromInts(1, 20))
	if err != nil || len(got.Items) != 1 || got.Items[0].OpType != "push" {
		t.Fatalf("%+v %v", got, err)
	}
}

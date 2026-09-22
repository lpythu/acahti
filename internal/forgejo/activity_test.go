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
			OpType:  "commit_repo",
			Content: `{"Commits":[{"Sha1":"abcdef1234567890abcdef1234567890abcdef12"}]}`,
			Repo:    &Repo{FullName: "saidc/tm-web"},
			Comment: &ActivityComment{PullRequestURL: "https://git.example/saidc/tm-web/pulls/4"},
		}})
	}))
	t.Cleanup(srv.Close)
	got, err := New(srv.URL, "admin").ListUserActivityFeeds("lipeiyang", "2026-09-21", page.FromInts(1, 20))
	if err != nil || len(got.Items) != 1 || got.Items[0].OpType != "commit_repo" {
		t.Fatalf("%+v %v", got, err)
	}
	if got.Items[0].Comment == nil || got.Items[0].Comment.PullRequestURL == "" {
		t.Fatalf("comment %+v", got.Items[0].Comment)
	}
}

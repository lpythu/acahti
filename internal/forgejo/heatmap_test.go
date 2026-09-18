package forgejo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserHeatmap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/users/lipeiyang/heatmap" || r.Header.Get("Sudo") != "lipeiyang" {
			t.Fatalf("%s sudo=%s", r.URL.Path, r.Header.Get("Sudo"))
		}
		_ = json.NewEncoder(w).Encode([]HeatPoint{{Timestamp: 1789689600, Contributions: 4}})
	}))
	t.Cleanup(srv.Close)
	got, err := New(srv.URL, "admin").UserHeatmap("lipeiyang")
	if err != nil || len(got) != 1 || got[0].Contributions != 4 {
		t.Fatalf("%v %v", got, err)
	}
}

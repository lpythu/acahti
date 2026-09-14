package forgejo

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApiErrorMessage(t *testing.T) {
	err := apiError([]byte(`{"message":"source_id and login_name must be specified together","url":"https://acahti.saidc.ai/api/swagger"}`))
	if err.Error() != "source_id and login_name must be specified together" {
		t.Fatalf("%v", err)
	}
	if err := apiError([]byte(`not-json`)); err.Error() != "not-json" {
		t.Fatalf("%v", err)
	}
}

func TestSetPasswordSendsAuthSource(t *testing.T) {
	var patch map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/admin/users":
			_ = json.NewEncoder(w).Encode([]User{{Login: "gaowenrong", LoginName: "gaowenrong", SourceID: 0}})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/admin/users/gaowenrong":
			b, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(b, &patch); err != nil {
				t.Fatal(err)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("%s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	c := New(srv.URL, "admin")
	if err := c.SetPassword("gaowenrong", "secret"); err != nil {
		t.Fatal(err)
	}
	if patch["source_id"] != float64(0) || patch["login_name"] != "gaowenrong" {
		t.Fatalf("identity %v", patch)
	}
	if patch["password"] != "secret" || patch["must_change_password"] != false {
		t.Fatalf("password %v", patch)
	}
}

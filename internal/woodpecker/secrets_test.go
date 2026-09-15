package woodpecker

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"acahti/internal/page"
)

func TestSecretsStripValue(t *testing.T) {
	var posts []string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		posts = append(posts, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/web-config.js"):
			_, _ = w.Write([]byte(`WOODPECKER_CSRF = "tok";`))
		case r.URL.Path == "/api/orgs/lookup/saidc":
			_, _ = w.Write([]byte(`{"id":3,"name":"saidc"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/orgs/3/secrets":
			_, _ = w.Write([]byte(`[{"name":"harbor_password","value":"super-secret","events":["push","tag"]}]`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/orgs/3/secrets":
			raw, _ := io.ReadAll(r.Body)
			var body map[string]any
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatal(err)
			}
			if body["value"] != "new" {
				t.Fatalf("value=%v", body["value"])
			}
			if body["plugins_only"] != false {
				t.Fatalf("plugins_only=%v", body["plugins_only"])
			}
			_, _ = w.Write([]byte(`{"name":"kubeconfig_office","value":"new","events":["push","tag","manual"]}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/orgs/3/secrets/harbor_password":
			_, _ = w.Write([]byte(`{"name":"harbor_password","value":"x","events":["push"]}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/orgs/3/secrets/harbor_password":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(s.Close)
	c := New(s.URL, "t")
	listed, err := c.ListOrgSecrets("saidc", page.Query{Page: 1, Size: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Items) != 1 || listed.Items[0].Name != "harbor_password" {
		t.Fatalf("%+v", listed.Items)
	}
	raw, _ := json.Marshal(listed.Items[0])
	if strings.Contains(string(raw), "super-secret") || strings.Contains(string(raw), `"value"`) {
		t.Fatalf("value leaked: %s", raw)
	}

	created, err := c.PutOrgSecret("saidc", "kubeconfig_office", "new", nil)
	if err != nil {
		t.Fatal(err)
	}
	if created.Name != "kubeconfig_office" {
		t.Fatalf("%+v", created)
	}
	cr, _ := json.Marshal(created)
	if strings.Contains(string(cr), `"value"`) {
		t.Fatalf("put leaked value: %s", cr)
	}

	updated, err := c.PutOrgSecret("saidc", "harbor_password", "x", []string{"push"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "harbor_password" {
		t.Fatalf("%+v", updated)
	}

	if err := c.DeleteOrgSecret("saidc", "harbor_password"); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(posts, "\n")
	if !strings.Contains(joined, "PATCH /api/orgs/3/secrets/harbor_password") {
		t.Fatalf("posts=%v", posts)
	}
	if !strings.Contains(joined, "POST /api/orgs/3/secrets") {
		t.Fatalf("posts=%v", posts)
	}
}

func TestRepoSecrets(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/web-config.js"):
			_, _ = w.Write([]byte(`WOODPECKER_CSRF = "tok";`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/repos/9/secrets":
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/repos/9/secrets":
			_, _ = w.Write([]byte(`{"name":"svc_token","events":["push","tag","manual"]}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(s.Close)
	c := New(s.URL, "t")
	c.ids["saidc/tm-web"] = 9
	got, err := c.PutRepoSecret("saidc/tm-web", "svc_token", "abc", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "svc_token" {
		t.Fatalf("%+v", got)
	}
	raw, _ := json.Marshal(got)
	if strings.Contains(string(raw), "abc") {
		t.Fatalf("value leaked: %s", raw)
	}
}

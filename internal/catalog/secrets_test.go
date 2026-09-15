package catalog

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"acahti/internal/config"
	"acahti/internal/forgejo"
	"acahti/internal/page"
	"acahti/internal/woodpecker"
)

func TestNormalizeSecretName(t *testing.T) {
	got, err := NormalizeSecretName("Harbor_Password")
	if err != nil || got != "harbor_password" {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := NormalizeSecretName(""); err == nil {
		t.Fatal("empty")
	}
	if _, err := NormalizeSecretName("HAS SPACE"); err == nil {
		t.Fatal("space")
	}
}

func TestSecretsACL(t *testing.T) {
	wpHits := 0
	fj := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sudo := r.Header.Get("Sudo")
		switch {
		case r.URL.Path == "/api/v1/user":
			admin := sudo == "alice"
			_ = json.NewEncoder(w).Encode(map[string]any{"login": sudo, "is_admin": admin})
		case strings.HasPrefix(r.URL.Path, "/api/v1/repos/saidc/tm-web"):
			admin := sudo == "alice" || sudo == "repoadmin"
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name": "tm-web", "full_name": "saidc/tm-web",
				"permissions": map[string]bool{"admin": admin, "push": true, "pull": true},
			})
		case strings.HasPrefix(r.URL.Path, "/api/v1/orgs/saidc/teams"):
			_ = json.NewEncoder(w).Encode([]any{})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(fj.Close)
	wp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wpHits++
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/web-config.js"):
			_, _ = w.Write([]byte(`WOODPECKER_CSRF = "tok";`))
		case r.URL.Path == "/api/orgs/lookup/saidc":
			_, _ = w.Write([]byte(`{"id":1,"name":"saidc"}`))
		case r.URL.Path == "/api/orgs/1/secrets":
			_, _ = w.Write([]byte(`[{"name":"harbor_password","value":"nope","events":["push"]}]`))
		case strings.Contains(r.URL.Path, "/api/repos/") && strings.HasSuffix(r.URL.Path, "/secrets"):
			_, _ = w.Write([]byte(`[{"name":"svc_token","value":"nope"}]`))
		case r.URL.Path == "/api/repos/lookup/saidc/tm-web":
			_, _ = w.Write([]byte(`{"id":9,"full_name":"saidc/tm-web"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(wp.Close)

	c := New(config.Config{Org: "saidc", AdminUser: "alice"}, forgejo.New(fj.URL, "t"), woodpecker.New(wp.URL, "t"), nil)

	if _, err := c.ListOrgSecrets("bob", page.Query{Page: 1, Size: 20}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("member org list: %v", err)
	}
	listed, err := c.ListOrgSecrets("alice", page.Query{Page: 1, Size: 20})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(listed)
	if strings.Contains(string(raw), "nope") || strings.Contains(string(raw), `"value"`) {
		t.Fatalf("value leaked: %s", raw)
	}
	if len(listed.Items) != 1 || listed.Items[0].Name != "harbor_password" {
		t.Fatalf("%+v", listed.Items)
	}

	if _, err := c.ListRepoSecrets("bob", "saidc", "tm-web", page.Query{Page: 1, Size: 20}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("member repo list: %v", err)
	}
	repoListed, err := c.ListRepoSecrets("repoadmin", "saidc", "tm-web", page.Query{Page: 1, Size: 20})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(repoListed)
	if strings.Contains(string(raw), "nope") {
		t.Fatalf("value leaked: %s", raw)
	}

	if _, err := c.PutOrgSecret("bob", "harbor_password", "x", nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("member put: %v", err)
	}
	if wpHits == 0 {
		t.Fatal("admin list never hit kernel")
	}

	head, err := c.RepoHeader("bob", "saidc", "tm-web", "dev")
	if err != nil {
		t.Fatal(err)
	}
	if head.Repo.CanManageSecrets {
		t.Fatal("pusher must not see secrets")
	}
	head, err = c.RepoHeader("repoadmin", "saidc", "tm-web", "dev")
	if err != nil {
		t.Fatal(err)
	}
	if !head.Repo.CanManageSecrets {
		t.Fatal("repo admin can_manage_secrets")
	}
}

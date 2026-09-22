package forgejo

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestEnsureNoreplyFillsEmptyAuthorKeepsSetName(t *testing.T) {
	var patches []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/admin/users":
			_ = json.NewEncoder(w).Encode([]User{
				{Login: "empty", LoginName: "empty", Email: "old@example.com"},
				{Login: "ada", LoginName: "ada", FullName: "Ada", Email: "ada@noreply.acahti.saidc.ai"},
			})
		case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/api/v1/admin/users/"):
			b, _ := io.ReadAll(r.Body)
			var patch map[string]any
			if err := json.Unmarshal(b, &patch); err != nil {
				t.Fatal(err)
			}
			patch["_login"] = strings.TrimPrefix(r.URL.Path, "/api/v1/admin/users/")
			patches = append(patches, patch)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("%s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	c := New(srv.URL, "admin")
	if err := c.EnsureNoreply("https://acahti.saidc.ai", "acahti.saidc.ai"); err != nil {
		t.Fatal(err)
	}
	if len(patches) != 1 || patches[0]["_login"] != "empty" {
		t.Fatalf("patches %v", patches)
	}
	if patches[0]["full_name"] != "empty" || patches[0]["email"] != "empty@noreply.acahti.saidc.ai" {
		t.Fatalf("empty %v", patches[0])
	}
}

func TestMergePRSudoesActor(t *testing.T) {
	var sudo string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/repos/saidc/docs/pulls/3/merge" {
			t.Fatalf("%s %s", r.Method, r.URL.Path)
		}
		sudo = r.Header.Get("Sudo")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	c := New(srv.URL, "admin")
	if err := c.MergePR("saidc", "docs", 3, "lipeiyang"); err != nil {
		t.Fatal(err)
	}
	if sudo != "lipeiyang" {
		t.Fatalf("sudo=%q", sudo)
	}
}

func TestDecodeCommitReadsGitAuthor(t *testing.T) {
	cm, err := decodeCommit([]byte(`{"sha":"242402ea","message":"feat","author":{"name":"兰佳硕","email":"lan@noreply.acahti.saidc.ai"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cm.SHA != "242402ea" || cm.Commit.Author.Name != "兰佳硕" || cm.Commit.Message != "feat" {
		t.Fatalf("%+v", cm)
	}
}

func TestDecodeCommitKeepsNestedAuthor(t *testing.T) {
	cm, err := decodeCommit([]byte(`{"sha":"abc","commit":{"message":"feat","author":{"name":"兰佳硕"}},"author":{"login":"lan","full_name":"兰佳硕"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cm.Commit.Author.Name != "兰佳硕" || cm.Author == nil || cm.Author.Login != "lan" {
		t.Fatalf("%+v", cm)
	}
}

func TestCreateUserSendsAuthorAndPassword(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/admin/users" {
			t.Fatalf("%s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(User{Login: "linshiyuee", FullName: "林诗月"})
	}))
	t.Cleanup(srv.Close)
	u, err := New(srv.URL, "admin").CreateUser("linshiyuee", "linshiyuee@noreply.example", "initpw", "林诗月", false)
	if err != nil || u.Login != "linshiyuee" {
		t.Fatalf("%+v %v", u, err)
	}
	if body["username"] != "linshiyuee" || body["full_name"] != "林诗月" || body["password"] != "initpw" {
		t.Fatalf("%v", body)
	}
}

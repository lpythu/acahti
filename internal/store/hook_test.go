package store

import (
	"testing"
)

func TestParseWoodpecker(t *testing.T) {
	raw := []byte(`{"repo":{"full_name":"saidc/demo"},"pipeline":{"number":4,"status":"running","workflows":[{"name":"ci","state":"running"}]}}`)
	got, ok := ParseWoodpecker(raw)
	if !ok || got.Repo != "saidc/demo" || got.Number != 4 || got.Pipeline.Status != "running" {
		t.Fatalf("%+v %v", got, ok)
	}
	if len(got.Pipeline.Jobs) != 1 || got.Pipeline.Jobs[0].Name != "ci" {
		t.Fatalf("jobs=%+v", got.Pipeline.Jobs)
	}
}

func TestParseWoodpeckerOwnerName(t *testing.T) {
	raw := []byte(`{"repo":{"owner":"saidc","name":"demo"},"pipeline":{"number":1,"status":"success"}}`)
	got, ok := ParseWoodpecker(raw)
	if !ok || got.Repo != "saidc/demo" {
		t.Fatalf("%+v %v", got, ok)
	}
}

func TestParseForgejoRepoEvent(t *testing.T) {
	action, repo, ok := ParseForgejoRepoEvent(map[string]any{
		"action": "created",
		"repository": map[string]any{
			"full_name":      "saidc/demo",
			"default_branch": "dev",
			"description":    "x",
		},
	})
	if !ok || action != "created" || repo.FullName != "saidc/demo" || repo.DefaultBranch != "dev" {
		t.Fatalf("%s %+v %v", action, repo, ok)
	}
	action, repo, ok = ParseForgejoRepoEvent(map[string]any{
		"ref_type": "branch",
		"repository": map[string]any{
			"full_name":    "saidc/demo",
			"updated_unix": 1700000000.0,
		},
	})
	if !ok || action != "push" || repo.Updated != 1700000000 {
		t.Fatalf("branch touch %s %+v %v", action, repo, ok)
	}
	action, repo, ok = ParseForgejoRepoEvent(map[string]any{
		"ref":     "refs/heads/dev",
		"commits": []any{},
		"repository": map[string]any{
			"full_name":    "saidc/argos-pack",
			"updated_unix": 1710000000,
		},
	})
	if !ok || action != "push" || repo.FullName != "saidc/argos-pack" || repo.Updated != 1710000000 {
		t.Fatalf("push %s %+v %v", action, repo, ok)
	}
	action, repo, ok = ParseForgejoRepoEvent(map[string]any{
		"action":     "deleted",
		"repository": map[string]any{"full_name": "saidc/demo"},
	})
	if !ok || action != "deleted" || repo.FullName != "saidc/demo" {
		t.Fatalf("delete %s %+v %v", action, repo, ok)
	}
	action, repo, ok = ParseForgejoRepoEvent(map[string]any{
		"action":     "archived",
		"repository": map[string]any{"full_name": "saidc/old"},
	})
	if !ok || action != "archived" || !repo.Archived {
		t.Fatalf("archived %s %+v %v", action, repo, ok)
	}
	action, repo, ok = ParseForgejoRepoEvent(map[string]any{
		"action":     "unarchived",
		"repository": map[string]any{"full_name": "saidc/old", "archived": true},
	})
	if !ok || action != "unarchived" || repo.Archived {
		t.Fatalf("unarchived %s %+v %v", action, repo, ok)
	}
}

func TestParseForgejoStatus(t *testing.T) {
	cases := []struct {
		url    string
		number int64
	}{
		{"http://woodpecker:8000/ci/repos/saidc/demo/pipeline/12", 12},
		{"https://acahti.example/repos/saidc/tm-web/pipelines/147", 147},
		{"https://acahti.example/repos/20/148/pipelines/1", 148},
		{"/repos/20/148/pipelines/1", 148},
	}
	for _, tc := range cases {
		repo, n, ok := ParseForgejoStatus(map[string]any{
			"target_url": tc.url,
			"repository": map[string]any{"full_name": "saidc/tm-web"},
		})
		if !ok || repo != "saidc/tm-web" || n != tc.number {
			t.Fatalf("%s -> %s %d %v want %d", tc.url, repo, n, ok, tc.number)
		}
	}
}

func TestPipelineNumberFromURL(t *testing.T) {
	if got := pipelineNumberFromURL("https://x/repos/saidc/tm-web/pipelines/147"); got != 147 {
		t.Fatalf("named %d", got)
	}
	if got := pipelineNumberFromURL("/repos/20/148/pipelines/1"); got != 148 {
		t.Fatalf("kernel %d", got)
	}
	if got := pipelineNumberFromURL("/ci/repos/saidc/demo/pipeline/12"); got != 12 {
		t.Fatalf("ci %d", got)
	}
	if got := pipelineNumberFromURL(""); got != 0 {
		t.Fatalf("empty %d", got)
	}
}

func TestForgejoActor(t *testing.T) {
	if got := ForgejoActor(map[string]any{"sender": map[string]any{"login": "ada"}}); got != "ada" {
		t.Fatalf("sender %q", got)
	}
	if got := ForgejoActor(map[string]any{"pusher": map[string]any{"username": "bob"}}); got != "bob" {
		t.Fatalf("pusher %q", got)
	}
	if ForgejoActor(nil) != "" || ForgejoActor(map[string]any{}) != "" {
		t.Fatal("empty")
	}
}

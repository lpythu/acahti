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

func TestParseForgejoStatus(t *testing.T) {
	repo, n, ok := ParseForgejoStatus(map[string]any{
		"target_url": "http://woodpecker:8000/ci/repos/saidc/demo/pipeline/12",
		"repository": map[string]any{"full_name": "saidc/demo"},
	})
	if !ok || repo != "saidc/demo" || n != 12 {
		t.Fatalf("%s %d %v", repo, n, ok)
	}
}

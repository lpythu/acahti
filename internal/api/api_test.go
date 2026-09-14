package api

import (
	"net/http"
	"testing"
)

func TestRoute(t *testing.T) {
	cases := []struct {
		path, method, tool string
		extra              map[string]any
	}{
		{"/repos/acme/demo/pipelines", http.MethodGet, "pipeline_list", map[string]any{"owner": "acme", "name": "demo", "repo": "acme/demo"}},
		{"/repos/acme/demo/pipelines", http.MethodPost, "pipeline_trigger", map[string]any{"owner": "acme", "name": "demo", "repo": "acme/demo"}},
		{"/repos/acme/demo/pipelines/12", http.MethodGet, "pipeline_get", map[string]any{"owner": "acme", "name": "demo", "repo": "acme/demo", "number": int64(12)}},
		{"/repos/acme/demo/pipelines/12/log", http.MethodGet, "pipeline_log", map[string]any{"owner": "acme", "name": "demo", "repo": "acme/demo", "number": int64(12)}},
		{"/repos/acme/demo/pipelines/12/rerun", http.MethodPost, "pipeline_rerun", map[string]any{"owner": "acme", "name": "demo", "repo": "acme/demo", "number": int64(12)}},
		{"/repos/acme/demo/pipelines/12/cancel", http.MethodPost, "pipeline_cancel", map[string]any{"owner": "acme", "name": "demo", "repo": "acme/demo", "number": int64(12)}},
		{"/repos/acme/demo/pipelines/12/approve", http.MethodPost, "deploy_approve", map[string]any{"owner": "acme", "name": "demo", "repo": "acme/demo", "number": int64(12)}},
		{"/repos/acme/demo/pulls/3/comments", http.MethodGet, "pr_comments", map[string]any{"owner": "acme", "name": "demo", "number": 3}},
		{"/repos/acme/demo/pulls/3/comments", http.MethodPost, "pr_comment", map[string]any{"owner": "acme", "name": "demo", "number": 3}},
		{"/repos/acme/demo/checks/abc", http.MethodGet, "checks_wait", map[string]any{"owner": "acme", "name": "demo", "sha": "abc"}},
		{"/inbox", http.MethodGet, "inbox", map[string]any{}},
		{"/pipelines/acme/12/log", http.MethodGet, "", nil},
	}
	for _, tc := range cases {
		tool, extra := route(tc.path, tc.method, map[string]any{})
		if tool != tc.tool {
			t.Fatalf("%s %s tool=%q want %q", tc.method, tc.path, tool, tc.tool)
		}
		for k, want := range tc.extra {
			if extra[k] != want {
				t.Fatalf("%s %s extra[%s]=%v want %v", tc.method, tc.path, k, extra[k], want)
			}
		}
	}
}

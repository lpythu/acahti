package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWoodpeckerForgeOAuthIsInternal(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	body, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if strings.Contains(text, "WOODPECKER_EXPERT_FORGE_OAUTH_HOST") {
		t.Fatal("forge token refresh must use WOODPECKER_FORGEJO_URL; public gateway closes /login/oauth")
	}
	if !strings.Contains(text, "WOODPECKER_HOST: http://localhost:8000/ci") {
		t.Fatal("WOODPECKER_HOST must be the published loopback /ci (localhost, not a raw IP)")
	}
	if !strings.Contains(text, "WOODPECKER_FORGEJO_URL: http://forgejo:3000") {
		t.Fatal("missing compose-net Forgejo URL")
	}
	if !strings.Contains(text, "WOODPECKER_GRPC_SECRET:") {
		t.Fatal("WOODPECKER_GRPC_SECRET must be persisted; a random one drops every agent on recreate")
	}
	if !strings.Contains(text, `WOODPECKER_DEFAULT_PIPELINE_CONFIGS: ".acahti/pipelines/"`) {
		t.Fatal("server default pipeline path must be the .acahti/pipelines/ directory")
	}
}

package woodpecker

import (
	"strings"
	"testing"
)

func TestParseArgosLog(t *testing.T) {
	sid, url := ParseArgosLog("run out/x\nargos k7m2p9q1ab3c\ndash https://argos.saidc.ai/runs/k7m2p9q1ab3c\n")
	if sid != "k7m2p9q1ab3c" || url != "https://argos.saidc.ai/runs/k7m2p9q1ab3c" {
		t.Fatalf("sid=%q url=%q", sid, url)
	}
	sid, url = ParseArgosLog("dash  https://argos.saidc.ai/runs/abcd1234efgh\n")
	if sid != "abcd1234efgh" || !strings.Contains(url, "abcd1234efgh") {
		t.Fatalf("two-space dash sid=%q url=%q", sid, url)
	}
	sid, url = ParseArgosLog("argos xyzxyzxyzxyz\n")
	if sid != "xyzxyzxyzxyz" || url != "" {
		t.Fatalf("sid only: %q %q", sid, url)
	}
	if !E2EJob("e2e.office") || E2EJob("ci") || !E2EJob("e2e") {
		t.Fatal("E2EJob")
	}
}

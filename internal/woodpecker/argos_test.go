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
	sid, url = ParseArgosLog("argos 51p4z87i88ae\ndash ********/runs/51p4z87i88ae\n")
	if sid != "51p4z87i88ae" || url != "" {
		t.Fatalf("redacted dash: %q %q", sid, url)
	}
	if got := DashRunURL("https://argos.saidc.ai/", "51p4z87i88ae"); got != "https://argos.saidc.ai/runs/51p4z87i88ae" {
		t.Fatalf("DashRunURL %q", got)
	}
	if DashRunURL("", "51p4z87i88ae") != "" {
		t.Fatal("empty base")
	}
	if !E2EJob("e2e.office") || E2EJob("ci") || !E2EJob("e2e") {
		t.Fatal("E2EJob")
	}
	if ArgosJob(Job{Name: "ci"}) || !ArgosJob(Job{Name: "cd.office", Steps: []Step{{Name: "e2e"}}}) {
		t.Fatal("ArgosJob")
	}
}

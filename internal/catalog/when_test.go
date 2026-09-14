package catalog

import "testing"

func TestMatchesWhen(t *testing.T) {
	src := "when:\n  - event: push\n    branch: [dev, test]\n"
	if !matchesWhen(src, "push", "dev") {
		t.Fatal("dev push")
	}
	if matchesWhen(src, "push", "main") {
		t.Fatal("main")
	}
	if matchesWhen(src, "cron", "dev") {
		t.Fatal("cron")
	}
	if !matchesWhen("steps:\n  - echo\n", "push", "dev") {
		t.Fatal("no when")
	}
}

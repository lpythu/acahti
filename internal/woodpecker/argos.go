package woodpecker

import (
	"regexp"
	"strings"
)

var (
	argosSIDLine  = regexp.MustCompile(`(?m)^argos[ \t]+([0-9a-z]{12})[ \t]*$`)
	argosDashLine = regexp.MustCompile(`(?m)^dash[ \t]+(https?://\S+/runs/([0-9a-z]{12}|[0-9a-f]{8}-[0-9a-f-]{27,}))\s*$`)
	argosRedacted = regexp.MustCompile(`(?m)^dash[ \t]+\*+/runs/([0-9a-z]{12})\s*$`)
)

func E2EJob(name string) bool {
	name = strings.TrimSpace(name)
	return name == "e2e" || strings.HasPrefix(name, "e2e.")
}

func DashRunURL(base, sid string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	sid = strings.TrimSpace(sid)
	if base == "" || sid == "" {
		return ""
	}
	return base + "/runs/" + sid
}

func ParseArgosLog(text string) (sid, url string) {
	if m := argosDashLine.FindStringSubmatch(text); len(m) == 3 {
		return m[2], strings.TrimSpace(m[1])
	}
	if m := argosSIDLine.FindStringSubmatch(text); len(m) == 2 {
		return m[1], ""
	}
	if m := argosRedacted.FindStringSubmatch(text); len(m) == 2 {
		return m[1], ""
	}
	return "", ""
}

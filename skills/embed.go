package skills

import (
	_ "embed"
	"strings"
)

//go:embed acahti/SKILL.md
var raw string

func Acahti(rootURL, org, domain string) string {
	s := raw
	s = strings.ReplaceAll(s, "$ROOT_URL", rootURL)
	s = strings.ReplaceAll(s, "$ACAHTI_ORG", org)
	s = strings.ReplaceAll(s, "$DOMAIN", domain)
	return s
}

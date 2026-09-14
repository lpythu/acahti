package skills

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
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

func SHA(rootURL, org, domain string) string {
	sum := sha256.Sum256([]byte(Acahti(rootURL, org, domain)))
	return hex.EncodeToString(sum[:])[:12]
}

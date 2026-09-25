package skills

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"strings"
)

//go:embed acahti/SKILL.md
var raw string

//go:embed acahti-demo/SKILL.md
var rawDemo string

func Acahti(rootURL, org, domain string) string {
	return fill(raw, rootURL, org, domain)
}

func Demo(rootURL, org, domain string) string {
	return fill(rawDemo, rootURL, org, domain)
}

func SHA(rootURL, org, domain string) string {
	sum := sha256.Sum256([]byte(Acahti(rootURL, org, domain)))
	return hex.EncodeToString(sum[:])[:12]
}

func DemoSHA(rootURL, org, domain string) string {
	sum := sha256.Sum256([]byte(Demo(rootURL, org, domain)))
	return hex.EncodeToString(sum[:])[:12]
}

// AgentPrompt is the one-shot text a human pastes into a coding agent.
func AgentPrompt(rootURL, org string) string {
	root := strings.TrimRight(rootURL, "/")
	return strings.TrimSpace(`Install ` + root + `/skill.md
Install ` + root + `/demo.md

Run the Acahti demo loop on this instance now. Connect MCP with OAuth. Do not paste tokens. Do not change global git config. Create or reuse a disposable demo-* repo under ` + org + `, push a trivial branch with a smoke pipeline, wait for checks, open a PR, merge when green (or stop with evidence if no Runner). Reply with repo URL, PR number, and pipeline number. I will only watch the Board.`)
}

func fill(tmpl, rootURL, org, domain string) string {
	s := tmpl
	s = strings.ReplaceAll(s, "$ROOT_URL", rootURL)
	s = strings.ReplaceAll(s, "$ACAHTI_ORG", org)
	s = strings.ReplaceAll(s, "$DOMAIN", domain)
	return s
}

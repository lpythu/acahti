package catalog

import (
	"regexp"
	"strings"
)

var (
	whenHead  = regexp.MustCompile(`^when:\s*$`)
	whenInline = regexp.MustCompile(`^when:\s+\S`)
)

func yamlScalars(raw string) []string {
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, "["); i >= 0 {
		if j := strings.LastIndex(raw, "]"); j > i {
			raw = raw[i+1 : j]
		}
	}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		s := strings.TrimSpace(p)
		s = strings.Trim(s, `"'`)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func parseWhen(content string) [][2][]string {
	lines := strings.Split(content, "\n")
	start := -1
	for i, line := range lines {
		line = strings.TrimRight(line, "\r")
		if whenHead.MatchString(line) || whenInline.MatchString(line) {
			start = i
			break
		}
	}
	if start < 0 {
		return [][2][]string{{{}}}
	}
	var block []string
	for _, line := range lines[start+1:] {
		line = strings.TrimRight(line, "\r")
		if line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			break
		}
		block = append(block, line)
	}
	var rules [][2][]string
	var cur *[2][]string
	for _, line := range block {
		if strings.Contains(line, "- ") || strings.TrimSpace(line) == "-" {
			rules = append(rules, [2][]string{})
			cur = &rules[len(rules)-1]
		}
		if cur == nil {
			rules = append(rules, [2][]string{})
			cur = &rules[len(rules)-1]
		}
		trim := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))
		if ev, ok := strings.CutPrefix(trim, "event:"); ok {
			cur[0] = yamlScalars(ev)
		}
		if br, ok := strings.CutPrefix(trim, "branch:"); ok {
			cur[1] = yamlScalars(br)
		}
	}
	if len(rules) == 0 {
		return [][2][]string{{{}}}
	}
	return rules
}

func matchesWhen(content, event, branch string) bool {
	ev := strings.ToLower(event)
	br := strings.TrimPrefix(strings.TrimPrefix(branch, "refs/heads/"), "refs/tags/")
	for _, rule := range parseWhen(content) {
		events, branches := rule[0], rule[1]
		eventOK := len(events) == 0
		for _, e := range events {
			e = strings.ToLower(e)
			if e == ev || (ev == "release" && e == "tag") {
				eventOK = true
				break
			}
		}
		branchOK := len(branches) == 0
		for _, b := range branches {
			if b == br {
				branchOK = true
				break
			}
		}
		if eventOK && branchOK {
			return true
		}
	}
	return false
}

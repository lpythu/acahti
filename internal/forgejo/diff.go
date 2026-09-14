package forgejo

import (
	"strings"
)

func ParseUnifiedDiff(raw string) []CommitFile {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	chunks := strings.Split(raw, "\ndiff --git ")
	var out []CommitFile
	for i, chunk := range chunks {
		if i == 0 {
			if strings.HasPrefix(chunk, "diff --git ") {
				chunk = strings.TrimPrefix(chunk, "diff --git ")
			} else {
				continue
			}
		}
		f := parseDiffFile(chunk)
		if f.Filename != "" {
			out = append(out, f)
		}
	}
	return out
}

func parseDiffFile(chunk string) CommitFile {
	lines := strings.Split(chunk, "\n")
	f := CommitFile{Status: "changed"}
	var body []string
	inHunk := false
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "new file mode"):
			f.Status = "added"
		case strings.HasPrefix(line, "deleted file mode"):
			f.Status = "removed"
		case strings.HasPrefix(line, "rename from "):
			f.Status = "renamed"
			f.PreviousFilename = strings.TrimPrefix(line, "rename from ")
		case strings.HasPrefix(line, "rename to "):
			f.Filename = strings.TrimPrefix(line, "rename to ")
		case strings.HasPrefix(line, "--- "):
			p := strings.TrimPrefix(line, "--- ")
			if p == "/dev/null" {
				f.Status = "added"
			} else {
				p = strings.TrimPrefix(p, "a/")
				if f.PreviousFilename == "" && f.Status == "renamed" {
					f.PreviousFilename = p
				}
			}
		case strings.HasPrefix(line, "+++ "):
			p := strings.TrimPrefix(line, "+++ ")
			if p == "/dev/null" {
				f.Status = "removed"
			} else {
				f.Filename = strings.TrimPrefix(p, "b/")
			}
		case strings.HasPrefix(line, "Binary files "):
			body = append(body, line)
		case strings.HasPrefix(line, "@@"):
			inHunk = true
			body = append(body, line)
		default:
			if inHunk {
				body = append(body, line)
				if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
					f.Additions++
				}
				if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
					f.Deletions++
				}
			}
		}
	}
	if f.Filename == "" {
		// "a/foo b/bar" on first line
		if len(lines) > 0 {
			parts := strings.Fields(lines[0])
			if len(parts) >= 2 {
				f.Filename = strings.TrimPrefix(parts[len(parts)-1], "b/")
			}
		}
	}
	f.Patch = strings.Join(body, "\n")
	f.Changes = f.Additions + f.Deletions
	return f
}

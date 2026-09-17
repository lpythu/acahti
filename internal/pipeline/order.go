package pipeline

import (
	"path"
	"strings"

	"acahti/internal/order"

	"gopkg.in/yaml.v3"
)

func JobName(file string) string {
	base := path.Base(strings.TrimSpace(file))
	base = strings.TrimSuffix(base, ".yaml")
	base = strings.TrimSuffix(base, ".yml")
	return base
}

func DependsOn(data string) []string {
	var doc struct {
		DependsOn any `yaml:"depends_on"`
	}
	if err := yaml.Unmarshal([]byte(data), &doc); err != nil {
		return nil
	}
	return asNameList(doc.DependsOn)
}

func asNameList(v any) []string {
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return nil
		}
		return []string{s}
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			s, _ := item.(string)
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// orderByDepends puts parents before children so Woodpecker PIDs (and pills) run left to right.
func orderByDepends(files []fileMeta) []fileMeta {
	if len(files) < 2 {
		return files
	}
	names := make([]string, len(files))
	byName := map[string]fileMeta{}
	deps := map[string][]string{}
	for i, f := range files {
		n := JobName(f.Name)
		if n == "" {
			n = f.Name
		}
		for byName[n].Name != "" {
			n = n + "#"
		}
		names[i] = n
		byName[n] = f
		deps[n] = DependsOn(f.Data)
	}
	ordered := order.Names(names, deps)
	out := make([]fileMeta, 0, len(files))
	for _, n := range ordered {
		out = append(out, byName[n])
	}
	return out
}

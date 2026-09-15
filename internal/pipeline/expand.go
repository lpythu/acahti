package pipeline

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var pipeRef = regexp.MustCompile(`^([a-z][a-z0-9-]*)@v([0-9]+)$`)

var official = map[string]struct{}{
	"docker-login": {},
	"docker-build": {},
	"helm":         {},
	"wait-http":    {},
	"argos":        {},
	"npm-publish":  {},
	"pypi-publish": {},
	"oss-put":      {},
	"oci-gc":       {},
}

// Expand rewrites pipe: name@v1 steps into image: bash + acahti-pipe <name>.
func Expand(src []byte) ([]byte, error) {
	if len(strings.TrimSpace(string(src))) == 0 {
		return src, nil
	}
	var doc map[string]any
	if err := yaml.Unmarshal(src, &doc); err != nil {
		return nil, fmt.Errorf("pipeline yaml: %w", err)
	}
	if doc == nil {
		return src, nil
	}
	if _, ok := doc["uses"]; ok {
		return nil, fmt.Errorf("uses: is not supported; use pipe:")
	}
	if steps, ok := doc["steps"]; ok {
		if err := expandSteps(steps); err != nil {
			return nil, err
		}
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func expandSteps(steps any) error {
	switch s := steps.(type) {
	case map[string]any:
		for name, raw := range s {
			step, ok := raw.(map[string]any)
			if !ok {
				return fmt.Errorf("step %s: expected a mapping", name)
			}
			if err := expandStep(name, step); err != nil {
				return err
			}
		}
	case []any:
		for i, raw := range s {
			step, ok := raw.(map[string]any)
			if !ok {
				return fmt.Errorf("step %d: expected a mapping", i)
			}
			name := strconv.Itoa(i)
			if n, ok := step["name"].(string); ok && n != "" {
				name = n
			}
			if err := expandStep(name, step); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("steps: expected a mapping or list")
	}
	return nil
}

var bannedWith = map[string]struct{}{
	"env_file":      {},
	"token_file":    {},
	"password_file": {},
	"auth_file":     {},
	"kubeconfig":    {},
	"secrets":       {},
}

func stepEnv(step map[string]any) (map[string]any, error) {
	env := map[string]any{}
	if existing, ok := step["environment"]; ok && existing != nil {
		e, ok := existing.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("environment: expected a mapping")
		}
		for k, v := range e {
			env[k] = v
		}
	}
	return env, nil
}

func rewriteSecrets(name string, step map[string]any) error {
	raw, ok := step["secrets"]
	if !ok || raw == nil {
		return nil
	}
	switch s := raw.(type) {
	case []any:
		names := make([]any, 0, len(s))
		for i, v := range s {
			sec, ok := v.(string)
			if !ok || strings.TrimSpace(sec) == "" {
				return fmt.Errorf("step %s: secrets[%d] must be a name", name, i)
			}
			names = append(names, strings.TrimSpace(sec))
		}
		step["secrets"] = names
		return nil
	case map[string]any:
		env, err := stepEnv(step)
		if err != nil {
			return fmt.Errorf("step %s: %w", name, err)
		}
		for k, v := range s {
			sec, ok := v.(string)
			if !ok || strings.TrimSpace(sec) == "" {
				return fmt.Errorf("step %s: secrets.%s must be a secret name", name, k)
			}
			if _, exists := env[k]; exists {
				return fmt.Errorf("step %s: environment %s conflicts with secrets", name, k)
			}
			env[k] = map[string]any{"from_secret": strings.TrimSpace(sec)}
		}
		step["environment"] = env
		delete(step, "secrets")
		return nil
	default:
		return fmt.Errorf("step %s: secrets: expected a list or mapping", name)
	}
}

func expandStep(name string, step map[string]any) error {
	if _, ok := step["uses"]; ok {
		return fmt.Errorf("step %s: uses: is not supported; use pipe:", name)
	}
	if err := rewriteSecrets(name, step); err != nil {
		return err
	}
	raw, ok := step["pipe"]
	if !ok {
		return nil
	}
	ref, ok := raw.(string)
	if !ok || strings.TrimSpace(ref) == "" {
		return fmt.Errorf("step %s: pipe: must be name@v1", name)
	}
	m := pipeRef.FindStringSubmatch(strings.TrimSpace(ref))
	if m == nil {
		return fmt.Errorf("step %s: pipe: %q must be name@v1", name, ref)
	}
	pipeName, ver := m[1], m[2]
	if ver != "1" {
		return fmt.Errorf("step %s: pipe %s@v%s is not supported (only @v1)", name, pipeName, ver)
	}
	if _, ok := official[pipeName]; !ok {
		return fmt.Errorf("step %s: unknown pipe %s", name, pipeName)
	}
	if _, ok := step["commands"]; ok {
		return fmt.Errorf("step %s: pipe: cannot mix with commands:", name)
	}
	env, err := stepEnv(step)
	if err != nil {
		return fmt.Errorf("step %s: %w", name, err)
	}
	if withRaw, ok := step["with"]; ok && withRaw != nil {
		with, ok := withRaw.(map[string]any)
		if !ok {
			return fmt.Errorf("step %s: with: expected a mapping", name)
		}
		for k, v := range with {
			lk := strings.ToLower(strings.ReplaceAll(k, "-", "_"))
			if _, ban := bannedWith[lk]; ban {
				return fmt.Errorf("step %s: with.%s is not supported; use secrets", name, k)
			}
			key := "INPUT_" + strings.ToUpper(strings.ReplaceAll(k, "-", "_"))
			env[key] = stringify(v)
		}
	}
	step["image"] = "bash"
	step["commands"] = []any{"acahti-pipe " + pipeName}
	if len(env) > 0 {
		step["environment"] = env
	}
	delete(step, "pipe")
	delete(step, "with")
	return nil
}

func stringify(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case uint64:
		return strconv.FormatUint(t, 10)
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		b, err := yaml.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return strings.TrimSpace(string(b))
	}
}

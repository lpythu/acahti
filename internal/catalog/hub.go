package catalog

import (
	"acahti/internal/events"
	"acahti/internal/woodpecker"
)

func (c *Catalog) AllowHubEvent(user string) func(events.Event) bool {
	if c == nil {
		return nil
	}
	return func(ev events.Event) bool {
		switch ev.Type {
		case "catalog.updated", "forgejo":
			return true
		case "pipeline.updated":
			return c.canSeeIndexedRepo(user, hubEventRepo(ev.Data))
		default:
			return false
		}
	}
}

func hubEventRepo(data any) string {
	switch v := data.(type) {
	case woodpecker.Pipeline:
		return v.Repo
	case *woodpecker.Pipeline:
		if v == nil {
			return ""
		}
		return v.Repo
	case map[string]any:
		s, _ := v["repo"].(string)
		return s
	default:
		return ""
	}
}

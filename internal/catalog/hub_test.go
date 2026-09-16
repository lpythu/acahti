package catalog

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"acahti/internal/config"
	"acahti/internal/events"
	"acahti/internal/forgejo"
	"acahti/internal/woodpecker"
)

func TestHubEventRepo(t *testing.T) {
	if hubEventRepo(woodpecker.Pipeline{Repo: "saidc/demo"}) != "saidc/demo" {
		t.Fatal("struct")
	}
	p := &woodpecker.Pipeline{Repo: "saidc/demo"}
	if hubEventRepo(p) != "saidc/demo" {
		t.Fatal("ptr")
	}
	if hubEventRepo((*woodpecker.Pipeline)(nil)) != "" {
		t.Fatal("nil ptr")
	}
	if hubEventRepo(map[string]any{"repo": "saidc/demo"}) != "saidc/demo" {
		t.Fatal("map")
	}
	if hubEventRepo("x") != "" || hubEventRepo(nil) != "" {
		t.Fatal("other")
	}
}

func TestAllowHubEventNilCatalog(t *testing.T) {
	var c *Catalog
	if c.AllowHubEvent("bob") != nil {
		t.Fatal("nil catalog")
	}
}

func TestAllowHubEventUnindexedMember(t *testing.T) {
	c := New(config.Config{AdminUser: "alice"}, forgejo.New("http://127.0.0.1:9", "t"), nil, nil)
	allow := c.AllowHubEvent("bob")
	if !allow(events.Event{Type: "catalog.updated", Data: map[string]any{"ok": true}}) {
		t.Fatal("catalog.updated")
	}
	if !allow(events.Event{Type: "forgejo", Data: map[string]any{"ok": true}}) {
		t.Fatal("forgejo")
	}
	if allow(events.Event{Type: "pipeline.updated", Data: woodpecker.Pipeline{Repo: "saidc/secret", Number: 1}}) {
		t.Fatal("member must not see unindexed pipeline")
	}
	if allow(events.Event{Type: "pipeline.updated", Data: woodpecker.Pipeline{Number: 1}}) {
		t.Fatal("empty repo")
	}
	if allow(events.Event{Type: "mystery"}) {
		t.Fatal("unknown")
	}
}

func TestAllowHubEventOrgAdmin(t *testing.T) {
	fj := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(fj.Close)
	c := New(config.Config{Org: "saidc", AdminUser: "alice"}, forgejo.New(fj.URL, "t"), nil, nil)
	allow := c.AllowHubEvent("alice")
	if !allow(events.Event{Type: "pipeline.updated", Data: woodpecker.Pipeline{Repo: "saidc/secret", Number: 1}}) {
		t.Fatal("admin should see pipeline.updated")
	}
	if allow(events.Event{Type: "mystery"}) {
		t.Fatal("admin unknown")
	}
}

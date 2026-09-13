package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"acahti/internal/config"
	"acahti/internal/events"
	"acahti/internal/forgejo"
	"acahti/internal/woodpecker"
)

func TestMuxRegisters(t *testing.T) {
	h := New(config.Config{SessionSecret: "test", RootURL: "http://127.0.0.1", Org: "acme"},
		forgejo.New("http://127.0.0.1:9", ""),
		woodpecker.New("http://127.0.0.1:9", ""),
		events.New())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("healthz %d", rr.Code)
	}
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/login", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("login %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Acahti") {
		t.Fatalf("login page missing brand")
	}
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/repos/acme/demo", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("spa /repos %d", rr.Code)
	}
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("spa /admin %d", rr.Code)
	}
}

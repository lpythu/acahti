package main

import (
	"log"
	"net/http"
	"time"

	"acahti/internal/catalog"
	"acahti/internal/config"
	"acahti/internal/events"
	"acahti/internal/forgejo"
	"acahti/internal/server"
	"acahti/internal/store"
	"acahti/internal/woodpecker"
)

func main() {
	cfg := config.Load()
	fj := forgejo.New(cfg.ForgejoURL, cfg.AdminToken)
	if err := fj.EnsureNoreply(cfg.RootURL, cfg.Domain); err != nil {
		log.Printf("noreply migrate: %v", err)
	}
	wp := woodpecker.New(cfg.WoodpeckerURL, cfg.WoodpeckerTok)
	idx, err := store.Open(cfg.DatabaseURL)
	if err != nil {
		log.Printf("pipeline index: %v", err)
	}
	cat := catalog.New(cfg, fj, wp, idx)
	hub := events.New()
	h := server.New(cfg, cat, hub)
	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("acahti-gateway listen %s root=%s org=%s", cfg.Listen, cfg.RootURL, cfg.Org)
	log.Fatal(srv.ListenAndServe())
}

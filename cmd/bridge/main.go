package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/sessions"

	"oidc-bridge/internal/config"
	"oidc-bridge/internal/db"
	oidcbridge "oidc-bridge/internal/oidc"
	"oidc-bridge/internal/server"
	"oidc-bridge/internal/web"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	store, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatal(err)
	}
	defer store.DB.Close()

	var provider *oidcbridge.Provider
	if cfg.OIDCConfigured() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		provider, err = oidcbridge.NewProvider(ctx, cfg)
		if err != nil {
			log.Printf("oidc provider init skipped: %v", err)
		}
	}

	renderer, err := web.NewRenderer(".")
	if err != nil {
		log.Fatal(err)
	}

	sessionStore := sessions.NewCookieStore([]byte(cfg.SessionSecret))
	sessionStore.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((24 * time.Hour).Seconds()),
	}

	r := server.NewRouter(cfg, store, provider, renderer, sessionStore)
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("oidc bridge listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}

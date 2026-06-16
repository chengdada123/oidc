package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          int
	BaseURL       string
	SessionSecret string
	DBPath        string

	AdminPassword string

	VPS8Issuer       string
	VPS8ClientID     string
	VPS8ClientSecret string
	VPS8Scopes       []string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	port := 8080
	if v := strings.TrimSpace(os.Getenv("PORT")); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid PORT: %w", err)
		}
		port = parsed
	}

	cfg := &Config{
		Port:             port,
		BaseURL:          getEnv("BASE_URL", "http://localhost:8080"),
		SessionSecret:    getRequiredEnv("SESSION_SECRET"),
		DBPath:           getEnv("DB_PATH", "data/bridge.db"),
		AdminPassword:    getRequiredEnv("ADMIN_PASSWORD"),
		VPS8Issuer:       strings.TrimSpace(os.Getenv("VPS8_OIDC_ISSUER")),
		VPS8ClientID:     strings.TrimSpace(os.Getenv("VPS8_OIDC_CLIENT_ID")),
		VPS8ClientSecret: strings.TrimSpace(os.Getenv("VPS8_OIDC_CLIENT_SECRET")),
		VPS8Scopes:       splitScopes(getEnv("VPS8_OIDC_SCOPES", "openid email profile")),
	}

	if cfg.SessionSecret == "" {
		return nil, fmt.Errorf("SESSION_SECRET is required")
	}
	if cfg.AdminPassword == "" {
		return nil, fmt.Errorf("ADMIN_PASSWORD is required")
	}

	return cfg, nil
}

func (c *Config) OIDCConfigured() bool {
	return c.VPS8Issuer != "" && c.VPS8ClientID != "" && c.VPS8ClientSecret != ""
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getRequiredEnv(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func splitScopes(v string) []string {
	parts := strings.Fields(v)
	if len(parts) == 0 {
		return []string{"openid", "email", "profile"}
	}
	return parts
}

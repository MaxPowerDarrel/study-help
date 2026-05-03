package config

import (
	"log"
	"os"
)

type Config struct {
	Addr          string
	DatabaseURL   string
	SessionSecret string
	ESVAPIKey     string
}

func Load() Config {
	cfg := Config{
		Addr:          envOr("ADDR", ":8080"),
		DatabaseURL:   envOr("DATABASE_URL", "./sqlite.db"),
		SessionSecret: os.Getenv("SESSION_SECRET"),
		ESVAPIKey:     os.Getenv("ESV_API_KEY"),
	}
	if cfg.SessionSecret == "" {
		log.Fatal("SESSION_SECRET is required")
	}
	if cfg.ESVAPIKey == "" {
		log.Fatal("ESV_API_KEY is required")
	}
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

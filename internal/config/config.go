package config

import (
	"log"
	"os"
)

type Config struct {
	Addr             string
	DatabaseURL      string
	SessionSecret    string
	ESVAPIKey        string
	YouVersionAppKey string
	Env              string
}

func Load() Config {
	cfg := Config{
		Addr:             envOr("ADDR", ":8080"),
		DatabaseURL:      envOr("DATABASE_URL", "./sqlite.db"),
		SessionSecret:    os.Getenv("SESSION_SECRET"),
		ESVAPIKey:        os.Getenv("ESV_API_KEY"),
		YouVersionAppKey: os.Getenv("YOUVERSION_APP_KEY"),
		Env:              envOr("ENV", "prod"),
	}
	if cfg.SessionSecret == "" {
		log.Fatal("SESSION_SECRET is required")
	}
	if cfg.ESVAPIKey == "" {
		log.Fatal("ESV_API_KEY is required")
	}
	if cfg.YouVersionAppKey == "" {
		log.Fatal("YOUVERSION_APP_KEY is required")
	}
	return cfg
}

// IsDev reports whether the app is running in development mode (ENV=dev).
// Used to relax cookie attributes (e.g. drop Secure on plain-HTTP localhost).
func (c Config) IsDev() bool {
	return c.Env == "dev"
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

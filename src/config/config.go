package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ListenAddr  string
	Env         string
	QuoteTTL    time.Duration
	DatabaseURL string
	OMP         OMPConfig
}

type OMPConfig struct {
	BaseURL string
	Token   string
}

// LoadFromEnv reads configuration from environment variables with fallback defaults.
// It also loads `.env` if present (for local development).
func LoadFromEnv() *Config {
	// Load .env if exists, ignore error if no file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file loaded, relying on environment variables")
	}

	listenAddr := getEnv("LISTEN_ADDR", ":8080")
	env := getEnv("ENV", "dev")
	ttlStr := getEnv("QUOTE_TTL", "5m")
	databaseURL := os.Getenv("DATABASE_URL")
	log.Printf("DATABASE_URL: %s", databaseURL)
	if databaseURL == "" {
		log.Fatal("[FATAL] DATABASE_URL is required")
	}

	ttl, err := time.ParseDuration(ttlStr)
	if err != nil {
		log.Fatalf("[FATAL] Invalid QUOTE_TTL duration: %v", err)
	}

	return &Config{
		ListenAddr:  listenAddr,
		Env:         env,
		QuoteTTL:    ttl,
		DatabaseURL: databaseURL,
		OMP: OMPConfig{
			BaseURL: getEnv("OMP_BASE_URL", "https://api.ompfinex.com"),
			Token:   getEnv("OMP_TOKEN", ""),
		},
	}
}

// helper to get env with default fallback
func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

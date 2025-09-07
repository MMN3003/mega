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
	Wallex      WallexConfig
	Ethereum    EthereumConfig
}
type EthereumConfig struct {
	RPCURL                 string
	AdminKey               string
	TreasuryKey            string
	PhoenixContractAddress string
	USDTContractAddress    string
}
type OMPConfig struct {
	BaseURL string
	Token   string
}

type WallexConfig struct {
	BaseURL string
	APIKey  string
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
	sepoliaRPCURL := os.Getenv("SEPOLIA_RPC_URL")
	adminPrivateKey := os.Getenv("SEPOLIA_ADMIN_PRIVATE_KEY")
	contractAddress := os.Getenv("SEPOLIA_PHOENIX_CONTRACT_ADDRESS")
	usdtContractAddress := os.Getenv("SEPOLIA_USDT_CONTRACT_ADDRESS")
	treasuryKey := os.Getenv("SEPOLIA_TREASURY_PRIVATE_KEY")

	return &Config{
		ListenAddr:  listenAddr,
		Env:         env,
		QuoteTTL:    ttl,
		DatabaseURL: databaseURL,
		OMP: OMPConfig{
			BaseURL: getEnv("OMP_BASE_URL", "https://api.ompfinex.com"),
			Token:   getEnv("OMP_TOKEN", ""),
		},
		Wallex: WallexConfig{
			BaseURL: getEnv("WALLEX_BASE_URL", "https://api.wallex.ir"),
			APIKey:  getEnv("WALLEX_API_KEY", ""),
		},
		Ethereum: EthereumConfig{
			RPCURL:                 sepoliaRPCURL,
			AdminKey:               adminPrivateKey,
			TreasuryKey:            treasuryKey,
			PhoenixContractAddress: contractAddress,
			USDTContractAddress:    usdtContractAddress,
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

package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Load reads configuration from environment variables and .env.
func Load() (Config, error) {
	_ = godotenv.Load()

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		return Config{}, fmt.Errorf("DB_DSN is required")
	}

	return Config{DBDSN: dsn}, nil
}

// Config contains application configuration.
type Config struct {
	DBDSN string
}
package config

import "os"

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Wall     WallConfig
}

type ServerConfig struct {
	Port string
	Mode string
}

type DatabaseConfig struct {
	DSN string
}

type WallConfig struct {
	WebhookURL string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8081"),
			Mode: getEnv("GIN_MODE", "debug"),
		},
		Database: DatabaseConfig{
			DSN: getEnv("DATABASE_URL", "postgres://hydra:hydra_secret@localhost:5432/hydra_pay?sslmode=disable"),
		},
		Wall: WallConfig{
			WebhookURL: getEnv("WALL_WEBHOOK_URL", "http://localhost:8080/api/v1/webhooks/payment"),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

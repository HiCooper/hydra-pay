package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/hydra/pay-service/pkg/logger"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Wall     WallConfig
	Alipay   AlipayConfig
	Wechat   WechatConfig
}

type ServerConfig struct {
	Port            string
	Mode            string
	AdminAPIPath    string
	CORSOrigins     string
	MaxBodyBytes    int64
	ReadTimeout     string
	WriteTimeout    string
	IdleTimeout     string
	WebhookPoolSize int
}

type DatabaseConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime string // e.g. "5m"
	ConnMaxIdleTime string // e.g. "1m"
}

type WallConfig struct {
	WebhookURL    string
	WebhookSecret string // HMAC-SHA256 signing secret for webhook signatures
}

type AlipayConfig struct {
	AppID               string
	PrivateKey          string
	AlipayPublicKey     string
	NotifyURL           string
	ReturnURL           string
	GatewayHost         string
	IsSandbox           bool
	PrivateKeyPath      string
	AlipayPublicKeyPath string
	OnboardingNotifyURL string
}

type WechatConfig struct {
	MchID               string
	APIv3Key            string
	SerialNo            string
	PrivateKey          string
	PrivateKeyPath      string
	NotifyURL           string
	IsSandbox           bool
	OnboardingNotifyURL string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            getEnv("PORT", "8081"),
			Mode:            getEnv("GIN_MODE", "debug"),
			AdminAPIPath:    getEnv("ADMIN_API_PATH", "/api/admin"),
			CORSOrigins:     getEnv("CORS_ORIGINS", "http://localhost:5173,http://localhost:5174,http://localhost:5175"),
			MaxBodyBytes:    int64(getEnvInt("MAX_BODY_BYTES", 1<<20)),
			ReadTimeout:     getEnv("SERVER_READ_TIMEOUT", "30s"),
			WriteTimeout:    getEnv("SERVER_WRITE_TIMEOUT", "30s"),
			IdleTimeout:     getEnv("SERVER_IDLE_TIMEOUT", "120s"),
			WebhookPoolSize: getEnvInt("WEBHOOK_POOL_SIZE", 10),
		},
		Database: DatabaseConfig{
			DSN:             getEnv("DATABASE_URL", "postgres://hydra:hydra_secret@localhost:5432/hydra_pay?sslmode=disable"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getEnv("DB_CONN_MAX_LIFETIME", "5m"),
			ConnMaxIdleTime: getEnv("DB_CONN_MAX_IDLE_TIME", "1m"),
		},
		Wall: WallConfig{
			WebhookURL:    getEnv("WALL_WEBHOOK_URL", "http://localhost:8080/api/v1/webhooks/payment"),
			WebhookSecret: getEnv("WALL_WEBHOOK_SECRET", ""),
		},
		Alipay: AlipayConfig{
			AppID:               getEnv("ALIPAY_APP_ID", ""),
			PrivateKey:          resolveKey("ALIPAY_PRIVATE_KEY", "ALIPAY_PRIVATE_KEY_PATH"),
			AlipayPublicKey:     resolveKey("ALIPAY_ALIPAY_PUBLIC_KEY", "ALIPAY_ALIPAY_PUBLIC_KEY_PATH"),
			NotifyURL:           getEnv("ALIPAY_NOTIFY_URL", ""),
			ReturnURL:           getEnv("ALIPAY_RETURN_URL", ""),
			GatewayHost:         getEnv("ALIPAY_GATEWAY_HOST", "openapi.alipay.com"),
			IsSandbox:           getEnv("ALIPAY_SANDBOX", "false") == "true",
			PrivateKeyPath:      getEnv("ALIPAY_PRIVATE_KEY_PATH", ""),
			AlipayPublicKeyPath: getEnv("ALIPAY_ALIPAY_PUBLIC_KEY_PATH", ""),
			OnboardingNotifyURL: getEnv("ALIPAY_ONBOARDING_NOTIFY_URL", ""),
		},
		Wechat: WechatConfig{
			MchID:               getEnv("WECHAT_MCH_ID", ""),
			APIv3Key:            getEnv("WECHAT_API_V3_KEY", ""),
			SerialNo:            getEnv("WECHAT_SERIAL_NO", ""),
			PrivateKey:          resolveKey("WECHAT_PRIVATE_KEY", "WECHAT_PRIVATE_KEY_PATH"),
			PrivateKeyPath:      getEnv("WECHAT_PRIVATE_KEY_PATH", ""),
			NotifyURL:           getEnv("WECHAT_NOTIFY_URL", ""),
			IsSandbox:           getEnv("WECHAT_SANDBOX", "false") == "true",
			OnboardingNotifyURL: getEnv("WECHAT_ONBOARDING_NOTIFY_URL", ""),
		},
	}
}

// Validate checks critical configuration and logs warnings for potential misconfig.
// Returns a non-fatal error listing all issues found (for startup logging).
func (c *Config) Validate() error {
	var issues []string

	if c.Database.DSN == "" {
		issues = append(issues, "DATABASE_URL is not set")
	}
	if c.Database.DSN == "postgres://hydra:hydra_secret@localhost:5432/hydra_pay?sslmode=disable" &&
		c.Server.Mode == "release" {
		issues = append(issues, "DATABASE_URL appears to be the default dev value in release mode")
	}

	if c.Alipay.AppID != "" && c.Alipay.PrivateKey == "" {
		issues = append(issues, "ALIPAY_APP_ID is set but ALIPAY_PRIVATE_KEY is missing")
	}
	if c.Wechat.MchID != "" && c.Wechat.APIv3Key == "" {
		issues = append(issues, "WECHAT_MCH_ID is set but WECHAT_API_V3_KEY is missing")
	}

	if len(issues) > 0 {
		for _, issue := range issues {
			logger.Warn(context.Background(), "config validation", "issue", issue)
		}
		return fmt.Errorf("config validation: %s", strings.Join(issues, "; "))
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

// resolveKey reads the key from file if *_PATH is set, otherwise from env var.
func resolveKey(envKey, pathKey string) string {
	path := os.Getenv(pathKey)
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			logger.Warn(context.Background(), "failed to read key file", "path", path, "error", err)
			return ""
		}
		return string(data)
	}
	return os.Getenv(envKey)
}

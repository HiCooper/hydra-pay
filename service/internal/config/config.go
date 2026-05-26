package config

import (
	"log"
	"os"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Wall     WallConfig
	Alipay   AlipayConfig
	Wechat   WechatConfig
}

type ServerConfig struct {
	Port         string
	Mode         string
	AdminAPIPath string
	CORSOrigins  string
}

type DatabaseConfig struct {
	DSN string
}

type WallConfig struct {
	WebhookURL    string
	WebhookSecret string // HMAC-SHA256 signing secret for webhook signatures
}

type AlipayConfig struct {
	AppID              string
	PrivateKey         string
	AlipayPublicKey    string
	NotifyURL          string
	ReturnURL          string
	GatewayHost        string
	IsSandbox          bool
	PrivateKeyPath     string
	AlipayPublicKeyPath string
	OnboardingNotifyURL string
}

type WechatConfig struct {
	MchID          string
	APIv3Key       string
	SerialNo       string
	PrivateKey     string
	PrivateKeyPath string
	NotifyURL      string
	IsSandbox      bool
	OnboardingNotifyURL string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8081"),
			Mode:         getEnv("GIN_MODE", "debug"),
			AdminAPIPath: getEnv("ADMIN_API_PATH", "/api/admin"),
			CORSOrigins:  getEnv("CORS_ORIGINS", "http://localhost:5173,http://localhost:5174"),
		},
		Database: DatabaseConfig{
			DSN: getEnv("DATABASE_URL", "postgres://hydra:hydra_secret@localhost:5432/hydra_pay?sslmode=disable"),
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
			MchID:          getEnv("WECHAT_MCH_ID", ""),
			APIv3Key:       getEnv("WECHAT_API_V3_KEY", ""),
			SerialNo:       getEnv("WECHAT_SERIAL_NO", ""),
			PrivateKey:     resolveKey("WECHAT_PRIVATE_KEY", "WECHAT_PRIVATE_KEY_PATH"),
			PrivateKeyPath: getEnv("WECHAT_PRIVATE_KEY_PATH", ""),
			NotifyURL:      getEnv("WECHAT_NOTIFY_URL", ""),
			IsSandbox:            getEnv("WECHAT_SANDBOX", "false") == "true",
			OnboardingNotifyURL: getEnv("WECHAT_ONBOARDING_NOTIFY_URL", ""),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// resolveKey reads the key from file if *_PATH is set, otherwise from env var.
func resolveKey(envKey, pathKey string) string {
	path := os.Getenv(pathKey)
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("[config] WARNING: failed to read key file %s: %v", path, err)
			return ""
		}
		return string(data)
	}
	return os.Getenv(envKey)
}
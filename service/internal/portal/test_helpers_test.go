package portal

import (
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/config"
	"github.com/hydra/pay-service/internal/model"
)

// openTestDB creates an in-memory SQLite database with tables for portal tests.
// Uses SQLite-compatible DDL since gen_random_uuid() is PostgreSQL-specific.
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	ddl := []string{
		// Merchant
		`CREATE TABLE IF NOT EXISTS merchants (
			id TEXT PRIMARY KEY, name TEXT NOT NULL, email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL, contact_name TEXT, contact_phone TEXT,
			status TEXT DEFAULT 'active', alipay_p_id TEXT, wechat_sub_mchid TEXT,
			wechat_sub_appid TEXT, created_at DATETIME, updated_at DATETIME
		)`,
		// App
		`CREATE TABLE IF NOT EXISTS apps (
			id TEXT PRIMARY KEY, merchant_id TEXT NOT NULL, name TEXT NOT NULL,
			api_key TEXT NOT NULL UNIQUE, status TEXT DEFAULT 'active',
			webhook_url TEXT, webhook_secret TEXT, created_at DATETIME, updated_at DATETIME
		)`,
		// Payment (matches all GORM model fields)
		`CREATE TABLE IF NOT EXISTS payments (
			id TEXT PRIMARY KEY, trade_no TEXT NOT NULL UNIQUE, app_id TEXT NOT NULL,
			user_id TEXT NOT NULL, plan_id TEXT, amount INTEGER NOT NULL DEFAULT 0,
			currency TEXT DEFAULT 'CNY', channel TEXT, trade_type TEXT,
			status TEXT DEFAULT 'pending', external_id TEXT, payment_url TEXT,
			qr_code_url TEXT, description TEXT,
			success_url TEXT, cancel_url TEXT, open_id TEXT, channel_app_id TEXT,
			sub_merchant_id TEXT, sub_channel_app_id TEXT, client_ip TEXT,
			notify_url TEXT, metadata TEXT, paid_at DATETIME,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		// CheckoutSession
		`CREATE TABLE IF NOT EXISTS checkout_sessions (
			id TEXT PRIMARY KEY, app_id TEXT NOT NULL, amount INTEGER NOT NULL,
			currency TEXT DEFAULT 'CNY', description TEXT,
			status TEXT DEFAULT 'open', success_url TEXT, cancel_url TEXT,
			metadata TEXT, payment_id TEXT,
			expires_at DATETIME, created_at DATETIME, updated_at DATETIME
		)`,
		// MerchantOnboarding
		`CREATE TABLE IF NOT EXISTS merchant_onboardings (
			id TEXT PRIMARY KEY, merchant_id TEXT NOT NULL, channel TEXT NOT NULL,
			out_request_no TEXT UNIQUE, applyment_id TEXT,
			status TEXT DEFAULT 'pending', sub_merchant_id TEXT,
			sign_url TEXT, qr_code_url TEXT, request_data TEXT,
			response_data TEXT, callback_data TEXT, error_message TEXT,
			created_at DATETIME, updated_at DATETIME
		)`,
		// Subscription
		`CREATE TABLE IF NOT EXISTS subscriptions (
			id TEXT PRIMARY KEY, app_id TEXT NOT NULL, user_id TEXT NOT NULL,
			plan_id TEXT NOT NULL, status TEXT DEFAULT 'active',
			current_period_start DATETIME NOT NULL, current_period_end DATETIME NOT NULL,
			cancelled_at DATETIME, created_at DATETIME, updated_at DATETIME
		)`,
	}
	for _, s := range ddl {
		if err := db.Exec(s).Error; err != nil {
			t.Fatalf("failed to create table: %v\nSQL: %s", err, s)
		}
	}
	return db
}

func seedMerchant(t *testing.T, db *gorm.DB) *model.Merchant {
	t.Helper()
	m := &model.Merchant{
		ID:    uuid.New(),
		Name:  "Test Merchant",
		Email: "test@test.com",
	}
	m.SetPassword("password")
	db.Create(m)
	return m
}

func seedApp(t *testing.T, db *gorm.DB, merchantID uuid.UUID, name string) *model.App {
	t.Helper()
	app := &model.App{
		ID:         uuid.New(),
		MerchantID: merchantID,
		Name:       name,
		APIKey:     model.GenerateAPIKey(),
		Status:     "active",
	}
	db.Create(app)
	return app
}

func newTestHandler(db *gorm.DB) *Handler {
	return NewHandler(db, &config.Config{})
}

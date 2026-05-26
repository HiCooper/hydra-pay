package database

import (
	"context"
	"time"

	"github.com/hydra/pay-service/internal/config"
	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{
		PrepareStmt: true,
		Logger:      gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&model.Payment{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.App{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.PaymentEvent{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.AlipayCallback{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.WeChatCallback{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.Refund{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.ScheduledTask{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.CheckoutSession{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.IdempotencyRecord{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.SubscriptionPlan{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.Subscription{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.MerchantOnboarding{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.Merchant{}); err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)

	if lifetime, err := time.ParseDuration(cfg.Database.ConnMaxLifetime); err == nil {
		sqlDB.SetConnMaxLifetime(lifetime)
	}
	if idleTime, err := time.ParseDuration(cfg.Database.ConnMaxIdleTime); err == nil {
		sqlDB.SetConnMaxIdleTime(idleTime)
	}

	logger.Info(context.Background(), "database connected and migrated",
		"max_open_conns", cfg.Database.MaxOpenConns,
		"max_idle_conns", cfg.Database.MaxIdleConns,
	)
	return db, nil
}

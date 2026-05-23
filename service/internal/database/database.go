package database

import (
	"log"

	"github.com/hydra/pay-service/internal/config"
	"github.com/hydra/pay-service/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{
		PrepareStmt: true,
		Logger:      logger.Default.LogMode(logger.Warn),
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

	log.Println("[db] connected and migrated")
	return db, nil
}

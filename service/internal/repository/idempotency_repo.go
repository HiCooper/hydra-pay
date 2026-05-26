package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/model"
)

type IdempotencyRepository struct {
	db *gorm.DB
}

func NewIdempotencyRepository(db *gorm.DB) *IdempotencyRepository {
	return &IdempotencyRepository{db: db}
}

// FindByKey returns a non-expired idempotency record for the given app and key.
func (r *IdempotencyRepository) FindByKey(appID uuid.UUID, key string) (*model.IdempotencyRecord, error) {
	var record model.IdempotencyRecord
	err := r.db.Where("app_id = ? AND idempotency_key = ? AND expires_at > ?",
		appID, key, time.Now()).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// Create stores a new idempotency record with a 24-hour expiry.
func (r *IdempotencyRepository) Create(record *model.IdempotencyRecord) error {
	if record.ExpiresAt.IsZero() {
		record.ExpiresAt = time.Now().Add(24 * time.Hour)
	}
	return r.db.Create(record).Error
}

// DeleteExpired removes records past their expiry. Called periodically.
func (r *IdempotencyRepository) DeleteExpired() error {
	return r.db.Where("expires_at <= ?", time.Now()).Delete(&model.IdempotencyRecord{}).Error
}

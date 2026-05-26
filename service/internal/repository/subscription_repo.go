package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/model"
)

type SubscriptionRepository struct {
	db *gorm.DB
}

func NewSubscriptionRepository(db *gorm.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) Create(sub *model.Subscription) error {
	return r.db.Create(sub).Error
}

func (r *SubscriptionRepository) GetByID(id uuid.UUID) (*model.Subscription, error) {
	var sub model.Subscription
	err := r.db.Where("id = ?", id).First(&sub).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *SubscriptionRepository) ListByApp(appID uuid.UUID, page, pageSize int) ([]model.Subscription, int64, error) {
	var subs []model.Subscription
	var total int64

	query := r.db.Model(&model.Subscription{}).Where("app_id = ?", appID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&subs).Error
	return subs, total, err
}

func (r *SubscriptionRepository) ListByUser(appID uuid.UUID, userID string) ([]model.Subscription, error) {
	var subs []model.Subscription
	err := r.db.Where("app_id = ? AND user_id = ?", appID, userID).
		Order("created_at DESC").Find(&subs).Error
	return subs, err
}

func (r *SubscriptionRepository) Cancel(id uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&model.Subscription{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":       model.SubscriptionStatusCancelled,
		"cancelled_at": now,
	}).Error
}

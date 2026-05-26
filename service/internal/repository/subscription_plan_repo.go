package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/model"
)

type SubscriptionPlanRepository struct {
	db *gorm.DB
}

func NewSubscriptionPlanRepository(db *gorm.DB) *SubscriptionPlanRepository {
	return &SubscriptionPlanRepository{db: db}
}

func (r *SubscriptionPlanRepository) Create(plan *model.SubscriptionPlan) error {
	return r.db.Create(plan).Error
}

func (r *SubscriptionPlanRepository) GetByID(id uuid.UUID) (*model.SubscriptionPlan, error) {
	var plan model.SubscriptionPlan
	err := r.db.Where("id = ?", id).First(&plan).Error
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func (r *SubscriptionPlanRepository) ListActive() ([]model.SubscriptionPlan, error) {
	var plans []model.SubscriptionPlan
	err := r.db.Where("status = ?", model.PlanStatusActive).Order("created_at DESC").Find(&plans).Error
	return plans, err
}

func (r *SubscriptionPlanRepository) ListAll() ([]model.SubscriptionPlan, error) {
	var plans []model.SubscriptionPlan
	err := r.db.Order("created_at DESC").Find(&plans).Error
	return plans, err
}

func (r *SubscriptionPlanRepository) Update(id uuid.UUID, updates map[string]interface{}) error {
	return r.db.Model(&model.SubscriptionPlan{}).Where("id = ?", id).Updates(updates).Error
}

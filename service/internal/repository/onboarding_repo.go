package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/model"
)

type OnboardingRepository struct {
	db *gorm.DB
}

func NewOnboardingRepository(db *gorm.DB) *OnboardingRepository {
	return &OnboardingRepository{db: db}
}

func (r *OnboardingRepository) Create(ob *model.MerchantOnboarding) error {
	return r.db.Create(ob).Error
}

func (r *OnboardingRepository) GetByID(id uuid.UUID) (*model.MerchantOnboarding, error) {
	var ob model.MerchantOnboarding
	if err := r.db.Where("id = ?", id).First(&ob).Error; err != nil {
		return nil, err
	}
	return &ob, nil
}

func (r *OnboardingRepository) GetByMerchantID(merchantID uuid.UUID) ([]model.MerchantOnboarding, error) {
	var obs []model.MerchantOnboarding
	err := r.db.Where("merchant_id = ?", merchantID).Order("created_at DESC").Find(&obs).Error
	return obs, err
}

func (r *OnboardingRepository) GetByApplymentID(channel, applymentID string) (*model.MerchantOnboarding, error) {
	var ob model.MerchantOnboarding
	err := r.db.Where("channel = ? AND applyment_id = ?", channel, applymentID).First(&ob).Error
	if err != nil {
		return nil, err
	}
	return &ob, nil
}

func (r *OnboardingRepository) GetByOutRequestNo(outRequestNo string) (*model.MerchantOnboarding, error) {
	var ob model.MerchantOnboarding
	err := r.db.Where("out_request_no = ?", outRequestNo).First(&ob).Error
	if err != nil {
		return nil, err
	}
	return &ob, nil
}

func (r *OnboardingRepository) UpdateStatus(id uuid.UUID, status string) error {
	return r.db.Model(&model.MerchantOnboarding{}).Where("id = ?", id).Update("status", status).Error
}

func (r *OnboardingRepository) MarkApproved(id uuid.UUID, subMerchantID string) error {
	return r.db.Model(&model.MerchantOnboarding{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":          model.OnboardingStatusApproved,
		"sub_merchant_id": subMerchantID,
	}).Error
}

func (r *OnboardingRepository) MarkRejected(id uuid.UUID, reason string) error {
	return r.db.Model(&model.MerchantOnboarding{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":        model.OnboardingStatusRejected,
		"error_message": reason,
	}).Error
}

func (r *OnboardingRepository) UpdateSignURL(id uuid.UUID, signURL, qrCodeURL string) error {
	return r.db.Model(&model.MerchantOnboarding{}).Where("id = ?", id).Updates(map[string]interface{}{
		"sign_url":    signURL,
		"qr_code_url": qrCodeURL,
	}).Error
}

func (r *OnboardingRepository) List(appID, channel, status string, page, pageSize int) ([]model.MerchantOnboarding, int64, error) {
	var obs []model.MerchantOnboarding
	var total int64

	query := r.db.Model(&model.MerchantOnboarding{})
	if appID != "" {
		query = query.Where("merchant_id = ?", appID)
	}
	if channel != "" {
		query = query.Where("channel = ?", channel)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&obs).Error
	if err != nil {
		return nil, 0, err
	}

	return obs, total, nil
}

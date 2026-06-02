package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/model"
)

type CheckoutSessionRepository struct {
	db *gorm.DB
}

func NewCheckoutSessionRepository(db *gorm.DB) *CheckoutSessionRepository {
	return &CheckoutSessionRepository{db: db}
}

func (r *CheckoutSessionRepository) Create(s *model.CheckoutSession) error {
	return r.db.Create(s).Error
}

func (r *CheckoutSessionRepository) GetByID(id uuid.UUID) (*model.CheckoutSession, error) {
	var s model.CheckoutSession
	err := r.db.Where("id = ?", id).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// LinkPayment associates a payment with the session without changing its status.
// The session stays "open" until the payment callback confirms success.
func (r *CheckoutSessionRepository) LinkPayment(id uuid.UUID, paymentID uuid.UUID) error {
	return r.db.Model(&model.CheckoutSession{}).Where("id = ?", id).Update("payment_id", paymentID).Error
}

func (r *CheckoutSessionRepository) MarkCompleted(id uuid.UUID, paymentID uuid.UUID) error {
	return r.db.Model(&model.CheckoutSession{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     model.CheckoutSessionCompleted,
		"payment_id": paymentID,
	}).Error
}

// CompleteByPaymentID marks all sessions linked to the given payment as completed.
func (r *CheckoutSessionRepository) CompleteByPaymentID(paymentID uuid.UUID) error {
	return r.db.Model(&model.CheckoutSession{}).
		Where("payment_id = ? AND status = ?", paymentID, model.CheckoutSessionOpen).
		Update("status", model.CheckoutSessionCompleted).Error
}

func (r *CheckoutSessionRepository) ListByAppID(appID uuid.UUID) ([]model.CheckoutSession, error) {
	var sessions []model.CheckoutSession
	err := r.db.Where("app_id = ?", appID).Order("created_at DESC").Find(&sessions).Error
	return sessions, err
}

func (r *CheckoutSessionRepository) Expire(id uuid.UUID) error {
	return r.db.Model(&model.CheckoutSession{}).Where("id = ?", id).Update("status", model.CheckoutSessionExpired).Error
}

func (r *CheckoutSessionRepository) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&model.CheckoutSession{}).Error
}

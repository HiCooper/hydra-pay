package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/model"
)

type PaymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Create(payment *model.Payment) error {
	return r.db.Create(payment).Error
}

func (r *PaymentRepository) GetByID(id uuid.UUID) (*model.Payment, error) {
	var payment model.Payment
	err := r.db.Where("id = ?", id).First(&payment).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

func (r *PaymentRepository) GetByTradeNo(tradeNo string) (*model.Payment, error) {
	var payment model.Payment
	err := r.db.Where("trade_no = ?", tradeNo).First(&payment).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

func (r *PaymentRepository) ListByApp(appID uuid.UUID, page, pageSize int) ([]model.Payment, int64, error) {
	var payments []model.Payment
	var total int64

	query := r.db.Model(&model.Payment{})
	if appID != uuid.Nil {
		query = query.Where("app_id = ?", appID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&payments).Error
	if err != nil {
		return nil, 0, err
	}

	return payments, total, nil
}

func (r *PaymentRepository) UpdateStatus(id uuid.UUID, status, externalID string) error {
	updates := map[string]interface{}{
		"status":      status,
		"external_id": externalID,
	}
	return r.db.Model(&model.Payment{}).Where("id = ?", id).Updates(updates).Error
}

func (r *PaymentRepository) MarkPaid(id uuid.UUID, externalID string) error {
	return r.db.Model(&model.Payment{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      model.PaymentStatusPaid,
			"external_id": externalID,
			"paid_at":     gorm.Expr("NOW()"),
		}).Error
}

func (r *PaymentRepository) UpdateChannel(id uuid.UUID, channel string) error {
	return r.db.Model(&model.Payment{}).Where("id = ?", id).Update("channel", channel).Error
}

func (r *PaymentRepository) UpdateChannelURLs(id uuid.UUID, paymentURL, qrCodeURL string) error {
	return r.db.Model(&model.Payment{}).Where("id = ?", id).Updates(map[string]interface{}{
		"payment_url": paymentURL,
		"qr_code_url": qrCodeURL,
	}).Error
}

// MarkPaidIfPending atomically marks payment as paid only if currently pending or processing.
// Returns true if the update was applied, false if already in a terminal state.
func (r *PaymentRepository) MarkPaidIfPending(id uuid.UUID, externalID string) (bool, error) {
	result := r.db.Model(&model.Payment{}).
		Where("id = ? AND status IN ?", id, []string{model.PaymentStatusPending, model.PaymentStatusProcessing}).
		Updates(map[string]interface{}{
			"status":      model.PaymentStatusPaid,
			"external_id": externalID,
			"paid_at":     gorm.Expr("NOW()"),
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}
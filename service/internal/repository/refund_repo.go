package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/model"
)

type RefundRepository struct {
	db *gorm.DB
}

func NewRefundRepository(db *gorm.DB) *RefundRepository {
	return &RefundRepository{db: db}
}

func (r *RefundRepository) Create(refund *model.Refund) error {
	return r.db.Create(refund).Error
}

func (r *RefundRepository) GetByID(id uuid.UUID) (*model.Refund, error) {
	var refund model.Refund
	if err := r.db.Where("id = ?", id).First(&refund).Error; err != nil {
		return nil, err
	}
	return &refund, nil
}

func (r *RefundRepository) GetByOutRequestNo(outReqNo string) (*model.Refund, error) {
	var refund model.Refund
	if err := r.db.Where("out_request_no = ?", outReqNo).First(&refund).Error; err != nil {
		return nil, err
	}
	return &refund, nil
}

func (r *RefundRepository) ListByPayment(paymentID uuid.UUID) ([]model.Refund, error) {
	var refunds []model.Refund
	if err := r.db.Where("payment_id = ?", paymentID).Order("created_at DESC").Find(&refunds).Error; err != nil {
		return nil, err
	}
	return refunds, nil
}

func (r *RefundRepository) UpdateStatus(id uuid.UUID, status, channelRefundID string, refundFee int64, errorMsg string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if channelRefundID != "" {
		updates["channel_refund_id"] = channelRefundID
	}
	if refundFee > 0 {
		updates["refund_fee"] = refundFee
	}
	if errorMsg != "" {
		updates["error_msg"] = errorMsg
	}
	return r.db.Model(&model.Refund{}).Where("id = ?", id).Updates(updates).Error
}

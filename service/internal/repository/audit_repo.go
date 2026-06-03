package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/pkg/audit"
)

// AuditRepository persists audit entries to the database.
type AuditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) SaveAuditEntry(_ context.Context, e *audit.Entry) error {
	id, err := uuid.Parse(e.ID)
	if err != nil {
		return err
	}
	m := &model.AuditLog{
		ID:       id,
		Action:   e.Action,
		Actor:    e.Actor,
		Target:   e.Target,
		TargetID: e.TargetID,
		OldValue: e.OldValue,
		NewValue: e.NewValue,
		TraceID:  e.TraceID,
		Result:   e.Result,
		Error:    e.Error,
	}
	return r.db.Create(m).Error
}

package model

import (
	"time"

	"github.com/google/uuid"
)

// AuditLog is the persisted record of a security-relevant or business-significant action.
// It is append-only — rows are never updated or deleted by application code.
type AuditLog struct {
	ID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Action   string    `gorm:"type:varchar(100);not null;index"`
	Actor    string    `gorm:"type:varchar(100);not null;index"`
	Target   string    `gorm:"type:varchar(100);not null;index"`
	TargetID string    `gorm:"type:varchar(64);not null;index"`
	OldValue string    `gorm:"type:text"`
	NewValue string    `gorm:"type:text"`
	TraceID  string    `gorm:"type:varchar(64)"`
	Result   string    `gorm:"type:varchar(20);not null;default:'success'"`
	Error    string    `gorm:"type:text"`

	CreatedAt time.Time `gorm:"not null;index"`
}

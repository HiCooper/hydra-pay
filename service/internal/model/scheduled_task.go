package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	TaskTypeOrderTimeout = "order_timeout"
	TaskStatusPending    = "pending"
	TaskStatusDone       = "done"
	TaskStatusCancelled  = "cancelled"
)

// ScheduledTask is a lightweight delayed task — used for order timeout checks.
type ScheduledTask struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TaskType    string    `gorm:"type:varchar(64);not null;index"`
	ReferenceID uuid.UUID `gorm:"type:uuid;not null;index"` // payment_id
	ExecuteAt   time.Time `gorm:"not null;index"`
	Status      string    `gorm:"type:varchar(32);default:pending;index"` // pending / done / cancelled
	CreatedAt   time.Time
}
package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/model"
)

type ScheduledTaskRepository struct {
	db *gorm.DB
}

func NewScheduledTaskRepository(db *gorm.DB) *ScheduledTaskRepository {
	return &ScheduledTaskRepository{db: db}
}

func (r *ScheduledTaskRepository) Create(task *model.ScheduledTask) error {
	return r.db.Create(task).Error
}

// CancelByReference marks all pending tasks for a payment as cancelled.
func (r *ScheduledTaskRepository) CancelByReference(paymentID uuid.UUID) {
	r.db.Model(&model.ScheduledTask{}).
		Where("reference_id = ? AND status = ?", paymentID, model.TaskStatusPending).
		Update("status", model.TaskStatusCancelled)
}

// FetchDue returns pending tasks that have reached their execution time.
func (r *ScheduledTaskRepository) FetchDue(limit int) ([]model.ScheduledTask, error) {
	var tasks []model.ScheduledTask
	err := r.db.
		Where("status = ? AND execute_at <= ?", model.TaskStatusPending, time.Now()).
		Order("execute_at ASC").
		Limit(limit).
		Find(&tasks).Error
	return tasks, err
}

// MarkDone marks a task as done.
func (r *ScheduledTaskRepository) MarkDone(id uuid.UUID) {
	r.db.Model(&model.ScheduledTask{}).Where("id = ?", id).Update("status", model.TaskStatusDone)
}
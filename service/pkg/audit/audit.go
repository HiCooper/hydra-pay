package audit

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Action constants for auditable operations.
const (
	ActionPaymentCreated   = "payment.created"
	ActionPaymentCompleted = "payment.completed"
	ActionPaymentFailed    = "payment.failed"
	ActionRefundCreated    = "refund.created"
	ActionRefundCompleted  = "refund.completed"
	ActionRefundFailed     = "refund.failed"
	ActionMerchantUpdated  = "merchant.updated"
	ActionAppCreated       = "app.created"
	ActionAppUpdated       = "app.updated"
	ActionAPIKeyRotated    = "apikey.rotated"
	ActionPlanCreated      = "plan.created"
	ActionPlanUpdated      = "plan.updated"
	ActionChannelUpdated   = "channel.updated"
)

// Entry represents a single audit log record.
type Entry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	Actor     string    `json:"actor"`
	Target    string    `json:"target"`
	TargetID  string    `json:"target_id"`
	OldValue  string    `json:"old_value,omitempty"`
	NewValue  string    `json:"new_value,omitempty"`
	TraceID   string    `json:"trace_id,omitempty"`
	RequestID string    `json:"request_id,omitempty"`
	Result    string    `json:"result,omitempty"` // "success" or "error"
	Error     string    `json:"error,omitempty"`
}

// Logger writes audit entries to stdout as structured JSON.
// A DB-backed writer can be layered on top via the Recorder interface.
type Logger struct {
	db Recorder
}

// Recorder is an optional interface for persisting audit entries to a database.
type Recorder interface {
	SaveAuditEntry(ctx context.Context, e *Entry) error
}

var defaultLogger *Logger

// Init initializes the package-level audit logger.
// Pass nil for db to log to stdout only (useful in dev/testing). Pass a Recorder
// to additionally persist entries to a database.
func Init(db Recorder) {
	defaultLogger = &Logger{db: db}
}

// Log records an audit entry using the package-level logger.
// It is a no-op if Init has not been called.
func Log(ctx context.Context, e *Entry) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Record(ctx, e)
}

// Record logs an audit entry. It always writes to stdout as structured JSON,
// and additionally persists to the database if a Recorder is configured.
func (l *Logger) Record(ctx context.Context, e *Entry) {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	attrs := []any{
		"audit_id", e.ID,
		"action", e.Action,
		"actor", e.Actor,
		"target", e.Target,
		"target_id", e.TargetID,
	}
	if e.OldValue != "" {
		attrs = append(attrs, "old_value", e.OldValue)
	}
	if e.NewValue != "" {
		attrs = append(attrs, "new_value", e.NewValue)
	}
	if e.Result != "" {
		attrs = append(attrs, "result", e.Result)
	}
	if e.Error != "" {
		attrs = append(attrs, "error", e.Error)
	}

	slog.InfoContext(ctx, "audit: "+e.Action, attrs...)

	if l.db != nil {
		if err := l.db.SaveAuditEntry(ctx, e); err != nil {
			slog.WarnContext(ctx, "audit db write failed", "action", e.Action, "error", err)
		}
	}
}

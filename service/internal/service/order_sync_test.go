package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hydra/pay-service/internal/channel"
	"github.com/hydra/pay-service/internal/model"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	driver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()

	ctx := context.Background()
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("hydra_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	db, err := gorm.Open(driver.Open(connStr), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to postgres: %v", err)
	}

	if err := db.AutoMigrate(
		&model.Payment{},
		&model.PaymentEvent{},
		&model.ScheduledTask{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(5)
	sqlDB.SetMaxIdleConns(2)

	cleanup := func() {
		sqlDB.Close()
		pgContainer.Terminate(ctx)
	}

	return db, cleanup
}

// MockAdapter for testing
type MockAdapter struct {
	status      error
	resultStr   string
	resultErr   error
}

func (m *MockAdapter) Name() string {
	return "mock"
}

func (m *MockAdapter) GetPaymentStatus(ctx context.Context, channelTxID string) (string, error) {
	if m.status != nil {
		return "", m.status
	}
	return m.resultStr, m.resultErr
}

func (m *MockAdapter) CreatePayment(ctx context.Context, req *channel.CreatePaymentRequest) (*channel.CreatePaymentResponse, error) {
	return nil, nil
}

func (m *MockAdapter) VerifyCallback(ctx context.Context, data *channel.CallbackData) (*channel.CallbackResult, error) {
	return nil, nil
}

func (m *MockAdapter) Refund(ctx context.Context, req *channel.RefundRequest) (*channel.RefundResponse, error) {
	return nil, nil
}

func TestSyncExpiredOrders_PendingTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create pending payment that should timeout
	paymentID := uuid.New()
	payment := &model.Payment{
		ID:        paymentID,
		TradeNo:   "trade_" + uuid.New().String()[:8],
		Channel:   "alipay",
		Amount:    1000,
		Status:    model.PaymentStatusPending,
		CreatedAt: time.Now().Add(-30 * time.Minute), // 30 min ago
	}
	if err := db.Create(payment).Error; err != nil {
		t.Fatalf("failed to create payment: %v", err)
	}

	// Create scheduled task
	task := &model.ScheduledTask{
		ID:          uuid.New(),
		ReferenceID: paymentID,
		TaskType:    model.TaskTypeOrderTimeout,
		ExecuteAt:   time.Now().Add(-5 * time.Minute),
		Status:      model.TaskStatusPending,
	}
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Run sync
	getAdapter := func(channel string) (channel.Adapter, error) {
		return &MockAdapter{}, nil
	}
	SyncExpiredOrders(db, getAdapter)

	// Verify payment is now expired
	updated := &model.Payment{}
	if err := db.First(updated, paymentID).Error; err != nil {
		t.Fatalf("failed to fetch updated payment: %v", err)
	}
	if updated.Status != model.PaymentStatusExpired {
		t.Errorf("expected status %s, got %s", model.PaymentStatusExpired, updated.Status)
	}

	// Verify task is marked done
	updatedTask := &model.ScheduledTask{}
	if err := db.First(updatedTask, task.ID).Error; err != nil {
		t.Fatalf("failed to fetch task: %v", err)
	}
	if updatedTask.Status != "done" {
		t.Errorf("expected task status done, got %s", updatedTask.Status)
	}
}

func TestSyncExpiredOrders_ProcessingToFailed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create processing payment
	paymentID := uuid.New()
	payment := &model.Payment{
		ID:         paymentID,
		TradeNo:    "trade_" + uuid.New().String()[:8],
		Channel:    "alipay",
		Amount:     1000,
		Status:     model.PaymentStatusProcessing,
		ExternalID: "ext_12345",
		CreatedAt:  time.Now().Add(-30 * time.Minute),
	}
	if err := db.Create(payment).Error; err != nil {
		t.Fatalf("failed to create payment: %v", err)
	}

	// Create scheduled task
	task := &model.ScheduledTask{
		ID:          uuid.New(),
		ReferenceID: paymentID,
		TaskType:    model.TaskTypeOrderTimeout,
		ExecuteAt:   time.Now().Add(-5 * time.Minute),
		Status:      model.TaskStatusPending,
	}
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Run sync with adapter returning failed status
	getAdapter := func(ch string) (channel.Adapter, error) {
		return &MockAdapter{
			resultStr: model.PaymentStatusFailed,
		}, nil
	}
	SyncExpiredOrders(db, getAdapter)

	// Verify payment is now failed
	updated := &model.Payment{}
	if err := db.First(updated, paymentID).Error; err != nil {
		t.Fatalf("failed to fetch updated payment: %v", err)
	}
	if updated.Status != model.PaymentStatusFailed {
		t.Errorf("expected status %s, got %s", model.PaymentStatusFailed, updated.Status)
	}
}

func TestSyncExpiredOrders_ProcessingToPaid(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create processing payment (late callback scenario)
	paymentID := uuid.New()
	payment := &model.Payment{
		ID:         paymentID,
		TradeNo:    "trade_" + uuid.New().String()[:8],
		Channel:    "alipay",
		Amount:     1000,
		Status:     model.PaymentStatusProcessing,
		ExternalID: "ext_12345",
		CreatedAt:  time.Now().Add(-30 * time.Minute),
	}
	if err := db.Create(payment).Error; err != nil {
		t.Fatalf("failed to create payment: %v", err)
	}

	// Create scheduled task
	task := &model.ScheduledTask{
		ID:          uuid.New(),
		ReferenceID: paymentID,
		TaskType:    model.TaskTypeOrderTimeout,
		ExecuteAt:   time.Now().Add(-5 * time.Minute),
		Status:      model.TaskStatusPending,
	}
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Run sync with adapter returning paid status
	getAdapter := func(ch string) (channel.Adapter, error) {
		return &MockAdapter{
			resultStr: model.PaymentStatusPaid,
		}, nil
	}
	SyncExpiredOrders(db, getAdapter)

	// Verify payment is now paid
	updated := &model.Payment{}
	if err := db.First(updated, paymentID).Error; err != nil {
		t.Fatalf("failed to fetch updated payment: %v", err)
	}
	if updated.Status != model.PaymentStatusPaid {
		t.Errorf("expected status %s, got %s", model.PaymentStatusPaid, updated.Status)
	}
}

func TestSyncExpiredOrders_ChannelQueryError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create processing payment
	paymentID := uuid.New()
	payment := &model.Payment{
		ID:         paymentID,
		TradeNo:    "trade_" + uuid.New().String()[:8],
		Channel:    "alipay",
		Amount:     1000,
		Status:     model.PaymentStatusProcessing,
		ExternalID: "ext_12345",
		CreatedAt:  time.Now().Add(-30 * time.Minute),
	}
	if err := db.Create(payment).Error; err != nil {
		t.Fatalf("failed to create payment: %v", err)
	}

	// Create scheduled task
	task := &model.ScheduledTask{
		ID:          uuid.New(),
		ReferenceID: paymentID,
		TaskType:    model.TaskTypeOrderTimeout,
		ExecuteAt:   time.Now().Add(-5 * time.Minute),
		Status:      model.TaskStatusPending,
	}
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Run sync with adapter query error
	getAdapter := func(ch string) (channel.Adapter, error) {
		return &MockAdapter{
			status: errors.New("network timeout"),
		}, nil
	}
	SyncExpiredOrders(db, getAdapter)

	// Verify payment is expired due to query error
	updated := &model.Payment{}
	if err := db.First(updated, paymentID).Error; err != nil {
		t.Fatalf("failed to fetch updated payment: %v", err)
	}
	if updated.Status != model.PaymentStatusExpired {
		t.Errorf("expected status %s, got %s", model.PaymentStatusExpired, updated.Status)
	}
}

func TestSyncExpiredOrders_ChannelUnavailable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create processing payment with unavailable channel
	paymentID := uuid.New()
	payment := &model.Payment{
		ID:         paymentID,
		TradeNo:    "trade_" + uuid.New().String()[:8],
		Channel:    "unknown_channel",
		Amount:     1000,
		Status:     model.PaymentStatusProcessing,
		ExternalID: "ext_12345",
		CreatedAt:  time.Now().Add(-30 * time.Minute),
	}
	if err := db.Create(payment).Error; err != nil {
		t.Fatalf("failed to create payment: %v", err)
	}

	// Create scheduled task
	task := &model.ScheduledTask{
		ID:          uuid.New(),
		ReferenceID: paymentID,
		TaskType:    model.TaskTypeOrderTimeout,
		ExecuteAt:   time.Now().Add(-5 * time.Minute),
		Status:      model.TaskStatusPending,
	}
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Run sync with unavailable channel
	getAdapter := func(ch string) (channel.Adapter, error) {
		return nil, errors.New("channel not configured")
	}
	SyncExpiredOrders(db, getAdapter)

	// Verify payment is expired
	updated := &model.Payment{}
	if err := db.First(updated, paymentID).Error; err != nil {
		t.Fatalf("failed to fetch updated payment: %v", err)
	}
	if updated.Status != model.PaymentStatusExpired {
		t.Errorf("expected status %s, got %s", model.PaymentStatusExpired, updated.Status)
	}
}

func TestSyncExpiredOrders_SkipTerminalStates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create paid payment
	paymentID := uuid.New()
	payment := &model.Payment{
		ID:        paymentID,
		TradeNo:   "trade_" + uuid.New().String()[:8],
		Channel:   "alipay",
		Amount:    1000,
		Status:    model.PaymentStatusPaid, // Already terminal
		CreatedAt: time.Now().Add(-30 * time.Minute),
	}
	if err := db.Create(payment).Error; err != nil {
		t.Fatalf("failed to create payment: %v", err)
	}

	// Create scheduled task
	task := &model.ScheduledTask{
		ID:          uuid.New(),
		ReferenceID: paymentID,
		TaskType:    model.TaskTypeOrderTimeout,
		ExecuteAt:   time.Now().Add(-5 * time.Minute),
		Status:      model.TaskStatusPending,
	}
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Run sync
	getAdapter := func(ch string) (channel.Adapter, error) {
		return &MockAdapter{}, nil
	}
	SyncExpiredOrders(db, getAdapter)

	// Verify payment status unchanged
	updated := &model.Payment{}
	if err := db.First(updated, paymentID).Error; err != nil {
		t.Fatalf("failed to fetch updated payment: %v", err)
	}
	if updated.Status != model.PaymentStatusPaid {
		t.Errorf("expected status %s, got %s", model.PaymentStatusPaid, updated.Status)
	}

	// Verify task is still marked done
	updatedTask := &model.ScheduledTask{}
	if err := db.First(updatedTask, task.ID).Error; err != nil {
		t.Fatalf("failed to fetch task: %v", err)
	}
	if updatedTask.Status != "done" {
		t.Errorf("expected task status done, got %s", updatedTask.Status)
	}
}

func TestSyncExpiredOrders_MultipleTasks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create 3 payments with different statuses
	payments := []struct {
		id     uuid.UUID
		status string
		expect string
	}{
		{uuid.New(), model.PaymentStatusPending, model.PaymentStatusExpired},
		{uuid.New(), model.PaymentStatusProcessing, model.PaymentStatusFailed},
		{uuid.New(), model.PaymentStatusPaid, model.PaymentStatusPaid}, // unchanged
	}

	for i, p := range payments {
		payment := &model.Payment{
			ID:         p.id,
			TradeNo:    "trade_" + uuid.New().String()[:8],
			Channel:    "alipay",
			Amount:     1000,
			Status:     p.status,
			ExternalID: "ext_" + string(rune(i)),
			CreatedAt:  time.Now().Add(-30 * time.Minute),
		}
		if err := db.Create(payment).Error; err != nil {
			t.Fatalf("failed to create payment: %v", err)
		}

		task := &model.ScheduledTask{
			ID:          uuid.New(),
			ReferenceID: p.id,
			TaskType:    model.TaskTypeOrderTimeout,
			ExecuteAt:   time.Now().Add(-5 * time.Minute),
			Status:      model.TaskStatusPending,
		}
		if err := db.Create(task).Error; err != nil {
			t.Fatalf("failed to create task: %v", err)
		}
	}

	// Run sync with adapter returning failed for processing
	getAdapter := func(ch string) (channel.Adapter, error) {
		return &MockAdapter{
			resultStr: model.PaymentStatusFailed,
		}, nil
	}
	SyncExpiredOrders(db, getAdapter)

	// Verify all payments have expected status
	for i, p := range payments {
		updated := &model.Payment{}
		if err := db.First(updated, p.id).Error; err != nil {
			t.Fatalf("failed to fetch updated payment %d: %v", i, err)
		}
		if updated.Status != p.expect {
			t.Errorf("payment %d: expected status %s, got %s", i, p.expect, updated.Status)
		}
	}
}

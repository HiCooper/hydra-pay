package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hydra/pay-service/internal/config"
	"github.com/hydra/pay-service/internal/handler"
	"github.com/hydra/pay-service/internal/middleware"
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
		&model.App{},
		&model.Refund{},
		&model.IdempotencyRecord{},
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

func TestPaymentCreateFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	gin.SetMode(gin.TestMode)
	db, cleanup := setupTestDB(t)
	defer cleanup()

	appID := uuid.New()
	apiKey := "sk_test_" + uuid.New().String()
	app := &model.App{
		ID:     appID,
		Name:   "Test App",
		APIKey: apiKey,
		Status: "active",
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	cfg := &config.Config{Server: config.ServerConfig{Mode: "test"}}
	payHandler := handler.NewPaymentHandler(db, cfg)

	r := gin.New()
	r.Use(middleware.RequestID())

	v1 := r.Group("/v1")
	v1.Use(func(c *gin.Context) {
		c.Set(middleware.ContextAppID, appID)
		c.Next()
	})
	{
		v1.POST("/payments/create", payHandler.CreatePayment)
		v1.GET("/payments/:id", payHandler.GetPayment)
	}

	// Step 1: Create a payment record directly (simulating pending payment)
	paymentID := uuid.New()
	payment := &model.Payment{
		ID:          paymentID,
		AppID:       appID,
		UserID:      "user_001",
		Amount:      100,
		Currency:    "CNY",
		Channel:     "alipay",
		Description: "Test payment",
		Status:      model.PaymentStatusPending,
		TradeNo:     strings.ReplaceAll(uuid.New().String(), "-", ""),
	}
	if err := db.Create(payment).Error; err != nil {
		t.Fatalf("failed to create payment: %v", err)
	}

	// Step 2: Verify payment exists via API
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/payments/"+paymentID.String(), nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data model.Payment `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Data.Status != model.PaymentStatusPending {
		t.Fatalf("expected status pending, got %s", resp.Data.Status)
	}

	// Step 3: Simulate callback - mark as paid
	payment.Status = model.PaymentStatusPaid
	payment.ExternalID = "alipay_tx_001"
	if err := db.Save(payment).Error; err != nil {
		t.Fatalf("failed to update payment: %v", err)
	}

	// Step 4: Verify payment status changed
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/v1/payments/"+paymentID.String(), nil)
	r.ServeHTTP(w, req)

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Data.Status != model.PaymentStatusPaid {
		t.Fatalf("expected status paid, got %s", resp.Data.Status)
	}

	t.Log("payment lifecycle test passed: pending -> paid")
}

func TestPaymentListRefunds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	gin.SetMode(gin.TestMode)
	db, cleanup := setupTestDB(t)
	defer cleanup()

	appID := uuid.New()
	db.Create(&model.App{
		ID: appID, Name: "Test App",
		APIKey: "sk_test_" + uuid.New().String(), Status: "active",
	})

	paymentID := uuid.New()
	payment := &model.Payment{
		ID: paymentID, AppID: appID,
		UserID: "user_001", Amount: 200, Currency: "CNY",
		Channel: "alipay", Status: model.PaymentStatusPaid,
		TradeNo: strings.ReplaceAll(uuid.New().String(), "-", ""),
	}
	db.Create(payment)

	refund := &model.Refund{
		ID:           uuid.New(),
		PaymentID:    paymentID,
		Channel:      "alipay",
		RefundAmount: "50",
		OutRequestNo: "rf_" + uuid.New().String(),
		Status:       model.RefundStatusSuccess,
	}
	db.Create(refund)

	cfg := &config.Config{Server: config.ServerConfig{Mode: "test"}}
	refundHandler := handler.NewRefundHandler(db, cfg)

	r := gin.New()
	r.Use(middleware.RequestID())
	v1 := r.Group("/v1")
	v1.Use(func(c *gin.Context) {
		c.Set(middleware.ContextAppID, appID)
		c.Next()
	})
	{
		v1.GET("/payments/:id/refunds", refundHandler.ListPaymentRefunds)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/payments/"+paymentID.String()+"/refunds", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			PaymentID string `json:"payment_id"`
			Refunds   []struct {
				RefundAmount string `json:"refund_amount"`
			} `json:"refunds"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(resp.Data.Refunds) != 1 {
		t.Fatalf("expected 1 refund, got %d", len(resp.Data.Refunds))
	}
	if resp.Data.Refunds[0].RefundAmount != "50" {
		t.Fatalf("expected refund amount 50, got %s", resp.Data.Refunds[0].RefundAmount)
	}

	t.Log("refund list test passed")
}

func TestIdempotencyKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	gin.SetMode(gin.TestMode)
	db, cleanup := setupTestDB(t)
	defer cleanup()

	appID := uuid.New()
	key := "idem_" + uuid.New().String()
	record := &model.IdempotencyRecord{
		ID:             uuid.New(),
		IdempotencyKey: key,
		AppID:          appID,
		ResponseStatus: 200,
		ResponseBody:   `{"status":"ok"}`,
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}
	if err := db.Create(record).Error; err != nil {
		t.Fatalf("failed to create idempotency record: %v", err)
	}

	duplicate := &model.IdempotencyRecord{
		ID:             uuid.New(),
		IdempotencyKey: key,
		AppID:          appID,
		ResponseStatus: 200,
		ResponseBody:   `{"status":"ok"}`,
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}
	if err := db.Create(duplicate).Error; err == nil {
		t.Fatal("expected duplicate key error, got nil")
	}

	t.Log("idempotency test passed")
}

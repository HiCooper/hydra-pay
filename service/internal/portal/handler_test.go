package portal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hydra/pay-service/internal/model"
)

func setMerchantContext(c *gin.Context, merchantID uuid.UUID) {
	c.Set(merchantContextKey, merchantID)
}

// ---- merchantAppIDs ----

func TestMerchantAppIDs(t *testing.T) {
	db := openTestDB(t)
	m := seedMerchant(t, db)
	app1 := seedApp(t, db, m.ID, "App 1")
	app2 := seedApp(t, db, m.ID, "App 2")

	h := newTestHandler(db)
	ids := h.merchantAppIDs(m.ID)

	if len(ids) != 2 {
		t.Fatalf("expected 2 app IDs, got %d", len(ids))
	}
	found := map[string]bool{}
	for _, id := range ids {
		found[id.String()] = true
	}
	if !found[app1.ID.String()] || !found[app2.ID.String()] {
		t.Fatal("expected both app IDs in result")
	}
}

func TestMerchantAppIDsNoApps(t *testing.T) {
	db := openTestDB(t)
	m := seedMerchant(t, db)

	h := newTestHandler(db)
	ids := h.merchantAppIDs(m.ID)

	if len(ids) != 0 {
		t.Fatalf("expected 0 app IDs, got %d", len(ids))
	}
}

func TestMerchantAppIDsDifferentMerchants(t *testing.T) {
	db := openTestDB(t)
	m1 := seedMerchant(t, db)
	m2 := &model.Merchant{Name: "M2", Email: "m2@test.com"}
	m2.SetPassword("password")
	db.Create(m2)
	seedApp(t, db, m1.ID, "M1 App")

	h := newTestHandler(db)
	ids := h.merchantAppIDs(m2.ID)
	if len(ids) != 0 {
		t.Fatalf("expected 0 app IDs for merchant 2, got %d", len(ids))
	}
}

// ---- appBelongsToMerchant ----

func TestAppBelongsToMerchant(t *testing.T) {
	db := openTestDB(t)
	m := seedMerchant(t, db)
	app := seedApp(t, db, m.ID, "Test App")

	h := newTestHandler(db)

	if !h.appBelongsToMerchant(app.ID, m.ID) {
		t.Fatal("expected app to belong to merchant")
	}

	otherMerchant := &model.Merchant{Name: "Other", Email: "other@test.com"}
	otherMerchant.SetPassword("password")
	db.Create(otherMerchant)

	if h.appBelongsToMerchant(app.ID, otherMerchant.ID) {
		t.Fatal("expected app NOT to belong to other merchant")
	}
}

// ---- Me Handler ----

func TestMe(t *testing.T) {
	db := openTestDB(t)
	m := seedMerchant(t, db)
	seedApp(t, db, m.ID, "App 1")

	h := newTestHandler(db)
	r := gin.New()
	r.GET("/me", func(c *gin.Context) {
		setMerchantContext(c, m.ID)
		h.Me(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/me", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Merchant map[string]interface{} `json:"merchant"`
			Apps     []interface{}          `json:"apps"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Success {
		t.Fatal("expected success")
	}
	if resp.Data.Merchant["Email"] != "test@test.com" {
		t.Fatalf("expected email test@test.com, got %v", resp.Data.Merchant["Email"])
	}
	// Password hash should not be returned
	if resp.Data.Merchant["PasswordHash"] != "" {
		t.Fatal("PasswordHash should be empty")
	}
	if len(resp.Data.Apps) != 1 {
		t.Fatalf("expected 1 app, got %d", len(resp.Data.Apps))
	}
}

// ---- Dashboard Handler ----

func TestDashboard(t *testing.T) {
	db := openTestDB(t)
	m := seedMerchant(t, db)
	app := seedApp(t, db, m.ID, "Test App")

	// Create some payments
	now := time.Now()
	db.Create(&model.Payment{
		AppID: app.ID, UserID: "u1", Amount: 1000, Status: model.PaymentStatusPaid,
		TradeNo: "T001", Channel: model.ChannelAlipay, CreatedAt: now,
	})
	db.Create(&model.Payment{
		AppID: app.ID, UserID: "u2", Amount: 500, Status: model.PaymentStatusPending,
		TradeNo: "T002", Channel: model.ChannelWechat, CreatedAt: now,
	})

	h := newTestHandler(db)
	r := gin.New()
	r.GET("/dashboard", func(c *gin.Context) {
		setMerchantContext(c, m.ID)
		h.Dashboard(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/dashboard", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			TodayOrders  int64   `json:"today_orders"`
			TodayPaid    int64   `json:"today_paid"`
			TodayRevenue float64 `json:"today_revenue"`
			SuccessRate  float64 `json:"success_rate"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Success {
		t.Fatal("expected success")
	}
	// Today's payments should include both (both created now)
	if resp.Data.TodayOrders != 2 {
		t.Fatalf("expected 2 today_orders, got %d", resp.Data.TodayOrders)
	}
	if resp.Data.TodayPaid != 1 {
		t.Fatalf("expected 1 today_paid, got %d", resp.Data.TodayPaid)
	}
	// Revenue: 1000 / 100 = 10.0
	if resp.Data.TodayRevenue != 10.0 {
		t.Fatalf("expected 10.0 revenue, got %f", resp.Data.TodayRevenue)
	}
	if resp.Data.SuccessRate != 50.0 {
		t.Fatalf("expected 50.0 success rate, got %f", resp.Data.SuccessRate)
	}
}

func TestDashboardNoPayments(t *testing.T) {
	db := openTestDB(t)
	m := seedMerchant(t, db)

	h := newTestHandler(db)
	r := gin.New()
	r.GET("/dashboard", func(c *gin.Context) {
		setMerchantContext(c, m.ID)
		h.Dashboard(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/dashboard", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			TodayOrders  int64   `json:"today_orders"`
			SuccessRate  float64 `json:"success_rate"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.TodayOrders != 0 {
		t.Fatalf("expected 0 orders, got %d", resp.Data.TodayOrders)
	}
	if resp.Data.SuccessRate != 0 {
		t.Fatalf("expected 0 success rate, got %f", resp.Data.SuccessRate)
	}
}

// ---- CreateApp Handler ----

func TestCreateApp(t *testing.T) {
	db := openTestDB(t)
	m := seedMerchant(t, db)

	h := newTestHandler(db)
	r := gin.New()
	r.POST("/apps", func(c *gin.Context) {
		setMerchantContext(c, m.ID)
		h.CreateApp(c)
	})

	body := `{"name":"My New App","webhook_url":"https://example.com/hook"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/apps", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Name       string `json:"Name"`
			APIKey     string `json:"APIKey"`
			Status     string `json:"Status"`
			WebhookURL string `json:"WebhookURL"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Success {
		t.Fatal("expected success")
	}
	if resp.Data.Name != "My New App" {
		t.Fatalf("expected 'My New App', got %s", resp.Data.Name)
	}
	if resp.Data.APIKey == "" {
		t.Fatal("expected API key to be generated")
	}
	if resp.Data.Status != "active" {
		t.Fatalf("expected status 'active', got %s", resp.Data.Status)
	}
	if resp.Data.WebhookURL != "https://example.com/hook" {
		t.Fatalf("expected webhook_url to be set")
	}

	// Verify app was saved in DB
	var app model.App
	if err := db.First(&app, "name = ?", "My New App").Error; err != nil {
		t.Fatalf("app not found in DB: %v", err)
	}
	if app.MerchantID != m.ID {
		t.Fatal("app.MerchantID should match the merchant")
	}
}

func TestCreateAppMissingName(t *testing.T) {
	db := openTestDB(t)
	m := seedMerchant(t, db)

	h := newTestHandler(db)
	r := gin.New()
	r.POST("/apps", func(c *gin.Context) {
		setMerchantContext(c, m.ID)
		h.CreateApp(c)
	})

	body := `{"name":""}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/apps", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ---- UpdateSettings Handler ----

func TestUpdateSettings(t *testing.T) {
	db := openTestDB(t)
	m := seedMerchant(t, db)

	h := newTestHandler(db)
	r := gin.New()
	r.PUT("/settings", func(c *gin.Context) {
		setMerchantContext(c, m.ID)
		h.UpdateSettings(c)
	})

	body := `{"contact_name":"John Doe","contact_phone":"13800138000"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify updated
	var merchant model.Merchant
	db.First(&merchant, "id = ?", m.ID)
	if merchant.ContactName != "John Doe" {
		t.Fatalf("expected ContactName 'John Doe', got %s", merchant.ContactName)
	}
	if merchant.ContactPhone != "13800138000" {
		t.Fatalf("expected ContactPhone '13800138000', got %s", merchant.ContactPhone)
	}
}

func TestUpdateSettingsPassword(t *testing.T) {
	db := openTestDB(t)
	m := seedMerchant(t, db)

	h := newTestHandler(db)
	r := gin.New()
	r.PUT("/settings", func(c *gin.Context) {
		setMerchantContext(c, m.ID)
		h.UpdateSettings(c)
	})

	body := `{"password":"new-password-123"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify password was changed
	var merchant model.Merchant
	db.First(&merchant, "id = ?", m.ID)
	if !merchant.CheckPassword("new-password-123") {
		t.Fatal("password should have been updated")
	}
	if merchant.CheckPassword("password") {
		t.Fatal("old password should no longer work")
	}
}

// ---- ListApps Handler ----

func TestListApps(t *testing.T) {
	db := openTestDB(t)
	m := seedMerchant(t, db)
	seedApp(t, db, m.ID, "App A")
	seedApp(t, db, m.ID, "App B")

	h := newTestHandler(db)
	r := gin.New()
	r.GET("/apps", func(c *gin.Context) {
		setMerchantContext(c, m.ID)
		h.ListApps(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/apps", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Success bool          `json:"success"`
		Data    []model.App   `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 apps, got %d", len(resp.Data))
	}
}

// ---- Payment Links (CreatePaymentLink) ----

func TestCreatePaymentLink(t *testing.T) {
	db := openTestDB(t)
	m := seedMerchant(t, db)
	app := seedApp(t, db, m.ID, "Test App")

	h := newTestHandler(db)
	r := gin.New()
	r.POST("/payment-links", func(c *gin.Context) {
		setMerchantContext(c, m.ID)
		h.CreatePaymentLink(c)
	})

	body := `{"app_id":"` + app.ID.String() + `","amount":100,"description":"test link"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/payment-links", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			ID          string `json:"id"`
			Amount      int64  `json:"amount"`
			Status      string `json:"Status"`
			CheckoutURL string `json:"checkout_url"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Success {
		t.Fatal("expected success")
	}
	if resp.Data.Amount != 100 {
		t.Fatalf("expected amount 100, got %d", resp.Data.Amount)
	}
	if resp.Data.Status != "open" {
		t.Fatalf("expected status 'open', got %s", resp.Data.Status)
	}
	if resp.Data.CheckoutURL == "" {
		t.Fatal("expected checkout_url")
	}
}

func TestCreatePaymentLinkWrongApp(t *testing.T) {
	db := openTestDB(t)
	m := seedMerchant(t, db)
	otherM := &model.Merchant{Name: "Other", Email: "other@test.com"}
	otherM.SetPassword("password")
	db.Create(otherM)
	otherApp := seedApp(t, db, otherM.ID, "Other App")

	h := newTestHandler(db)
	r := gin.New()
	r.POST("/payment-links", func(c *gin.Context) {
		setMerchantContext(c, m.ID)
		h.CreatePaymentLink(c)
	})

	// Try to create payment link for another merchant's app
	body := `{"app_id":"` + otherApp.ID.String() + `","amount":100,"description":"test"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/payment-links", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for wrong app, got %d: %s", w.Code, w.Body.String())
	}
}

// ---- Subscriptions ----

func TestListSubscriptions(t *testing.T) {
	db := openTestDB(t)
	m := seedMerchant(t, db)
	app := seedApp(t, db, m.ID, "Test App")

	// Create a subscription
	now := time.Now()
	planID := uuid.New()
	sub := model.Subscription{
		AppID:              app.ID,
		UserID:             "user1",
		PlanID:             planID,
		Status:             model.SubscriptionStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour),
	}
	db.Create(&sub)

	h := newTestHandler(db)
	r := gin.New()
	r.GET("/subscriptions", func(c *gin.Context) {
		setMerchantContext(c, m.ID)
		h.ListSubscriptions(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/subscriptions", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

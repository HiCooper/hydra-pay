package portal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/hydra/pay-service/internal/model"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---- Token Tests ----

func TestSignAndVerifyToken(t *testing.T) {
	merchantID := uuid.New().String()

	token := signToken(merchantID)
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	got, ok := verifyToken(token)
	if !ok {
		t.Fatal("expected valid token to verify successfully")
	}
	if got != merchantID {
		t.Fatalf("expected merchant ID %s, got %s", merchantID, got)
	}
}

func TestVerifyTokenInvalidFormat(t *testing.T) {
	cases := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"no dots", "justonestring"},
		{"too many dots", "a.b.c"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := verifyToken(tc.token)
			if ok {
				t.Fatal("expected invalid token to fail verification")
			}
		})
	}
}

func TestVerifyTokenBadBase64(t *testing.T) {
	_, ok := verifyToken("!!!not-base64.invalid")
	if ok {
		t.Fatal("expected bad base64 token to fail")
	}
}

func TestVerifyTokenTampered(t *testing.T) {
	merchantID := uuid.New().String()
	token := signToken(merchantID)

	parts := strings.SplitN(token, ".", 2)
	tamperedToken := "dGFtcGVyZWQ=" + "." + parts[1]
	_, ok := verifyToken(tamperedToken)
	if ok {
		t.Fatal("expected tampered token to fail verification")
	}
}

func TestSignTokenDeterministic(t *testing.T) {
	merchantID := uuid.New().String()
	t1 := signToken(merchantID)
	t2 := signToken(merchantID)
	if t1 != t2 {
		t.Fatal("signToken should be deterministic for the same merchant ID")
	}
}

func TestSignTokenDifferentIDs(t *testing.T) {
	id1 := uuid.New().String()
	id2 := uuid.New().String()
	t1 := signToken(id1)
	t2 := signToken(id2)
	if t1 == t2 {
		t.Fatal("different merchant IDs should produce different tokens")
	}
}

func newTestRouter() *gin.Engine {
	return gin.New()
}

// ---- Login Tests ----

func TestLoginSuccess(t *testing.T) {
	db := openTestDB(t)
	m := seedMerchant(t, db)

	r := newTestRouter()
	r.POST("/portal/api/login", func(c *gin.Context) {
		Login(c, db)
	})

	w := httptest.NewRecorder()
	body := `{"email":"test@test.com","password":"password"}`
	req, _ := http.NewRequest("POST", "/portal/api/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Token        string `json:"token"`
			MerchantID   string `json:"merchant_id"`
			MerchantName string `json:"merchant_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success response")
	}
	if resp.Data.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if resp.Data.MerchantID != m.ID.String() {
		t.Fatalf("expected merchant ID %s, got %s", m.ID.String(), resp.Data.MerchantID)
	}

	// Verify the token is valid
	id, ok := verifyToken(resp.Data.Token)
	if !ok {
		t.Fatal("returned token should be verifiable")
	}
	if id != m.ID.String() {
		t.Fatalf("token merchant ID mismatch: %s vs %s", id, m.ID.String())
	}
}

func TestLoginWrongPassword(t *testing.T) {
	db := openTestDB(t)
	seedMerchant(t, db)

	r := newTestRouter()
	r.POST("/portal/api/login", func(c *gin.Context) {
		Login(c, db)
	})

	w := httptest.NewRecorder()
	body := `{"email":"test@test.com","password":"wrong-password"}`
	req, _ := http.NewRequest("POST", "/portal/api/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestLoginInactiveMerchant(t *testing.T) {
	db := openTestDB(t)
	m := &model.Merchant{
		ID:     uuid.New(),
		Name:   "Inactive Merchant",
		Email:  "inactive@example.com",
		Status: "disabled",
	}
	m.SetPassword("password123")
	db.Create(m)

	r := newTestRouter()
	r.POST("/portal/api/login", func(c *gin.Context) {
		Login(c, db)
	})

	w := httptest.NewRecorder()
	body := `{"email":"inactive@example.com","password":"password123"}`
	req, _ := http.NewRequest("POST", "/portal/api/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for inactive merchant, got %d", w.Code)
	}
}

func TestLoginMissingFields(t *testing.T) {
	db := openTestDB(t)

	r := newTestRouter()
	r.POST("/portal/api/login", func(c *gin.Context) {
		Login(c, db)
	})

	t.Run("missing email", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/portal/api/login", strings.NewReader(`{"password":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("missing password", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/portal/api/login", strings.NewReader(`{"email":"test@test.com"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}

func TestLoginNonexistentEmail(t *testing.T) {
	db := openTestDB(t)

	r := newTestRouter()
	r.POST("/portal/api/login", func(c *gin.Context) {
		Login(c, db)
	})

	w := httptest.NewRecorder()
	body := `{"email":"noone@example.com","password":"password"}`
	req, _ := http.NewRequest("POST", "/portal/api/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for nonexistent email, got %d", w.Code)
	}
}

// ---- MerchantAuth Middleware Tests ----

func TestMerchantAuthNoHeader(t *testing.T) {
	db := openTestDB(t)

	r := newTestRouter()
	r.Use(MerchantAuth(db))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestMerchantAuthInvalidToken(t *testing.T) {
	db := openTestDB(t)

	r := newTestRouter()
	r.Use(MerchantAuth(db))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestMerchantAuthValidToken(t *testing.T) {
	db := openTestDB(t)
	m := seedMerchant(t, db)

	r := newTestRouter()
	r.Use(MerchantAuth(db))
	r.GET("/test", func(c *gin.Context) {
		id, _ := c.Get(merchantContextKey)
		c.JSON(http.StatusOK, gin.H{"merchant_id": id.(uuid.UUID).String()})
	})

	token := signToken(m.ID.String())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMerchantAuthInactiveMerchant(t *testing.T) {
	db := openTestDB(t)
	m := &model.Merchant{
		ID:     uuid.New(),
		Name:   "Inactive",
		Email:  "inactive-auth@example.com",
		Status: "disabled",
	}
	m.SetPassword("password123")
	db.Create(m)

	r := newTestRouter()
	r.Use(MerchantAuth(db))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := signToken(m.ID.String())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for inactive merchant, got %d", w.Code)
	}
}

func TestMerchantAuthTokenForDeletedMerchant(t *testing.T) {
	db := openTestDB(t)

	unknownID := uuid.New().String()
	token := signToken(unknownID)

	r := newTestRouter()
	r.Use(MerchantAuth(db))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for token with no matching merchant, got %d", w.Code)
	}
}

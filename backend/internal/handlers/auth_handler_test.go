package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/wbmccurry20/jbs-internal-portal/internal/auth"
	"github.com/wbmccurry20/jbs-internal-portal/internal/database"
	"github.com/wbmccurry20/jbs-internal-portal/internal/models"
)

func setupJBSTestDB(t *testing.T) {
	// For JBS portal, we'll use PostgreSQL connection or mock it
	// This is a mock setup - in real tests you might use a test container
	// Note: For actual testing, you'd want to use a test database
	// For now, we'll demonstrate the test structure
	database.DB = &sql.DB{} // Mock - replace with actual test DB connection
}

func teardownJBSTestDB() {
	if database.DB != nil {
		database.DB.Close()
		database.DB = nil
	}
}

// Mock test - structure for authentication tests
func TestJBSLogin_Success(t *testing.T) {
	t.Skip("Skipping integration test - requires test database setup")
	
	setupJBSTestDB(t)
	defer teardownJBSTestDB()

	os.Setenv("JWT_SECRET", "test-secret-key-for-testing")
	defer os.Unsetenv("JWT_SECRET")

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.POST("/api/auth/login", Login)

	loginReq := models.LoginRequest{
		Email:    "test@jbs.com",
		Password: "TestPassword123!",
	}
	body, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestJBSLogin_InvalidPassword(t *testing.T) {
	t.Skip("Skipping integration test - requires test database setup")
	
	setupJBSTestDB(t)
	defer teardownJBSTestDB()

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.POST("/api/auth/login", Login)

	loginReq := models.LoginRequest{
		Email:    "test@jbs.com",
		Password: "WrongPassword123!",
	}
	body, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestJBSLogin_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.POST("/api/auth/login", Login)

	// Send invalid JSON
	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["error"] != "Invalid request" {
		t.Errorf("Expected error message 'Invalid request', got %s", response["error"])
	}
}

func TestJBSGetCurrentUser_NotFound(t *testing.T) {
	t.Skip("Skipping integration test - requires test database setup")
	
	setupJBSTestDB(t)
	defer teardownJBSTestDB()

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/api/auth/me", func(c *gin.Context) {
		c.Set("userID", 9999)
		GetCurrentUser(c)
	})

	req, _ := http.NewRequest("GET", "/api/auth/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// Unit tests for auth module
func TestGenerateToken(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-for-testing")
	defer os.Unsetenv("JWT_SECRET")

	token, err := auth.GenerateToken(1, "test@jbs.com", "admin")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if token == "" {
		t.Error("Expected non-empty token")
	}
}

func TestValidateToken(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-for-testing")
	defer os.Unsetenv("JWT_SECRET")

	token, _ := auth.GenerateToken(1, "test@jbs.com", "admin")
	
	claims, err := auth.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.Email != "test@jbs.com" {
		t.Errorf("Expected email test@jbs.com, got %s", claims.Email)
	}
	if claims.Role != "admin" {
		t.Errorf("Expected role admin, got %s", claims.Role)
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-for-testing")
	defer os.Unsetenv("JWT_SECRET")

	_, err := auth.ValidateToken("invalid.token.here")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	t.Skip("Requires time manipulation for proper testing")
}

func TestCheckPasswordHash_Success(t *testing.T) {
	password := "TestPassword123!"
	hash, _ := auth.HashPassword(password)
	
	if !auth.CheckPasswordHash(password, hash) {
		t.Error("Expected password to match hash")
	}
}

func TestCheckPasswordHash_Failure(t *testing.T) {
	password := "TestPassword123!"
	hash, _ := auth.HashPassword(password)
	
	if auth.CheckPasswordHash("WrongPassword", hash) {
		t.Error("Expected password to not match hash")
	}
}

func TestHashPassword(t *testing.T) {
	password := "TestPassword123!"
	hash1, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	hash2, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Hashes should be different (salt is random)
	if hash1 == hash2 {
		t.Error("Expected different hashes for same password")
	}

	// But both should verify
	if !auth.CheckPasswordHash(password, hash1) || !auth.CheckPasswordHash(password, hash2) {
		t.Error("Expected both hashes to verify")
	}
}

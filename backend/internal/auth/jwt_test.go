package auth

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestHashPassword(t *testing.T) {
	password := "SecurePassword123!"
	
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	
	if hash == "" {
		t.Error("Expected non-empty hash")
	}
	
	if hash == password {
		t.Error("Hash should not equal plaintext password")
	}
}

func TestHashPassword_DifferentHashes(t *testing.T) {
	password := "SecurePassword123!"
	
	hash1, _ := HashPassword(password)
	hash2, _ := HashPassword(password)
	
	// Due to salt, hashes should be different
	if hash1 == hash2 {
		t.Error("Expected different hashes for same password (salt should be random)")
	}
}

func TestCheckPasswordHash_Success(t *testing.T) {
	password := "SecurePassword123!"
	hash, _ := HashPassword(password)
	
	if !CheckPasswordHash(password, hash) {
		t.Error("Expected password to match hash")
	}
}

func TestCheckPasswordHash_Failure(t *testing.T) {
	password := "SecurePassword123!"
	wrongPassword := "WrongPassword456!"
	hash, _ := HashPassword(password)
	
	if CheckPasswordHash(wrongPassword, hash) {
		t.Error("Expected wrong password to not match hash")
	}
}

func TestCheckPasswordHash_InvalidHash(t *testing.T) {
	password := "SecurePassword123!"
	invalidHash := "not-a-valid-bcrypt-hash"
	
	if CheckPasswordHash(password, invalidHash) {
		t.Error("Expected invalid hash to fail")
	}
}

func TestGenerateToken_Success(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-minimum-32-chars-long")
	defer os.Unsetenv("JWT_SECRET")
	
	token, err := GenerateToken(1, "test@jbs.com", "admin")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	
	if token == "" {
		t.Error("Expected non-empty token")
	}
}

func TestGenerateToken_MissingSecret(t *testing.T) {
	os.Unsetenv("JWT_SECRET")
	
	_, err := GenerateToken(1, "test@jbs.com", "admin")
	if err == nil {
		t.Error("Expected error when JWT_SECRET is not set")
	}
	
	if err.Error() != "JWT_SECRET not set" {
		t.Errorf("Expected 'JWT_SECRET not set' error, got: %v", err)
	}
}

func TestGenerateToken_DifferentUsers(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-minimum-32-chars-long")
	defer os.Unsetenv("JWT_SECRET")
	
	token1, _ := GenerateToken(1, "user1@jbs.com", "admin")
	token2, _ := GenerateToken(2, "user2@jbs.com", "user")
	
	if token1 == token2 {
		t.Error("Expected different tokens for different users")
	}
}

func TestValidateToken_Success(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-minimum-32-chars-long")
	defer os.Unsetenv("JWT_SECRET")
	
	token, _ := GenerateToken(1, "test@jbs.com", "admin")
	
	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}
	
	if claims.UserID != 1 {
		t.Errorf("Expected UserID 1, got %d", claims.UserID)
	}
	if claims.Email != "test@jbs.com" {
		t.Errorf("Expected email test@jbs.com, got %s", claims.Email)
	}
	if claims.Role != "admin" {
		t.Errorf("Expected role admin, got %s", claims.Role)
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-minimum-32-chars-long")
	defer os.Unsetenv("JWT_SECRET")
	
	_, err := ValidateToken("invalid.token.here")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	os.Setenv("JWT_SECRET", "original-secret-key")
	token, _ := GenerateToken(1, "test@jbs.com", "admin")
	
	// Change secret
	os.Setenv("JWT_SECRET", "different-secret-key")
	defer os.Unsetenv("JWT_SECRET")
	
	_, err := ValidateToken(token)
	if err == nil {
		t.Error("Expected error when validating with wrong secret")
	}
}

func TestValidateToken_MissingSecret(t *testing.T) {
	os.Unsetenv("JWT_SECRET")
	
	_, err := ValidateToken("any.token.here")
	if err == nil {
		t.Error("Expected error when JWT_SECRET is not set")
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-minimum-32-chars-long")
	defer os.Unsetenv("JWT_SECRET")
	
	// Create an expired token manually
	claims := Claims{
		UserID: 1,
		Email:  "test@jbs.com",
		Role:   "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired 1 hour ago
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte("test-secret-key-minimum-32-chars-long"))
	
	_, err := ValidateToken(tokenString)
	if err == nil {
		t.Error("Expected error for expired token")
	}
}

func TestValidateToken_NoneAlgorithmAttack(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-minimum-32-chars-long")
	defer os.Unsetenv("JWT_SECRET")
	
	// Try to create a token with 'none' algorithm (security vulnerability)
	claims := Claims{
		UserID: 1,
		Email:  "attacker@jbs.com",
		Role:   "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	
	_, err := ValidateToken(tokenString)
	if err == nil {
		t.Error("Expected error - should reject 'none' algorithm tokens")
	}
}

func TestClaims_Serialization(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-minimum-32-chars-long")
	defer os.Unsetenv("JWT_SECRET")
	
	originalUserID := 42
	originalEmail := "test@jbs.com"
	originalRole := "accountant"
	
	token, _ := GenerateToken(originalUserID, originalEmail, originalRole)
	claims, _ := ValidateToken(token)
	
	if claims.UserID != originalUserID {
		t.Errorf("UserID mismatch: expected %d, got %d", originalUserID, claims.UserID)
	}
	if claims.Email != originalEmail {
		t.Errorf("Email mismatch: expected %s, got %s", originalEmail, claims.Email)
	}
	if claims.Role != originalRole {
		t.Errorf("Role mismatch: expected %s, got %s", originalRole, claims.Role)
	}
}

func TestGenerateToken_SpecialCharacters(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-minimum-32-chars-long")
	defer os.Unsetenv("JWT_SECRET")
	
	email := "test+special@jbs.com"
	token, err := GenerateToken(1, email, "user")
	if err != nil {
		t.Fatalf("Failed to generate token with special characters: %v", err)
	}
	
	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token with special characters: %v", err)
	}
	
	if claims.Email != email {
		t.Errorf("Expected email %s, got %s", email, claims.Email)
	}
}

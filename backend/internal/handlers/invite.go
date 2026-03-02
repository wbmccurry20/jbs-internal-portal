package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wbmccurry20/jbs-internal-portal/internal/database"
	"golang.org/x/crypto/bcrypt"
)

// InviteUserRequest is the request body for inviting a new user
type InviteUserRequest struct {
	Email string `json:"email" binding:"required,email"`
	Name  string `json:"name" binding:"required"`
	Role  string `json:"role" binding:"required"`
}

// AcceptInviteRequest is the request body for accepting an invite
type AcceptInviteRequest struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required,min=10"`
}

// ValidateTokenResponse is returned when checking if a token is valid
type ValidateTokenResponse struct {
	Valid bool   `json:"valid"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// generateSecureToken creates a cryptographically random 32-byte hex token
func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// InviteUser creates a new user in pending status and sends an invite email
func InviteUser(c *gin.Context) {
	var req InviteUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request. Name, email, and role are required."})
		return
	}

	// Validate role
	validRoles := map[string]bool{
		"executive": true, "hr_admin": true, "finance": true,
		"project_manager": true, "construction_admin": true,
		"trainee": true,
	}
	if !validRoles[req.Role] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
		return
	}

	// Normalize email
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// Check if user already exists
	var existingID int
	var existingStatus string
	err := database.DB.QueryRow(
		"SELECT id, status FROM users WHERE email = $1", req.Email,
	).Scan(&existingID, &existingStatus)

	if err == nil {
		// User exists
		if existingStatus == "active" {
			c.JSON(http.StatusConflict, gin.H{"error": "A user with this email is already active"})
			return
		}
		// If pending, allow re-invite (generate new token below)
	} else if err != sql.ErrNoRows {
		log.Printf("Database error checking user existence: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Get the inviter's user ID
	inviterID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Begin transaction
	tx, err := database.DB.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var userID int

	if existingID > 0 {
		// User already exists in pending state — update name/role
		_, err = tx.Exec(
			"UPDATE users SET name = $1, role = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3",
			req.Name, req.Role, existingID,
		)
		if err != nil {
			log.Printf("Failed to update pending user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
			return
		}
		userID = existingID

		// Invalidate any existing tokens for this user
		_, err = tx.Exec(
			"DELETE FROM invite_tokens WHERE user_id = $1 AND used_at IS NULL",
			userID,
		)
		if err != nil {
			log.Printf("Failed to clean old tokens: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
	} else {
		// Create new user with placeholder password and pending status
		// The placeholder password is a random bcrypt hash that can never be guessed
		placeholderBytes := make([]byte, 32)
		rand.Read(placeholderBytes)
		placeholderHash, _ := bcrypt.GenerateFromPassword(placeholderBytes, 14)

		err = tx.QueryRow(
			`INSERT INTO users (email, password, name, role, status) 
			 VALUES ($1, $2, $3, $4, 'pending') RETURNING id`,
			req.Email, string(placeholderHash), req.Name, req.Role,
		).Scan(&userID)
		if err != nil {
			log.Printf("Failed to create pending user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}
	}

	// Generate secure invite token
	token, err := generateSecureToken()
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate invite token"})
		return
	}

	// Token expires in 72 hours
	expiresAt := time.Now().Add(72 * time.Hour)

	_, err = tx.Exec(
		`INSERT INTO invite_tokens (user_id, token, expires_at, created_by) 
		 VALUES ($1, $2, $3, $4)`,
		userID, token, expiresAt, inviterID,
	)
	if err != nil {
		log.Printf("Failed to create invite token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create invite"})
		return
	}

	if err = tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Send invite email (non-blocking — don't fail the request if email fails)
	go func() {
		if err := sendInviteEmail(req.Email, req.Name, token); err != nil {
			log.Printf("WARNING: Failed to send invite email to %s: %v", req.Email, err)
		} else {
			log.Printf("✓ Invite email sent to %s", req.Email)
		}
	}()

	c.JSON(http.StatusCreated, gin.H{
		"message": fmt.Sprintf("Invite sent to %s", req.Email),
		"user_id": userID,
	})
}

// ValidateInviteToken checks if a token is valid and not expired
func ValidateInviteToken(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token is required"})
		return
	}

	var name, email string
	var expiresAt time.Time
	var usedAt sql.NullTime

	err := database.DB.QueryRow(`
		SELECT u.name, u.email, it.expires_at, it.used_at
		FROM invite_tokens it
		JOIN users u ON u.id = it.user_id
		WHERE it.token = $1
	`, token).Scan(&name, &email, &expiresAt, &usedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, ValidateTokenResponse{Valid: false})
		return
	}
	if err != nil {
		log.Printf("Error validating invite token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Check if already used
	if usedAt.Valid {
		c.JSON(http.StatusOK, ValidateTokenResponse{Valid: false})
		return
	}

	// Check if expired
	if time.Now().After(expiresAt) {
		c.JSON(http.StatusOK, ValidateTokenResponse{Valid: false})
		return
	}

	c.JSON(http.StatusOK, ValidateTokenResponse{
		Valid: true,
		Name:  name,
		Email: email,
	})
}

// AcceptInvite allows a user to set their password and activate their account
func AcceptInvite(c *gin.Context) {
	var req AcceptInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token and password (min 10 characters) are required"})
		return
	}

	// Validate password strength
	if len(req.Password) < 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 10 characters"})
		return
	}
	if !hasUppercase(req.Password) || !hasLowercase(req.Password) || !hasDigit(req.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password must contain uppercase, lowercase, and a number"})
		return
	}

	// Look up token
	var userID int
	var expiresAt time.Time
	var usedAt sql.NullTime

	err := database.DB.QueryRow(`
		SELECT it.user_id, it.expires_at, it.used_at
		FROM invite_tokens it
		WHERE it.token = $1
	`, req.Token).Scan(&userID, &expiresAt, &usedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired invite link"})
		return
	}
	if err != nil {
		log.Printf("Error looking up invite token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Check if already used
	if usedAt.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This invite has already been used"})
		return
	}

	// Check if expired
	if time.Now().After(expiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This invite has expired. Please ask your administrator to resend it."})
		return
	}

	// Hash the new password with high cost
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 14)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}

	// Begin transaction
	tx, err := database.DB.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Activate user and set password
	_, err = tx.Exec(
		`UPDATE users SET password = $1, status = 'active', updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
		string(hashedPassword), userID,
	)
	if err != nil {
		log.Printf("Failed to activate user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to activate account"})
		return
	}

	// Mark token as used
	_, err = tx.Exec(
		`UPDATE invite_tokens SET used_at = CURRENT_TIMESTAMP WHERE token = $1`,
		req.Token,
	)
	if err != nil {
		log.Printf("Failed to mark token as used: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if err = tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	log.Printf("✓ User %d accepted invite and activated account", userID)

	c.JSON(http.StatusOK, gin.H{"message": "Account activated! You can now log in."})
}

// ResendInvite generates a new token and resends the invite email
func ResendInvite(c *gin.Context) {
	userID := c.Param("id")

	// Get user info
	var email, name, status string
	err := database.DB.QueryRow(
		"SELECT email, name, status FROM users WHERE id = $1", userID,
	).Scan(&email, &name, &status)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		log.Printf("Error fetching user for resend: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User is already active"})
		return
	}

	inviterID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Invalidate old tokens
	_, err = database.DB.Exec(
		"DELETE FROM invite_tokens WHERE user_id = $1 AND used_at IS NULL", userID,
	)
	if err != nil {
		log.Printf("Failed to clean old tokens: %v", err)
	}

	// Generate new token
	token, err := generateSecureToken()
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate invite token"})
		return
	}

	expiresAt := time.Now().Add(72 * time.Hour)
	_, err = database.DB.Exec(
		`INSERT INTO invite_tokens (user_id, token, expires_at, created_by) VALUES ($1, $2, $3, $4)`,
		userID, token, expiresAt, inviterID,
	)
	if err != nil {
		log.Printf("Failed to create invite token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create invite"})
		return
	}

	// Send email
	go func() {
		if err := sendInviteEmail(email, name, token); err != nil {
			log.Printf("WARNING: Failed to resend invite email to %s: %v", email, err)
		} else {
			log.Printf("✓ Invite email resent to %s", email)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Invite resent to %s", email)})
}

// sendInviteEmail sends the invite email via SMTP
func sendInviteEmail(toEmail, name, token string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASSWORD")
	fromEmail := os.Getenv("SMTP_FROM")
	frontendURL := os.Getenv("FRONTEND_URL")

	if smtpHost == "" || smtpUser == "" || smtpPass == "" {
		return fmt.Errorf("SMTP not configured (SMTP_HOST, SMTP_USER, SMTP_PASSWORD required)")
	}

	if smtpPort == "" {
		smtpPort = "587"
	}
	if fromEmail == "" {
		fromEmail = smtpUser
	}
	if frontendURL == "" {
		frontendURL = "https://jbs-portal.up.railway.app"
	}

	inviteLink := fmt.Sprintf("%s/accept-invite?token=%s", frontendURL, token)

	subject := "You're invited to the JBS Construction Portal"
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; color: #333;">
  <div style="background: #1e40af; padding: 24px; border-radius: 8px 8px 0 0; text-align: center;">
    <h1 style="color: white; margin: 0; font-size: 24px;">JBS Construction Portal</h1>
  </div>
  <div style="background: #f9fafb; padding: 32px; border: 1px solid #e5e7eb; border-top: none; border-radius: 0 0 8px 8px;">
    <h2 style="margin-top: 0;">Welcome, %s!</h2>
    <p>You've been invited to join the JBS Construction internal portal. Click the button below to set up your password and activate your account.</p>
    <div style="text-align: center; margin: 32px 0;">
      <a href="%s" style="background: #1e40af; color: white; padding: 14px 32px; text-decoration: none; border-radius: 6px; font-weight: 600; font-size: 16px; display: inline-block;">Set Up My Account</a>
    </div>
    <p style="color: #6b7280; font-size: 14px;">This link expires in <strong>72 hours</strong>. If it expires, ask your administrator to resend the invite.</p>
    <p style="color: #6b7280; font-size: 14px;">If the button doesn't work, copy this link into your browser:</p>
    <p style="color: #6b7280; font-size: 12px; word-break: break-all;">%s</p>
    <hr style="border: none; border-top: 1px solid #e5e7eb; margin: 24px 0;">
    <p style="color: #9ca3af; font-size: 12px; margin: 0;">If you didn't expect this invitation, you can safely ignore this email.</p>
  </div>
</body>
</html>`, name, inviteLink, inviteLink)

	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n",
		fromEmail, toEmail, subject)

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	err := smtp.SendMail(
		smtpHost+":"+smtpPort,
		auth,
		fromEmail,
		[]string{toEmail},
		[]byte(headers+body),
	)

	return err
}

// Password validation helpers
func hasUppercase(s string) bool {
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			return true
		}
	}
	return false
}

func hasLowercase(s string) bool {
	for _, c := range s {
		if c >= 'a' && c <= 'z' {
			return true
		}
	}
	return false
}

func hasDigit(s string) bool {
	for _, c := range s {
		if c >= '0' && c <= '9' {
			return true
		}
	}
	return false
}

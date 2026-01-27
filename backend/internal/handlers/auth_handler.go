package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wbmccurry20/jbs-internal-portal/internal/auth"
	"github.com/wbmccurry20/jbs-internal-portal/internal/database"
	"github.com/wbmccurry20/jbs-internal-portal/internal/models"
)

// Login handles user login
func Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Find user by email
	var user models.User
	err := database.DB.QueryRow(`
		SELECT id, email, password, name, role, created_at, updated_at
		FROM users
		WHERE email = $1
	`, req.Email).Scan(&user.ID, &user.Email, &user.Password, &user.Name, 
		&user.Role, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Verify password
	if !auth.CheckPasswordHash(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT token
	token, err := auth.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, models.LoginResponse{
		Token: token,
		User:  user,
	})
}

// GetCurrentUser returns the current authenticated user
func GetCurrentUser(c *gin.Context) {
	userID, _ := c.Get("userID")

	var user models.User
	err := database.DB.QueryRow(`
		SELECT id, email, name, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`, userID).Scan(&user.ID, &user.Email, &user.Name, &user.Role, 
		&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// Health check endpoint
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"service": "jbs-internal-portal",
	})
}

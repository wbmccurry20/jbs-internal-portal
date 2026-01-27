package main

import (
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/wbmccurry20/jbs-internal-portal/internal/database"
	"github.com/wbmccurry20/jbs-internal-portal/internal/handlers"
	"github.com/wbmccurry20/jbs-internal-portal/internal/middleware"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Set Gin mode
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Connect to database
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	if err := database.Connect(databaseURL); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	// Initialize Gin router
	router := gin.Default()

	// Security middleware
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.InputValidation())

	// CORS middleware
	router.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowedOrigins := getAllowedOrigins()

		for _, allowed := range allowedOrigins {
			if origin == allowed {
				c.Header("Access-Control-Allow-Origin", origin)
				break
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Public routes
	router.GET("/health", handlers.Health)
	router.POST("/api/login", handlers.Login)

	// Protected routes
	api := router.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
		// User routes
		api.GET("/user", handlers.GetCurrentUser)

		// Concur conversion routes
		api.POST("/concur/upload", handlers.UploadConcurFile)
		api.GET("/concur/history", handlers.GetConversionHistory)
		api.GET("/download/:id", handlers.DownloadConversionResult)

		// Reconciliation routes
		api.POST("/reconciliation/upload", handlers.UploadReconciliationFiles)
		api.GET("/reconciliation/history", handlers.GetReconciliationHistory)
		api.GET("/reconciliation/download/:id", handlers.DownloadReconciliationResult)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func getAllowedOrigins() []string {
	origins := os.Getenv("ALLOWED_ORIGINS")
	if origins == "" {
		return []string{"http://localhost:4321"}
	}
	return strings.Split(origins, ",")
}

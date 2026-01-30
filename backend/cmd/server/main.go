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

	// Security middleware (applied to all routes)
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.RateLimitGeneral())

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
	router.POST("/api/login", middleware.RateLimitLogin(), handlers.Login)

	// Protected routes
	api := router.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
		// User routes
		api.GET("/user", handlers.GetCurrentUser)
		api.POST("/change-password", handlers.ChangeOwnPassword)

		// User management routes (owner and support only)
		api.GET("/users", middleware.RequireRole("owner", "support"), handlers.ListUsers)
		api.POST("/users", middleware.RequireRole("owner", "support"), handlers.CreateUser)
		api.DELETE("/users/:id", middleware.RequireRole("owner", "support"), handlers.DeleteUser)
		api.POST("/users/:id/reset-password", middleware.RequireRole("owner", "support"), handlers.ResetUserPassword)

		// Concur conversion routes (finance and owner only, with upload rate limiting)
		api.POST("/concur/upload", middleware.RequireRole("finance", "owner", "support"), middleware.RateLimitUpload(), handlers.UploadConcurFile)
		api.GET("/concur/history", middleware.RequireRole("finance", "owner", "support"), handlers.GetConversionHistory)
		api.GET("/download/:id", middleware.RequireRole("finance", "owner", "support"), handlers.DownloadConversionResult)
		api.DELETE("/concur/:id", middleware.RequireRole("finance", "owner", "support"), handlers.DeleteConversionJob)

		// Reconciliation routes (finance and owner only, with upload rate limiting)
		api.POST("/reconciliation/upload", middleware.RequireRole("finance", "owner", "support"), middleware.RateLimitUpload(), handlers.UploadReconciliationFiles)
		api.GET("/reconciliation/history", middleware.RequireRole("finance", "owner", "support"), handlers.GetReconciliationHistory)
		api.GET("/reconciliation/download/:id", middleware.RequireRole("finance", "owner", "support"), handlers.DownloadReconciliationResult)
		api.DELETE("/reconciliation/:id", middleware.RequireRole("finance", "owner", "support"), handlers.DeleteReconciliationJob)

		// Job routes (delete requires owner or support)
		api.GET("/jobs", handlers.ListJobs)
		api.GET("/jobs/:id", handlers.GetJob)
		api.POST("/jobs", handlers.CreateJob)
		api.PUT("/jobs/:id", handlers.UpdateJob)
		api.DELETE("/jobs/:id", middleware.RequireRole("owner", "support"), handlers.DeleteJob)
		api.POST("/jobs/:id/updates", handlers.CreateJobUpdate)
		api.GET("/jobs/:id/updates", handlers.ListJobUpdates)

		// Bid routes (archive requires owner or support)
		api.GET("/bids", handlers.ListBids)
		api.GET("/bids/:id", handlers.GetBid)
		api.POST("/bids", handlers.CreateBid)
		api.PUT("/bids/:id", handlers.UpdateBid)
		api.DELETE("/bids/:id", middleware.RequireRole("owner", "support"), handlers.DeleteBid)

		// Lodging routes (delete requires owner or support)
		api.GET("/lodging", handlers.ListLodging)
		api.GET("/lodging/:id", handlers.GetLodging)
		api.POST("/lodging", handlers.CreateLodging)
		api.PUT("/lodging/:id", handlers.UpdateLodging)
		api.DELETE("/lodging/:id", middleware.RequireRole("owner", "support"), handlers.DeleteLodging)
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
		// In production, ALLOWED_ORIGINS must be explicitly set
		if os.Getenv("GIN_MODE") == "release" {
			log.Fatal("ALLOWED_ORIGINS environment variable is required in production mode")
		}
		// Development fallback
		return []string{"http://localhost:4321"}
	}
	return strings.Split(origins, ",")
}

package main

import (
	"log"
	"net/http"
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

	// Limit request body size to 10MB to prevent abuse
	router.MaxMultipartMemory = 10 << 20 // 10 MB
	router.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20) // 10MB
		c.Next()
	})

	// Security middleware (applied to all routes)
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.RateLimitGeneral())
	// Temporarily disabled - investigating Railway deployment issues
	// router.Use(middleware.QueryLogger(500 * time.Millisecond))

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

		// License routes (delete requires owner or support)
		api.GET("/licenses", handlers.ListLicenses)
		api.GET("/licenses/state-summary", handlers.GetStateSummary)
		api.POST("/licenses", handlers.CreateLicense)
		api.PUT("/licenses/:id", handlers.UpdateLicense)
		api.DELETE("/licenses/:id", middleware.RequireRole("owner", "support"), handlers.DeleteLicense)

		// Training portal routes
		// Trainee view (any authenticated user)
		api.GET("/training/my-calendar", handlers.GetMyTrainingCalendar)

		// Admin training management (owner and support only)
		api.GET("/training/programs", middleware.RequireRole("owner", "support"), handlers.ListTrainingPrograms)
		api.GET("/training/programs/:id", middleware.RequireRole("owner", "support"), handlers.GetTrainingProgram)
		api.POST("/training/programs", middleware.RequireRole("owner", "support"), handlers.CreateTrainingProgram)
		api.PUT("/training/programs/:id", middleware.RequireRole("owner", "support"), handlers.UpdateTrainingProgram)
		api.DELETE("/training/programs/:id", middleware.RequireRole("owner", "support"), handlers.DeleteTrainingProgram)

		// Schedule items (owner and support only)
		api.POST("/training/programs/:id/items", middleware.RequireRole("owner", "support"), handlers.CreateScheduleItem)
		api.PUT("/training/programs/:id/items/:itemId", middleware.RequireRole("owner", "support"), handlers.UpdateScheduleItem)
		api.DELETE("/training/programs/:id/items/:itemId", middleware.RequireRole("owner", "support"), handlers.DeleteScheduleItem)
		api.PUT("/training/programs/:id/items", middleware.RequireRole("owner", "support"), handlers.BulkUpdateScheduleItems)

		// Resources (owner and support only)
		api.POST("/training/programs/:id/resources", middleware.RequireRole("owner", "support"), handlers.CreateResource)
		api.PUT("/training/programs/:id/resources/:resourceId", middleware.RequireRole("owner", "support"), handlers.UpdateResource)
		api.DELETE("/training/programs/:id/resources/:resourceId", middleware.RequireRole("owner", "support"), handlers.DeleteResource)

		// Trainee assignments (owner and support only)
		api.GET("/training/assignments", middleware.RequireRole("owner", "support"), handlers.ListAssignments)
		api.POST("/training/assignments", middleware.RequireRole("owner", "support"), handlers.CreateAssignment)
		api.PUT("/training/assignments/:id", middleware.RequireRole("owner", "support"), handlers.UpdateAssignment)
		api.DELETE("/training/assignments/:id", middleware.RequireRole("owner", "support"), handlers.DeleteAssignment)
		api.GET("/training/assignments/:id/calendar", middleware.RequireRole("owner", "support"), handlers.GetTraineeCalendar)
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

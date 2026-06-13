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

	// Initialize SharePoint / Graph API client
	handlers.InitGraphClient()

	// Initialize dedicated mail Graph client (separate token + Mail scopes, DB attached)
	handlers.InitMailGraphClient(database.DB)

	// Initialize email OAuth handler (public routes — Microsoft redirects here without JWT)
	emailAuthHandler := &handlers.EmailAuthHandler{
		GraphClient: handlers.GetMailGraphClient(),
		DB:          database.DB,
	}

	// Initialize email automation handler (JWT-protected API)
	emailAutomationHandler := &handlers.EmailAutomationHandler{
		DB: database.DB,
	}

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

	// Public invite routes (no auth required — users click link from email)
	router.GET("/api/invite/validate/:token", handlers.ValidateInviteToken)
	router.POST("/api/invite/accept", handlers.AcceptInvite)

	// Public SharePoint OAuth callback (Microsoft redirects here after login)
	router.GET("/api/sharepoint/callback", handlers.HandleSharePointCallback)

	// Public Email OAuth routes (Microsoft redirects here — no JWT required)
	router.GET("/api/email/auth", emailAuthHandler.GetAuthURL)
	router.GET("/api/email/callback", emailAuthHandler.HandleCallback)

	// Public payment application endpoints (no auth required — called from buildwithjbs.com)
	// RateLimitSubmission: 10 submissions per hour per IP (stricter than general)
	v1 := router.Group("/api/v1")
	v1.POST("/payment-applications", middleware.RateLimitSubmission(), handlers.CreatePaymentApplication)
	v1.GET("/payment-applications/:submissionToken", middleware.RateLimitGeneral(), handlers.GetPaymentApplication)

	// Stripe webhook — raw route, no auth middleware, no JSON body parsing.
	// Stripe-Signature header is verified inside the handler using STRIPE_WEBHOOK_SIGNING_SECRET.
	router.POST("/api/v1/stripe/webhook", handlers.StripeWebhook)

	// Protected routes
	api := router.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
		// User routes
		api.GET("/user", handlers.GetCurrentUser)
		api.POST("/change-password", handlers.ChangeOwnPassword)

		// User management routes (executive, hr_admin, and support only)
		api.GET("/users", middleware.RequireRole("executive", "hr_admin", "support"), handlers.ListUsers)
		api.POST("/users", middleware.RequireRole("executive", "hr_admin", "support"), handlers.CreateUser)
		api.POST("/users/invite", middleware.RequireRole("executive", "hr_admin", "support"), handlers.InviteUser)
		api.POST("/users/:id/resend-invite", middleware.RequireRole("executive", "hr_admin", "support"), handlers.ResendInvite)
		api.DELETE("/users/:id", middleware.RequireRole("executive", "support"), handlers.DeleteUser)
		api.POST("/users/:id/reset-password", middleware.RequireRole("executive", "hr_admin", "support"), handlers.ResetUserPassword)
		api.PUT("/users/:id/role", middleware.RequireRole("executive", "support"), handlers.UpdateUserRole)

		// Concur conversion routes (finance, executive, and support only)
		api.POST("/concur/upload", middleware.RequireRole("finance", "executive", "support"), middleware.RateLimitUpload(), handlers.UploadConcurFile)
		api.GET("/concur/history", middleware.RequireRole("finance", "executive", "support"), handlers.GetConversionHistory)
		api.GET("/download/:id", middleware.RequireRole("finance", "executive", "support"), handlers.DownloadConversionResult)
		api.DELETE("/concur/:id", middleware.RequireRole("finance", "executive", "support"), handlers.DeleteConversionJob)

		// Reconciliation routes (finance, executive, and support only)
		api.POST("/reconciliation/upload", middleware.RequireRole("finance", "executive", "support"), middleware.RateLimitUpload(), handlers.UploadReconciliationFiles)
		api.GET("/reconciliation/history", middleware.RequireRole("finance", "executive", "support"), handlers.GetReconciliationHistory)
		api.GET("/reconciliation/download/:id", middleware.RequireRole("finance", "executive", "support"), handlers.DownloadReconciliationResult)
		api.DELETE("/reconciliation/:id", middleware.RequireRole("finance", "executive", "support"), handlers.DeleteReconciliationJob)

		// Job routes
		api.GET("/jobs", middleware.RequireRole("executive", "hr_admin", "finance", "project_manager", "construction_admin", "support"), handlers.ListJobs)
		api.GET("/jobs/:id", middleware.RequireRole("executive", "hr_admin", "finance", "project_manager", "construction_admin", "support"), handlers.GetJob)
		api.POST("/jobs", middleware.RequireRole("executive", "hr_admin", "project_manager", "support"), handlers.CreateJob)
		api.PUT("/jobs/:id", middleware.RequireRole("executive", "hr_admin", "project_manager", "support"), handlers.UpdateJob)
		api.DELETE("/jobs/:id", middleware.RequireRole("executive", "support"), handlers.DeleteJob)
		api.POST("/jobs/:id/updates", middleware.RequireRole("executive", "hr_admin", "project_manager", "support"), handlers.CreateJobUpdate)
		api.GET("/jobs/:id/updates", middleware.RequireRole("executive", "hr_admin", "finance", "project_manager", "construction_admin", "support"), handlers.ListJobUpdates)

		// Bid routes
		api.GET("/bids", middleware.RequireRole("executive", "hr_admin", "finance", "project_manager", "construction_admin", "support"), handlers.ListBids)
		api.GET("/bids/:id", middleware.RequireRole("executive", "hr_admin", "finance", "project_manager", "construction_admin", "support"), handlers.GetBid)
		api.POST("/bids", middleware.RequireRole("executive", "hr_admin", "project_manager", "support"), handlers.CreateBid)
		api.PUT("/bids/:id", middleware.RequireRole("executive", "hr_admin", "project_manager", "support"), handlers.UpdateBid)
		api.DELETE("/bids/:id", middleware.RequireRole("executive", "support"), handlers.DeleteBid)

		// Superintendent lookup (for dropdowns)
		api.GET("/superintendents", middleware.RequireRole("executive", "hr_admin", "construction_admin", "project_manager", "support"), handlers.ListSuperintendents)

		// Lodging routes
		api.GET("/lodging", middleware.RequireRole("executive", "hr_admin", "construction_admin", "project_manager", "support"), handlers.ListLodging)
		api.GET("/lodging/:id", middleware.RequireRole("executive", "hr_admin", "construction_admin", "project_manager", "support"), handlers.GetLodging)
		api.POST("/lodging", middleware.RequireRole("executive", "hr_admin", "construction_admin", "support"), handlers.CreateLodging)
		api.PUT("/lodging/:id", middleware.RequireRole("executive", "hr_admin", "construction_admin", "support"), handlers.UpdateLodging)
		api.DELETE("/lodging/:id", middleware.RequireRole("executive", "support"), handlers.DeleteLodging)

		// License routes
		api.GET("/licenses", middleware.RequireRole("executive", "hr_admin", "construction_admin", "project_manager", "support"), handlers.ListLicenses)
		api.GET("/licenses/state-summary", middleware.RequireRole("executive", "hr_admin", "construction_admin", "project_manager", "support"), handlers.GetStateSummary)

		// SharePoint auto-discover routes (must be before /licenses/:id)
		api.GET("/licenses/scan/states", middleware.RequireRole("executive", "hr_admin", "construction_admin", "support"), handlers.ListSharePointStates)
		api.POST("/licenses/scan", middleware.RequireRole("executive", "hr_admin", "construction_admin", "support"), handlers.ScanSharePointState)
		api.POST("/licenses/approve-suggestions", middleware.RequireRole("executive", "hr_admin", "construction_admin", "support"), handlers.ApproveSuggestions)

		api.POST("/licenses", middleware.RequireRole("executive", "hr_admin", "construction_admin", "support"), handlers.CreateLicense)
		api.PUT("/licenses/:id", middleware.RequireRole("executive", "hr_admin", "construction_admin", "support"), handlers.UpdateLicense)
		api.DELETE("/licenses/:id", middleware.RequireRole("executive", "support"), handlers.DeleteLicense)

		// License document linking routes
		api.GET("/licenses/:id/documents", middleware.RequireRole("executive", "hr_admin", "construction_admin", "project_manager", "support"), handlers.ListDocuments)
		api.POST("/licenses/:id/documents", middleware.RequireRole("executive", "hr_admin", "construction_admin", "support"), handlers.LinkDocument)
		api.DELETE("/licenses/:id/documents/:docId", middleware.RequireRole("executive", "hr_admin", "construction_admin", "support"), handlers.UnlinkDocument)

		// SharePoint folder browser routes (executive, hr_admin, construction_admin, support)
		api.GET("/sharepoint/status", middleware.RequireRole("executive", "hr_admin", "construction_admin", "support"), handlers.GetSharePointStatus)
		api.GET("/sharepoint/auth", middleware.RequireRole("executive", "hr_admin", "construction_admin", "support"), handlers.InitSharePointAuth)
		api.GET("/sharepoint/browse", middleware.RequireRole("executive", "hr_admin", "construction_admin", "support"), handlers.BrowseSharePoint)
		api.POST("/sharepoint/disconnect", middleware.RequireRole("executive", "hr_admin", "construction_admin", "support"), handlers.DisconnectSharePoint)

		// Email automation routes (executive, construction_admin, support)
		api.GET("/email-automation/status", middleware.RequireRole("executive", "construction_admin", "support"), emailAutomationHandler.GetStatus)
		api.GET("/email-automation/folder-mappings", middleware.RequireRole("executive", "construction_admin", "support"), emailAutomationHandler.ListFolderMappings)
		api.PUT("/email-automation/folder-mappings/:id", middleware.RequireRole("executive", "construction_admin", "support"), emailAutomationHandler.UpdateFolderMapping)
		api.GET("/email-automation/logs", middleware.RequireRole("executive", "construction_admin", "support"), emailAutomationHandler.ListLogs)
		api.POST("/email-automation/sync-folders", middleware.RequireRole("executive", "construction_admin", "support"), emailAutomationHandler.SyncFolders)

		// Training portal routes
		// Trainee view (any authenticated user)
		api.GET("/training/my-calendar", handlers.GetMyTrainingCalendar)

		// Admin training management (executive, hr_admin, and support only)
		api.GET("/training/programs", middleware.RequireRole("executive", "hr_admin", "support"), handlers.ListTrainingPrograms)
		api.GET("/training/programs/:id", middleware.RequireRole("executive", "hr_admin", "support"), handlers.GetTrainingProgram)
		api.POST("/training/programs", middleware.RequireRole("hr_admin", "support"), handlers.CreateTrainingProgram)
		api.PUT("/training/programs/:id", middleware.RequireRole("hr_admin", "support"), handlers.UpdateTrainingProgram)
		api.DELETE("/training/programs/:id", middleware.RequireRole("hr_admin", "support"), handlers.DeleteTrainingProgram)

		// Schedule items (hr_admin and support only)
		api.POST("/training/programs/:id/items", middleware.RequireRole("hr_admin", "support"), handlers.CreateScheduleItem)
		api.PUT("/training/programs/:id/items/:itemId", middleware.RequireRole("hr_admin", "support"), handlers.UpdateScheduleItem)
		api.DELETE("/training/programs/:id/items/:itemId", middleware.RequireRole("hr_admin", "support"), handlers.DeleteScheduleItem)
		api.PUT("/training/programs/:id/items", middleware.RequireRole("hr_admin", "support"), handlers.BulkUpdateScheduleItems)

		// Resources (hr_admin and support only)
		api.POST("/training/programs/:id/resources", middleware.RequireRole("hr_admin", "support"), handlers.CreateResource)
		api.PUT("/training/programs/:id/resources/:resourceId", middleware.RequireRole("hr_admin", "support"), handlers.UpdateResource)
		api.DELETE("/training/programs/:id/resources/:resourceId", middleware.RequireRole("hr_admin", "support"), handlers.DeleteResource)

		// Program preview (admin: see calendar with a simulated start date)
		api.GET("/training/programs/:id/preview", middleware.RequireRole("executive", "hr_admin", "support"), handlers.PreviewProgramCalendar)

		// Trainee assignments (hr_admin and support only)
		api.GET("/training/assignments", middleware.RequireRole("executive", "hr_admin", "support"), handlers.ListAssignments)
		api.POST("/training/assignments", middleware.RequireRole("hr_admin", "support"), handlers.CreateAssignment)
		api.PUT("/training/assignments/:id", middleware.RequireRole("hr_admin", "support"), handlers.UpdateAssignment)
		api.DELETE("/training/assignments/:id", middleware.RequireRole("hr_admin", "support"), handlers.DeleteAssignment)
		api.GET("/training/assignments/:id/calendar", middleware.RequireRole("executive", "hr_admin", "support"), handlers.GetTraineeCalendar)

		// Per-trainee schedule overrides (hr_admin and support only)
		api.GET("/training/assignments/:id/overrides", middleware.RequireRole("executive", "hr_admin", "support"), handlers.ListOverrides)
		api.POST("/training/assignments/:id/overrides", middleware.RequireRole("hr_admin", "support"), handlers.CreateOverride)
		api.PUT("/training/assignments/:id/overrides/:overrideId", middleware.RequireRole("hr_admin", "support"), handlers.UpdateOverride)
		api.DELETE("/training/assignments/:id/overrides/:overrideId", middleware.RequireRole("hr_admin", "support"), handlers.DeleteOverride)
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

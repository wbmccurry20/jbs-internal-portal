package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wbmccurry20/jbs-internal-portal/internal/database"
	"github.com/wbmccurry20/jbs-internal-portal/internal/models"
	"github.com/wbmccurry20/jbs-internal-portal/internal/services"
	"github.com/wbmccurry20/jbs-internal-portal/internal/utils"
)

// UploadReconciliationFiles handles bank and foundation file uploads for reconciliation
func UploadReconciliationFiles(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get tolerance days from form (default: 3)
	toleranceDaysStr := c.DefaultPostForm("tolerance_days", "3")
	toleranceDays, err := strconv.Atoi(toleranceDaysStr)
	if err != nil || toleranceDays < 0 {
		toleranceDays = 3
	}

	// Get uploaded files
	bankFile, err := c.FormFile("bank_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bank file is required"})
		return
	}

	foundationFile, err := c.FormFile("foundation_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Foundation file is required"})
		return
	}

	// Validate file extensions
	bankExt := filepath.Ext(bankFile.Filename)
	foundationExt := filepath.Ext(foundationFile.Filename)
	
	validExts := map[string]bool{".csv": true, ".xlsx": true, ".xls": true}
	if !validExts[bankExt] || !validExts[foundationExt] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only CSV and Excel files are supported"})
		return
	}

	// Sanitize filenames
	safeBankFilename := utils.SanitizeFilename(bankFile.Filename)
	safeFoundationFilename := utils.SanitizeFilename(foundationFile.Filename)

	// Create upload directory
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	os.MkdirAll(uploadDir, 0755)

	// Generate unique filenames
	timestamp := time.Now().Format("20060102_150405")
	bankFilename := fmt.Sprintf("bank_%s_%s", timestamp, safeBankFilename)
	foundationFilename := fmt.Sprintf("foundation_%s_%s", timestamp, safeFoundationFilename)
	bankPath := filepath.Join(uploadDir, bankFilename)
	foundationPath := filepath.Join(uploadDir, foundationFilename)

	// Save files
	if err := c.SaveUploadedFile(bankFile, bankPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save bank file"})
		return
	}

	if err := c.SaveUploadedFile(foundationFile, foundationPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save foundation file"})
		return
	}

	// Create reconciliation job record
	var jobID int
	err = database.DB.QueryRow(`
		INSERT INTO reconciliation_jobs (user_id, bank_filename, foundation_filename, status, tolerance_days)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, userID, bankFile.Filename, foundationFile.Filename, "processing", toleranceDays).Scan(&jobID)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job record"})
		return
	}

	// Process reconciliation
	outputFilename := fmt.Sprintf("reconciliation_%s.xlsx", timestamp)
	outputPath := filepath.Join(uploadDir, outputFilename)

	// Load transactions
	bankTrans, err := services.LoadBankTransactions(bankPath)
	if err != nil {
		database.DB.Exec(`
			UPDATE reconciliation_jobs 
			SET status = $1, error_log = $2, completed_at = $3
			WHERE id = $4
		`, "failed", fmt.Sprintf("Failed to load bank transactions: %s", err.Error()), time.Now(), jobID)

		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to load bank transactions: %s", err.Error())})
		return
	}

	foundationTrans, err := services.LoadFoundationTransactions(foundationPath)
	if err != nil {
		database.DB.Exec(`
			UPDATE reconciliation_jobs 
			SET status = $1, error_log = $2, completed_at = $3
			WHERE id = $4
		`, "failed", fmt.Sprintf("Failed to load foundation transactions: %s", err.Error()), time.Now(), jobID)

		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to load foundation transactions: %s", err.Error())})
		return
	}

	// Run reconciliation
	engine := services.NewReconciliationEngine(toleranceDays, 0.01)
	report := engine.Reconcile(bankTrans, foundationTrans)

	// Generate Excel report
	generator := services.NewExcelReportGenerator()
	err = generator.GenerateReconciliationExcel(report, outputPath)
	if err != nil {
		database.DB.Exec(`
			UPDATE reconciliation_jobs 
			SET status = $1, error_log = $2, completed_at = $3
			WHERE id = $4
		`, "failed", fmt.Sprintf("Failed to generate report: %s", err.Error()), time.Now(), jobID)

		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to generate report: %s", err.Error())})
		return
	}

	// Update job as completed
	now := time.Now()
	_, err = database.DB.Exec(`
		UPDATE reconciliation_jobs 
		SET status = $1, matched_count = $2, void_count = $3, 
		    ambiguous_void_count = $4, output_file_path = $5, completed_at = $6
		WHERE id = $7
	`, "completed", report.TotalMatched(), len(report.VoidPairs), 
		len(report.AmbiguousVoids), outputPath, now, jobID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update job record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id":              jobID,
		"matched_count":       report.TotalMatched(),
		"bank_only_count":     report.TotalBankOnly(),
		"foundation_only_count": report.TotalFoundationOnly(),
		"void_count":          len(report.VoidPairs),
		"ambiguous_void_count": len(report.AmbiguousVoids),
		"potential_match_count": len(report.PotentialMatches),
		"match_rate":          fmt.Sprintf("%.1f%%", report.MatchRate()),
		"output_file":         outputFilename,
		"download_url":        fmt.Sprintf("/api/reconciliation/download/%d", jobID),
	})
}

// GetReconciliationHistory returns reconciliation job history for the user
func GetReconciliationHistory(c *gin.Context) {
	userID, _ := c.Get("userID")

	rows, err := database.DB.Query(`
		SELECT id, bank_filename, foundation_filename, status, tolerance_days,
		       matched_count, void_count, ambiguous_void_count, created_at, completed_at
		FROM reconciliation_jobs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 50
	`, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
		return
	}
	defer rows.Close()

	jobs := make([]models.ReconciliationJob, 0)
	for rows.Next() {
		var job models.ReconciliationJob
		err := rows.Scan(&job.ID, &job.BankFilename, &job.FoundationFilename,
			&job.Status, &job.ToleranceDays, &job.MatchedCount, &job.VoidCount,
			&job.AmbiguousVoidCount, &job.CreatedAt, &job.CompletedAt)
		if err != nil {
			continue
		}
		jobs = append(jobs, job)
	}

	c.JSON(http.StatusOK, jobs)
}

// DownloadReconciliationResult downloads the reconciliation Excel report
func DownloadReconciliationResult(c *gin.Context) {
	jobIDStr := c.Param("id")
	jobID, err := strconv.Atoi(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	userID, _ := c.Get("userID")

	// Verify job belongs to user and get file path
	var outputPath string
	err = database.DB.QueryRow(`
		SELECT output_file_path
		FROM reconciliation_jobs
		WHERE id = $1 AND user_id = $2 AND status = 'completed'
	`, jobID, userID).Scan(&outputPath)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found or not completed"})
		return
	}

	// Check file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Output file not found"})
		return
	}

	// Serve the file
	c.FileAttachment(outputPath, filepath.Base(outputPath))
}

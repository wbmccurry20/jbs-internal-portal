package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wbmccurry20/jbs-internal-portal/internal/database"
	"github.com/wbmccurry20/jbs-internal-portal/internal/models"
	"github.com/wbmccurry20/jbs-internal-portal/internal/services"
	"github.com/wbmccurry20/jbs-internal-portal/internal/utils"
)

// UploadConcurFile handles Concur Excel file upload and conversion
func UploadConcurFile(c *gin.Context) {
	// Get user from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get vendor ID from form (default: 138)
	vendorID := c.DefaultPostForm("vendor_id", "138")

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// Validate file
	if err := utils.ValidateExcelFile(file); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Sanitize filename
	safeFilename := utils.SanitizeFilename(file.Filename)

	// Create upload directory if it doesn't exist
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(500, gin.H{"error": "Failed to create upload directory"})
		return
	}

	// Generate unique filename
	timestamp := time.Now().Format("20060102_150405")
	inputFilename := fmt.Sprintf("concur_%s_%s", timestamp, safeFilename)
	inputPath := filepath.Join(uploadDir, inputFilename)

	// Save uploaded file
	if err := c.SaveUploadedFile(file, inputPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Create conversion job record
	var jobID int
	err = database.DB.QueryRow(`
		INSERT INTO conversion_jobs (user_id, filename, status, vendor_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, userID, file.Filename, "processing", vendorID).Scan(&jobID)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job record"})
		return
	}

	// Process the file
	outputFilename := fmt.Sprintf("foundation_%s.csv", timestamp)
	outputPath := filepath.Join(uploadDir, outputFilename)

	converter := services.NewConcurConverter(vendorID)
	result, err := converter.ProcessFile(inputPath, outputPath)

	if err != nil {
		// Update job as failed
		database.DB.Exec(`
			UPDATE conversion_jobs 
			SET status = $1, error_log = $2, completed_at = $3
			WHERE id = $4
		`, "failed", err.Error(), time.Now(), jobID)

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update job as completed
	now := time.Now()
	_, err = database.DB.Exec(`
		UPDATE conversion_jobs 
		SET status = $1, rows_processed = $2, rows_skipped = $3, 
		    output_file_path = $4, completed_at = $5
		WHERE id = $6
	`, "completed", result.RowsProcessed, result.RowsSkipped, outputPath, now, jobID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update job record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id":         jobID,
		"rows_processed": result.RowsProcessed,
		"rows_skipped":   result.RowsSkipped,
		"issues":         result.Issues,
		"output_file":    outputFilename,
		"download_url":   fmt.Sprintf("/api/download/%d", jobID),
	})
}

// GetConversionHistory returns conversion job history for the user
func GetConversionHistory(c *gin.Context) {
	userID, _ := c.Get("userID")

	rows, err := database.DB.Query(`
		SELECT id, filename, status, vendor_id, rows_processed, rows_skipped, 
		       created_at, completed_at
		FROM conversion_jobs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 50
	`, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
		return
	}
	defer rows.Close()

	jobs := make([]models.ConversionJob, 0)
	for rows.Next() {
		var job models.ConversionJob
		err := rows.Scan(&job.ID, &job.Filename, &job.Status, &job.VendorID,
			&job.RowsProcessed, &job.RowsSkipped, &job.CreatedAt, &job.CompletedAt)
		if err != nil {
			continue
		}
		jobs = append(jobs, job)
	}

	c.JSON(http.StatusOK, jobs)
}

// DownloadConversionResult downloads the converted CSV file
func DownloadConversionResult(c *gin.Context) {
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
		FROM conversion_jobs
		WHERE id = $1 AND user_id = $2 AND status = 'completed'
	`, jobID, userID).Scan(&outputPath)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found or not completed"})
		return
	}

	// Validate path to prevent directory traversal
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	cleanPath := filepath.Clean(outputPath)
	absUploadDir, _ := filepath.Abs(uploadDir)
	absOutputPath, _ := filepath.Abs(cleanPath)
	
	// Ensure file is within upload directory
	if !strings.HasPrefix(absOutputPath, absUploadDir) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Check file exists
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Output file not found"})
		return
	}

	// Serve the file
	c.FileAttachment(cleanPath, filepath.Base(cleanPath))
}

// DeleteConversionJob deletes a conversion job and its associated files
func DeleteConversionJob(c *gin.Context) {
	jobIDStr := c.Param("id")
	jobID, err := strconv.Atoi(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	userID, _ := c.Get("userID")

	// Get job details to verify ownership and get file paths
	var outputPath string
	var ownerID int
	err = database.DB.QueryRow(`
		SELECT user_id, output_file_path
		FROM conversion_jobs
		WHERE id = $1
	`, jobID).Scan(&ownerID, &outputPath)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	// Verify user owns this job
	if ownerID != userID.(int) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own jobs"})
		return
	}

	// Delete the output file if it exists
	if outputPath != "" {
		uploadDir := os.Getenv("UPLOAD_DIR")
		if uploadDir == "" {
			uploadDir = "./uploads"
		}
		cleanPath := filepath.Clean(outputPath)
		absUploadDir, _ := filepath.Abs(uploadDir)
		absOutputPath, _ := filepath.Abs(cleanPath)
		
		// Only delete if within upload directory
		if strings.HasPrefix(absOutputPath, absUploadDir) {
			os.Remove(cleanPath)
		}
	}

	// Delete the database record
	_, err = database.DB.Exec(`DELETE FROM conversion_jobs WHERE id = $1`, jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete job"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job deleted successfully"})
}

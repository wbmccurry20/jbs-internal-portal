package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ONEDRIVE SYNC HANDLERS
// To be implemented when Azure credentials are available
//
// Prerequisites:
// - Azure App Registration with OneDrive API access
// - Client ID, Client Secret, Tenant ID in .env
// - OneDrive folder path containing license PDFs
//
// Implementation plan:
// 1. Authenticate with Azure using OAuth 2.0
// 2. List files in configured OneDrive folder
// 3. For each PDF:
//    - Download file temporarily
//    - Parse metadata (state, type, number, dates) from filename or OCR
//    - Check if license exists in DB (by state + license_number)
//    - Insert new or update existing record
//    - Store onedrive_file_path for reference
// 4. Return sync summary (added, updated, errors)

// SyncFromOneDrive triggers a full sync from OneDrive to database
func SyncFromOneDrive(c *gin.Context) {
	// TODO: Implement when Azure credentials available
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "OneDrive sync not yet configured",
		"message": "Waiting for Azure credentials. Use manual CRUD for now.",
		"status":  "pending_credentials",
	})
}

// GetOneDriveSyncStatus returns the last sync info
func GetOneDriveSyncStatus(c *gin.Context) {
	// TODO: Query onedrive_sync_log table (from migration 009)
	c.JSON(http.StatusOK, gin.H{
		"status":        "not_configured",
		"last_sync":     nil,
		"total_files":   0,
		"last_error":    nil,
		"configuration": "pending_azure_credentials",
	})
}

// UploadToOneDrive uploads a license PDF to OneDrive (future)
func UploadToOneDrive(c *gin.Context) {
	// TODO: Implement bi-directional sync
	// When user uploads a license PDF through portal, also save to OneDrive
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "OneDrive upload not yet implemented",
	})
}

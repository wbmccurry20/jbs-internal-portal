package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wbmccurry20/jbs-internal-portal/internal/database"
)

// LICENSE MANAGEMENT HANDLERS
//
// Phase 1 (done): Portal CRUD — manual license entry, US map, stats
// Phase 2 (done): SharePoint folder browser — browse JBS Licensing folder from portal
// Phase 3 (in progress): Auto-discover + document linking
// Phase 4: Auto-suggest — scan folder hierarchy to propose new license records

// ListLicenses returns all licenses with optional filters
func ListLicenses(c *gin.Context) {
	query := `
		SELECT
			sl.id, sl.state, sl.license_type, sl.license_number, sl.entity_name,
			sl.issue_date, sl.expiration_date, sl.status, sl.renewal_fee, sl.notes,
			sl.city, sl.is_city_license, sl.is_active,
			sl.created_at, sl.updated_at,
			COALESCE(dc.doc_count, 0) AS document_count
		FROM state_licenses sl
		LEFT JOIN (
			SELECT license_id, COUNT(*) AS doc_count FROM license_documents GROUP BY license_id
		) dc ON dc.license_id = sl.id
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	// Filter by state
	if state := c.Query("state"); state != "" {
		query += fmt.Sprintf(" AND sl.state = $%d", argPos)
		args = append(args, state)
		argPos++
	}

	// Filter by status
	if status := c.Query("status"); status != "" {
		query += fmt.Sprintf(" AND sl.status = $%d", argPos)
		args = append(args, status)
		argPos++
	}

	// Filter by expiring soon (days parameter) - parse to int to prevent SQL injection
	if days := c.Query("expiring_days"); days != "" {
		daysInt, err := strconv.Atoi(days)
		if err != nil || daysInt < 0 || daysInt > 365 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid expiring_days parameter"})
			return
		}
		query += fmt.Sprintf(" AND sl.expiration_date <= CURRENT_DATE + INTERVAL '%d days'", daysInt)
		query += " AND sl.expiration_date >= CURRENT_DATE"
	}

	query += " ORDER BY sl.state, sl.expiration_date"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		log.Printf("Error fetching licenses: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch licenses"})
		return
	}
	defer rows.Close()

	// Initialize as empty slice so JSON serializes to [] not null
	licenses := make([]map[string]interface{}, 0)
	for rows.Next() {
		var (
			id                                                      int
			state, licenseType                                      string
			licenseNumber, entityName                               sql.NullString
			issueDate, expirationDate                               sql.NullTime
			status                                                  sql.NullString
			renewalFee                                              sql.NullFloat64
			notes                                                   sql.NullString
			city                                                    sql.NullString
			isCityLicense, isActive                                 sql.NullBool
			createdAt, updatedAt                                    time.Time
			documentCount                                           int
		)

		err := rows.Scan(
			&id, &state, &licenseType, &licenseNumber, &entityName,
			&issueDate, &expirationDate, &status, &renewalFee, &notes,
			&city, &isCityLicense, &isActive,
			&createdAt, &updatedAt,
			&documentCount,
		)
		if err != nil {
			log.Printf("Error scanning license row: %v", err)
			continue
		}

		license := map[string]interface{}{
			"id":              id,
			"state":           state,
			"license_type":    licenseType,
			"license_number":  licenseNumber.String,
			"entity_name":     entityName.String,
			"status":          status.String,
			"renewal_fee":     renewalFee.Float64,
			"notes":           notes.String,
			"city":            city.String,
			"is_city_license": isCityLicense.Bool,
			"is_active":       isActive.Bool,
			"document_count":  documentCount,
			"created_at":      createdAt,
			"updated_at":      updatedAt,
		}

		if issueDate.Valid {
			license["issue_date"] = issueDate.Time.Format("2006-01-02")
		}
		if expirationDate.Valid {
			license["expiration_date"] = expirationDate.Time.Format("2006-01-02")
		}

		licenses = append(licenses, license)
	}

	c.JSON(http.StatusOK, licenses)
}

// GetStateSummary returns a summary of all 50 states
func GetStateSummary(c *gin.Context) {
	query := `
		SELECT 
			state,
			COUNT(*) as total_licenses,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active,
			COUNT(CASE WHEN status = 'expiring' THEN 1 END) as expiring,
			COUNT(CASE WHEN status = 'expired' THEN 1 END) as expired,
			COUNT(CASE WHEN is_city_license = true THEN 1 END) as city_licenses,
			COUNT(CASE WHEN is_city_license = false OR is_city_license IS NULL THEN 1 END) as state_licenses
		FROM state_licenses
		GROUP BY state
		ORDER BY state
	`

	rows, err := database.DB.Query(query)
	if err != nil {
		log.Printf("Error fetching state summary: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch state summary"})
		return
	}
	defer rows.Close()

	summary := make([]map[string]interface{}, 0)
	for rows.Next() {
		var state string
		var total, active, expiring, expired, cityLicenses, stateLicenses int

		err := rows.Scan(&state, &total, &active, &expiring, &expired, &cityLicenses, &stateLicenses)
		if err != nil {
			log.Printf("ERROR: license state summary scan failed: %v", err)
			continue
		}

		summary = append(summary, map[string]interface{}{
			"state":          state,
			"total":          total,
			"active":         active,
			"expiring":       expiring,
			"expired":        expired,
			"city_licenses":  cityLicenses,
			"state_licenses": stateLicenses,
		})
	}

	c.JSON(http.StatusOK, summary)
}

// CreateLicense creates a new license
func CreateLicense(c *gin.Context) {
	var req struct {
		State          string  `json:"state" binding:"required"`
		LicenseType    string  `json:"license_type" binding:"required"`
		LicenseNumber  string  `json:"license_number"`
		EntityName     string  `json:"entity_name"`
		IssueDate      string  `json:"issue_date"`
		ExpirationDate string  `json:"expiration_date"`
		Status         string  `json:"status"`
		RenewalFee     float64 `json:"renewal_fee"`
		Notes          string  `json:"notes"`
		City           string  `json:"city"`
		IsCityLicense  bool    `json:"is_city_license"`
		IsActive       *bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Convert empty strings to null for optional fields
	var licenseNumber, entityName, issueDate, expirationDate, notes, city interface{}
	
	if req.LicenseNumber != "" {
		licenseNumber = req.LicenseNumber
	}
	if req.EntityName != "" {
		entityName = req.EntityName
	}
	if req.IssueDate != "" {
		issueDate = req.IssueDate
	}
	if req.ExpirationDate != "" {
		expirationDate = req.ExpirationDate
	}
	if req.Notes != "" {
		notes = req.Notes
	}
	if req.City != "" {
		city = req.City
	}

	// Default is_active to true if not provided
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	// Auto-calculate status if not provided
	status := req.Status
	if status == "" && req.ExpirationDate != "" {
		expDate, err := time.Parse("2006-01-02", req.ExpirationDate)
		if err == nil {
			daysUntilExpiration := int(time.Until(expDate).Hours() / 24)
			if daysUntilExpiration < 0 {
				status = "expired"
			} else if daysUntilExpiration <= 90 {
				status = "expiring"
			} else {
				status = "active"
			}
		}
	}
	
	// Default to active if still empty
	if status == "" {
		status = "active"
	}

	query := `
		INSERT INTO state_licenses 
			(state, license_type, license_number, entity_name, issue_date, 
			 expiration_date, status, renewal_fee, notes, city, is_city_license, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`

	var id int
	err := database.DB.QueryRow(
		query,
		req.State, req.LicenseType, licenseNumber, entityName,
		issueDate, expirationDate, status, req.RenewalFee, notes,
		city, req.IsCityLicense, isActive,
	).Scan(&id)

	if err != nil {
		log.Printf("Error creating license: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create license"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "License created successfully"})
}

// UpdateLicense updates an existing license
func UpdateLicense(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		State          string  `json:"state"`
		LicenseType    string  `json:"license_type"`
		LicenseNumber  string  `json:"license_number"`
		EntityName     string  `json:"entity_name"`
		IssueDate      string  `json:"issue_date"`
		ExpirationDate string  `json:"expiration_date"`
		Status         string  `json:"status"`
		RenewalFee     float64 `json:"renewal_fee"`
		Notes          string  `json:"notes"`
		City           string  `json:"city"`
		IsCityLicense  bool    `json:"is_city_license"`
		IsActive       *bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Default is_active to true if not provided
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	// Convert empty strings to null for optional fields
	var licenseNumber, entityName, issueDate, expirationDate, notes, city interface{}
	if req.LicenseNumber != "" {
		licenseNumber = req.LicenseNumber
	}
	if req.EntityName != "" {
		entityName = req.EntityName
	}
	if req.IssueDate != "" {
		issueDate = req.IssueDate
	}
	if req.ExpirationDate != "" {
		expirationDate = req.ExpirationDate
	}
	if req.Notes != "" {
		notes = req.Notes
	}
	if req.City != "" {
		city = req.City
	}

	query := `
		UPDATE state_licenses
		SET state = $1, license_type = $2, license_number = $3, entity_name = $4,
		    issue_date = $5, expiration_date = $6, status = $7, renewal_fee = $8,
		    notes = $9, city = $10, is_city_license = $11, is_active = $12,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $13
	`

	result, err := database.DB.Exec(
		query,
		req.State, req.LicenseType, licenseNumber, entityName,
		issueDate, expirationDate, req.Status, req.RenewalFee, notes,
		city, req.IsCityLicense, isActive,
		id,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update license"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "License not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "License updated successfully"})
}

// DeleteLicense deletes a license
func DeleteLicense(c *gin.Context) {
	id := c.Param("id")

	result, err := database.DB.Exec("DELETE FROM state_licenses WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete license"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "License not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "License deleted successfully"})
}

// ============================================================
// AUTO-DISCOVER: Scan SharePoint folders to create license records
// ============================================================

// ListSharePointStates returns the list of state folders on SharePoint
func ListSharePointStates(c *gin.Context) {
	if graphClient == nil || !graphClient.IsConnected() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SharePoint not connected"})
		return
	}

	folders, err := graphClient.ListStateFolders()
	if err != nil {
		log.Printf("Error listing SharePoint state folders: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list state folders"})
		return
	}

	// Return simplified list for the dropdown
	states := make([]map[string]interface{}, 0, len(folders))
	for _, f := range folders {
		states = append(states, map[string]interface{}{
			"name":    f.Name,
			"web_url": f.WebURL,
		})
	}

	c.JSON(http.StatusOK, states)
}

// ScanSharePointState scans a single state's SharePoint folder and returns suggestions
func ScanSharePointState(c *gin.Context) {
	if graphClient == nil || !graphClient.IsConnected() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SharePoint not connected"})
		return
	}

	var req struct {
		FolderName string `json:"folder_name" binding:"required"`
		WebURL     string `json:"web_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "folder_name is required"})
		return
	}

	suggestions, skipped, err := graphClient.ScanStateFolder(req.FolderName, req.WebURL)
	if err != nil {
		log.Printf("Error scanning state folder %q: %v", req.FolderName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan folder: " + err.Error()})
		return
	}

	if len(suggestions) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"suggestions":     []interface{}{},
			"already_tracked": []interface{}{},
			"skipped_folders": skipped,
		})
		return
	}

	// Cross-reference with existing licenses for this state
	stateCode := suggestions[0].State
	rows, err := database.DB.Query(
		"SELECT state, COALESCE(city, '') as city, COALESCE(is_city_license, false) as is_city FROM state_licenses WHERE state = $1",
		stateCode,
	)
	if err != nil {
		log.Printf("Error querying existing licenses for state %s: %v", stateCode, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing licenses"})
		return
	}
	defer rows.Close()

	// Build a set of existing (state, city, is_city) combos
	type licenseKey struct {
		city   string
		isCity bool
	}
	existing := make(map[licenseKey]bool)
	for rows.Next() {
		var state, city string
		var isCity bool
		if err := rows.Scan(&state, &city, &isCity); err != nil {
			continue
		}
		existing[licenseKey{city: city, isCity: isCity}] = true
	}

	// Split suggestions into new vs already tracked
	newSuggestions := make([]map[string]interface{}, 0)
	alreadyTracked := make([]map[string]interface{}, 0)
	for _, s := range suggestions {
		item := map[string]interface{}{
			"state":           s.State,
			"state_name":      s.StateName,
			"city":            s.City,
			"folder_path":     s.FolderPath,
			"web_url":         s.WebURL,
			"is_city_license": s.IsCityLicense,
		}

		key := licenseKey{city: s.City, isCity: s.IsCityLicense}
		if existing[key] {
			alreadyTracked = append(alreadyTracked, item)
		} else {
			newSuggestions = append(newSuggestions, item)
		}
	}

	// Log the scan
	database.DB.Exec(
		"INSERT INTO onedrive_sync_log (sync_type, records_processed, records_created, errors, started_at, completed_at) VALUES ($1, $2, 0, $3, NOW(), NOW())",
		"state_scan:"+stateCode, len(suggestions), fmt.Sprintf("skipped: %v", skipped),
	)

	c.JSON(http.StatusOK, gin.H{
		"suggestions":     newSuggestions,
		"already_tracked": alreadyTracked,
		"skipped_folders": skipped,
	})
}

// ApproveSuggestions creates license records from approved SharePoint scan results
func ApproveSuggestions(c *gin.Context) {
	var req struct {
		Suggestions []struct {
			State         string `json:"state" binding:"required"`
			City          string `json:"city"`
			IsCityLicense bool   `json:"is_city_license"`
			FolderPath    string `json:"folder_path"`
			LicenseType   string `json:"license_type"`
			EntityName    string `json:"entity_name"`
			Notes         string `json:"notes"`
		} `json:"suggestions" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if len(req.Suggestions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No suggestions to approve"})
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	ids := make([]int, 0, len(req.Suggestions))
	for _, s := range req.Suggestions {
		// Default license type and entity name
		licenseType := s.LicenseType
		if licenseType == "" {
			licenseType = "General Contractor"
		}
		entityName := s.EntityName
		if entityName == "" {
			entityName = "JBS Construction Group"
		}

		var city interface{}
		if s.City != "" {
			city = s.City
		}

		var notes interface{}
		if s.Notes != "" {
			notes = s.Notes
		}

		var folderPath interface{}
		if s.FolderPath != "" {
			folderPath = s.FolderPath
		}

		var id int
		err := tx.QueryRow(`
			INSERT INTO state_licenses
				(state, license_type, entity_name, city, is_city_license, is_active, status, onedrive_file_path, last_synced_at, notes)
			VALUES ($1, $2, $3, $4, $5, true, 'active', $6, NOW(), $7)
			RETURNING id
		`, s.State, licenseType, entityName, city, s.IsCityLicense, folderPath, notes).Scan(&id)
		if err != nil {
			log.Printf("Error creating license from suggestion: %v", err)
			continue
		}
		ids = append(ids, id)
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit licenses"})
		return
	}

	// Update sync log
	if len(ids) > 0 {
		database.DB.Exec(
			"UPDATE onedrive_sync_log SET records_created = $1 WHERE id = (SELECT id FROM onedrive_sync_log ORDER BY id DESC LIMIT 1)",
			len(ids),
		)
	}

	c.JSON(http.StatusCreated, gin.H{
		"created": len(ids),
		"ids":     ids,
	})
}

// ============================================================
// DOCUMENT LINKING: Attach SharePoint files to license records
// ============================================================

// ListDocuments returns all documents linked to a specific license
func ListDocuments(c *gin.Context) {
	licenseID := c.Param("id")

	rows, err := database.DB.Query(
		`SELECT id, license_id, document_name, document_url, document_type, folder_path, linked_at
		 FROM license_documents WHERE license_id = $1 ORDER BY linked_at DESC`,
		licenseID,
	)
	if err != nil {
		log.Printf("Error fetching documents for license %s: %v", licenseID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch documents"})
		return
	}
	defer rows.Close()

	docs := make([]map[string]interface{}, 0)
	for rows.Next() {
		var (
			id, lid    int
			name, url  string
			docType    sql.NullString
			folderPath sql.NullString
			linkedAt   time.Time
		)
		if err := rows.Scan(&id, &lid, &name, &url, &docType, &folderPath, &linkedAt); err != nil {
			continue
		}
		docs = append(docs, map[string]interface{}{
			"id":            id,
			"license_id":    lid,
			"document_name": name,
			"document_url":  url,
			"document_type": docType.String,
			"folder_path":   folderPath.String,
			"linked_at":     linkedAt,
		})
	}

	c.JSON(http.StatusOK, docs)
}

// LinkDocument attaches a SharePoint document to a license record
func LinkDocument(c *gin.Context) {
	licenseID := c.Param("id")

	var req struct {
		DocumentName      string `json:"document_name" binding:"required"`
		DocumentURL       string `json:"document_url" binding:"required"`
		DocumentType      string `json:"document_type"`
		SharePointItemID  string `json:"sharepoint_item_id"`
		FolderPath        string `json:"folder_path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document_name and document_url are required"})
		return
	}

	// Default document type
	docType := req.DocumentType
	if docType == "" {
		docType = "other"
	}

	// Get user ID from auth context
	userIDVal, _ := c.Get("userID")
	var linkedBy interface{}
	if uid, ok := userIDVal.(int); ok {
		linkedBy = uid
	}

	var id int
	err := database.DB.QueryRow(
		`INSERT INTO license_documents (license_id, document_name, document_url, document_type, sharepoint_item_id, folder_path, linked_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		licenseID, req.DocumentName, req.DocumentURL, docType, req.SharePointItemID, req.FolderPath, linkedBy,
	).Scan(&id)
	if err != nil {
		log.Printf("Error linking document to license %s: %v", licenseID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to link document"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Document linked successfully"})
}

// UnlinkDocument removes a document link from a license
func UnlinkDocument(c *gin.Context) {
	licenseID := c.Param("id")
	docID := c.Param("docId")

	result, err := database.DB.Exec(
		"DELETE FROM license_documents WHERE id = $1 AND license_id = $2",
		docID, licenseID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unlink document"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Document unlinked"})
}

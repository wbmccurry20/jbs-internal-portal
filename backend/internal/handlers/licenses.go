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
// Current: Portal-first CRUD operations for manual license entry
// Future: OneDrive sync integration
//   - Auto-import license PDFs from Azure OneDrive folder
//   - Parse metadata from file names or embedded text
//   - Store onedrive_file_path for reference
//   - Sync status tracking (last_synced_at)
//   - Bi-directional sync: portal updates → OneDrive, OneDrive changes → portal

// ListLicenses returns all licenses with optional filters
func ListLicenses(c *gin.Context) {
	query := `
		SELECT 
			id, state, license_type, license_number, entity_name,
			issue_date, expiration_date, status, renewal_fee, notes,
			city, is_city_license, is_active,
			created_at, updated_at
		FROM state_licenses
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	// Filter by state
	if state := c.Query("state"); state != "" {
		query += fmt.Sprintf(" AND state = $%d", argPos)
		args = append(args, state)
		argPos++
	}

	// Filter by status
	if status := c.Query("status"); status != "" {
		query += fmt.Sprintf(" AND status = $%d", argPos)
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
		query += fmt.Sprintf(" AND expiration_date <= CURRENT_DATE + INTERVAL '%d days'", daysInt)
		query += " AND expiration_date >= CURRENT_DATE"
	}

	query += " ORDER BY state, expiration_date"

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
		)

		err := rows.Scan(
			&id, &state, &licenseType, &licenseNumber, &entityName,
			&issueDate, &expirationDate, &status, &renewalFee, &notes,
			&city, &isCityLicense, &isActive,
			&createdAt, &updatedAt,
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

package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wbmccurry20/jbs-internal-portal/internal/database"
)

// ListLicenses returns all licenses with optional filters
func ListLicenses(c *gin.Context) {
	query := `
		SELECT 
			id, state, license_type, license_number, entity_name,
			issue_date, expiration_date, status, renewal_fee, notes,
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

	// Filter by expiring soon (days parameter)
	if days := c.Query("expiring_days"); days != "" {
		query += fmt.Sprintf(" AND expiration_date <= CURRENT_DATE + INTERVAL '%s days'", days)
		query += " AND expiration_date >= CURRENT_DATE"
	}

	query += " ORDER BY state, expiration_date"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch licenses"})
		return
	}
	defer rows.Close()

	var licenses []map[string]interface{}
	for rows.Next() {
		var (
			id, state, licenseType, licenseNumber, entityName       string
			issueDate, expirationDate                               sql.NullTime
			status                                                   sql.NullString
			renewalFee                                              sql.NullFloat64
			notes                                                    sql.NullString
			createdAt, updatedAt                                    time.Time
		)

		err := rows.Scan(
			&id, &state, &licenseType, &licenseNumber, &entityName,
			&issueDate, &expirationDate, &status, &renewalFee, &notes,
			&createdAt, &updatedAt,
		)
		if err != nil {
			continue
		}

		license := map[string]interface{}{
			"id":             id,
			"state":          state,
			"license_type":   licenseType,
			"license_number": licenseNumber,
			"entity_name":    entityName,
			"status":         status.String,
			"renewal_fee":    renewalFee.Float64,
			"notes":          notes.String,
			"created_at":     createdAt,
			"updated_at":     updatedAt,
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
			COUNT(CASE WHEN status = 'expired' THEN 1 END) as expired
		FROM state_licenses
		GROUP BY state
		ORDER BY state
	`

	rows, err := database.DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch state summary"})
		return
	}
	defer rows.Close()

	var summary []map[string]interface{}
	for rows.Next() {
		var state string
		var total, active, expiring, expired int

		err := rows.Scan(&state, &total, &active, &expiring, &expired)
		if err != nil {
			continue
		}

		summary = append(summary, map[string]interface{}{
			"state":    state,
			"total":    total,
			"active":   active,
			"expiring": expiring,
			"expired":  expired,
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
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert empty strings to null for optional fields
	var licenseNumber, entityName, issueDate, expirationDate, notes interface{}
	
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
			 expiration_date, status, renewal_fee, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	var id int
	err := database.DB.QueryRow(
		query,
		req.State, req.LicenseType, licenseNumber, entityName,
		issueDate, expirationDate, status, req.RenewalFee, notes,
	).Scan(&id)

	if err != nil {
		fmt.Printf("Error creating license: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create license", "details": err.Error()})
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
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `
		UPDATE state_licenses
		SET state = $1, license_type = $2, license_number = $3, entity_name = $4,
		    issue_date = $5, expiration_date = $6, status = $7, renewal_fee = $8,
		    notes = $9, updated_at = CURRENT_TIMESTAMP
		WHERE id = $10
	`

	result, err := database.DB.Exec(
		query,
		req.State, req.LicenseType, req.LicenseNumber, req.EntityName,
		req.IssueDate, req.ExpirationDate, req.Status, req.RenewalFee, req.Notes,
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

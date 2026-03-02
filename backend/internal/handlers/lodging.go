package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wbmccurry20/jbs-internal-portal/internal/database"
	"github.com/wbmccurry20/jbs-internal-portal/internal/models"
)

// ListLodging returns a list of lodging records with optional filtering
func ListLodging(c *gin.Context) {
	query := `
		SELECT 
			l.id, l.superintendent_id, l.job_id, l.location,
			l.check_in_date, l.check_out_date, l.property_link,
			l.address, l.jobsite_address,
			l.created_at, l.updated_at,
			s.id as s_id, s.name as s_name,
			j.id as j_id, j.job_number, j.job_name
		FROM superintendent_lodging l
		LEFT JOIN superintendents s ON l.superintendent_id = s.id
		LEFT JOIN jobs j ON l.job_id = j.id
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	// Apply filters
	if superintendentID := c.Query("superintendent_id"); superintendentID != "" {
		query += fmt.Sprintf(" AND l.superintendent_id = $%d", argPos)
		args = append(args, superintendentID)
		argPos++
	}
	if jobID := c.Query("job_id"); jobID != "" {
		query += fmt.Sprintf(" AND l.job_id = $%d", argPos)
		args = append(args, jobID)
		argPos++
	}

	// Active lodging only (currently checked in)
	if active := c.Query("active"); active == "true" {
		query += " AND l.check_in_date <= NOW() AND (l.check_out_date IS NULL OR l.check_out_date >= NOW())"
	}

	query += " ORDER BY l.check_in_date DESC NULLS LAST"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch lodging"})
		return
	}
	defer rows.Close()

	var lodgings []map[string]interface{}
	for rows.Next() {
		var l models.SuperintendentLodging
		var sID, jID sql.NullInt64
		var sName, jNumber, jName, location, propertyLink, address, jobsiteAddress sql.NullString

		err := rows.Scan(
			&l.ID, &l.SuperintendentID, &l.JobID, &location,
			&l.CheckInDate, &l.CheckOutDate, &propertyLink,
			&address, &jobsiteAddress,
			&l.CreatedAt, &l.UpdatedAt,
			&sID, &sName, &jID, &jNumber, &jName,
		)
		if err != nil {
			continue
		}

		lodging := map[string]interface{}{
			"id":               l.ID,
			"superintendent_id": l.SuperintendentID,
			"job_id":           l.JobID,
			"location":         location.String,
			"check_in_date":    l.CheckInDate,
			"check_out_date":   l.CheckOutDate,
			"property_link":    propertyLink.String,
			"address":          address.String,
			"jobsite_address":  jobsiteAddress.String,
			"created_at":       l.CreatedAt,
			"updated_at":       l.UpdatedAt,
		}

		if sName.Valid {
			lodging["superintendent"] = map[string]interface{}{
				"id":   sID.Int64,
				"name": sName.String,
			}
		}
		if jNumber.Valid {
			lodging["job"] = map[string]interface{}{
				"id":         jID.Int64,
				"job_number": jNumber.String,
				"job_name":   jName.String,
			}
		}

		lodgings = append(lodgings, lodging)
	}

	c.JSON(http.StatusOK, lodgings)
}

// GetLodging returns a single lodging record by ID
func GetLodging(c *gin.Context) {
	id := c.Param("id")

	query := `
		SELECT 
			l.id, l.superintendent_id, l.job_id, l.location,
			l.check_in_date, l.check_out_date, l.property_link,
			l.address, l.jobsite_address,
			l.created_at, l.updated_at,
			s.id as s_id, s.name as s_name, s.phone as s_phone, s.email as s_email,
			j.id as j_id, j.job_number, j.job_name, j.city, j.state
		FROM superintendent_lodging l
		LEFT JOIN superintendents s ON l.superintendent_id = s.id
		LEFT JOIN jobs j ON l.job_id = j.id
		WHERE l.id = $1
	`

	var l models.SuperintendentLodging
	var sID, jID sql.NullInt64
	var sName, sPhone, sEmail, jNumber, jName, jCity, jState sql.NullString

	err := database.DB.QueryRow(query, id).Scan(
		&l.ID, &l.SuperintendentID, &l.JobID, &l.Location,
		&l.CheckInDate, &l.CheckOutDate, &l.PropertyLink,
		&l.Address, &l.JobsiteAddress,
		&l.CreatedAt, &l.UpdatedAt,
		&sID, &sName, &sPhone, &sEmail,
		&jID, &jNumber, &jName, &jCity, &jState,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Lodging not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch lodging"})
		return
	}

	lodging := map[string]interface{}{
		"id":               l.ID,
		"superintendent_id": l.SuperintendentID,
		"job_id":           l.JobID,
		"location":         l.Location,
		"check_in_date":    l.CheckInDate,
		"check_out_date":   l.CheckOutDate,
		"property_link":    l.PropertyLink,
		"address":          l.Address,
		"jobsite_address":  l.JobsiteAddress,
		"created_at":       l.CreatedAt,
		"updated_at":       l.UpdatedAt,
	}

	if sName.Valid {
		lodging["superintendent"] = map[string]interface{}{
			"id":    sID.Int64,
			"name":  sName.String,
			"phone": sPhone.String,
			"email": sEmail.String,
		}
	}
	if jNumber.Valid {
		lodging["job"] = map[string]interface{}{
			"id":         jID.Int64,
			"job_number": jNumber.String,
			"job_name":   jName.String,
			"city":       jCity.String,
			"state":      jState.String,
		}
	}

	c.JSON(http.StatusOK, lodging)
}

// CreateLodging creates a new lodging record
func CreateLodging(c *gin.Context) {
	var req models.CreateLodgingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate required fields
	if req.SuperintendentID == 0 || req.Location == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "superintendent_id and location are required"})
		return
	}

	query := `
		INSERT INTO superintendent_lodging (
			superintendent_id, job_id, location,
			check_in_date, check_out_date, property_link,
			address, jobsite_address,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	var lodgingID int
	var createdAt, updatedAt string
	err := database.DB.QueryRow(query,
		req.SuperintendentID, req.JobID, req.Location,
		req.CheckInDate, req.CheckOutDate, req.PropertyLink,
		req.Address, req.JobsiteAddress,
	).Scan(&lodgingID, &createdAt, &updatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create lodging"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         lodgingID,
		"location":   req.Location,
		"created_at": createdAt,
	})
}

// UpdateLodging updates an existing lodging record
func UpdateLodging(c *gin.Context) {
	id := c.Param("id")

	var req models.UpdateLodgingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if req.SuperintendentID != 0 {
		updates = append(updates, fmt.Sprintf("superintendent_id = $%d", argPos))
		args = append(args, req.SuperintendentID)
		argPos++
	}
	if req.JobID != nil {
		updates = append(updates, fmt.Sprintf("job_id = $%d", argPos))
		args = append(args, *req.JobID)
		argPos++
	}
	if req.Location != "" {
		updates = append(updates, fmt.Sprintf("location = $%d", argPos))
		args = append(args, req.Location)
		argPos++
	}
	if req.CheckInDate != nil {
		updates = append(updates, fmt.Sprintf("check_in_date = $%d", argPos))
		args = append(args, *req.CheckInDate)
		argPos++
	}
	if req.CheckOutDate != nil {
		updates = append(updates, fmt.Sprintf("check_out_date = $%d", argPos))
		args = append(args, *req.CheckOutDate)
		argPos++
	}
	if req.PropertyLink != "" {
		updates = append(updates, fmt.Sprintf("property_link = $%d", argPos))
		args = append(args, req.PropertyLink)
		argPos++
	}
	if req.Address != "" {
		updates = append(updates, fmt.Sprintf("address = $%d", argPos))
		args = append(args, req.Address)
		argPos++
	}
	if req.JobsiteAddress != "" {
		updates = append(updates, fmt.Sprintf("jobsite_address = $%d", argPos))
		args = append(args, req.JobsiteAddress)
		argPos++
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	updates = append(updates, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE superintendent_lodging SET %s WHERE id = $%d", strings.Join(updates, ", "), argPos)

	result, err := database.DB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update lodging"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Lodging not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Lodging updated successfully"})
}

// DeleteLodging soft-deletes a lodging record (marks as archived)
func DeleteLodging(c *gin.Context) {
	id := c.Param("id")

	result, err := database.DB.Exec("UPDATE superintendent_lodging SET archived = true WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to archive lodging"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Lodging not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Lodging archived successfully"})
}

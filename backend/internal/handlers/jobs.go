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

// ListJobs returns a list of jobs with optional filtering
func ListJobs(c *gin.Context) {
	// Build query with filters
	query := `
		SELECT 
			j.id, j.job_number, j.job_name, j.location, j.city, j.state,
			j.latitude, j.longitude, j.client_id, j.status, 
			j.contract_value, j.revised_contract_value,
			j.start_date, j.projected_end_date, j.actual_end_date,
			j.project_manager_id, j.apm_id, j.superintendent_id,
			j.created_at, j.updated_at,
			c.id as c_id, c.name as c_name,
			s.id as s_id, s.name as s_name
		FROM jobs j
		LEFT JOIN clients c ON j.client_id = c.id
		LEFT JOIN superintendents s ON j.superintendent_id = s.id
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	// Apply filters
	if status := c.Query("status"); status != "" {
		query += fmt.Sprintf(" AND j.status = $%d", argPos)
		args = append(args, status)
		argPos++
	}
	if clientID := c.Query("client_id"); clientID != "" {
		query += fmt.Sprintf(" AND j.client_id = $%d", argPos)
		args = append(args, clientID)
		argPos++
	}
	if superintendentID := c.Query("superintendent_id"); superintendentID != "" {
		query += fmt.Sprintf(" AND j.superintendent_id = $%d", argPos)
		args = append(args, superintendentID)
		argPos++
	}

	// Search by job number, name, or location
	if search := c.Query("search"); search != "" {
		searchTerm := "%" + search + "%"
		query += fmt.Sprintf(" AND (j.job_number ILIKE $%d OR j.job_name ILIKE $%d OR j.city ILIKE $%d OR j.location ILIKE $%d)", argPos, argPos, argPos, argPos)
		args = append(args, searchTerm)
		argPos++
	}

	// Filter archived jobs (default: show only non-archived)
	if archived := c.Query("archived"); archived != "" {
		isArchived := archived == "true"
		query += fmt.Sprintf(" AND j.archived = $%d", argPos)
		args = append(args, isArchived)
		argPos++
	} else {
		// Default: only show non-archived
		query += fmt.Sprintf(" AND j.archived = $%d", argPos)
		args = append(args, false)
		argPos++
	}

	query += " ORDER BY j.created_at DESC"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch jobs"})
		return
	}
	defer rows.Close()

	var jobs []map[string]interface{}
	for rows.Next() {
		var j models.Job
		var cID, sID sql.NullInt64
		var cName, sName sql.NullString

		err := rows.Scan(
			&j.ID, &j.JobNumber, &j.JobName, &j.Location, &j.City, &j.State,
			&j.Latitude, &j.Longitude, &j.ClientID, &j.Status,
			&j.ContractValue, &j.RevisedContractValue,
			&j.StartDate, &j.ProjectedEndDate, &j.ActualEndDate,
			&j.ProjectManagerID, &j.APMID, &j.SuperintendentID,
			&j.CreatedAt, &j.UpdatedAt,
			&cID, &cName, &sID, &sName,
		)
		if err != nil {
			continue
		}

		job := map[string]interface{}{
			"id":                   j.ID,
			"job_number":           j.JobNumber,
			"job_name":             j.JobName,
			"location":             j.Location,
			"city":                 j.City,
			"state":                j.State,
			"latitude":             j.Latitude,
			"longitude":            j.Longitude,
			"client_id":            j.ClientID,
			"status":               j.Status,
			"contract_value":       j.ContractValue,
			"revised_contract_value": j.RevisedContractValue,
			"start_date":           j.StartDate,
			"projected_end_date":   j.ProjectedEndDate,
			"actual_end_date":      j.ActualEndDate,
			"project_manager_id":   j.ProjectManagerID,
			"apm_id":               j.APMID,
			"superintendent_id":    j.SuperintendentID,
			"created_at":           j.CreatedAt,
			"updated_at":           j.UpdatedAt,
		}

		if cName.Valid {
			job["client"] = map[string]interface{}{
				"id":   cID.Int64,
				"name": cName.String,
			}
		}
		if sName.Valid {
			job["superintendent"] = map[string]interface{}{
				"id":   sID.Int64,
				"name": sName.String,
			}
		}

		jobs = append(jobs, job)
	}

	c.JSON(http.StatusOK, jobs)
}

// GetJob returns a single job by ID with all related data
func GetJob(c *gin.Context) {
	id := c.Param("id")

	query := `
		SELECT 
			j.id, j.job_number, j.job_name, j.location, j.city, j.state,
			j.latitude, j.longitude, j.client_id, j.status, 
			j.contract_value, j.revised_contract_value,
			j.start_date, j.projected_end_date, j.actual_end_date,
			j.project_manager_id, j.apm_id, j.superintendent_id,
			j.created_at, j.updated_at,
			c.id as c_id, c.name as c_name, c.contact_name, c.contact_phone, c.contact_email,
			s.id as s_id, s.name as s_name
		FROM jobs j
		LEFT JOIN clients c ON j.client_id = c.id
		LEFT JOIN superintendents s ON j.superintendent_id = s.id
		WHERE j.id = $1
	`

	var j models.Job
	var cID sql.NullInt64
	var cName, cContact, cPhone, cEmail sql.NullString
	var sID sql.NullInt64
	var sName sql.NullString

	err := database.DB.QueryRow(query, id).Scan(
		&j.ID, &j.JobNumber, &j.JobName, &j.Location, &j.City, &j.State,
		&j.Latitude, &j.Longitude, &j.ClientID, &j.Status,
		&j.ContractValue, &j.RevisedContractValue,
		&j.StartDate, &j.ProjectedEndDate, &j.ActualEndDate,
		&j.ProjectManagerID, &j.APMID, &j.SuperintendentID,
		&j.CreatedAt, &j.UpdatedAt,
		&cID, &cName, &cContact, &cPhone, &cEmail,
		&sID, &sName,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch job"})
		return
	}

	job := map[string]interface{}{
		"id":                     j.ID,
		"job_number":             j.JobNumber,
		"job_name":               j.JobName,
		"location":               j.Location,
		"city":                   j.City,
		"state":                  j.State,
		"latitude":               j.Latitude,
		"longitude":              j.Longitude,
		"client_id":              j.ClientID,
		"status":                 j.Status,
		"contract_value":         j.ContractValue,
		"revised_contract_value": j.RevisedContractValue,
		"start_date":             j.StartDate,
		"projected_end_date":     j.ProjectedEndDate,
		"actual_end_date":        j.ActualEndDate,
		"project_manager_id":     j.ProjectManagerID,
		"apm_id":                 j.APMID,
		"superintendent_id":      j.SuperintendentID,
		"created_at":             j.CreatedAt,
		"updated_at":             j.UpdatedAt,
	}

	if cName.Valid {
		job["client"] = map[string]interface{}{
			"id":            cID.Int64,
			"name":          cName.String,
			"contact_name":  cContact.String,
			"contact_phone": cPhone.String,
			"contact_email": cEmail.String,
		}
	}
	if sName.Valid {
		job["superintendent"] = map[string]interface{}{
			"id":   sID.Int64,
			"name": sName.String,
		}
	}

	// Get updates
	updatesQuery := `
		SELECT id, job_id, update_text, created_at
		FROM job_updates
		WHERE job_id = $1
		ORDER BY created_at DESC
	`
	rows, err := database.DB.Query(updatesQuery, id)
	if err == nil {
		defer rows.Close()
		var updates []map[string]interface{}
		for rows.Next() {
			var u models.JobUpdate
			if err := rows.Scan(&u.ID, &u.JobID, &u.UpdateText, &u.CreatedAt); err == nil {
				updates = append(updates, map[string]interface{}{
					"id":          u.ID,
					"job_id":      u.JobID,
					"update_text": u.UpdateText,
				})
			}
		}
		job["updates"] = updates
	}

	c.JSON(http.StatusOK, job)
}

// CreateJob creates a new job
func CreateJob(c *gin.Context) {
	var req models.CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate required fields
	if req.JobNumber == "" || req.JobName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job_number and job_name are required"})
		return
	}

	// Check for duplicate job number
	var exists bool
	err := database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM jobs WHERE job_number = $1)", req.JobNumber).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Job number already exists"})
		return
	}

	// Insert job
	query := `
		INSERT INTO jobs (
			job_number, job_name, location, city, state,
			client_id, status, contract_value, 
			start_date, projected_end_date,
			project_manager_id, apm_id, superintendent_id,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	var jobID int
	var createdAt, updatedAt string
	err = database.DB.QueryRow(query,
		req.JobNumber, req.JobName, req.Location, req.City, req.State,
		req.ClientID, req.Status, req.ContractValue,
		req.StartDate, req.ProjectedEndDate,
		req.ProjectManagerID, req.APMID, req.SuperintendentID,
	).Scan(&jobID, &createdAt, &updatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          jobID,
		"job_number":  req.JobNumber,
		"job_name":    req.JobName,
		"created_at":  createdAt,
	})
}

// UpdateJob updates an existing job
func UpdateJob(c *gin.Context) {
	id := c.Param("id")

	var req models.UpdateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if req.JobName != nil {
		updates = append(updates, fmt.Sprintf("job_name = $%d", argPos))
		args = append(args, *req.JobName)
		argPos++
	}
	if req.Location != nil {
		updates = append(updates, fmt.Sprintf("location = $%d", argPos))
		args = append(args, *req.Location)
		argPos++
	}
	if req.City != nil {
		updates = append(updates, fmt.Sprintf("city = $%d", argPos))
		args = append(args, *req.City)
		argPos++
	}
	if req.State != nil {
		updates = append(updates, fmt.Sprintf("state = $%d", argPos))
		args = append(args, *req.State)
		argPos++
	}
	if req.ClientID != nil {
		updates = append(updates, fmt.Sprintf("client_id = $%d", argPos))
		args = append(args, *req.ClientID)
		argPos++
	}
	if req.Status != nil {
		updates = append(updates, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *req.Status)
		argPos++
	}
	if req.ContractValue != nil {
		updates = append(updates, fmt.Sprintf("contract_value = $%d", argPos))
		args = append(args, *req.ContractValue)
		argPos++
	}
	if req.RevisedContractValue != nil {
		updates = append(updates, fmt.Sprintf("revised_contract_value = $%d", argPos))
		args = append(args, *req.RevisedContractValue)
		argPos++
	}
	if req.StartDate != nil {
		updates = append(updates, fmt.Sprintf("start_date = $%d", argPos))
		args = append(args, *req.StartDate)
		argPos++
	}
	if req.ProjectedEndDate != nil {
		updates = append(updates, fmt.Sprintf("projected_end_date = $%d", argPos))
		args = append(args, *req.ProjectedEndDate)
		argPos++
	}
	if req.ActualEndDate != nil {
		updates = append(updates, fmt.Sprintf("actual_end_date = $%d", argPos))
		args = append(args, *req.ActualEndDate)
		argPos++
	}
	if req.SuperintendentID != nil {
		updates = append(updates, fmt.Sprintf("superintendent_id = $%d", argPos))
		args = append(args, *req.SuperintendentID)
		argPos++
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	updates = append(updates, fmt.Sprintf("updated_at = NOW()"))
	args = append(args, id)

	query := fmt.Sprintf("UPDATE jobs SET %s WHERE id = $%d", strings.Join(updates, ", "), argPos)

	result, err := database.DB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update job"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job updated successfully"})
}

// DeleteJob soft-deletes a job (marks as archived)
func DeleteJob(c *gin.Context) {
	id := c.Param("id")

	result, err := database.DB.Exec("UPDATE jobs SET archived = true WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to archive job"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job archived successfully"})
}

// CreateJobUpdate adds a new update/note to a job
func CreateJobUpdate(c *gin.Context) {
	jobID := c.Param("id")

	// Verify job exists
	var exists bool
	err := database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM jobs WHERE id = $1)", jobID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	var req struct {
		UpdateText string `json:"update_text" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	query := `
		INSERT INTO job_updates (job_id, update_text, created_at)
		VALUES ($1, $2, NOW())
		RETURNING id, created_at
	`

	var updateID int
	var createdAt string
	err = database.DB.QueryRow(query, jobID, req.UpdateText).Scan(&updateID, &createdAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create update"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          updateID,
		"job_id":      jobID,
		"update_text": req.UpdateText,
		"created_at":  createdAt,
	})
}

// ListJobUpdates returns all updates for a job
func ListJobUpdates(c *gin.Context) {
	jobID := c.Param("id")

	query := `
		SELECT id, job_id, update_text, created_at
		FROM job_updates
		WHERE job_id = $1
		ORDER BY created_at DESC
	`

	rows, err := database.DB.Query(query, jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updates"})
		return
	}
	defer rows.Close()

	var updates []map[string]interface{}
	for rows.Next() {
		var u models.JobUpdate
		if err := rows.Scan(&u.ID, &u.JobID, &u.UpdateText, &u.CreatedAt); err != nil {
			continue
		}
		updates = append(updates, map[string]interface{}{
			"id":          u.ID,
			"job_id":      u.JobID,
			"update_text": u.UpdateText,
			"created_at":  u.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, updates)
}

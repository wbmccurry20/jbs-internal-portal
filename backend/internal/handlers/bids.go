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

// ListBids returns a list of bids with optional filtering
func ListBids(c *gin.Context) {
	query := `
		SELECT 
			b.id, b.client_id, b.location, b.city, b.state,
			b.due_date, b.assigned_to_id, b.status,
			b.building_connected_date, b.plan_hub_date,
			b.awarded, b.bid_amount, b.notes, b.job_id, b.archived,
			b.created_at, b.updated_at,
			c.id as c_id, c.name as c_name,
			j.id as j_id, j.job_number, j.job_name
		FROM bids b
		LEFT JOIN clients c ON b.client_id = c.id
		LEFT JOIN jobs j ON b.job_id = j.id
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	// Apply filters
	if status := c.Query("status"); status != "" {
		query += fmt.Sprintf(" AND b.status = $%d", argPos)
		args = append(args, status)
		argPos++
	}
	if clientID := c.Query("client_id"); clientID != "" {
		query += fmt.Sprintf(" AND b.client_id = $%d", argPos)
		args = append(args, clientID)
		argPos++
	}
	if awarded := c.Query("awarded"); awarded != "" {
		query += fmt.Sprintf(" AND b.awarded = $%d", argPos)
		args = append(args, awarded)
		argPos++
	}
	if archived := c.Query("archived"); archived != "" {
		isArchived := archived == "true"
		query += fmt.Sprintf(" AND b.archived = $%d", argPos)
		args = append(args, isArchived)
		argPos++
	} else {
		// Default: only show non-archived
		query += fmt.Sprintf(" AND b.archived = $%d", argPos)
		args = append(args, false)
		argPos++
	}

	// Search by location or city
	if search := c.Query("search"); search != "" {
		searchTerm := "%" + search + "%"
		query += fmt.Sprintf(" AND (b.location ILIKE $%d OR b.city ILIKE $%d)", argPos, argPos)
		args = append(args, searchTerm)
		argPos++
	}

	query += " ORDER BY b.due_date ASC NULLS LAST, b.created_at DESC"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bids"})
		return
	}
	defer rows.Close()

	var bids []map[string]interface{}
	for rows.Next() {
		var b models.Bid
		var cID, jID sql.NullInt64
		var cName, jNumber, jName, awarded sql.NullString

		err := rows.Scan(
			&b.ID, &b.ClientID, &b.Location, &b.City, &b.State,
			&b.DueDate, &b.AssignedToID, &b.Status,
			&b.BuildingConnectedDate, &b.PlanHubDate,
			&awarded, &b.BidAmount, &b.Notes, &b.JobID, &b.Archived,
			&b.CreatedAt, &b.UpdatedAt,
			&cID, &cName, &jID, &jNumber, &jName,
		)
		if err != nil {
			fmt.Printf("Scan error: %v\n", err)
			continue
		}

		bid := map[string]interface{}{
			"id":                     b.ID,
			"client_id":              b.ClientID,
			"location":               b.Location,
			"city":                   b.City,
			"state":                  b.State,
			"due_date":               b.DueDate,
			"assigned_to_id":         b.AssignedToID,
			"status":                 b.Status,
			"building_connected_date": b.BuildingConnectedDate,
			"plan_hub_date":          b.PlanHubDate,
			"awarded":                awarded.String,
			"bid_amount":             b.BidAmount,
			"notes":                  b.Notes,
			"job_id":                 b.JobID,
			"archived":               b.Archived,
			"created_at":             b.CreatedAt,
			"updated_at":             b.UpdatedAt,
		}

		if cName.Valid {
			bid["client"] = map[string]interface{}{
				"id":   cID.Int64,
				"name": cName.String,
			}
		}
		if jNumber.Valid {
			bid["job"] = map[string]interface{}{
				"id":         jID.Int64,
				"job_number": jNumber.String,
				"job_name":   jName.String,
			}
		}

		bids = append(bids, bid)
	}

	c.JSON(http.StatusOK, bids)
}

// GetBid returns a single bid by ID
func GetBid(c *gin.Context) {
	id := c.Param("id")

	query := `
		SELECT 
			b.id, b.client_id, b.location, b.city, b.state,
			b.due_date, b.assigned_to_id, b.status,
			b.building_connected_date, b.plan_hub_date,
			b.awarded, b.job_id, b.archived,
			b.created_at, b.updated_at,
			c.id as c_id, c.name as c_name, c.contact_name, c.contact_phone, c.contact_email,
			j.id as j_id, j.job_number, j.job_name
		FROM bids b
		LEFT JOIN clients c ON b.client_id = c.id
		LEFT JOIN jobs j ON b.job_id = j.id
		WHERE b.id = $1
	`

	var b models.Bid
	var cID sql.NullInt64
	var cName, cContact, cPhone, cEmail sql.NullString
	var jID sql.NullInt64
	var jNumber, jName, awarded sql.NullString

	err := database.DB.QueryRow(query, id).Scan(
		&b.ID, &b.ClientID, &b.Location, &b.City, &b.State,
		&b.DueDate, &b.AssignedToID, &b.Status,
		&b.BuildingConnectedDate, &b.PlanHubDate,
		&awarded, &b.JobID, &b.Archived,
		&b.CreatedAt, &b.UpdatedAt,
		&cID, &cName, &cContact, &cPhone, &cEmail,
		&jID, &jNumber, &jName,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bid not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bid"})
		return
	}

	bid := map[string]interface{}{
		"id":                     b.ID,
		"client_id":              b.ClientID,
		"location":               b.Location,
		"city":                   b.City,
		"state":                  b.State,
		"due_date":               b.DueDate,
		"assigned_to_id":         b.AssignedToID,
		"status":                 b.Status,
		"building_connected_date": b.BuildingConnectedDate,
		"plan_hub_date":          b.PlanHubDate,
		"awarded":                awarded.String,
		"job_id":                 b.JobID,
		"archived":               b.Archived,
		"created_at":             b.CreatedAt,
		"updated_at":             b.UpdatedAt,
	}

	if cName.Valid {
		bid["client"] = map[string]interface{}{
			"id":            cID.Int64,
			"name":          cName.String,
			"contact_name":  cContact.String,
			"contact_phone": cPhone.String,
			"contact_email": cEmail.String,
		}
	}
	if jNumber.Valid {
		bid["job"] = map[string]interface{}{
			"id":         jID.Int64,
			"job_number": jNumber.String,
			"job_name":   jName.String,
		}
	}

	c.JSON(http.StatusOK, bid)
}

// CreateBid creates a new bid
func CreateBid(c *gin.Context) {
	var req models.CreateBidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate required fields
	if req.Location == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "location is required"})
		return
	}

	query := `
		INSERT INTO bids (
			client_id, location, city, state, due_date,
			assigned_to_id, status, building_connected_date, plan_hub_date,
			awarded, archived, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, false, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	var bidID int
	var createdAt, updatedAt string
	err := database.DB.QueryRow(query,
		req.ClientID, req.Location, req.City, req.State, req.DueDate,
		req.AssignedToID, req.Status, req.BuildingConnectedDate, req.PlanHubDate,
		req.Awarded,
	).Scan(&bidID, &createdAt, &updatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bid"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         bidID,
		"location":   req.Location,
		"created_at": createdAt,
	})
}

// UpdateBid updates an existing bid
func UpdateBid(c *gin.Context) {
	id := c.Param("id")

	var req models.UpdateBidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if req.ClientID != nil {
		updates = append(updates, fmt.Sprintf("client_id = $%d", argPos))
		args = append(args, *req.ClientID)
		argPos++
	}
	if req.Location != "" {
		updates = append(updates, fmt.Sprintf("location = $%d", argPos))
		args = append(args, req.Location)
		argPos++
	}
	if req.City != "" {
		updates = append(updates, fmt.Sprintf("city = $%d", argPos))
		args = append(args, req.City)
		argPos++
	}
	if req.State != "" {
		updates = append(updates, fmt.Sprintf("state = $%d", argPos))
		args = append(args, req.State)
		argPos++
	}
	if req.DueDate != nil {
		updates = append(updates, fmt.Sprintf("due_date = $%d", argPos))
		args = append(args, *req.DueDate)
		argPos++
	}
	if req.AssignedToID != nil {
		updates = append(updates, fmt.Sprintf("assigned_to_id = $%d", argPos))
		args = append(args, *req.AssignedToID)
		argPos++
	}
	if req.Status != "" {
		updates = append(updates, fmt.Sprintf("status = $%d", argPos))
		args = append(args, req.Status)
		argPos++
	}
	if req.BuildingConnectedDate != nil {
		updates = append(updates, fmt.Sprintf("building_connected_date = $%d", argPos))
		args = append(args, *req.BuildingConnectedDate)
		argPos++
	}
	if req.PlanHubDate != nil {
		updates = append(updates, fmt.Sprintf("plan_hub_date = $%d", argPos))
		args = append(args, *req.PlanHubDate)
		argPos++
	}
	if req.Awarded != "" {
		updates = append(updates, fmt.Sprintf("awarded = $%d", argPos))
		args = append(args, req.Awarded)
		argPos++
	}
	if req.JobID != nil {
		updates = append(updates, fmt.Sprintf("job_id = $%d", argPos))
		args = append(args, *req.JobID)
		argPos++
	}
	if req.Archived != nil {
		updates = append(updates, fmt.Sprintf("archived = $%d", argPos))
		args = append(args, *req.Archived)
		argPos++
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	updates = append(updates, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE bids SET %s WHERE id = $%d", strings.Join(updates, ", "), argPos)

	result, err := database.DB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bid"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bid not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Bid updated successfully"})
}

// DeleteBid archives a bid
func DeleteBid(c *gin.Context) {
	id := c.Param("id")

	result, err := database.DB.Exec("UPDATE bids SET archived = true WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to archive bid"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bid not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Bid archived successfully"})
}

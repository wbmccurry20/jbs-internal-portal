package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wbmccurry20/jbs-internal-portal/internal/database"
	"github.com/wbmccurry20/jbs-internal-portal/internal/models"
)

// normaliseTimePtr converts empty-string time pointers to nil for clean DB storage.
func normaliseTimePtr(t *string) *string {
	if t == nil || *t == "" {
		return nil
	}
	return t
}

// ===================== Training Programs =====================

// ListTrainingPrograms returns all programs (admin view)
func ListTrainingPrograms(c *gin.Context) {
	rows, err := database.DB.Query(`
		SELECT p.id, p.name, p.description, p.duration_weeks, p.created_by, p.is_active, p.created_at, p.updated_at,
			   COALESCE(u.name, '') as creator_name,
			   (SELECT COUNT(*) FROM trainee_assignments a WHERE a.program_id = p.id) as assignment_count
		FROM training_programs p
		LEFT JOIN users u ON p.created_by = u.id
		ORDER BY p.created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch programs"})
		return
	}
	defer rows.Close()

	type ProgramWithMeta struct {
		models.TrainingProgram
		CreatorName     string `json:"creator_name"`
		AssignmentCount int    `json:"assignment_count"`
	}

	var programs []ProgramWithMeta
	for rows.Next() {
		var p ProgramWithMeta
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.DurationWeeks, &p.CreatedBy,
			&p.IsActive, &p.CreatedAt, &p.UpdatedAt, &p.CreatorName, &p.AssignmentCount); err != nil {
			continue
		}
		programs = append(programs, p)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Warning: error iterating program rows: %v", err)
	}

	if programs == nil {
		programs = []ProgramWithMeta{}
	}

	c.JSON(http.StatusOK, gin.H{"programs": programs})
}

// GetTrainingProgram returns a single program with all its schedule items and resources
func GetTrainingProgram(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid program ID"})
		return
	}

	// Get program
	var p models.TrainingProgram
	err = database.DB.QueryRow(`
		SELECT id, name, description, duration_weeks, created_by, is_active, created_at, updated_at
		FROM training_programs WHERE id = $1
	`, id).Scan(&p.ID, &p.Name, &p.Description, &p.DurationWeeks, &p.CreatedBy, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Program not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch program"})
		return
	}

	// Get schedule items
	rows, err := database.DB.Query(`
		SELECT id, program_id, week_number, day_of_week, COALESCE(day_title, ''), title, COALESCE(description, ''),
			   COALESCE(link_url, ''), COALESCE(link_label, ''), time_slot, sort_order, is_highlight,
			   start_time, end_time, created_at, updated_at
		FROM training_schedule_items
		WHERE program_id = $1
		ORDER BY week_number, day_of_week, sort_order
	`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch schedule items"})
		return
	}
	defer rows.Close()

	p.Items = []models.TrainingScheduleItem{}
	for rows.Next() {
		var item models.TrainingScheduleItem
		if err := rows.Scan(&item.ID, &item.ProgramID, &item.WeekNumber, &item.DayOfWeek, &item.DayTitle,
			&item.Title, &item.Description, &item.LinkURL, &item.LinkLabel, &item.TimeSlot,
			&item.SortOrder, &item.IsHighlight, &item.StartTime, &item.EndTime,
			&item.CreatedAt, &item.UpdatedAt); err != nil {
			continue
		}
		p.Items = append(p.Items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Warning: error iterating schedule item rows: %v", err)
	}

	// Get resources
	resRows, err := database.DB.Query(`
		SELECT id, program_id, title, url, category, COALESCE(description, ''), sort_order, created_at
		FROM training_resources
		WHERE program_id = $1
		ORDER BY sort_order, title
	`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch resources"})
		return
	}
	defer resRows.Close()

	p.Resources = []models.TrainingResource{}
	for resRows.Next() {
		var r models.TrainingResource
		if err := resRows.Scan(&r.ID, &r.ProgramID, &r.Title, &r.URL, &r.Category, &r.Description, &r.SortOrder, &r.CreatedAt); err != nil {
			continue
		}
		p.Resources = append(p.Resources, r)
	}
	if err := resRows.Err(); err != nil {
		log.Printf("Warning: error iterating resource rows: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"program": p})
}

// CreateTrainingProgram creates a new program
func CreateTrainingProgram(c *gin.Context) {
	var req struct {
		Name          string `json:"name" binding:"required"`
		Description   string `json:"description"`
		DurationWeeks int    `json:"duration_weeks"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name is required"})
		return
	}

	if req.DurationWeeks <= 0 {
		req.DurationWeeks = 4
	}

	userID, _ := c.Get("userID")

	var id int
	err := database.DB.QueryRow(`
		INSERT INTO training_programs (name, description, duration_weeks, created_by)
		VALUES ($1, $2, $3, $4) RETURNING id
	`, req.Name, req.Description, req.DurationWeeks, userID).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create program"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Program created"})
}

// UpdateTrainingProgram updates program details
func UpdateTrainingProgram(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid program ID"})
		return
	}

	var req struct {
		Name          string `json:"name" binding:"required"`
		Description   string `json:"description"`
		DurationWeeks int    `json:"duration_weeks"`
		IsActive      bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name is required"})
		return
	}

	if req.DurationWeeks <= 0 {
		req.DurationWeeks = 4
	}

	_, err = database.DB.Exec(`
		UPDATE training_programs SET name = $1, description = $2, duration_weeks = $3, is_active = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
	`, req.Name, req.Description, req.DurationWeeks, req.IsActive, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update program"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Program updated"})
}

// DeleteTrainingProgram deletes a program and all its data
func DeleteTrainingProgram(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid program ID"})
		return
	}

	_, err = database.DB.Exec("DELETE FROM training_programs WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete program"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Program deleted"})
}

// ===================== Schedule Items =====================

// CreateScheduleItem adds a task to a program day
func CreateScheduleItem(c *gin.Context) {
	programID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid program ID"})
		return
	}

	var req struct {
		WeekNumber  int     `json:"week_number" binding:"required"`
		DayOfWeek   int     `json:"day_of_week" binding:"required"`
		DayTitle    string  `json:"day_title"`
		Title       string  `json:"title" binding:"required"`
		Description string  `json:"description"`
		LinkURL     string  `json:"link_url"`
		LinkLabel   string  `json:"link_label"`
		TimeSlot    string  `json:"time_slot"`
		StartTime   *string `json:"start_time"`
		EndTime     *string `json:"end_time"`
		SortOrder   int     `json:"sort_order"`
		IsHighlight bool    `json:"is_highlight"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Week number, day of week, and title are required"})
		return
	}

	if req.TimeSlot == "" {
		req.TimeSlot = "all-day"
	}

	// Normalise empty-string time pointers to nil
	req.StartTime = normaliseTimePtr(req.StartTime)
	req.EndTime = normaliseTimePtr(req.EndTime)

	var id int
	err = database.DB.QueryRow(`
		INSERT INTO training_schedule_items
			(program_id, week_number, day_of_week, day_title, title, description,
			 link_url, link_label, time_slot, start_time, end_time, sort_order, is_highlight)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`, programID, req.WeekNumber, req.DayOfWeek, req.DayTitle, req.Title, req.Description,
		req.LinkURL, req.LinkLabel, req.TimeSlot, req.StartTime, req.EndTime,
		req.SortOrder, req.IsHighlight).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create schedule item"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Schedule item created"})
}

// UpdateScheduleItem updates a schedule item
func UpdateScheduleItem(c *gin.Context) {
	programID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid program ID"})
		return
	}

	itemID, err := strconv.Atoi(c.Param("itemId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	// Verify item belongs to this program
	var ownerProgramID int
	if err := database.DB.QueryRow("SELECT program_id FROM training_schedule_items WHERE id = $1", itemID).Scan(&ownerProgramID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule item not found"})
		return
	}
	if ownerProgramID != programID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Item does not belong to this program"})
		return
	}

	var req struct {
		WeekNumber  int     `json:"week_number" binding:"required"`
		DayOfWeek   int     `json:"day_of_week" binding:"required"`
		DayTitle    string  `json:"day_title"`
		Title       string  `json:"title" binding:"required"`
		Description string  `json:"description"`
		LinkURL     string  `json:"link_url"`
		LinkLabel   string  `json:"link_label"`
		TimeSlot    string  `json:"time_slot"`
		StartTime   *string `json:"start_time"`
		EndTime     *string `json:"end_time"`
		SortOrder   int     `json:"sort_order"`
		IsHighlight bool    `json:"is_highlight"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Week number, day of week, and title are required"})
		return
	}

	if req.TimeSlot == "" {
		req.TimeSlot = "all-day"
	}

	req.StartTime = normaliseTimePtr(req.StartTime)
	req.EndTime = normaliseTimePtr(req.EndTime)

	_, err = database.DB.Exec(`
		UPDATE training_schedule_items
		SET week_number = $1, day_of_week = $2, day_title = $3, title = $4, description = $5,
			link_url = $6, link_label = $7, time_slot = $8, start_time = $9, end_time = $10,
			sort_order = $11, is_highlight = $12, updated_at = CURRENT_TIMESTAMP
		WHERE id = $13
	`, req.WeekNumber, req.DayOfWeek, req.DayTitle, req.Title, req.Description,
		req.LinkURL, req.LinkLabel, req.TimeSlot, req.StartTime, req.EndTime,
		req.SortOrder, req.IsHighlight, itemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update schedule item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule item updated"})
}

// DeleteScheduleItem removes a schedule item
func DeleteScheduleItem(c *gin.Context) {
	programID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid program ID"})
		return
	}

	itemID, err := strconv.Atoi(c.Param("itemId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	// Verify item belongs to this program
	var ownerProgramID int
	if err := database.DB.QueryRow("SELECT program_id FROM training_schedule_items WHERE id = $1", itemID).Scan(&ownerProgramID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule item not found"})
		return
	}
	if ownerProgramID != programID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Item does not belong to this program"})
		return
	}

	_, err = database.DB.Exec("DELETE FROM training_schedule_items WHERE id = $1", itemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete schedule item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule item deleted"})
}

// BulkUpdateScheduleItems replaces all items for a program (useful for drag-and-drop reorder)
func BulkUpdateScheduleItems(c *gin.Context) {
	programID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid program ID"})
		return
	}

	var req struct {
		Items []struct {
			WeekNumber  int     `json:"week_number"`
			DayOfWeek   int     `json:"day_of_week"`
			DayTitle    string  `json:"day_title"`
			Title       string  `json:"title"`
			Description string  `json:"description"`
			LinkURL     string  `json:"link_url"`
			LinkLabel   string  `json:"link_label"`
			TimeSlot    string  `json:"time_slot"`
			StartTime   *string `json:"start_time"`
			EndTime     *string `json:"end_time"`
			SortOrder   int     `json:"sort_order"`
			IsHighlight bool    `json:"is_highlight"`
		} `json:"items" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Items array is required"})
		return
	}

	// Limit bulk updates to prevent abuse (52 weeks * 5 days * ~5 items max = 1300)
	if len(req.Items) > 1500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Too many items (max 1500)"})
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Delete existing items
	_, err = tx.Exec("DELETE FROM training_schedule_items WHERE program_id = $1", programID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear existing items"})
		return
	}

	// Insert new items
	for _, item := range req.Items {
		timeSlot := item.TimeSlot
		if timeSlot == "" {
			timeSlot = "all-day"
		}
		startTime := normaliseTimePtr(item.StartTime)
		endTime := normaliseTimePtr(item.EndTime)

		_, err = tx.Exec(`
			INSERT INTO training_schedule_items
				(program_id, week_number, day_of_week, day_title, title, description,
				 link_url, link_label, time_slot, start_time, end_time, sort_order, is_highlight)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`, programID, item.WeekNumber, item.DayOfWeek, item.DayTitle, item.Title, item.Description,
			item.LinkURL, item.LinkLabel, timeSlot, startTime, endTime,
			item.SortOrder, item.IsHighlight)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert schedule item"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save items"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule updated", "items_count": len(req.Items)})
}

// ===================== Trainee Assignments =====================

// ListAssignments returns all trainee assignments
func ListAssignments(c *gin.Context) {
	rows, err := database.DB.Query(`
		SELECT a.id, a.program_id, a.user_id, a.start_date, a.status, COALESCE(a.notes, ''),
			   a.assigned_by, a.created_at, a.updated_at,
			   u.name as trainee_name, u.email as trainee_email,
			   p.name as program_name,
			   COALESCE(ab.name, '') as assigned_by_name
		FROM trainee_assignments a
		JOIN users u ON a.user_id = u.id
		JOIN training_programs p ON a.program_id = p.id
		LEFT JOIN users ab ON a.assigned_by = ab.id
		ORDER BY a.created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch assignments"})
		return
	}
	defer rows.Close()

	var assignments []models.TraineeAssignment
	for rows.Next() {
		var a models.TraineeAssignment
		var startDate time.Time
		if err := rows.Scan(&a.ID, &a.ProgramID, &a.UserID, &startDate, &a.Status, &a.Notes,
			&a.AssignedBy, &a.CreatedAt, &a.UpdatedAt,
			&a.TraineeName, &a.TraineeEmail, &a.ProgramName, &a.AssignedByName); err != nil {
			continue
		}
		a.StartDate = startDate.Format("2006-01-02")
		assignments = append(assignments, a)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Warning: error iterating assignment rows: %v", err)
	}

	if assignments == nil {
		assignments = []models.TraineeAssignment{}
	}

	c.JSON(http.StatusOK, gin.H{"assignments": assignments})
}

// CreateAssignment assigns a trainee to a program
func CreateAssignment(c *gin.Context) {
	var req struct {
		ProgramID int    `json:"program_id" binding:"required"`
		UserID    int    `json:"user_id" binding:"required"`
		StartDate string `json:"start_date" binding:"required"`
		Notes     string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Program ID, user ID, and start date are required"})
		return
	}

	// Validate start date
	_, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format (use YYYY-MM-DD)"})
		return
	}

	assignedBy, _ := c.Get("userID")

	var id int
	err = database.DB.QueryRow(`
		INSERT INTO trainee_assignments (program_id, user_id, start_date, notes, assigned_by)
		VALUES ($1, $2, $3, $4, $5) RETURNING id
	`, req.ProgramID, req.UserID, req.StartDate, req.Notes, assignedBy).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create assignment. User may already be assigned to this program."})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Trainee assigned to program"})
}

// UpdateAssignment updates assignment status/notes
func UpdateAssignment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assignment ID"})
		return
	}

	var req struct {
		StartDate string `json:"start_date"`
		Status    string `json:"status"`
		Notes     string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.StartDate != "" {
		if _, err := time.Parse("2006-01-02", req.StartDate); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format"})
			return
		}
	}

	// Validate status
	if req.Status != "" {
		validStatuses := map[string]bool{"active": true, "completed": true, "paused": true, "cancelled": true}
		if !validStatuses[req.Status] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status. Must be active, completed, paused, or cancelled"})
			return
		}
	}

	_, err = database.DB.Exec(`
		UPDATE trainee_assignments
		SET start_date = COALESCE(NULLIF($1, '')::date, start_date),
			status = COALESCE(NULLIF($2, ''), status),
			notes = $3,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $4
	`, req.StartDate, req.Status, req.Notes, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update assignment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Assignment updated"})
}

// DeleteAssignment removes a trainee assignment
func DeleteAssignment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assignment ID"})
		return
	}

	_, err = database.DB.Exec("DELETE FROM trainee_assignments WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete assignment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Assignment deleted"})
}

// ===================== Training Resources =====================

// CreateResource adds a resource link to a program
func CreateResource(c *gin.Context) {
	programID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid program ID"})
		return
	}

	var req struct {
		Title       string `json:"title" binding:"required"`
		URL         string `json:"url" binding:"required"`
		Category    string `json:"category"`
		Description string `json:"description"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Title and URL are required"})
		return
	}

	if req.Category == "" {
		req.Category = "General"
	}

	var id int
	err = database.DB.QueryRow(`
		INSERT INTO training_resources (program_id, title, url, category, description, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
	`, programID, req.Title, req.URL, req.Category, req.Description, req.SortOrder).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create resource"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Resource created"})
}

// UpdateResource updates a resource
func UpdateResource(c *gin.Context) {
	programID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid program ID"})
		return
	}

	resID, err := strconv.Atoi(c.Param("resourceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid resource ID"})
		return
	}

	// Verify resource belongs to this program
	var ownerProgramID int
	if err := database.DB.QueryRow("SELECT program_id FROM training_resources WHERE id = $1", resID).Scan(&ownerProgramID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
		return
	}
	if ownerProgramID != programID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Resource does not belong to this program"})
		return
	}

	var req struct {
		Title       string `json:"title" binding:"required"`
		URL         string `json:"url" binding:"required"`
		Category    string `json:"category"`
		Description string `json:"description"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Title and URL are required"})
		return
	}

	_, err = database.DB.Exec(`
		UPDATE training_resources SET title = $1, url = $2, category = $3, description = $4, sort_order = $5
		WHERE id = $6
	`, req.Title, req.URL, req.Category, req.Description, req.SortOrder, resID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update resource"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Resource updated"})
}

// DeleteResource removes a resource
func DeleteResource(c *gin.Context) {
	programID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid program ID"})
		return
	}

	resID, err := strconv.Atoi(c.Param("resourceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid resource ID"})
		return
	}

	// Verify resource belongs to this program
	var ownerProgramID int
	if err := database.DB.QueryRow("SELECT program_id FROM training_resources WHERE id = $1", resID).Scan(&ownerProgramID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
		return
	}
	if ownerProgramID != programID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Resource does not belong to this program"})
		return
	}

	_, err = database.DB.Exec("DELETE FROM training_resources WHERE id = $1", resID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete resource"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Resource deleted"})
}

// ===================== Trainee Calendar View =====================

// GetMyTrainingCalendar returns the logged-in trainee's calendar
func GetMyTrainingCalendar(c *gin.Context) {
	userID, _ := c.Get("userID")

	// Get active assignment for this user
	var a models.TraineeAssignment
	var startDate time.Time
	err := database.DB.QueryRow(`
		SELECT a.id, a.program_id, a.user_id, a.start_date, a.status, COALESCE(a.notes, ''),
			   a.assigned_by, a.created_at, a.updated_at,
			   u.name, u.email, p.name
		FROM trainee_assignments a
		JOIN users u ON a.user_id = u.id
		JOIN training_programs p ON a.program_id = p.id
		WHERE a.user_id = $1 AND a.status = 'active'
		ORDER BY a.created_at DESC LIMIT 1
	`, userID).Scan(&a.ID, &a.ProgramID, &a.UserID, &startDate, &a.Status, &a.Notes,
		&a.AssignedBy, &a.CreatedAt, &a.UpdatedAt,
		&a.TraineeName, &a.TraineeEmail, &a.ProgramName)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, gin.H{"calendar": nil, "message": "No active training program assigned"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch assignment"})
		return
	}
	a.StartDate = startDate.Format("2006-01-02")

	// Build calendar
	calendar, err := buildTraineeCalendar(a, startDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build calendar"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"calendar": calendar})
}

// GetTraineeCalendar returns calendar for a specific assignment (admin view)
func GetTraineeCalendar(c *gin.Context) {
	assignmentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assignment ID"})
		return
	}

	var a models.TraineeAssignment
	var startDate time.Time
	err = database.DB.QueryRow(`
		SELECT a.id, a.program_id, a.user_id, a.start_date, a.status, COALESCE(a.notes, ''),
			   a.assigned_by, a.created_at, a.updated_at,
			   u.name, u.email, p.name
		FROM trainee_assignments a
		JOIN users u ON a.user_id = u.id
		JOIN training_programs p ON a.program_id = p.id
		WHERE a.id = $1
	`, assignmentID).Scan(&a.ID, &a.ProgramID, &a.UserID, &startDate, &a.Status, &a.Notes,
		&a.AssignedBy, &a.CreatedAt, &a.UpdatedAt,
		&a.TraineeName, &a.TraineeEmail, &a.ProgramName)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch assignment"})
		return
	}
	a.StartDate = startDate.Format("2006-01-02")

	calendar, err := buildTraineeCalendar(a, startDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build calendar"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"calendar": calendar})
}

// buildTraineeCalendar computes actual dates from program template + start date
func buildTraineeCalendar(assignment models.TraineeAssignment, startDate time.Time) (*models.TraineeCalendar, error) {
	// Get program
	var program models.TrainingProgram
	err := database.DB.QueryRow(`
		SELECT id, name, description, duration_weeks, created_by, is_active, created_at, updated_at
		FROM training_programs WHERE id = $1
	`, assignment.ProgramID).Scan(&program.ID, &program.Name, &program.Description,
		&program.DurationWeeks, &program.CreatedBy, &program.IsActive, &program.CreatedAt, &program.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Get schedule items
	rows, err := database.DB.Query(`
		SELECT id, program_id, week_number, day_of_week, COALESCE(day_title, ''), title, COALESCE(description, ''),
			   COALESCE(link_url, ''), COALESCE(link_label, ''), time_slot, sort_order, is_highlight,
			   start_time, end_time, created_at, updated_at
		FROM training_schedule_items
		WHERE program_id = $1
		ORDER BY week_number, day_of_week, sort_order
	`, assignment.ProgramID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Group items by (week, day)
	type dayKey struct {
		week int
		day  int
	}
	itemsByDay := make(map[dayKey][]models.TrainingScheduleItem)
	dayTitles := make(map[dayKey]string)

	for rows.Next() {
		var item models.TrainingScheduleItem
		if err := rows.Scan(&item.ID, &item.ProgramID, &item.WeekNumber, &item.DayOfWeek, &item.DayTitle,
			&item.Title, &item.Description, &item.LinkURL, &item.LinkLabel, &item.TimeSlot,
			&item.SortOrder, &item.IsHighlight, &item.StartTime, &item.EndTime,
			&item.CreatedAt, &item.UpdatedAt); err != nil {
			continue
		}
		key := dayKey{item.WeekNumber, item.DayOfWeek}
		itemsByDay[key] = append(itemsByDay[key], item)
		if item.DayTitle != "" {
			dayTitles[key] = item.DayTitle
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Get resources
	resRows, err := database.DB.Query(`
		SELECT id, program_id, title, url, category, COALESCE(description, ''), sort_order, created_at
		FROM training_resources
		WHERE program_id = $1
		ORDER BY sort_order, title
	`, assignment.ProgramID)
	if err != nil {
		return nil, err
	}
	defer resRows.Close()

	var resources []models.TrainingResource
	for resRows.Next() {
		var r models.TrainingResource
		if err := resRows.Scan(&r.ID, &r.ProgramID, &r.Title, &r.URL, &r.Category, &r.Description, &r.SortOrder, &r.CreatedAt); err != nil {
			continue
		}
		resources = append(resources, r)
	}
	if err := resRows.Err(); err != nil {
		return nil, err
	}

	// Load per-trainee overrides (only for real assignments, not previews)
	type dayKey2 struct {
		week int
		day  int
	}
	overridesByDay := make(map[dayKey2][]models.ScheduleOverride)
	if assignment.ID > 0 {
		oRows, err := database.DB.Query(`
			SELECT id, assignment_id, week_number, day_of_week, action, base_item_id,
				   COALESCE(title, ''), COALESCE(description, ''), COALESCE(day_title, ''),
				   COALESCE(link_url, ''), COALESCE(link_label, ''),
				   COALESCE(time_slot, 'all-day'), start_time, end_time,
				   sort_order, is_highlight, created_by, created_at, updated_at
			FROM assignment_schedule_overrides
			WHERE assignment_id = $1
			ORDER BY week_number, day_of_week, sort_order
		`, assignment.ID)
		if err != nil {
			return nil, err
		}
		defer oRows.Close()

		for oRows.Next() {
			var o models.ScheduleOverride
			if err := oRows.Scan(&o.ID, &o.AssignmentID, &o.WeekNumber, &o.DayOfWeek, &o.Action, &o.BaseItemID,
				&o.Title, &o.Description, &o.DayTitle, &o.LinkURL, &o.LinkLabel,
				&o.TimeSlot, &o.StartTime, &o.EndTime,
				&o.SortOrder, &o.IsHighlight, &o.CreatedBy, &o.CreatedAt, &o.UpdatedAt); err != nil {
				continue
			}
			k := dayKey2{o.WeekNumber, o.DayOfWeek}
			overridesByDay[k] = append(overridesByDay[k], o)
		}
		if err := oRows.Err(); err != nil {
			return nil, err
		}
	}

	// Build calendar days
	today := time.Now().Truncate(24 * time.Hour)
	var days []models.CalendarDay
	currentWeek := 0

	// Find the Monday of the start week
	weekStart := startDate
	for weekStart.Weekday() != time.Monday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}

	for week := 1; week <= program.DurationWeeks; week++ {
		for dow := 1; dow <= 5; dow++ { // Mon-Fri
			dayOffset := (week-1)*7 + (dow - 1)
			date := weekStart.AddDate(0, 0, dayOffset)

			key := dayKey{week, dow}
			items := itemsByDay[key]
			if items == nil {
				items = []models.TrainingScheduleItem{}
			}

			// Apply per-trainee overrides if any exist
			oKey := dayKey2{week, dow}
			if dayOverrides, ok := overridesByDay[oKey]; ok && len(dayOverrides) > 0 {
				items = applyOverrides(items, dayOverrides)
			}

			// Check if an override provides a day_title
			dayTitle := dayTitles[key]
			for _, o := range overridesByDay[oKey] {
				if o.DayTitle != "" {
					dayTitle = o.DayTitle
					break
				}
			}

			day := models.CalendarDay{
				Date:      date.Format("2006-01-02"),
				DayTitle:  dayTitle,
				DayOfWeek: dow,
				Week:      week,
				Items:     items,
				IsToday:   date.Equal(today),
				IsPast:    date.Before(today),
			}
			days = append(days, day)

			if date.Equal(today) || (date.Before(today) && date.AddDate(0, 0, 7).After(today)) {
				currentWeek = week
			}
		}
	}

	if currentWeek == 0 {
		if today.Before(weekStart) {
			currentWeek = 1
		} else {
			currentWeek = program.DurationWeeks
		}
	}

	return &models.TraineeCalendar{
		Assignment:  assignment,
		Program:     program,
		Days:        days,
		Resources:   resources,
		CurrentWeek: currentWeek,
		TotalWeeks:  program.DurationWeeks,
	}, nil
}

// ===================== Program Preview =====================

// PreviewProgramCalendar lets admins preview a program as if a trainee started on a given date.
func PreviewProgramCalendar(c *gin.Context) {
	programID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid program ID"})
		return
	}

	startDateStr := c.Query("start_date")
	if startDateStr == "" {
		// Default to next Monday
		now := time.Now()
		daysUntilMon := (8 - int(now.Weekday())) % 7
		if daysUntilMon == 0 {
			daysUntilMon = 7
		}
		startDateStr = now.AddDate(0, 0, daysUntilMon).Format("2006-01-02")
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date (use YYYY-MM-DD)"})
		return
	}

	// Build a synthetic assignment for preview
	userName, _ := c.Get("userName")
	nameStr := "Preview"
	if n, ok := userName.(string); ok && n != "" {
		nameStr = n
	}

	fakeAssignment := models.TraineeAssignment{
		ProgramID:   programID,
		StartDate:   startDateStr,
		Status:      "active",
		TraineeName: nameStr,
	}

	calendar, err := buildTraineeCalendar(fakeAssignment, startDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build preview calendar"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"calendar": calendar})
}

// ===================== Schedule Overrides =====================

// ListOverrides returns all overrides for an assignment
func ListOverrides(c *gin.Context) {
	assignmentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assignment ID"})
		return
	}

	rows, err := database.DB.Query(`
		SELECT id, assignment_id, week_number, day_of_week, action, base_item_id,
			   COALESCE(title, ''), COALESCE(description, ''), COALESCE(day_title, ''),
			   COALESCE(link_url, ''), COALESCE(link_label, ''),
			   COALESCE(time_slot, 'all-day'), start_time, end_time,
			   sort_order, is_highlight, created_by, created_at, updated_at
		FROM assignment_schedule_overrides
		WHERE assignment_id = $1
		ORDER BY week_number, day_of_week, sort_order
	`, assignmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch overrides"})
		return
	}
	defer rows.Close()

	var overrides []models.ScheduleOverride
	for rows.Next() {
		var o models.ScheduleOverride
		if err := rows.Scan(&o.ID, &o.AssignmentID, &o.WeekNumber, &o.DayOfWeek, &o.Action, &o.BaseItemID,
			&o.Title, &o.Description, &o.DayTitle, &o.LinkURL, &o.LinkLabel,
			&o.TimeSlot, &o.StartTime, &o.EndTime,
			&o.SortOrder, &o.IsHighlight, &o.CreatedBy, &o.CreatedAt, &o.UpdatedAt); err != nil {
			continue
		}
		overrides = append(overrides, o)
	}
	if overrides == nil {
		overrides = []models.ScheduleOverride{}
	}

	c.JSON(http.StatusOK, gin.H{"overrides": overrides})
}

// CreateOverride adds a per-trainee schedule override
func CreateOverride(c *gin.Context) {
	assignmentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assignment ID"})
		return
	}

	var req struct {
		WeekNumber  int     `json:"week_number" binding:"required"`
		DayOfWeek   int     `json:"day_of_week" binding:"required"`
		Action      string  `json:"action" binding:"required"`
		BaseItemID  *int    `json:"base_item_id"`
		Title       string  `json:"title"`
		Description string  `json:"description"`
		DayTitle    string  `json:"day_title"`
		LinkURL     string  `json:"link_url"`
		LinkLabel   string  `json:"link_label"`
		TimeSlot    string  `json:"time_slot"`
		StartTime   *string `json:"start_time"`
		EndTime     *string `json:"end_time"`
		SortOrder   int     `json:"sort_order"`
		IsHighlight bool    `json:"is_highlight"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "week_number, day_of_week, and action are required"})
		return
	}

	validActions := map[string]bool{"add": true, "remove": true, "replace": true}
	if !validActions[req.Action] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be add, remove, or replace"})
		return
	}

	if (req.Action == "remove" || req.Action == "replace") && req.BaseItemID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "base_item_id is required for remove/replace actions"})
		return
	}

	if req.TimeSlot == "" {
		req.TimeSlot = "all-day"
	}
	req.StartTime = normaliseTimePtr(req.StartTime)
	req.EndTime = normaliseTimePtr(req.EndTime)

	userID, _ := c.Get("userID")

	var id int
	err = database.DB.QueryRow(`
		INSERT INTO assignment_schedule_overrides
			(assignment_id, week_number, day_of_week, action, base_item_id,
			 title, description, day_title, link_url, link_label,
			 time_slot, start_time, end_time, sort_order, is_highlight, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id
	`, assignmentID, req.WeekNumber, req.DayOfWeek, req.Action, req.BaseItemID,
		req.Title, req.Description, req.DayTitle, req.LinkURL, req.LinkLabel,
		req.TimeSlot, req.StartTime, req.EndTime, req.SortOrder, req.IsHighlight, userID).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create override"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Override created"})
}

// UpdateOverride updates an existing override
func UpdateOverride(c *gin.Context) {
	assignmentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assignment ID"})
		return
	}

	overrideID, err := strconv.Atoi(c.Param("overrideId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid override ID"})
		return
	}

	// Verify ownership
	var ownerID int
	if err := database.DB.QueryRow("SELECT assignment_id FROM assignment_schedule_overrides WHERE id = $1", overrideID).Scan(&ownerID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Override not found"})
		return
	}
	if ownerID != assignmentID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Override does not belong to this assignment"})
		return
	}

	var req struct {
		WeekNumber  int     `json:"week_number" binding:"required"`
		DayOfWeek   int     `json:"day_of_week" binding:"required"`
		Action      string  `json:"action" binding:"required"`
		BaseItemID  *int    `json:"base_item_id"`
		Title       string  `json:"title"`
		Description string  `json:"description"`
		DayTitle    string  `json:"day_title"`
		LinkURL     string  `json:"link_url"`
		LinkLabel   string  `json:"link_label"`
		TimeSlot    string  `json:"time_slot"`
		StartTime   *string `json:"start_time"`
		EndTime     *string `json:"end_time"`
		SortOrder   int     `json:"sort_order"`
		IsHighlight bool    `json:"is_highlight"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.TimeSlot == "" {
		req.TimeSlot = "all-day"
	}
	req.StartTime = normaliseTimePtr(req.StartTime)
	req.EndTime = normaliseTimePtr(req.EndTime)

	_, err = database.DB.Exec(`
		UPDATE assignment_schedule_overrides
		SET week_number = $1, day_of_week = $2, action = $3, base_item_id = $4,
			title = $5, description = $6, day_title = $7, link_url = $8, link_label = $9,
			time_slot = $10, start_time = $11, end_time = $12, sort_order = $13, is_highlight = $14,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $15
	`, req.WeekNumber, req.DayOfWeek, req.Action, req.BaseItemID,
		req.Title, req.Description, req.DayTitle, req.LinkURL, req.LinkLabel,
		req.TimeSlot, req.StartTime, req.EndTime, req.SortOrder, req.IsHighlight, overrideID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update override"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Override updated"})
}

// DeleteOverride removes an override
func DeleteOverride(c *gin.Context) {
	assignmentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assignment ID"})
		return
	}

	overrideID, err := strconv.Atoi(c.Param("overrideId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid override ID"})
		return
	}

	var ownerID int
	if err := database.DB.QueryRow("SELECT assignment_id FROM assignment_schedule_overrides WHERE id = $1", overrideID).Scan(&ownerID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Override not found"})
		return
	}
	if ownerID != assignmentID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Override does not belong to this assignment"})
		return
	}

	_, err = database.DB.Exec("DELETE FROM assignment_schedule_overrides WHERE id = $1", overrideID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete override"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Override deleted"})
}

// applyOverrides merges per-trainee overrides into a base set of schedule items for a given (week, day).
func applyOverrides(baseItems []models.TrainingScheduleItem, overrides []models.ScheduleOverride) []models.TrainingScheduleItem {
	// Build a set of base item IDs to remove
	removeSet := make(map[int]bool)
	replaceMap := make(map[int]models.ScheduleOverride)

	for _, o := range overrides {
		switch o.Action {
		case "remove":
			if o.BaseItemID != nil {
				removeSet[*o.BaseItemID] = true
			}
		case "replace":
			if o.BaseItemID != nil {
				replaceMap[*o.BaseItemID] = o
			}
		}
	}

	// Filter + replace base items
	var result []models.TrainingScheduleItem
	for _, item := range baseItems {
		if removeSet[item.ID] {
			continue
		}
		if repl, ok := replaceMap[item.ID]; ok {
			item.Title = repl.Title
			item.Description = repl.Description
			item.DayTitle = repl.DayTitle
			item.LinkURL = repl.LinkURL
			item.LinkLabel = repl.LinkLabel
			item.TimeSlot = repl.TimeSlot
			item.StartTime = repl.StartTime
			item.EndTime = repl.EndTime
			item.SortOrder = repl.SortOrder
			item.IsHighlight = repl.IsHighlight
		}
		result = append(result, item)
	}

	// Append 'add' overrides as new items (use negative IDs so frontend can distinguish)
	for i, o := range overrides {
		if o.Action == "add" {
			result = append(result, models.TrainingScheduleItem{
				ID:          -(o.ID), // negative = override-sourced
				ProgramID:   0,
				WeekNumber:  o.WeekNumber,
				DayOfWeek:   o.DayOfWeek,
				DayTitle:    o.DayTitle,
				Title:       o.Title,
				Description: o.Description,
				LinkURL:     o.LinkURL,
				LinkLabel:   o.LinkLabel,
				TimeSlot:    o.TimeSlot,
				StartTime:   o.StartTime,
				EndTime:     o.EndTime,
				SortOrder:   o.SortOrder + 1000 + i, // sort after base items
				IsHighlight: o.IsHighlight,
			})
		}
	}

	return result
}

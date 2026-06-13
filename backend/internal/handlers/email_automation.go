package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wbmccurry20/jbs-internal-portal/internal/services"
)

// EmailAutomationHandler serves the Email Automation feature API.
type EmailAutomationHandler struct {
	DB *sql.DB
}

// GetStatus returns the current email automation configuration and connection state.
// GET /api/email-automation/status
func (h *EmailAutomationHandler) GetStatus(c *gin.Context) {
	// Read config keys from DB (all may be absent or empty — must not error out).
	configKeys := []string{"enabled", "mailbox_address", "last_processed_at", "oauth_access_token"}
	cfg := make(map[string]string, len(configKeys))
	for _, k := range configKeys {
		var v string
		err := h.DB.QueryRow(
			`SELECT value FROM email_automation_config WHERE key = $1`, k,
		).Scan(&v)
		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read email config"})
			return
		}
		cfg[k] = v
	}

	client := GetMailGraphClient()
	clientConfigured := client != nil && client.IsConfigured()
	connected := clientConfigured && cfg["oauth_access_token"] != ""

	var processedToday int
	h.DB.QueryRow(
		`SELECT COUNT(*) FROM email_automation_log WHERE processed_at::date = CURRENT_DATE`,
	).Scan(&processedToday)

	c.JSON(http.StatusOK, gin.H{
		"configured":        clientConfigured,
		"connected":         connected,
		"enabled":           cfg["enabled"] == "true",
		"mailbox_address":   cfg["mailbox_address"],
		"last_processed_at": cfg["last_processed_at"],
		"processed_today":   processedToday,
	})
}

// ListFolderMappings returns all rows from email_folder_mappings ordered by folder_path.
// GET /api/email-automation/folder-mappings
func (h *EmailAutomationHandler) ListFolderMappings(c *gin.Context) {
	rows, err := h.DB.Query(`
		SELECT id, project_name, folder_name, folder_path,
		       parent_folder_name, is_bid_submitted_sub, bid_id
		FROM email_folder_mappings
		ORDER BY folder_path`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query folder mappings"})
		return
	}
	defer rows.Close()

	type folderMappingRow struct {
		ID                int     `json:"id"`
		ProjectName       string  `json:"project_name"`
		FolderName        string  `json:"folder_name"`
		FolderPath        string  `json:"folder_path"`
		ParentFolderName  string  `json:"parent_folder_name"`
		IsBidSubmittedSub bool    `json:"is_bid_submitted_sub"`
		BidID             *int    `json:"bid_id"`
	}

	result := []folderMappingRow{}
	for rows.Next() {
		var r folderMappingRow
		var parentFolderName sql.NullString
		var bidID sql.NullInt64
		if err := rows.Scan(
			&r.ID, &r.ProjectName, &r.FolderName, &r.FolderPath,
			&parentFolderName, &r.IsBidSubmittedSub, &bidID,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan folder mapping"})
			return
		}
		if parentFolderName.Valid {
			r.ParentFolderName = parentFolderName.String
		}
		if bidID.Valid {
			v := int(bidID.Int64)
			r.BidID = &v
		}
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to iterate folder mappings"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// UpdateFolderMapping updates the project_name of a folder mapping by ID.
// PUT /api/email-automation/folder-mappings/:id
func (h *EmailAutomationHandler) UpdateFolderMapping(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var body struct {
		ProjectName string `json:"project_name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	result, err := h.DB.Exec(
		`UPDATE email_folder_mappings SET project_name = $1, updated_at = NOW() WHERE id = $2`,
		body.ProjectName, id,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update folder mapping"})
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "folder mapping not found"})
		return
	}

	// Return the updated row.
	type folderMappingRow struct {
		ID                int        `json:"id"`
		ProjectName       string     `json:"project_name"`
		FolderName        string     `json:"folder_name"`
		FolderPath        string     `json:"folder_path"`
		ParentFolderName  string     `json:"parent_folder_name"`
		IsBidSubmittedSub bool       `json:"is_bid_submitted_sub"`
		BidID             *int       `json:"bid_id"`
		UpdatedAt         time.Time  `json:"updated_at"`
	}

	var r folderMappingRow
	var parentFolderName sql.NullString
	var bidID sql.NullInt64
	err = h.DB.QueryRow(`
		SELECT id, project_name, folder_name, folder_path,
		       parent_folder_name, is_bid_submitted_sub, bid_id, updated_at
		FROM email_folder_mappings WHERE id = $1`, id).
		Scan(&r.ID, &r.ProjectName, &r.FolderName, &r.FolderPath,
			&parentFolderName, &r.IsBidSubmittedSub, &bidID, &r.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read updated row"})
		return
	}
	if parentFolderName.Valid {
		r.ParentFolderName = parentFolderName.String
	}
	if bidID.Valid {
		v := int(bidID.Int64)
		r.BidID = &v
	}

	c.JSON(http.StatusOK, r)
}

// ListLogs returns email automation log entries with pagination.
// GET /api/email-automation/logs?limit=50&offset=0
func (h *EmailAutomationHandler) ListLogs(c *gin.Context) {
	limit := 50
	offset := 0

	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			if n < 1 {
				n = 1
			} else if n > 200 {
				n = 200
			}
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	rows, err := h.DB.Query(`
		SELECT id, message_id, subject, sender, action_taken,
		       destination_folder, rule_matched, error, processed_at
		FROM email_automation_log
		ORDER BY processed_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query logs"})
		return
	}
	defer rows.Close()

	type logRow struct {
		ID                int        `json:"id"`
		MessageID         string     `json:"message_id"`
		Subject           string     `json:"subject"`
		Sender            string     `json:"sender"`
		ActionTaken       string     `json:"action_taken"`
		DestinationFolder string     `json:"destination_folder"`
		RuleMatched       string     `json:"rule_matched"`
		Error             string     `json:"error"`
		ProcessedAt       time.Time  `json:"processed_at"`
	}

	result := []logRow{}
	for rows.Next() {
		var r logRow
		var subject, sender, actionTaken, destinationFolder, ruleMatched, errText sql.NullString
		if err := rows.Scan(
			&r.ID, &r.MessageID, &subject, &sender, &actionTaken,
			&destinationFolder, &ruleMatched, &errText, &r.ProcessedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan log row"})
			return
		}
		r.Subject = subject.String
		r.Sender = sender.String
		r.ActionTaken = actionTaken.String
		r.DestinationFolder = destinationFolder.String
		r.RuleMatched = ruleMatched.String
		r.Error = errText.String
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to iterate logs"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// SyncFolders triggers an immediate folder sync from the shared mailbox.
// POST /api/email-automation/sync-folders
func (h *EmailAutomationHandler) SyncFolders(c *gin.Context) {
	client := GetMailGraphClient()
	if client == nil || !client.IsConfigured() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email not connected — connect the mailbox first."})
		return
	}

	var mailbox string
	h.DB.QueryRow(
		`SELECT value FROM email_automation_config WHERE key = 'mailbox_address'`,
	).Scan(&mailbox)

	if mailbox == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email not connected — connect the mailbox first."})
		return
	}

	count, err := services.SyncFolderMappings(h.DB, client, mailbox)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"synced": count})
}

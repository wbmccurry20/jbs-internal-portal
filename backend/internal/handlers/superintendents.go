package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wbmccurry20/jbs-internal-portal/internal/database"
)

// ListSuperintendents returns all active superintendents (for dropdown population)
func ListSuperintendents(c *gin.Context) {
	query := `SELECT id, name, email, phone, active FROM superintendents WHERE active = true ORDER BY name`

	rows, err := database.DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch superintendents"})
		return
	}
	defer rows.Close()

	var supers []map[string]interface{}
	for rows.Next() {
		var id int
		var name, email, phone string
		var active bool
		if err := rows.Scan(&id, &name, &email, &phone, &active); err != nil {
			continue
		}
		supers = append(supers, map[string]interface{}{
			"id":     id,
			"name":   name,
			"email":  email,
			"phone":  phone,
			"active": active,
		})
	}

	if supers == nil {
		supers = []map[string]interface{}{}
	}

	c.JSON(http.StatusOK, supers)
}

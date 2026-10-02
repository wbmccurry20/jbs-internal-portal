package handlers

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wbmccurry20/jbs-internal-portal/internal/services"
	"github.com/wbmccurry20/jbs-internal-portal/internal/utils"
)

func ConvertReimbursementFile(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No Excel file uploaded"})
		return
	}

	extension := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if extension != ".xlsx" && extension != ".xlsm" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Upload an .xlsx or .xlsm reimbursement workbook"})
		return
	}
	if err := utils.ValidateExcelFile(fileHeader); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Excel upload: " + err.Error()})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to read uploaded workbook"})
		return
	}
	defer file.Close()

	description := strings.TrimSpace(c.PostForm("invoice_description"))
	if description == "" {
		description = "Employee Reimbursement"
	}
	conversion, err := services.ConvertReimbursementWorkbook(file, description)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", `attachment; filename="foundation_reimbursements.csv"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", conversion.CSV)
}

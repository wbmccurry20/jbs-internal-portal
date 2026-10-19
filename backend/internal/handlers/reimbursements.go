package handlers

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wbmccurry20/jbs-internal-portal/internal/services"
	"github.com/wbmccurry20/jbs-internal-portal/internal/utils"
)

func ConvertReimbursementFile(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No reimbursement file uploaded"})
		return
	}

	extension := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if extension != ".csv" && extension != ".xlsx" && extension != ".xlsm" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Upload an accounting .csv or .xlsx/.xlsm reimbursement workbook"})
		return
	}
	validate := utils.ValidateExcelFile
	if extension == ".csv" {
		switch fileHeader.Header.Get("Content-Type") {
		case "application/vnd.ms-excel", "application/octet-stream", "":
			fileHeader.Header.Set("Content-Type", "text/csv")
		}
		validate = utils.ValidateCSVFile
	}
	if err := validate(fileHeader); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reimbursement upload: " + err.Error()})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to read uploaded file"})
		return
	}
	defer file.Close()

	description := strings.TrimSpace(c.PostForm("invoice_description"))
	if description == "" {
		description = "Employee Reimbursement"
	}
	convert := services.ConvertReimbursementWorkbook
	if extension == ".csv" {
		convert = services.ConvertReimbursementCSV
	}
	conversion, err := convert(file, description)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", `attachment; filename="foundation_reimbursements.csv"`)
	c.Header("Access-Control-Expose-Headers", "X-Reimbursement-Shortened-Descriptions")
	c.Header("X-Reimbursement-Shortened-Descriptions", strconv.Itoa(conversion.ShortenedDescriptions))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", conversion.CSV)
}

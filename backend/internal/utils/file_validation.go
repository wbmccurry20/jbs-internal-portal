package utils

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
)

// AllowedExcelTypes are the MIME types we accept for Excel files
var AllowedExcelTypes = map[string]bool{
	"application/vnd.ms-excel":                                        true, // .xls
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true, // .xlsx
	"application/vnd.ms-excel.sheet.macroEnabled.12":                  true, // .xlsm
}

// AllowedCSVTypes are the MIME types we accept for CSV files
var AllowedCSVTypes = map[string]bool{
	"text/csv":          true,
	"application/csv":   true,
	"text/plain":        true, // Some systems send CSV as text/plain
}

// AllowedExtensions are file extensions we accept
var AllowedExtensions = map[string]bool{
	".xls":  true,
	".xlsx": true,
	".xlsm": true,
	".csv":  true,
}

// ValidateExcelFile validates that the uploaded file is a valid Excel file
func ValidateExcelFile(file *multipart.FileHeader) error {
	// Check file size (already done by middleware, but double-check)
	maxSize := int64(10 * 1024 * 1024) // 10MB
	if file.Size > maxSize {
		return fmt.Errorf("file size exceeds maximum allowed (10MB)")
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !AllowedExtensions[ext] {
		return fmt.Errorf("invalid file extension: %s (allowed: .xls, .xlsx, .xlsm, .csv)", ext)
	}

	// Check MIME type
	contentType := file.Header.Get("Content-Type")
	if !AllowedExcelTypes[contentType] && !AllowedCSVTypes[contentType] {
		return fmt.Errorf("invalid file type: %s", contentType)
	}

	// Check filename for malicious patterns
	if strings.Contains(file.Filename, "..") || strings.Contains(file.Filename, "/") || strings.Contains(file.Filename, "\\") {
		return fmt.Errorf("invalid filename: contains path traversal characters")
	}

	return nil
}

// ValidateCSVFile validates that the uploaded file is a valid CSV file
func ValidateCSVFile(file *multipart.FileHeader) error {
	// Check file size
	maxSize := int64(10 * 1024 * 1024) // 10MB
	if file.Size > maxSize {
		return fmt.Errorf("file size exceeds maximum allowed (10MB)")
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".csv" {
		return fmt.Errorf("invalid file extension: %s (must be .csv)", ext)
	}

	// Check MIME type
	contentType := file.Header.Get("Content-Type")
	if !AllowedCSVTypes[contentType] {
		return fmt.Errorf("invalid file type: %s", contentType)
	}

	// Check filename for malicious patterns
	if strings.Contains(file.Filename, "..") || strings.Contains(file.Filename, "/") || strings.Contains(file.Filename, "\\") {
		return fmt.Errorf("invalid filename: contains path traversal characters")
	}

	return nil
}

// SanitizeFilename removes potentially dangerous characters from filename
func SanitizeFilename(filename string) string {
	// Replace dangerous characters with underscores
	dangerous := []string{"/", "\\", "..", "<", ">", ":", "\"", "|", "?", "*"}
	sanitized := filename
	for _, char := range dangerous {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}
	return sanitized
}

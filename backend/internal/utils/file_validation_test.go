package utils

import (
	"mime/multipart"
	"net/textproto"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createMockFileHeader(filename, contentType string, size int64) *multipart.FileHeader {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", contentType)
	
	return &multipart.FileHeader{
		Filename: filename,
		Header:   header,
		Size:     size,
	}
}

func TestValidateExcelFile(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		contentType string
		size        int64
		shouldError bool
	}{
		{
			name:        "Valid XLSX file",
			filename:    "test.xlsx",
			contentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			size:        1024,
			shouldError: false,
		},
		{
			name:        "Valid XLS file",
			filename:    "test.xls",
			contentType: "application/vnd.ms-excel",
			size:        1024,
			shouldError: false,
		},
		{
			name:        "File too large",
			filename:    "test.xlsx",
			contentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			size:        11 * 1024 * 1024, // 11MB
			shouldError: true,
		},
		{
			name:        "Invalid extension",
			filename:    "test.exe",
			contentType: "application/octet-stream",
			size:        1024,
			shouldError: true,
		},
		{
			name:        "Path traversal attempt",
			filename:    "../../../etc/passwd",
			contentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			size:        1024,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := createMockFileHeader(tt.filename, tt.contentType, tt.size)
			err := ValidateExcelFile(file)
			
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal.xlsx", "normal.xlsx"},
		{"../../../etc/passwd", "______etc_passwd"},
		{"file<script>.xlsx", "file_script_.xlsx"},
		{"file|name.xlsx", "file_name.xlsx"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

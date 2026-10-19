package handlers

import (
	"bytes"
	"encoding/csv"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestConvertReimbursementFileAccountingCSV(t *testing.T) {
	fixture, err := os.ReadFile("../../../docs/data/USD_Synchronized_Accounting_Extract_2_20261016.csv")
	if err != nil {
		t.Fatal(err)
	}
	for _, contentType := range []string{"text/csv", "application/vnd.ms-excel", "application/octet-stream"} {
		t.Run(contentType, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			header := make(textproto.MIMEHeader)
			header.Set("Content-Disposition", `form-data; name="file"; filename="accounting.csv"`)
			header.Set("Content-Type", contentType)
			part, err := writer.CreatePart(header)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := io.Copy(part, bytes.NewReader(fixture)); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			router := gin.New()
			router.POST("/convert", ConvertReimbursementFile)
			request := httptest.NewRequest(http.MethodPost, "/convert", &body)
			request.Header.Set("Content-Type", writer.FormDataContentType())
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if response.Header().Get("Content-Disposition") == "" || response.Header().Get("X-Reimbursement-Shortened-Descriptions") == "0" {
				t.Fatal("missing download or shortening warning")
			}
			rows, err := csv.NewReader(response.Body).ReadAll()
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 60 {
				t.Fatalf("CSV has %d rows, want 14 template + 46 expenses", len(rows))
			}
		})
	}
}

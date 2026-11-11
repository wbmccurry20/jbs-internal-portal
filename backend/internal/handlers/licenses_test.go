package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/wbmccurry20/jbs-internal-portal/internal/database"
)

func TestLicenseJurisdictionValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, body := range []string{
		`{"state":"AL","license_type":"Business","city":"","is_city_license":true}`,
		`{"state":"AL","license_type":"Business","city":"Hueytown","is_city_license":false}`,
		`{"state":"  ","license_type":"Business"}`,
	} {
		for _, handler := range []gin.HandlerFunc{CreateLicense, UpdateLicense} {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")
			handler(c)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("body %s: got status %d, want 400", body, w.Code)
			}
		}
	}
}

func TestLicenseAPIWithPostgres(t *testing.T) {
	url := os.Getenv("TEST_LICENSE_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_LICENSE_DATABASE_URL is required for PostgreSQL integration test")
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`
					CREATE TEMP TABLE state_licenses (
						id serial primary key, state text, license_type text, license_number text,
						entity_name text, issue_date date, expiration_date date, status text,
						renewal_fee numeric, notes text, city text, is_city_license boolean,
						is_active boolean, created_at timestamp default now(), updated_at timestamp default now()
					);
					CREATE TEMP TABLE license_documents (license_id integer);
				`)
	if err != nil {
		t.Fatal(err)
	}
	previous := database.DB
	database.DB = db
	defer func() { database.DB = previous }()
	var expiration string
	if err := db.QueryRow(`SELECT ((CURRENT_TIMESTAMP AT TIME ZONE 'America/New_York')::date + 60)::text`).Scan(&expiration); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	call := func(handler gin.HandlerFunc, method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(method, path, strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		handler(c)
		return w
	}
	body := `{"state":"AL","license_type":"Business","city":"Hueytown","is_city_license":true,"expiration_date":"` + expiration + `"}`
	if w := call(CreateLicense, http.MethodPost, "/", body); w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}
	if w := call(CreateLicense, http.MethodPost, "/", `{"state":"GA","license_type":"Contractor"}`); w.Code != http.StatusCreated {
		t.Fatalf("create undated: %d %s", w.Code, w.Body)
	}
	w := call(ListLicenses, http.MethodGet, "/?status=expiring", "")
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d %s", w.Code, w.Body)
	}
	var licenses []struct {
		City   string `json:"city"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &licenses); err != nil {
		t.Fatal(err)
	}
	if len(licenses) != 1 || licenses[0].City != "Hueytown" || licenses[0].Status != "expiring" {
		t.Fatalf("unexpected filtered licenses: %+v", licenses)
	}
	w = call(ListLicenses, http.MethodGet, "/?status=unknown", "")
	if err := json.Unmarshal(w.Body.Bytes(), &licenses); err != nil || len(licenses) != 1 || licenses[0].Status != "unknown" {
		t.Fatalf("unexpected undated licenses: %+v, error=%v", licenses, err)
	}
	w = call(GetStateSummary, http.MethodGet, "/", "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"expiring":1`) {
		t.Fatalf("state summary: %d %s", w.Code, w.Body)
	}
}

func TestEffectiveLicenseStatus(t *testing.T) {
	url := os.Getenv("TEST_LICENSE_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_LICENSE_DATABASE_URL is required for PostgreSQL integration test")
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	query := `SELECT ` + effectiveLicenseStatus + `
		FROM (SELECT $1::text AS status,
			CASE WHEN $2::integer IS NULL THEN NULL
			ELSE (CURRENT_TIMESTAMP AT TIME ZONE 'America/New_York')::date + $2::integer END AS expiration_date) sl`
	for _, tc := range []struct {
		stored string
		days   interface{}
		want   string
	}{
		{"active", nil, "unknown"},
		{"pending", nil, "pending"},
		{"pending", 15, "pending"},
		{"active", -1, "expired"},
		{"active", 0, "expiring"},
		{"active", 60, "expiring"},
		{"expiring", 61, "active"},
	} {
		var got string
		if err := db.QueryRow(query, tc.stored, tc.days).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("stored=%s days=%v: got %s, want %s", tc.stored, tc.days, got, tc.want)
		}
	}
}

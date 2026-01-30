package models

import "time"

// StateLicense represents a state construction license
type StateLicense struct {
	ID               int        `json:"id"`
	State            string     `json:"state"`
	LicenseType      string     `json:"license_type"`
	LicenseNumber    string     `json:"license_number"`
	EntityName       string     `json:"entity_name"`
	IssueDate        *time.Time `json:"issue_date"`
	ExpirationDate   *time.Time `json:"expiration_date"`
	Status           string     `json:"status"`
	RenewalFee       *float64   `json:"renewal_fee"`
	Notes            string     `json:"notes"`
	OnedriveFilePath string     `json:"onedrive_file_path"`
	LastSyncedAt     *time.Time `json:"last_synced_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// StateSummary represents license status for a state (for map view)
type StateSummary struct {
	StateCode     string  `json:"state_code"`
	StateName     string  `json:"state_name"`
	Status        string  `json:"status"` // active, expiring, expired, none
	LicenseCount  int     `json:"license_count"`
	ExpiringCount int     `json:"expiring_count"`
	TotalFees     float64 `json:"total_fees"`
}

// OneDriveSyncLog tracks synchronization jobs
type OneDriveSyncLog struct {
	ID               int        `json:"id"`
	SyncType         string     `json:"sync_type"`
	RecordsProcessed int        `json:"records_processed"`
	RecordsCreated   int        `json:"records_created"`
	RecordsUpdated   int        `json:"records_updated"`
	Errors           string     `json:"errors"`
	StartedAt        *time.Time `json:"started_at"`
	CompletedAt      *time.Time `json:"completed_at"`
}

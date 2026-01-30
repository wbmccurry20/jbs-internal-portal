package models

import (
	"time"
)

// Job represents a construction job/project
type Job struct {
	ID                    int            `json:"id"`
	JobNumber             string         `json:"job_number"`
	JobName               string         `json:"job_name"`
	Location              string         `json:"location"`
	City                  string         `json:"city"`
	State                 string         `json:"state"`
	Latitude              *float64       `json:"latitude"`
	Longitude             *float64       `json:"longitude"`
	ClientID              *int           `json:"client_id"`
	Status                string         `json:"status"`
	ContractValue         *float64       `json:"contract_value"`
	RevisedContractValue  *float64       `json:"revised_contract_value"`
	StartDate             *time.Time     `json:"start_date"`
	ProjectedEndDate      *time.Time     `json:"projected_end_date"`
	ActualEndDate         *time.Time     `json:"actual_end_date"`
	ProjectManagerID      *int           `json:"project_manager_id"`
	APMID                 *int           `json:"apm_id"`
	SuperintendentID      *int           `json:"superintendent_id"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`

	// Relationships (populated via joins)
	Client          *Client         `json:"client,omitempty"`
	Superintendent  *Superintendent `json:"superintendent,omitempty"`
	ProjectManager  *User           `json:"project_manager,omitempty"`
	APM             *User           `json:"apm,omitempty"`
}

// JobWithDetails includes all related data
type JobWithDetails struct {
	Job
	Updates      []JobUpdate    `json:"updates"`
	ChangeOrders []ChangeOrder  `json:"change_orders"`
}

// JobListItem is a lightweight version for list views
type JobListItem struct {
	ID                   int        `json:"id"`
	JobNumber            string     `json:"job_number"`
	JobName              string     `json:"job_name"`
	City                 string     `json:"city"`
	State                string     `json:"state"`
	ClientName           string     `json:"client_name"`
	Status               string     `json:"status"`
	SuperintendentName   string     `json:"superintendent_name"`
	ProjectManagerName   string     `json:"project_manager_name"`
	ContractValue        *float64   `json:"contract_value"`
	StartDate            *time.Time `json:"start_date"`
	ProjectedEndDate     *time.Time `json:"projected_end_date"`
}

// JobUpdate represents meeting minutes and status updates
type JobUpdate struct {
	ID         int       `json:"id"`
	JobID      int       `json:"job_id"`
	AuthorID   *int      `json:"author_id"`
	AuthorName string    `json:"author_name"`
	UpdateText string    `json:"update_text"`
	CreatedAt  time.Time `json:"created_at"`
}

// ChangeOrder represents contract changes
type ChangeOrder struct {
	ID         int        `json:"id"`
	JobID      int        `json:"job_id"`
	Description string    `json:"description"`
	Amount     *float64   `json:"amount"`
	Status     string     `json:"status"`
	CreatedBy  *int       `json:"created_by"`
	ApprovedBy *int       `json:"approved_by"`
	CreatedAt  time.Time  `json:"created_at"`
	ApprovedAt *time.Time `json:"approved_at"`
}

// CreateJobRequest is the request body for creating a job
type CreateJobRequest struct {
	JobNumber            string     `json:"job_number" binding:"required"`
	JobName              string     `json:"job_name" binding:"required"`
	Location             string     `json:"location"`
	City                 string     `json:"city"`
	State                string     `json:"state"`
	ClientID             *int       `json:"client_id"`
	Status               string     `json:"status"`
	ContractValue        *float64   `json:"contract_value"`
	StartDate            *time.Time `json:"start_date"`
	ProjectedEndDate     *time.Time `json:"projected_end_date"`
	ProjectManagerID     *int       `json:"project_manager_id"`
	APMID                *int       `json:"apm_id"`
	SuperintendentID     *int       `json:"superintendent_id"`
}

// UpdateJobRequest is the request body for updating a job
type UpdateJobRequest struct {
	JobName              *string    `json:"job_name"`
	Location             *string    `json:"location"`
	City                 *string    `json:"city"`
	State                *string    `json:"state"`
	ClientID             *int       `json:"client_id"`
	Status               *string    `json:"status"`
	ContractValue        *float64   `json:"contract_value"`
	RevisedContractValue *float64   `json:"revised_contract_value"`
	StartDate            *time.Time `json:"start_date"`
	ProjectedEndDate     *time.Time `json:"projected_end_date"`
	ActualEndDate        *time.Time `json:"actual_end_date"`
	ProjectManagerID     *int       `json:"project_manager_id"`
	APMID                *int       `json:"apm_id"`
	SuperintendentID     *int       `json:"superintendent_id"`
}

// JobFilters for filtering job lists
type JobFilters struct {
	Status           string
	ClientID         *int
	SuperintendentID *int
	ProjectManagerID *int
	State            string
	Search           string // Search in job_number, job_name, location
}

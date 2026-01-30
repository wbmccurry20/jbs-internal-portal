package models

import "time"

// SuperintendentLodging represents temporary housing for superintendents
type SuperintendentLodging struct {
	ID                   int        `json:"id"`
	SuperintendentID     int        `json:"superintendent_id"`
	JobID                *int       `json:"job_id"`
	Location             string     `json:"location"`
	CheckInDate          *time.Time `json:"check_in_date"`
	CheckOutDate         *time.Time `json:"check_out_date"`
	PropertyLink         string     `json:"property_link"`
	Address              string     `json:"address"`
	JobsiteAddress       string     `json:"jobsite_address"`
	CostPerNight         *float64   `json:"cost_per_night"`
	TotalCost            *float64   `json:"total_cost"`
	BookingConfirmation  string     `json:"booking_confirmation"`
	Notes                string     `json:"notes"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`

	// Relationships
	Superintendent *Superintendent `json:"superintendent,omitempty"`
	Job            *Job            `json:"job,omitempty"`
}

// CreateLodgingRequest is the request body for creating lodging
type CreateLodgingRequest struct {
	SuperintendentID    int        `json:"superintendent_id" binding:"required"`
	JobID               *int       `json:"job_id"`
	Location            string     `json:"location"`
	CheckInDate         *time.Time `json:"check_in_date"`
	CheckOutDate        *time.Time `json:"check_out_date"`
	PropertyLink        string     `json:"property_link"`
	Address             string     `json:"address"`
	JobsiteAddress      string     `json:"jobsite_address"`
	CostPerNight        *float64   `json:"cost_per_night"`
	TotalCost           *float64   `json:"total_cost"`
	BookingConfirmation string     `json:"booking_confirmation"`
	Notes               string     `json:"notes"`
}

// UpdateLodgingRequest is the request body for updating lodging
type UpdateLodgingRequest struct {
	SuperintendentID    int        `json:"superintendent_id"`
	JobID               *int       `json:"job_id"`
	Location            string     `json:"location"`
	CheckInDate         *time.Time `json:"check_in_date"`
	CheckOutDate        *time.Time `json:"check_out_date"`
	PropertyLink        string     `json:"property_link"`
	Address             string     `json:"address"`
	JobsiteAddress      string     `json:"jobsite_address"`
	CostPerNight        *float64   `json:"cost_per_night"`
	TotalCost           *float64   `json:"total_cost"`
	BookingConfirmation string     `json:"booking_confirmation"`
	Notes               string     `json:"notes"`
}

// LodgingFilters for querying lodging
type LodgingFilters struct {
	SuperintendentID *int   `form:"superintendent_id"`
	JobID            *int   `form:"job_id"`
	CurrentOnly      bool   `form:"current_only"` // Only active reservations
	Search           string `form:"search"`
	Limit            int    `form:"limit"`
	Offset           int    `form:"offset"`
	SortBy           string `form:"sort_by"`
	SortOrder        string `form:"sort_order"`
}

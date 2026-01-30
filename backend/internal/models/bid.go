package models

import "time"

// Bid represents a bid opportunity in the sales pipeline
type Bid struct {
	ID                    int        `json:"id"`
	ClientID              *int       `json:"client_id"`
	Location              string     `json:"location"`
	City                  string     `json:"city"`
	State                 string     `json:"state"`
	DueDate               *time.Time `json:"due_date"`
	AssignedToID          *int       `json:"assigned_to_id"`
	AssignedToName        string     `json:"assigned_to_name"` // For legacy import
	Status                string     `json:"status"`
	BuildingConnectedDate *time.Time `json:"building_connected_date"`
	PlanHubDate           *time.Time `json:"plan_hub_date"`
	Awarded               string     `json:"awarded"` // "Yes", "No", or empty
	BidAmount             *float64   `json:"bid_amount"`
	Notes                 string     `json:"notes"`
	JobID                 *int       `json:"job_id"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	Archived              bool       `json:"archived"`

	// Relationships
	Client     *Client `json:"client,omitempty"`
	AssignedTo *User   `json:"assigned_to,omitempty"`
	Job        *Job    `json:"job,omitempty"`
}

// CreateBidRequest is the request body for creating a bid
type CreateBidRequest struct {
	ClientID              *int       `json:"client_id"`
	Location              string     `json:"location" binding:"required"`
	City                  string     `json:"city"`
	State                 string     `json:"state"`
	DueDate               *time.Time `json:"due_date"`
	AssignedToID          *int       `json:"assigned_to_id"`
	AssignedToName        string     `json:"assigned_to_name"`
	Status                string     `json:"status"`
	BuildingConnectedDate *time.Time `json:"building_connected_date"`
	PlanHubDate           *time.Time `json:"plan_hub_date"`
	Awarded               string     `json:"awarded"`
	BidAmount             *float64   `json:"bid_amount"`
	Notes                 string     `json:"notes"`
}

// UpdateBidRequest is the request body for updating a bid
type UpdateBidRequest struct {
	ClientID              *int       `json:"client_id"`
	Location              string     `json:"location"`
	City                  string     `json:"city"`
	State                 string     `json:"state"`
	DueDate               *time.Time `json:"due_date"`
	AssignedToID          *int       `json:"assigned_to_id"`
	AssignedToName        string     `json:"assigned_to_name"`
	Status                string     `json:"status"`
	BuildingConnectedDate *time.Time `json:"building_connected_date"`
	PlanHubDate           *time.Time `json:"plan_hub_date"`
	Awarded               string     `json:"awarded"`
	BidAmount             *float64   `json:"bid_amount"`
	Notes                 string     `json:"notes"`
	JobID                 *int       `json:"job_id"`
	Archived              *bool      `json:"archived"`
}

// BidFilters for querying bids
type BidFilters struct {
	ClientID     *int    `form:"client_id"`
	AssignedToID *int    `form:"assigned_to_id"`
	Status       string  `form:"status"`
	Awarded      string  `form:"awarded"`
	Archived     *bool   `form:"archived"`
	Search       string  `form:"search"`
	Limit        int     `form:"limit"`
	Offset       int     `form:"offset"`
	SortBy       string  `form:"sort_by"`
	SortOrder    string  `form:"sort_order"`
}

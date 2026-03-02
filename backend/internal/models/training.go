package models

import "time"

// TrainingProgram is a template for new-hire training (e.g., "APM 30-Day Onboarding")
type TrainingProgram struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	DurationWeeks int       `json:"duration_weeks"`
	CreatedBy     int       `json:"created_by"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Populated on detail queries
	Items     []TrainingScheduleItem `json:"items,omitempty"`
	Resources []TrainingResource     `json:"resources,omitempty"`
}

// TrainingScheduleItem is one bullet-point / task for a specific day
type TrainingScheduleItem struct {
	ID          int       `json:"id"`
	ProgramID   int       `json:"program_id"`
	WeekNumber  int       `json:"week_number"`
	DayOfWeek   int       `json:"day_of_week"` // 1=Mon … 5=Fri
	DayTitle    string    `json:"day_title"`    // optional theme like "Site Visit"
	Title       string    `json:"title"`
	Description string    `json:"description"`
	LinkURL     string    `json:"link_url"`
	LinkLabel   string    `json:"link_label"`
	TimeSlot    string    `json:"time_slot"` // morning, afternoon, all-day
	SortOrder   int       `json:"sort_order"`
	IsHighlight bool      `json:"is_highlight"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TraineeAssignment links a user (trainee) to a training program
type TraineeAssignment struct {
	ID         int       `json:"id"`
	ProgramID  int       `json:"program_id"`
	UserID     int       `json:"user_id"`
	StartDate  string    `json:"start_date"` // YYYY-MM-DD
	Status     string    `json:"status"`     // active, completed, paused
	Notes      string    `json:"notes"`
	AssignedBy int       `json:"assigned_by"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// Populated on queries
	TraineeName    string `json:"trainee_name,omitempty"`
	TraineeEmail   string `json:"trainee_email,omitempty"`
	ProgramName    string `json:"program_name,omitempty"`
	AssignedByName string `json:"assigned_by_name,omitempty"`
}

// TrainingResource is a general link/document attached to a program
type TrainingResource struct {
	ID          int       `json:"id"`
	ProgramID   int       `json:"program_id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
}

// CalendarDay represents one day on a trainee's calendar
type CalendarDay struct {
	Date      string                 `json:"date"` // YYYY-MM-DD
	DayTitle  string                 `json:"day_title"`
	DayOfWeek int                    `json:"day_of_week"` // 1=Mon … 5=Fri
	Week      int                    `json:"week"`
	Items     []TrainingScheduleItem `json:"items"`
	IsToday   bool                   `json:"is_today"`
	IsPast    bool                   `json:"is_past"`
}

// TraineeCalendar is the full calendar for a trainee's program
type TraineeCalendar struct {
	Assignment  TraineeAssignment `json:"assignment"`
	Program     TrainingProgram   `json:"program"`
	Days        []CalendarDay     `json:"days"`
	Resources   []TrainingResource `json:"resources"`
	CurrentWeek int               `json:"current_week"`
	TotalWeeks  int               `json:"total_weeks"`
}

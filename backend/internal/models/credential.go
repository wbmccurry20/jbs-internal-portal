package models

import "time"

// Credential represents an encrypted password/credential
type Credential struct {
	ID                int       `json:"id"`
	Category          string    `json:"category"`
	Name              string    `json:"name"`
	Username          string    `json:"username"`
	EncryptedPassword string    `json:"-"` // Never send to client
	Password          string    `json:"password,omitempty"` // Only populated when decrypted
	URL               string    `json:"url"`
	Notes             string    `json:"notes"`
	CreatedBy         *int      `json:"created_by"`
	SharedWith        []int     `json:"shared_with"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// CredentialListItem is for list views (no password)
type CredentialListItem struct {
	ID         int       `json:"id"`
	Category   string    `json:"category"`
	Name       string    `json:"name"`
	Username   string    `json:"username"`
	URL        string    `json:"url"`
	CreatedBy  *int      `json:"created_by"`
	SharedWith []int     `json:"shared_with"`
	CreatedAt  time.Time `json:"created_at"`
}

// CredentialAuditLog tracks who accessed what
type CredentialAuditLog struct {
	ID           int       `json:"id"`
	CredentialID int       `json:"credential_id"`
	UserID       int       `json:"user_id"`
	Action       string    `json:"action"` // create, view, update, delete, share
	IPAddress    string    `json:"ip_address"`
	CreatedAt    time.Time `json:"created_at"`
	
	// For display
	UserName       string `json:"user_name,omitempty"`
	CredentialName string `json:"credential_name,omitempty"`
}

// CreateCredentialRequest for API
type CreateCredentialRequest struct {
	Category string `json:"category" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Username string `json:"username"`
	Password string `json:"password" binding:"required"`
	URL      string `json:"url"`
	Notes    string `json:"notes"`
}

// UpdateCredentialRequest for API
type UpdateCredentialRequest struct {
	Category *string `json:"category"`
	Name     *string `json:"name"`
	Username *string `json:"username"`
	Password *string `json:"password"` // Only if changing password
	URL      *string `json:"url"`
	Notes    *string `json:"notes"`
}

// ShareCredentialRequest for API
type ShareCredentialRequest struct {
	UserIDs []int `json:"user_ids" binding:"required"`
}

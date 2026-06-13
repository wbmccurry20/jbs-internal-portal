package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MailMessage represents a single message returned from the Graph mail API.
type MailMessage struct {
	ID             string
	Subject        string
	Sender         string // emailAddress.address
	SenderName     string // emailAddress.name
	ReceivedAt     time.Time
	IsRead         bool
	IsFlagged      bool
	ConversationID string
	BodyPreview    string
	ParentFolderID string
}

// MailFolder represents an Outlook mail folder returned from the Graph mail API.
type MailFolder struct {
	ID               string
	DisplayName      string
	ParentFolderID   string
	ChildFolderCount int
	TotalItemCount   int
	UnreadItemCount  int
}

// ---------------------------------------------------------------------------
// Mail folder methods
// ---------------------------------------------------------------------------

// ListMailFolders returns the top-level mail folders of the given shared mailbox.
func (g *GraphClient) ListMailFolders(mailbox string) ([]MailFolder, error) {
	endpoint := fmt.Sprintf(
		"https://graph.microsoft.com/v1.0/users/%s/mailFolders?includeHiddenFolders=true&$top=100",
		url.PathEscape(mailbox),
	)

	body, err := g.graphGet(endpoint)
	if err != nil {
		return nil, fmt.Errorf("ListMailFolders: %w", err)
	}

	var resp struct {
		Value []struct {
			ID               string `json:"id"`
			DisplayName      string `json:"displayName"`
			ParentFolderID   string `json:"parentFolderId"`
			ChildFolderCount int    `json:"childFolderCount"`
			TotalItemCount   int    `json:"totalItemCount"`
			UnreadItemCount  int    `json:"unreadItemCount"`
		} `json:"value"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("ListMailFolders: parse response: %w", err)
	}

	folders := make([]MailFolder, 0, len(resp.Value))
	for _, f := range resp.Value {
		folders = append(folders, MailFolder{
			ID:               f.ID,
			DisplayName:      f.DisplayName,
			ParentFolderID:   f.ParentFolderID,
			ChildFolderCount: f.ChildFolderCount,
			TotalItemCount:   f.TotalItemCount,
			UnreadItemCount:  f.UnreadItemCount,
		})
	}
	return folders, nil
}

// ListSubFolders returns the child folders of folderID in the given shared mailbox.
func (g *GraphClient) ListSubFolders(mailbox, folderID string) ([]MailFolder, error) {
	endpoint := fmt.Sprintf(
		"https://graph.microsoft.com/v1.0/users/%s/mailFolders/%s/childFolders?$top=100",
		url.PathEscape(mailbox),
		url.PathEscape(folderID),
	)

	body, err := g.graphGet(endpoint)
	if err != nil {
		return nil, fmt.Errorf("ListSubFolders: %w", err)
	}

	var resp struct {
		Value []struct {
			ID               string `json:"id"`
			DisplayName      string `json:"displayName"`
			ParentFolderID   string `json:"parentFolderId"`
			ChildFolderCount int    `json:"childFolderCount"`
			TotalItemCount   int    `json:"totalItemCount"`
			UnreadItemCount  int    `json:"unreadItemCount"`
		} `json:"value"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("ListSubFolders: parse response: %w", err)
	}

	folders := make([]MailFolder, 0, len(resp.Value))
	for _, f := range resp.Value {
		folders = append(folders, MailFolder{
			ID:               f.ID,
			DisplayName:      f.DisplayName,
			ParentFolderID:   f.ParentFolderID,
			ChildFolderCount: f.ChildFolderCount,
			TotalItemCount:   f.TotalItemCount,
			UnreadItemCount:  f.UnreadItemCount,
		})
	}
	return folders, nil
}

// GetFolderByName returns the first folder whose DisplayName matches displayName
// (case-insensitive). When parentFolderID is empty it searches the top-level mailbox
// folders; otherwise it searches the children of parentFolderID.
// Returns nil, nil if no match is found.
func (g *GraphClient) GetFolderByName(mailbox, parentFolderID, displayName string) (*MailFolder, error) {
	var folders []MailFolder
	var err error

	if parentFolderID == "" {
		folders, err = g.ListMailFolders(mailbox)
	} else {
		folders, err = g.ListSubFolders(mailbox, parentFolderID)
	}
	if err != nil {
		return nil, fmt.Errorf("GetFolderByName: %w", err)
	}

	target := strings.ToLower(displayName)
	for i := range folders {
		if strings.ToLower(folders[i].DisplayName) == target {
			return &folders[i], nil
		}
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mail message methods
// ---------------------------------------------------------------------------

// ListNewMessages returns messages in folderID, optionally filtered to those received at or
// after sinceRFC3339. Follows @odata.nextLink pagination until all pages are consumed.
func (g *GraphClient) ListNewMessages(mailbox, folderID, sinceRFC3339 string) ([]MailMessage, error) {
	params := url.Values{
		"$top":     {"50"},
		"$orderby": {"receivedDateTime desc"},
	}
	if sinceRFC3339 != "" {
		params.Set("$filter", fmt.Sprintf("receivedDateTime ge %s", sinceRFC3339))
	}

	endpoint := fmt.Sprintf(
		"https://graph.microsoft.com/v1.0/users/%s/mailFolders/%s/messages?%s",
		url.PathEscape(mailbox),
		url.PathEscape(folderID),
		params.Encode(),
	)

	var all []MailMessage

	for endpoint != "" {
		body, err := g.graphGet(endpoint)
		if err != nil {
			return nil, fmt.Errorf("ListNewMessages: %w", err)
		}

		var resp struct {
			NextLink string `json:"@odata.nextLink"`
			Value    []struct {
				ID             string `json:"id"`
				Subject        string `json:"subject"`
				BodyPreview    string `json:"bodyPreview"`
				ConversationID string `json:"conversationId"`
				IsRead         bool   `json:"isRead"`
				ParentFolderID string `json:"parentFolderId"`
				ReceivedAt     string `json:"receivedDateTime"`
				From           *struct {
					EmailAddress struct {
						Name    string `json:"name"`
						Address string `json:"address"`
					} `json:"emailAddress"`
				} `json:"from"`
				Flag *struct {
					FlagStatus string `json:"flagStatus"`
				} `json:"flag"`
			} `json:"value"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("ListNewMessages: parse response: %w", err)
		}

		for _, m := range resp.Value {
			msg := MailMessage{
				ID:             m.ID,
				Subject:        m.Subject,
				BodyPreview:    m.BodyPreview,
				ConversationID: m.ConversationID,
				IsRead:         m.IsRead,
				ParentFolderID: m.ParentFolderID,
			}
			if m.From != nil {
				msg.Sender = m.From.EmailAddress.Address
				msg.SenderName = m.From.EmailAddress.Name
			}
			if m.Flag != nil {
				msg.IsFlagged = m.Flag.FlagStatus == "flagged"
			}
			if m.ReceivedAt != "" {
				if t, err := time.Parse(time.RFC3339, m.ReceivedAt); err == nil {
					msg.ReceivedAt = t
				}
			}
			all = append(all, msg)
		}

		// Follow the next page, or stop if there are no more pages.
		endpoint = resp.NextLink
	}

	return all, nil
}

// MoveMessage moves messageID to destinationFolderID in the given shared mailbox.
func (g *GraphClient) MoveMessage(mailbox, messageID, destinationFolderID string) error {
	endpoint := fmt.Sprintf(
		"https://graph.microsoft.com/v1.0/users/%s/messages/%s/move",
		url.PathEscape(mailbox),
		url.PathEscape(messageID),
	)

	payload, err := json.Marshal(map[string]string{"destinationId": destinationFolderID})
	if err != nil {
		return fmt.Errorf("MoveMessage: marshal body: %w", err)
	}

	if err := g.graphPost(endpoint, payload); err != nil {
		return fmt.Errorf("MoveMessage: %w", err)
	}
	return nil
}

// FlagMessage sets the flag status of messageID to "flagged" in the given shared mailbox.
func (g *GraphClient) FlagMessage(mailbox, messageID string) error {
	endpoint := fmt.Sprintf(
		"https://graph.microsoft.com/v1.0/users/%s/messages/%s",
		url.PathEscape(mailbox),
		url.PathEscape(messageID),
	)

	payload, err := json.Marshal(map[string]interface{}{
		"flag": map[string]string{"flagStatus": "flagged"},
	})
	if err != nil {
		return fmt.Errorf("FlagMessage: marshal body: %w", err)
	}

	if err := g.graphPatch(endpoint, payload); err != nil {
		return fmt.Errorf("FlagMessage: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Internal HTTP helpers for mail methods
// ---------------------------------------------------------------------------

// graphPost makes an authenticated POST to the Graph API with a JSON body.
// A 2xx response is considered success; anything else returns an error with the status and body.
func (g *GraphClient) graphPost(endpoint string, jsonBody []byte) error {
	accessToken, err := g.getAccessToken()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Graph API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("Graph API error (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// graphPatch makes an authenticated PATCH to the Graph API with a JSON body.
// A 2xx response is considered success; anything else returns an error with the status and body.
func (g *GraphClient) graphPatch(endpoint string, jsonBody []byte) error {
	accessToken, err := g.getAccessToken()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PATCH", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Graph API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("Graph API error (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

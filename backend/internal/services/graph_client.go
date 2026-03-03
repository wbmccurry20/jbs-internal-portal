package services

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// GraphClient handles Microsoft Graph API with delegated (user) auth
type GraphClient struct {
	ClientID     string
	ClientSecret string
	TenantID     string
	RedirectURI  string
	SiteHost     string // netorg4205680.sharepoint.com
	SitePath     string // /sites/JBS

	token      *OAuthToken
	oauthState string
	mu         sync.RWMutex
}

// OAuthToken holds Microsoft OAuth2 tokens
type OAuthToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	ExpiresAt    time.Time
}

// SharePointItem represents a file or folder from SharePoint
type SharePointItem struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	WebURL     string    `json:"web_url"`
	IsFolder   bool      `json:"is_folder"`
	MimeType   string    `json:"mime_type,omitempty"`
	ChildCount int       `json:"child_count,omitempty"`
	ModifiedAt time.Time `json:"modified_at"`
	ModifiedBy string    `json:"modified_by,omitempty"`
}

// NewGraphClient creates a Graph API client
func NewGraphClient(clientID, clientSecret, tenantID, redirectURI string) *GraphClient {
	return &GraphClient{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TenantID:     tenantID,
		RedirectURI:  redirectURI,
		SiteHost:     "netorg4205680.sharepoint.com",
		SitePath:     "/sites/JBS",
	}
}

// IsConfigured returns true if Azure credentials are set
func (g *GraphClient) IsConfigured() bool {
	return g.ClientID != "" && g.TenantID != ""
}

// IsConnected returns true if we have a valid token
func (g *GraphClient) IsConnected() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.token != nil && time.Now().Before(g.token.ExpiresAt)
}

// GetAuthURL returns the Microsoft OAuth2 login URL (delegated flow)
func (g *GraphClient) GetAuthURL() string {
	state := generateState()
	g.mu.Lock()
	g.oauthState = state
	g.mu.Unlock()

	params := url.Values{
		"client_id":     {g.ClientID},
		"response_type": {"code"},
		"redirect_uri":  {g.RedirectURI},
		"scope":         {"https://graph.microsoft.com/Sites.Read.All https://graph.microsoft.com/Files.Read.All offline_access"},
		"response_mode": {"query"},
		"state":         {state},
	}

	return fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize?%s",
		g.TenantID, params.Encode())
}

// ValidateState checks the OAuth state parameter
func (g *GraphClient) ValidateState(state string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.oauthState != "" && g.oauthState == state
}

// ExchangeCode exchanges an auth code for tokens
func (g *GraphClient) ExchangeCode(code string) error {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", g.TenantID)

	data := url.Values{
		"client_id":    {g.ClientID},
		"code":         {code},
		"redirect_uri": {g.RedirectURI},
		"grant_type":   {"authorization_code"},
		"scope":        {"https://graph.microsoft.com/Sites.Read.All https://graph.microsoft.com/Files.Read.All offline_access"},
	}
	// Only include client_secret if set (public client apps don't need it)
	if g.ClientSecret != "" {
		data.Set("client_secret", g.ClientSecret)
	}

	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("token exchange failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var token OAuthToken
	if err := json.Unmarshal(body, &token); err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}
	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)

	g.mu.Lock()
	g.token = &token
	g.oauthState = ""
	g.mu.Unlock()

	log.Printf("✅ SharePoint connected (token expires: %s)", token.ExpiresAt.Format(time.Kitchen))
	return nil
}

// refreshToken refreshes an expired access token
func (g *GraphClient) refreshToken() error {
	g.mu.RLock()
	refresh := ""
	if g.token != nil {
		refresh = g.token.RefreshToken
	}
	g.mu.RUnlock()

	if refresh == "" {
		return fmt.Errorf("no refresh token — user must re-authenticate")
	}

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", g.TenantID)
	data := url.Values{
		"client_id":     {g.ClientID},
		"refresh_token": {refresh},
		"grant_type":    {"refresh_token"},
		"scope":         {"https://graph.microsoft.com/Sites.Read.All https://graph.microsoft.com/Files.Read.All offline_access"},
	}
	if g.ClientSecret != "" {
		data.Set("client_secret", g.ClientSecret)
	}

	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("refresh failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("refresh failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var token OAuthToken
	if err := json.Unmarshal(body, &token); err != nil {
		return fmt.Errorf("failed to parse refresh response: %w", err)
	}
	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)

	g.mu.Lock()
	g.token = &token
	g.mu.Unlock()

	log.Printf("✅ SharePoint token refreshed (expires: %s)", token.ExpiresAt.Format(time.Kitchen))
	return nil
}

// getAccessToken returns a valid token, auto-refreshing if needed
func (g *GraphClient) getAccessToken() (string, error) {
	g.mu.RLock()
	token := g.token
	g.mu.RUnlock()

	if token == nil {
		return "", fmt.Errorf("not authenticated — connect SharePoint first")
	}

	if time.Now().Add(5 * time.Minute).After(token.ExpiresAt) {
		if err := g.refreshToken(); err != nil {
			return "", err
		}
		g.mu.RLock()
		token = g.token
		g.mu.RUnlock()
	}

	return token.AccessToken, nil
}

// graphGet makes an authenticated GET to the Graph API
func (g *GraphClient) graphGet(endpoint string) ([]byte, error) {
	accessToken, err := g.getAccessToken()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Graph API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Graph API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// ListFolder lists the children of a folder in the Shared Documents library
// folderPath is relative to Shared Documents root, e.g. "Licensing" or "Licensing/Alabama (AL)"
func (g *GraphClient) ListFolder(folderPath string) ([]SharePointItem, error) {
	// Graph API path: /sites/{host}:{sitePath}:/drive/root:/{path}:/children
	encodedPath := url.PathEscape(folderPath)
	// url.PathEscape encodes spaces as %20 but also encodes slashes, we need slashes preserved
	encodedPath = strings.ReplaceAll(encodedPath, "%2F", "/")

	endpoint := fmt.Sprintf(
		"https://graph.microsoft.com/v1.0/sites/%s:%s:/drive/root:/%s:/children?$orderby=name&$top=200",
		g.SiteHost, g.SitePath, encodedPath,
	)

	body, err := g.graphGet(endpoint)
	if err != nil {
		return nil, err
	}

	var response struct {
		Value []json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	items := make([]SharePointItem, 0, len(response.Value))
	for _, raw := range response.Value {
		var driveItem struct {
			ID                   string `json:"id"`
			Name                 string `json:"name"`
			Size                 int64  `json:"size"`
			WebURL               string `json:"webUrl"`
			LastModifiedDateTime string `json:"lastModifiedDateTime"`
			LastModifiedBy       *struct {
				User *struct {
					DisplayName string `json:"displayName"`
				} `json:"user"`
			} `json:"lastModifiedBy"`
			File *struct {
				MimeType string `json:"mimeType"`
			} `json:"file"`
			Folder *struct {
				ChildCount int `json:"childCount"`
			} `json:"folder"`
		}
		if err := json.Unmarshal(raw, &driveItem); err != nil {
			continue
		}

		item := SharePointItem{
			ID:     driveItem.ID,
			Name:   driveItem.Name,
			Size:   driveItem.Size,
			WebURL: driveItem.WebURL,
		}

		if t, err := time.Parse(time.RFC3339, driveItem.LastModifiedDateTime); err == nil {
			item.ModifiedAt = t
		}
		if driveItem.LastModifiedBy != nil && driveItem.LastModifiedBy.User != nil {
			item.ModifiedBy = driveItem.LastModifiedBy.User.DisplayName
		}
		if driveItem.Folder != nil {
			item.IsFolder = true
			item.ChildCount = driveItem.Folder.ChildCount
		}
		if driveItem.File != nil {
			item.MimeType = driveItem.File.MimeType
		}

		items = append(items, item)
	}

	return items, nil
}

// Disconnect clears the stored token
func (g *GraphClient) Disconnect() {
	g.mu.Lock()
	g.token = nil
	g.mu.Unlock()
	log.Println("SharePoint disconnected")
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

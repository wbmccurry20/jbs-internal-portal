package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wbmccurry20/jbs-internal-portal/internal/services"
)

// GetGraphClient returns the package-level GraphClient initialised by InitGraphClient.
// Used by main.go to wire the shared client into EmailAuthHandler.
func GetGraphClient() *services.GraphClient {
	return graphClient
}

// EmailAuthHandler handles OAuth2 for the shared mailbox (Mail.ReadWrite.Shared scope).
type EmailAuthHandler struct {
	GraphClient *services.GraphClient
	DB          *sql.DB
}

// GetAuthURL returns the Microsoft OAuth2 login URL for mail access.
// GET /api/email/auth
func (h *EmailAuthHandler) GetAuthURL(c *gin.Context) {
	tenantID := os.Getenv("AZURE_TENANT_ID")
	clientID := os.Getenv("AZURE_CLIENT_ID")
	redirectURI := os.Getenv("MAIL_REDIRECT_URI")

	if tenantID == "" || clientID == "" || redirectURI == "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Email auth not configured — set AZURE_TENANT_ID, AZURE_CLIENT_ID, MAIL_REDIRECT_URI",
		})
		return
	}

	state := generateEmailState()

	params := url.Values{
		"client_id":     {clientID},
		"response_type": {"code"},
		"redirect_uri":  {redirectURI},
		"scope":         {"https://graph.microsoft.com/Mail.ReadWrite.Shared offline_access"},
		"response_mode": {"query"},
		"state":         {state},
	}

	authURL := fmt.Sprintf(
		"https://login.microsoftonline.com/%s/oauth2/v2.0/authorize?%s",
		tenantID, params.Encode(),
	)

	// Store state in DB so it can be validated in HandleCallback even after a restart
	if err := upsertEmailAuthState(h.DB, state); err != nil {
		log.Printf("⚠️  Failed to persist email auth state: %v", err)
		// Non-fatal: fall through and return the URL — validation may fail on callback
	}

	c.JSON(http.StatusOK, gin.H{"auth_url": authURL})
}

// HandleCallback exchanges the Microsoft auth code for tokens and persists them.
// GET /api/email/callback
func (h *EmailAuthHandler) HandleCallback(c *gin.Context) {
	frontendURL := getFrontendURL()

	if errMsg := c.Query("error"); errMsg != "" {
		log.Printf("❌ Email OAuth error: %s — %s", errMsg, c.Query("error_description"))
		c.Redirect(http.StatusFound, frontendURL+"/email-automation?connected=error")
		return
	}

	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No authorization code"})
		return
	}

	if !validateAndConsumeEmailState(h.DB, state) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state parameter"})
		return
	}

	tenantID := os.Getenv("AZURE_TENANT_ID")
	clientID := os.Getenv("AZURE_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	redirectURI := os.Getenv("MAIL_REDIRECT_URI")

	tokenURL := fmt.Sprintf(
		"https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID,
	)

	data := url.Values{
		"client_id":    {clientID},
		"code":         {code},
		"redirect_uri": {redirectURI},
		"grant_type":   {"authorization_code"},
		"scope":        {"https://graph.microsoft.com/Mail.ReadWrite.Shared offline_access"},
	}
	if clientSecret != "" {
		data.Set("client_secret", clientSecret)
	}

	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		log.Printf("❌ Mail token exchange failed: %v", err)
		c.Redirect(http.StatusFound, frontendURL+"/email-automation?connected=error")
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		log.Printf("❌ Mail token exchange HTTP %d: %s", resp.StatusCode, string(body))
		c.Redirect(http.StatusFound, frontendURL+"/email-automation?connected=error")
		return
	}

	var raw struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		log.Printf("❌ Failed to parse mail token response: %v", err)
		c.Redirect(http.StatusFound, frontendURL+"/email-automation?connected=error")
		return
	}

	token := &services.OAuthToken{
		AccessToken:  raw.AccessToken,
		RefreshToken: raw.RefreshToken,
		ExpiresIn:    raw.ExpiresIn,
		ExpiresAt:    time.Now().Add(time.Duration(raw.ExpiresIn) * time.Second),
	}

	if err := h.GraphClient.SaveMailTokenToDB(token); err != nil {
		log.Printf("❌ Failed to persist mail token: %v", err)
		c.Redirect(http.StatusFound, frontendURL+"/email-automation?connected=error")
		return
	}

	log.Printf("✅ Mail OAuth token saved (expires: %s)", token.ExpiresAt.Format(time.Kitchen))
	c.Redirect(http.StatusFound, frontendURL+"/email-automation?connected=true")
}

// ---------------------------------------------------------------------------
// Internal helpers — email auth state stored in email_automation_config
// ---------------------------------------------------------------------------

const emailAuthStateKey = "oauth_pending_state"

func generateEmailState() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// upsertEmailAuthState persists the CSRF state token in the config table.
func upsertEmailAuthState(db *sql.DB, state string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	_, err := db.Exec(`
		INSERT INTO email_automation_config (key, value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (key) DO UPDATE
		  SET value = EXCLUDED.value, updated_at = NOW()`,
		emailAuthStateKey, state,
	)
	return err
}

// validateAndConsumeEmailState checks the state param and clears it from the DB (one-time use).
func validateAndConsumeEmailState(db *sql.DB, state string) bool {
	if db == nil || state == "" {
		return false
	}

	var stored string
	err := db.QueryRow(
		"SELECT value FROM email_automation_config WHERE key = $1",
		emailAuthStateKey,
	).Scan(&stored)
	if err != nil {
		return false
	}

	if stored != state || stored == "" {
		return false
	}

	// Consume: clear the pending state
	_, _ = db.Exec(`
		INSERT INTO email_automation_config (key, value, updated_at)
		VALUES ($1, '', NOW())
		ON CONFLICT (key) DO UPDATE
		  SET value = '', updated_at = NOW()`,
		emailAuthStateKey,
	)

	return true
}

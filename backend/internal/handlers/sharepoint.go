package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/wbmccurry20/jbs-internal-portal/internal/services"
)

// Graph client singleton — initialized on startup
var graphClient *services.GraphClient

// InitGraphClient sets up the Graph client from env vars
func InitGraphClient() {
	graphClient = services.NewGraphClient(
		os.Getenv("AZURE_CLIENT_ID"),
		os.Getenv("AZURE_CLIENT_SECRET"),
		os.Getenv("AZURE_TENANT_ID"),
		os.Getenv("SHAREPOINT_REDIRECT_URI"),
	)

	if graphClient.IsConfigured() {
		log.Println("✅ SharePoint integration configured")
	} else {
		log.Println("⚠️  SharePoint not configured (set AZURE_CLIENT_ID, AZURE_TENANT_ID)")
	}
}

// GetSharePointStatus returns connection status
func GetSharePointStatus(c *gin.Context) {
	if graphClient == nil || !graphClient.IsConfigured() {
		c.JSON(http.StatusOK, gin.H{
			"configured": false,
			"connected":  false,
			"message":    "Set AZURE_CLIENT_ID, AZURE_TENANT_ID, and SHAREPOINT_REDIRECT_URI in environment.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"configured": true,
		"connected":  graphClient.IsConnected(),
	})
}

// InitSharePointAuth returns the Microsoft login URL
func InitSharePointAuth(c *gin.Context) {
	if graphClient == nil || !graphClient.IsConfigured() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SharePoint not configured"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"auth_url": graphClient.GetAuthURL(),
	})
}

// HandleSharePointCallback handles the OAuth redirect from Microsoft
func HandleSharePointCallback(c *gin.Context) {
	if errMsg := c.Query("error"); errMsg != "" {
		log.Printf("❌ SharePoint OAuth error: %s — %s", errMsg, c.Query("error_description"))
		frontendURL := getFrontendURL()
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/licensing?sp=error", frontendURL))
		return
	}

	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No authorization code"})
		return
	}

	if !graphClient.ValidateState(state) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state parameter"})
		return
	}

	if err := graphClient.ExchangeCode(code); err != nil {
		log.Printf("❌ Token exchange failed: %v", err)
		frontendURL := getFrontendURL()
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/licensing?sp=error", frontendURL))
		return
	}

	frontendURL := getFrontendURL()
	c.Redirect(http.StatusFound, fmt.Sprintf("%s/licensing?sp=connected", frontendURL))
}

// BrowseSharePoint lists files/folders at a given path in the Licensing folder
func BrowseSharePoint(c *gin.Context) {
	if graphClient == nil || !graphClient.IsConnected() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "SharePoint not connected"})
		return
	}

	// path param is relative to Licensing folder, e.g. "" or "Alabama (AL)" or "Alabama (AL)/Birmingham"
	subPath := c.Query("path")
	folderPath := "Licensing"
	if subPath != "" {
		folderPath = "Licensing/" + subPath
	}

	items, err := graphClient.ListFolder(folderPath)
	if err != nil {
		log.Printf("❌ SharePoint browse error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to browse SharePoint: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"path":  subPath,
		"items": items,
	})
}

// DisconnectSharePoint clears the stored token
func DisconnectSharePoint(c *gin.Context) {
	if graphClient != nil {
		graphClient.Disconnect()
	}
	c.JSON(http.StatusOK, gin.H{"message": "Disconnected"})
}

func getFrontendURL() string {
	url := os.Getenv("FRONTEND_URL")
	if url == "" {
		return "http://localhost:4321"
	}
	return url
}

package gcal

import (
	"context"
	"fmt"

	"github.com/omriShneor/project_alfred/internal/database"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// Client wraps the Google Calendar API client
type Client struct {
	userID          int64 // User who owns this client
	service         *calendar.Service
	config          *oauth2.Config
	credentialsFile string
	token           *oauth2.Token
	db              *database.DB // For token storage
}

// tryInitService attempts to initialize the service, refreshing the token if needed
func (c *Client) tryInitService() error {
	if c.token == nil {
		return fmt.Errorf("no token available")
	}

	ctx := context.Background()

	// If token is expired but we have a refresh token, try to refresh
	if !c.token.Valid() && c.token.RefreshToken != "" {
		tokenSource := c.config.TokenSource(ctx, c.token)
		newToken, err := tokenSource.Token()
		if err != nil {
			return fmt.Errorf("failed to refresh token: %w", err)
		}
		c.token = newToken
		// Save refreshed token (database for multi-user, file for single-user)
		if err := c.saveToken(newToken); err != nil {
			fmt.Printf("Warning: could not save refreshed token: %v\n", err)
		}
	}

	return c.initService(ctx)
}

// saveToken saves the token to database
func (c *Client) saveToken(token *oauth2.Token) error {
	if c.db == nil || c.userID == 0 {
		return fmt.Errorf("database and userID are required")
	}
	return c.db.UpdateGoogleToken(c.userID, token)
}

// IsAuthenticated returns true if the client is authenticated
func (c *Client) IsAuthenticated() bool {
	return c.service != nil
}

// GetAuthURL returns the OAuth authorization URL
func (c *Client) GetAuthURL() string {
	return c.config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// GetAuthURLWithRedirect returns the OAuth authorization URL with a custom redirect URI
// This is used for mobile apps that use deep links for OAuth callbacks
func (c *Client) GetAuthURLWithRedirect(redirectURI string) string {
	// Create a copy of the config with the custom redirect URI
	configCopy := *c.config
	configCopy.RedirectURL = redirectURI
	return configCopy.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// initService initializes the Calendar service with the current token
func (c *Client) initService(ctx context.Context) error {
	if c.token == nil {
		return fmt.Errorf("no token available")
	}

	httpClient := c.config.Client(ctx, c.token)
	service, err := calendar.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("failed to create calendar service: %w", err)
	}

	c.service = service
	return nil
}

// ExchangeCode exchanges an authorization code for a token and saves it
func (c *Client) ExchangeCode(ctx context.Context, code string) error {
	token, err := c.config.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("failed to exchange code for token: %w", err)
	}

	c.token = token
	if err := c.saveInitialToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	return c.initService(ctx)
}

// ExchangeCodeWithRedirect exchanges an authorization code for a token using a custom redirect URI
// This is used for mobile apps that use deep links for OAuth callbacks
func (c *Client) ExchangeCodeWithRedirect(ctx context.Context, code, redirectURI string) error {
	// Create a copy of the config with the custom redirect URI
	configCopy := *c.config
	configCopy.RedirectURL = redirectURI

	token, err := configCopy.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("failed to exchange code for token: %w", err)
	}

	c.token = token
	if err := c.saveInitialToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	return c.initService(ctx)
}

// saveInitialToken saves a newly exchanged token (includes full token data)
func (c *Client) saveInitialToken(token *oauth2.Token) error {
	if c.db == nil || c.userID == 0 {
		return fmt.Errorf("database and userID are required")
	}
	// Note: email will be set later when we have user info
	return c.db.SaveGoogleToken(c.userID, token, "")
}

// Disconnect removes the stored token and clears the service
func (c *Client) Disconnect() error {
	if c.db != nil && c.userID != 0 {
		if err := c.db.DeleteGoogleToken(c.userID); err != nil {
			return fmt.Errorf("failed to delete token from database: %w", err)
		}
	}

	// Clear internal state
	c.token = nil
	c.service = nil

	return nil
}

// NewClientForUser creates a Google Calendar client for a specific user (multi-user mode)
// Token storage is handled via database instead of file
func NewClientForUser(userID int64, credentialsFile string, db *database.DB) (*Client, error) {
	fmt.Printf("[GCal Client] Creating client for user %d\n", userID)
	config, err := loadOAuthConfig(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load OAuth config: %w", err)
	}

	client := &Client{
		userID:          userID,
		config:          config,
		credentialsFile: credentialsFile,
		db:              db,
	}

	// Try to load existing token from database
	token, err := db.GetGoogleToken(userID)
	fmt.Printf("[GCal Client] Token from DB: exists=%v, err=%v\n", token != nil, err)
	if err == nil && token != nil {
		client.token = token
		fmt.Printf("[GCal Client] Token details: valid=%v, hasRefresh=%v, expiry=%v\n",
			token.Valid(), token.RefreshToken != "", token.Expiry)
		// Try to initialize the service with the existing token
		if err := client.tryInitService(); err != nil {
			fmt.Printf("[GCal Client] ERROR: Could not initialize calendar service for user %d: %v\n", userID, err)
		} else {
			fmt.Printf("[GCal Client] Service initialized successfully for user %d\n", userID)
		}
	} else {
		fmt.Printf("[GCal Client] No token found for user %d\n", userID)
	}

	return client, nil
}


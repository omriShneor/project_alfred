package gcal

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// Client wraps the Google Calendar API client
type Client struct {
	service         *calendar.Service
	config          *oauth2.Config
	credentialsFile string
	tokenFile       string
	token           *oauth2.Token
}

// NewClient creates a new Google Calendar client
func NewClient(credentialsFile, tokenFile string) (*Client, error) {
	config, err := loadOAuthConfig(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load OAuth config: %w", err)
	}

	client := &Client{
		config:          config,
		credentialsFile: credentialsFile,
		tokenFile:       tokenFile,
	}

	// Try to load existing token and initialize service
	token, err := loadToken(tokenFile)
	if err == nil {
		client.token = token
		// Try to initialize the service with the existing token
		if err := client.tryInitService(); err != nil {
			// Token might be expired, but that's OK - user will need to re-auth
			fmt.Printf("Note: Could not initialize calendar service with existing token: %v\n", err)
		}
	}

	return client, nil
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
		if err := saveToken(c.tokenFile, newToken); err != nil {
			fmt.Printf("Warning: could not save refreshed token: %v\n", err)
		}
	}

	return c.initService(ctx)
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
	if err := saveToken(c.tokenFile, token); err != nil {
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
	if err := saveToken(c.tokenFile, token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	return c.initService(ctx)
}

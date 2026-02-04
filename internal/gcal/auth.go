package gcal

import (
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

const (
	oauthCallbackPort = 8089
	callbackPath      = "/oauth/callback"
)

// getOAuthCallbackURL returns the OAuth callback URL, using ALFRED_BASE_URL if set
func getOAuthCallbackURL() string {
	if baseURL := os.Getenv("ALFRED_BASE_URL"); baseURL != "" {
		return baseURL + callbackPath
	}
	return fmt.Sprintf("http://localhost:%d%s", oauthCallbackPort, callbackPath)
}

// OAuthScopes contains only Calendar scopes
// Gmail scopes should be requested separately via incremental auth
var OAuthScopes = []string{
	calendar.CalendarScope,
}

// loadOAuthConfig loads OAuth2 configuration from credentials file or environment variable
func loadOAuthConfig(credentialsFile string) (*oauth2.Config, error) {
	// Try environment variable first (useful for container deployments)
	if credJSON := os.Getenv("GOOGLE_CREDENTIALS_JSON"); credJSON != "" {
		config, err := google.ConfigFromJSON([]byte(credJSON), OAuthScopes...)
		if err == nil {
			config.RedirectURL = getOAuthCallbackURL()
			return config, nil
		}
	}

	// Try specified file
	if credentialsFile != "" {
		if config, err := loadConfigFromFile(credentialsFile); err == nil {
			return config, nil
		}
	}

	// Try default credentials.json in current directory
	if config, err := loadConfigFromFile("./credentials.json"); err == nil {
		return config, nil
	}

	return nil, fmt.Errorf("no credentials file found - please provide credentials.json or set GOOGLE_CREDENTIALS_JSON env var")
}

// GetOAuthConfig returns the OAuth config for use by other packages (e.g., Gmail)
func (c *Client) GetOAuthConfig() *oauth2.Config {
	return c.config
}

// GetToken returns the current OAuth token for use by other packages (e.g., Gmail)
func (c *Client) GetToken() *oauth2.Token {
	return c.token
}

// loadConfigFromFile attempts to load OAuth config from a file
func loadConfigFromFile(path string) (*oauth2.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config, err := google.ConfigFromJSON(data, OAuthScopes...)
	if err != nil {
		return nil, err
	}

	config.RedirectURL = getOAuthCallbackURL()
	return config, nil
}

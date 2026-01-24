package gcal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

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

// loadOAuthConfig loads OAuth2 configuration from credentials file or environment variable
func loadOAuthConfig(credentialsFile string) (*oauth2.Config, error) {
	// Try environment variable first (useful for container deployments)
	if credJSON := os.Getenv("GOOGLE_CREDENTIALS_JSON"); credJSON != "" {
		config, err := google.ConfigFromJSON([]byte(credJSON), calendar.CalendarScope)
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

// loadConfigFromFile attempts to load OAuth config from a file
func loadConfigFromFile(path string) (*oauth2.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config, err := google.ConfigFromJSON(data, calendar.CalendarScope)
	if err != nil {
		return nil, err
	}

	config.RedirectURL = getOAuthCallbackURL()
	return config, nil
}

// loadToken loads an OAuth token from a file
func loadToken(path string) (*oauth2.Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}

	return &token, nil
}

// saveToken saves an OAuth token to a file
func saveToken(path string, token *oauth2.Token) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// StartCallbackServer starts the OAuth callback server and returns immediately.
// It returns a channel that will receive the authorization code when received.
// The caller should call ExchangeCode with the received code.
// redirectURL is where to redirect after successful authorization.
func (c *Client) StartCallbackServer(ctx context.Context, redirectURL string) (<-chan string, <-chan error) {
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", oauthCallbackPort),
		Handler: mux,
	}

	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no authorization code received")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Authorization failed: no code received"))
			return
		}

		// Send code and shutdown server before redirect
		codeChan <- code
		go srv.Shutdown(context.Background())

		// Redirect back to onboarding page
		http.Redirect(w, r, redirectURL, http.StatusFound)
	})

	// Start the callback server in a goroutine
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	// Auto-shutdown after timeout (5 minutes)
	go func() {
		select {
		case <-ctx.Done():
			srv.Shutdown(context.Background())
		case <-time.After(5 * time.Minute):
			srv.Shutdown(context.Background())
		}
	}()

	return codeChan, errChan
}

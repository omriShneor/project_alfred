package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/omriShneor/project_alfred/internal/auth"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// AuthConfig holds configuration for authentication
type AuthConfig struct {
	CredentialsFile string
	CredentialsJSON string
	RedirectURL     string // e.g., "alfred://oauth/callback" for mobile deep link
}

// initAuth initializes the authentication service
// This should be called during server setup
func (s *Server) initAuth(cfg AuthConfig) error {
	// Load OAuth credentials
	var credJSON []byte
	var err error

	if cfg.CredentialsJSON != "" {
		credJSON = []byte(cfg.CredentialsJSON)
	} else if cfg.CredentialsFile != "" {
		credJSON, err = os.ReadFile(cfg.CredentialsFile)
		if err != nil {
			return fmt.Errorf("failed to read credentials file: %w", err)
		}
	} else {
		// Try default locations
		if envJSON := os.Getenv("GOOGLE_CREDENTIALS_JSON"); envJSON != "" {
			credJSON = []byte(envJSON)
		} else {
			credJSON, err = os.ReadFile("./credentials.json")
			if err != nil {
				return fmt.Errorf("no credentials found: %w", err)
			}
		}
	}

	// Parse and create OAuth config
	oauthConfig, err := google.ConfigFromJSON(credJSON, auth.OAuthScopes...)
	if err != nil {
		return fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Set redirect URL for auth flow
	if cfg.RedirectURL != "" {
		oauthConfig.RedirectURL = cfg.RedirectURL
	} else if baseURL := os.Getenv("ALFRED_BASE_URL"); baseURL != "" {
		oauthConfig.RedirectURL = baseURL + "/oauth/callback"
	} else {
		// Default for mobile deep link
		oauthConfig.RedirectURL = "alfred://oauth/callback"
	}

	// Create auth service
	authService, err := auth.NewService(s.db.DB, oauthConfig)
	if err != nil {
		return fmt.Errorf("failed to create auth service: %w", err)
	}

	s.authService = authService
	s.authMiddleware = auth.NewMiddleware(authService)

	return nil
}

// handleAuthGoogle initiates Google OAuth login
// GET /api/auth/google?redirect_uri=...
func (s *Server) handleAuthGoogle(w http.ResponseWriter, r *http.Request) {
	if s.authService == nil {
		respondError(w, http.StatusServiceUnavailable, "authentication not configured")
		return
	}

	// Get optional redirect URI override from query params
	redirectURI := r.URL.Query().Get("redirect_uri")
	if redirectURI != "" {
		// For mobile apps, allow specifying a deep link redirect
		// Create a modified OAuth config with the custom redirect
		config := s.authService.GetOAuthConfig()
		modifiedConfig := &oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			Endpoint:     config.Endpoint,
			RedirectURL:  redirectURI,
			Scopes:       config.Scopes,
		}
		authURL := modifiedConfig.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
		respondJSON(w, http.StatusOK, map[string]string{"auth_url": authURL})
		return
	}

	// Use default redirect URL
	authURL := s.authService.GetAuthURL("state")
	respondJSON(w, http.StatusOK, map[string]string{"auth_url": authURL})
}

// handleAuthGoogleCallback handles the OAuth callback and creates a session
// POST /api/auth/google/callback
// Body: { "code": "...", "redirect_uri": "..." }
func (s *Server) handleAuthGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if s.authService == nil {
		respondError(w, http.StatusServiceUnavailable, "authentication not configured")
		return
	}

	var req struct {
		Code        string `json:"code"`
		RedirectURI string `json:"redirect_uri"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Code == "" {
		respondError(w, http.StatusBadRequest, "missing authorization code")
		return
	}

	// Get device info from headers
	deviceInfo := r.Header.Get("X-Device-Info")
	if deviceInfo == "" {
		deviceInfo = r.Header.Get("User-Agent")
	}

	// If a custom redirect URI was used, we need to create a temporary config
	var user *auth.User
	var sessionToken string
	var err error

	// Note: For custom redirect URIs, the mobile app should use the same redirect_uri
	// that was used to generate the auth URL. The auth service will handle the exchange.
	user, sessionToken, err = s.authService.ExchangeCodeAndLogin(r.Context(), req.Code, deviceInfo)

	if err != nil {
		respondError(w, http.StatusBadRequest, "authentication failed: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"session_token": sessionToken,
		"user": map[string]interface{}{
			"id":         user.ID,
			"email":      user.Email,
			"name":       user.Name,
			"avatar_url": user.AvatarURL,
		},
	})
}

// handleAuthLogout invalidates the current session
// POST /api/auth/logout
func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if s.authService == nil {
		respondError(w, http.StatusServiceUnavailable, "authentication not configured")
		return
	}

	// Get token from Authorization header
	token := extractBearerToken(r)
	if token == "" {
		respondError(w, http.StatusBadRequest, "missing authorization token")
		return
	}

	if err := s.authService.Logout(token); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to logout")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

// handleAuthMe returns the current authenticated user
// GET /api/auth/me
func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":         user.ID,
		"email":      user.Email,
		"name":       user.Name,
		"avatar_url": user.AvatarURL,
	})
}

// extractBearerToken extracts the token from the Authorization header
func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	const prefix = "Bearer "
	if len(authHeader) > len(prefix) && authHeader[:len(prefix)] == prefix {
		return authHeader[len(prefix):]
	}
	return ""
}

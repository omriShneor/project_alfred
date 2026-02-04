package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

	// Parse and create OAuth config with profile scopes only (for login)
	// Gmail and Calendar scopes are requested separately via incremental auth
	oauthConfig, err := google.ConfigFromJSON(credJSON, auth.ProfileScopes...)
	if err != nil {
		return fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Set redirect URL for auth flow
	if cfg.RedirectURL != "" {
		oauthConfig.RedirectURL = cfg.RedirectURL
	} else if baseURL := os.Getenv("ALFRED_BASE_URL"); baseURL != "" {
		oauthConfig.RedirectURL = baseURL + "/api/auth/callback"
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

// handleAuthGoogleLogin initiates Google OAuth login with profile scopes
// POST /api/auth/google/login
// Body: { "redirect_uri": "..." } (optional)
func (s *Server) handleAuthGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if s.authService == nil {
		respondError(w, http.StatusServiceUnavailable, "authentication not configured")
		return
	}

	var req struct {
		RedirectURI string `json:"redirect_uri"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req) // Optional body

	fmt.Printf("[Login] Generating OAuth URL for profile scopes\n")
	fmt.Printf("[Login] Custom redirect URI: %s\n", req.RedirectURI)

	// Create OAuth config with profile scopes only
	config := s.authService.GetOAuthConfig()

	redirectURI := req.RedirectURI
	if redirectURI == "" {
		redirectURI = config.RedirectURL
	}

	modifiedConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Endpoint:     config.Endpoint,
		RedirectURL:  redirectURI,
		Scopes:       auth.ProfileScopes, // Only profile scopes for login
	}

	authURL := modifiedConfig.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Printf("[Login] Generated auth URL: %s\n", authURL)

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

	// Pass the redirect URI so the token exchange uses the same URI as the auth URL
	user, sessionToken, err = s.authService.ExchangeCodeAndLogin(r.Context(), req.Code, deviceInfo, req.RedirectURI)

	if err != nil {
		respondError(w, http.StatusBadRequest, "authentication failed: "+err.Error())
		return
	}

	// Check if returning user has sources configured - start services immediately
	if s.userServiceManager != nil {
		hasSources, _ := s.db.UserHasAnySources(user.ID)
		if hasSources {
			go s.userServiceManager.StartServicesForUser(user.ID)
		}
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

// handleAuthOAuthCallback handles the OAuth callback from Google (browser redirect)
// Google redirects here with ?code=..., then we redirect to the mobile app's deep link
// This allows using a standard http(s) URL as the Google OAuth redirect URI
// GET /api/auth/callback?code=...
func (s *Server) handleAuthOAuthCallback(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[OAuth Callback] Received callback from Google\n")
	fmt.Printf("[OAuth Callback] Query params: %v\n", r.URL.Query())

	code := r.URL.Query().Get("code")
	errorParam := r.URL.Query().Get("error")

	// Deep link URL for the mobile app
	deepLinkBase := "alfred://oauth/callback"

	// Handle OAuth errors
	if errorParam != "" {
		errorDesc := r.URL.Query().Get("error_description")
		fmt.Printf("[OAuth Callback] ERROR from Google: %s - %s\n", errorParam, errorDesc)
		redirectURL := fmt.Sprintf("%s?error=%s&error_description=%s",
			deepLinkBase, url.QueryEscape(errorParam), url.QueryEscape(errorDesc))
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}

	// Handle missing code
	if code == "" {
		fmt.Printf("[OAuth Callback] ERROR: No code in callback\n")
		http.Redirect(w, r, deepLinkBase+"?error=no_code", http.StatusFound)
		return
	}

	// Redirect to mobile app with the auth code
	redirectURL := fmt.Sprintf("%s?code=%s", deepLinkBase, url.QueryEscape(code))
	fmt.Printf("[OAuth Callback] Redirecting to deep link: %s\n", redirectURL)

	// Simple HTTP redirect - no HTML page needed
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// handleRequestAdditionalScopes initiates incremental OAuth authorization
// POST /api/auth/google/add-scopes
// Body: { "scopes": ["gmail"] } or { "scopes": ["calendar"] }, "redirect_uri": "..." }
func (s *Server) handleRequestAdditionalScopes(w http.ResponseWriter, r *http.Request) {
	if s.authService == nil {
		respondError(w, http.StatusServiceUnavailable, "authentication not configured")
		return
	}

	var req struct {
		Scopes      []string `json:"scopes"`       // "gmail" or "calendar"
		RedirectURI string   `json:"redirect_uri"` // Optional custom redirect
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Scopes) == 0 {
		respondError(w, http.StatusBadRequest, "scopes required")
		return
	}

	fmt.Printf("[Incremental Auth] Received request for scopes: %v\n", req.Scopes)
	fmt.Printf("[Incremental Auth] Custom redirect URI: %s\n", req.RedirectURI)

	// Map scope names to actual OAuth scopes
	var requestedScopes []string
	for _, scope := range req.Scopes {
		switch scope {
		case "gmail":
			requestedScopes = append(requestedScopes, auth.GmailScopes...)
		case "calendar":
			requestedScopes = append(requestedScopes, auth.CalendarScopes...)
		default:
			respondError(w, http.StatusBadRequest, "invalid scope: "+scope)
			return
		}
	}

	// Use custom redirect URI if provided, otherwise use default
	redirectURI := req.RedirectURI
	if redirectURI == "" {
		redirectURI = s.authService.GetOAuthConfig().RedirectURL
	}

	fmt.Printf("[Incremental Auth] Using redirect URI: %s\n", redirectURI)
	fmt.Printf("[Incremental Auth] Requested scopes: %v\n", requestedScopes)

	// Create OAuth config with custom redirect and incremental scopes
	config := s.authService.GetOAuthConfig()
	modifiedConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Endpoint:     config.Endpoint,
		RedirectURL:  redirectURI,
		Scopes:       requestedScopes,
	}

	// Generate incremental auth URL with include_granted_scopes=true
	authURL := modifiedConfig.AuthCodeURL("state",
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
		oauth2.SetAuthURLParam("include_granted_scopes", "true"),
	)

	fmt.Printf("[Incremental Auth] Generated auth URL: %s\n", authURL)

	respondJSON(w, http.StatusOK, map[string]string{"auth_url": authURL})
}

// handleAddScopesCallback handles the OAuth callback for incremental authorization
// POST /api/auth/google/add-scopes/callback
// Body: { "code": "...", "redirect_uri": "...", "scopes": ["gmail"] }
func (s *Server) handleAddScopesCallback(w http.ResponseWriter, r *http.Request) {
	if s.authService == nil {
		respondError(w, http.StatusServiceUnavailable, "authentication not configured")
		return
	}

	userID := getUserID(r)
	if userID == 0 {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req struct {
		Code        string   `json:"code"`
		RedirectURI string   `json:"redirect_uri"`
		Scopes      []string `json:"scopes"` // "gmail" or "calendar"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	fmt.Printf("[Add Scopes Callback] User %d exchanging code\n", userID)
	fmt.Printf("[Add Scopes Callback] Scopes: %v\n", req.Scopes)
	if len(req.Code) > 10 {
		fmt.Printf("[Add Scopes Callback] Code (first 10 chars): %s...\n", req.Code[:10])
	}

	if req.Code == "" {
		respondError(w, http.StatusBadRequest, "missing authorization code")
		return
	}

	// Map scope names to actual OAuth scopes
	var newScopes []string
	for _, scope := range req.Scopes {
		switch scope {
		case "gmail":
			newScopes = append(newScopes, auth.GmailScopes...)
		case "calendar":
			newScopes = append(newScopes, auth.CalendarScopes...)
		}
	}

	// Exchange code and add scopes
	if err := s.authService.ExchangeCodeAndAddScopes(r.Context(), userID, req.Code, newScopes); err != nil {
		fmt.Printf("[Add Scopes Callback] ERROR: Failed to add scopes for user %d: %v\n", userID, err)
		respondError(w, http.StatusBadRequest, "failed to add scopes: "+err.Error())
		return
	}

	fmt.Printf("[Add Scopes Callback] SUCCESS: Scopes added for user %d\n", userID)
	respondJSON(w, http.StatusOK, map[string]string{"status": "scopes_added"})
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

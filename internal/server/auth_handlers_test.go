package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/omriShneor/project_alfred/internal/auth"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/sse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// createTestServerWithAuth creates a test server with authentication configured
func createTestServerWithAuth(t *testing.T) *Server {
	t.Helper()

	// Set up test encryption key
	os.Setenv("ALFRED_ENCRYPTION_KEY", "test-key-for-auth-handlers")
	t.Cleanup(func() {
		os.Unsetenv("ALFRED_ENCRYPTION_KEY")
	})

	db := database.NewTestDB(t)
	state := sse.NewState()

	// Create OAuth config for testing
	config := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "alfred://oauth/callback",
		Scopes:       auth.ProfileScopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}

	authService, err := auth.NewService(db.DB, config)
	require.NoError(t, err)

	return &Server{
		db:              db,
		onboardingState: state,
		state:           state,
		authService:     authService,
		authMiddleware:  auth.NewMiddleware(authService),
	}
}

func TestHandleAuthGoogleLogin(t *testing.T) {
	s := createTestServerWithAuth(t)

	t.Run("returns valid OAuth URL with profile scopes", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/auth/google/login", nil)
		w := httptest.NewRecorder()

		s.handleAuthGoogleLogin(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		authURL := response["auth_url"]
		assert.NotEmpty(t, authURL)
		assert.Contains(t, authURL, "https://accounts.google.com/o/oauth2/auth")
		assert.Contains(t, authURL, "client_id=test-client-id")
		assert.Contains(t, authURL, "userinfo.email")
		assert.Contains(t, authURL, "userinfo.profile")
		assert.Contains(t, authURL, "access_type=offline")
	})

	t.Run("uses custom redirect URI when provided", func(t *testing.T) {
		body := map[string]string{
			"redirect_uri": "https://example.com/callback",
		}
		bodyJSON, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/api/auth/google/login", bytes.NewReader(bodyJSON))
		w := httptest.NewRecorder()

		s.handleAuthGoogleLogin(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		authURL := response["auth_url"]
		assert.Contains(t, authURL, "redirect_uri=https%3A%2F%2Fexample.com%2Fcallback")
	})

	t.Run("fails when auth service not configured", func(t *testing.T) {
		serverWithoutAuth := createTestServer(t)
		req := httptest.NewRequest("POST", "/api/auth/google/login", nil)
		w := httptest.NewRecorder()

		serverWithoutAuth.handleAuthGoogleLogin(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
}

func TestHandleRequestAdditionalScopes(t *testing.T) {
	s := createTestServerWithAuth(t)

	t.Run("returns OAuth URL for Gmail scope", func(t *testing.T) {
		body := map[string]interface{}{
			"scopes": []string{"gmail"},
		}
		bodyJSON, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/api/auth/google/add-scopes", bytes.NewReader(bodyJSON))
		w := httptest.NewRecorder()

		s.handleRequestAdditionalScopes(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		authURL := response["auth_url"]
		assert.NotEmpty(t, authURL)
		assert.Contains(t, authURL, "gmail.readonly")
		assert.Contains(t, authURL, "include_granted_scopes=true")
	})

	t.Run("returns OAuth URL for Calendar scope", func(t *testing.T) {
		body := map[string]interface{}{
			"scopes": []string{"calendar"},
		}
		bodyJSON, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/api/auth/google/add-scopes", bytes.NewReader(bodyJSON))
		w := httptest.NewRecorder()

		s.handleRequestAdditionalScopes(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		authURL := response["auth_url"]
		assert.Contains(t, authURL, "calendar")
	})

	t.Run("returns OAuth URL for both Gmail and Calendar scopes", func(t *testing.T) {
		body := map[string]interface{}{
			"scopes": []string{"gmail", "calendar"},
		}
		bodyJSON, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/api/auth/google/add-scopes", bytes.NewReader(bodyJSON))
		w := httptest.NewRecorder()

		s.handleRequestAdditionalScopes(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		authURL := response["auth_url"]
		assert.Contains(t, authURL, "gmail.readonly")
		assert.Contains(t, authURL, "calendar")
		assert.Contains(t, authURL, "include_granted_scopes=true")
	})

	t.Run("rejects invalid scope", func(t *testing.T) {
		body := map[string]interface{}{
			"scopes": []string{"invalid-scope"},
		}
		bodyJSON, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/api/auth/google/add-scopes", bytes.NewReader(bodyJSON))
		w := httptest.NewRecorder()

		s.handleRequestAdditionalScopes(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("rejects empty scopes", func(t *testing.T) {
		body := map[string]interface{}{
			"scopes": []string{},
		}
		bodyJSON, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/api/auth/google/add-scopes", bytes.NewReader(bodyJSON))
		w := httptest.NewRecorder()

		s.handleRequestAdditionalScopes(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("rejects invalid request body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/auth/google/add-scopes", bytes.NewReader([]byte("invalid json")))
		w := httptest.NewRecorder()

		s.handleRequestAdditionalScopes(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("uses custom redirect URI when provided", func(t *testing.T) {
		body := map[string]interface{}{
			"scopes":       []string{"gmail"},
			"redirect_uri": "https://example.com/callback",
		}
		bodyJSON, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/api/auth/google/add-scopes", bytes.NewReader(bodyJSON))
		w := httptest.NewRecorder()

		s.handleRequestAdditionalScopes(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		authURL := response["auth_url"]
		assert.Contains(t, authURL, "redirect_uri=https%3A%2F%2Fexample.com%2Fcallback")
	})
}

func TestHandleAuthOAuthCallback(t *testing.T) {
	s := createTestServerWithAuth(t)

	t.Run("redirects to deep link with code", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/callback?code=test-code-123", nil)
		w := httptest.NewRecorder()

		s.handleAuthOAuthCallback(w, req)

		assert.Equal(t, http.StatusFound, w.Code)
		location := w.Header().Get("Location")
		assert.Equal(t, "alfred://oauth/callback?code=test-code-123", location)
	})

	t.Run("redirects to deep link with error", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/callback?error=access_denied&error_description=User+cancelled", nil)
		w := httptest.NewRecorder()

		s.handleAuthOAuthCallback(w, req)

		assert.Equal(t, http.StatusFound, w.Code)
		location := w.Header().Get("Location")
		assert.Contains(t, location, "alfred://oauth/callback?error=access_denied")
		assert.Contains(t, location, "error_description=User")
	})

	t.Run("redirects with error when no code", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/callback", nil)
		w := httptest.NewRecorder()

		s.handleAuthOAuthCallback(w, req)

		assert.Equal(t, http.StatusFound, w.Code)
		location := w.Header().Get("Location")
		assert.Equal(t, "alfred://oauth/callback?error=no_code", location)
	})

	t.Run("URL encodes code parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/callback?code=code+with+spaces%26special", nil)
		w := httptest.NewRecorder()

		s.handleAuthOAuthCallback(w, req)

		assert.Equal(t, http.StatusFound, w.Code)
		location := w.Header().Get("Location")
		// Verify URL encoding is preserved
		assert.Contains(t, location, "code=code")
	})
}

func TestHandleAuthGoogleCallback(t *testing.T) {
	s := createTestServerWithAuth(t)

	t.Run("rejects missing code", func(t *testing.T) {
		body := map[string]string{}
		bodyJSON, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/api/auth/google/callback", bytes.NewReader(bodyJSON))
		w := httptest.NewRecorder()

		s.handleAuthGoogleCallback(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "missing authorization code")
	})

	t.Run("rejects invalid request body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/auth/google/callback", bytes.NewReader([]byte("invalid")))
		w := httptest.NewRecorder()

		s.handleAuthGoogleCallback(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid request body")
	})

	t.Run("fails with invalid authorization code", func(t *testing.T) {
		body := map[string]string{
			"code": "invalid-code",
		}
		bodyJSON, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/api/auth/google/callback", bytes.NewReader(bodyJSON))
		w := httptest.NewRecorder()

		s.handleAuthGoogleCallback(w, req)

		// Will fail because invalid code can't be exchanged with Google
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "authentication failed")
	})
}

func TestHandleAddScopesCallback(t *testing.T) {
	s := createTestServerWithAuth(t)

	t.Run("rejects unauthenticated request", func(t *testing.T) {
		body := map[string]interface{}{
			"code":   "test-code",
			"scopes": []string{"gmail"},
		}
		bodyJSON, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/api/auth/google/add-scopes/callback", bytes.NewReader(bodyJSON))
		w := httptest.NewRecorder()

		s.handleAddScopesCallback(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "authentication required")
	})

	t.Run("rejects missing code", func(t *testing.T) {
		user := database.CreateTestUser(t, s.db)

		body := map[string]interface{}{
			"scopes": []string{"gmail"},
		}
		bodyJSON, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/api/auth/google/add-scopes/callback", bytes.NewReader(bodyJSON))
		req = addUserToContext(req, user.ID)
		w := httptest.NewRecorder()

		s.handleAddScopesCallback(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "missing authorization code")
	})

	t.Run("rejects invalid request body", func(t *testing.T) {
		user := database.CreateTestUser(t, s.db)

		req := httptest.NewRequest("POST", "/api/auth/google/add-scopes/callback", bytes.NewReader([]byte("invalid")))
		req = addUserToContext(req, user.ID)
		w := httptest.NewRecorder()

		s.handleAddScopesCallback(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid request body")
	})

	t.Run("fails with invalid authorization code", func(t *testing.T) {
		user := database.CreateTestUser(t, s.db)

		// First create an initial token with profile scopes
		token := &oauth2.Token{
			AccessToken:  "test-access",
			RefreshToken: "test-refresh",
			TokenType:    "Bearer",
		}
		err := s.db.SaveGoogleToken(user.ID, token, "test@example.com")
		require.NoError(t, err)

		body := map[string]interface{}{
			"code":   "invalid-code",
			"scopes": []string{"gmail"},
		}
		bodyJSON, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/api/auth/google/add-scopes/callback", bytes.NewReader(bodyJSON))
		req = addUserToContext(req, user.ID)
		w := httptest.NewRecorder()

		s.handleAddScopesCallback(w, req)

		// Will fail because invalid code can't be exchanged
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "failed to add scopes")
	})
}

func TestExtractBearerToken(t *testing.T) {
	t.Run("extracts valid bearer token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer test-token-123")

		token := extractBearerToken(req)
		assert.Equal(t, "test-token-123", token)
	})

	t.Run("returns empty for missing header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)

		token := extractBearerToken(req)
		assert.Empty(t, token)
	})

	t.Run("returns empty for invalid format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Invalid token-123")

		token := extractBearerToken(req)
		assert.Empty(t, token)
	})

	t.Run("returns empty for bearer without token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer")

		token := extractBearerToken(req)
		assert.Empty(t, token)
	})
}

// addUserToContext is a helper to simulate an authenticated request
func addUserToContext(req *http.Request, userID int64) *http.Request {
	// Store user in request context (simulating auth middleware)
	user := &auth.User{
		ID:        userID,
		GoogleID:  "test-google-id",
		Email:     "test@example.com",
		Name:      "Test User",
		AvatarURL: "",
	}
	ctx := req.Context()
	ctx = auth.SetUserInContext(ctx, user)
	return req.WithContext(ctx)
}

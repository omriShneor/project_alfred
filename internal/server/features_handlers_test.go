package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/omriShneor/project_alfred/internal/auth"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/sse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestHandleGetAppStatusStartsServicesWhenGmailScopeCached(t *testing.T) {
	os.Setenv("ALFRED_ENCRYPTION_KEY", "test-key-app-status")
	t.Cleanup(func() {
		os.Unsetenv("ALFRED_ENCRYPTION_KEY")
	})

	db := database.NewTestDB(t)
	state := sse.NewState()
	testUser := database.CreateTestUser(t, db)

	// Seed token with Gmail scope
	token := &oauth2.Token{
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(1 * time.Hour),
	}
	err := db.SaveGoogleToken(testUser.ID, token, testUser.Email, []string{"https://www.googleapis.com/auth/gmail.readonly"})
	require.NoError(t, err)

	oauthCfg := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "alfred://oauth/callback",
		Scopes:       auth.ProfileScopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}

	authService, err := auth.NewService(db.DB, oauthCfg)
	require.NoError(t, err)

	s := &Server{
		db:              db,
		onboardingState: state,
		state:           state,
		authService:     authService,
		authMiddleware:  auth.NewMiddleware(authService),
	}

	userServiceManager := NewUserServiceManager(UserServiceManagerConfig{
		DB: db,
	})
	s.SetUserServiceManager(userServiceManager)

	req := httptest.NewRequest("GET", "/api/app/status", nil)
	req = req.WithContext(auth.SetUserInContext(req.Context(), &auth.User{
		ID:    testUser.ID,
		Email: testUser.Email,
		Name:  testUser.Name,
	}))

	w := httptest.NewRecorder()
	s.handleGetAppStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	require.Eventually(t, func() bool {
		return userServiceManager.IsRunningForUser(testUser.ID)
	}, 2*time.Second, 20*time.Millisecond)
}

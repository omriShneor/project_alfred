package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/omriShneor/project_alfred/internal/auth"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestHandleCreateEmailSourceRequiresGmailScope(t *testing.T) {
	s := createTestServerWithAuth(t)
	user := database.CreateTestUser(t, s.db)

	reqBody := map[string]string{
		"type":       "sender",
		"identifier": "user@example.com",
		"name":       "User",
	}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/gmail/sources", bytes.NewReader(body))
	req = withAuthContext(req, user)
	w := httptest.NewRecorder()

	s.handleCreateEmailSource(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleCreateEmailSourceWithGmailScope(t *testing.T) {
	s := createTestServerWithAuth(t)
	user := database.CreateTestUser(t, s.db)

	token := &oauth2.Token{
		AccessToken:  "access",
		RefreshToken: "refresh",
	}
	err := s.db.SaveGoogleToken(user.ID, token, user.Email, []string{auth.GmailScopes[0]})
	require.NoError(t, err)

	reqBody := map[string]string{
		"type":       "sender",
		"identifier": "user@example.com",
		"name":       "User",
	}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/gmail/sources", bytes.NewReader(body))
	req = withAuthContext(req, user)
	w := httptest.NewRecorder()

	s.handleCreateEmailSource(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandleAddCustomSourceRequiresGmailScope(t *testing.T) {
	s := createTestServerWithAuth(t)
	user := database.CreateTestUser(t, s.db)

	reqBody := map[string]string{
		"value": "custom@example.com",
	}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/gmail/sources/custom", bytes.NewReader(body))
	req = withAuthContext(req, user)
	w := httptest.NewRecorder()

	s.handleAddCustomSource(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

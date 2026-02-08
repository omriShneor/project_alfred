package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestHandleUpdateEmailSource_UserScoped(t *testing.T) {
	s := createTestServer(t)
	owner := database.CreateTestUser(t, s.db)
	otherUser := database.CreateTestUser(t, s.db)

	created, err := s.db.CreateEmailSource(owner.ID, database.EmailSourceTypeSender, "owner@example.com", "Owner Source")
	require.NoError(t, err)

	t.Run("cannot update another user's source", func(t *testing.T) {
		body, err := json.Marshal(map[string]any{
			"name":    "Hacked Name",
			"enabled": false,
		})
		require.NoError(t, err)

		req := httptest.NewRequest("PUT", "/api/gmail/sources/"+strconv.FormatInt(created.ID, 10), bytes.NewReader(body))
		req.SetPathValue("id", strconv.FormatInt(created.ID, 10))
		req = withAuthContext(req, otherUser)
		w := httptest.NewRecorder()

		s.handleUpdateEmailSource(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		unchanged, err := s.db.GetEmailSourceByID(created.ID)
		require.NoError(t, err)
		require.NotNil(t, unchanged)
		assert.Equal(t, "Owner Source", unchanged.Name)
		assert.True(t, unchanged.Enabled)
	})

	t.Run("owner can update own source", func(t *testing.T) {
		body, err := json.Marshal(map[string]any{
			"name":    "Updated Name",
			"enabled": false,
		})
		require.NoError(t, err)

		req := httptest.NewRequest("PUT", "/api/gmail/sources/"+strconv.FormatInt(created.ID, 10), bytes.NewReader(body))
		req.SetPathValue("id", strconv.FormatInt(created.ID, 10))
		req = withAuthContext(req, owner)
		w := httptest.NewRecorder()

		s.handleUpdateEmailSource(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		updated, err := s.db.GetEmailSourceByID(created.ID)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, "Updated Name", updated.Name)
		assert.False(t, updated.Enabled)
	})
}

func TestHandleDeleteEmailSource_UserScoped(t *testing.T) {
	s := createTestServer(t)
	owner := database.CreateTestUser(t, s.db)
	otherUser := database.CreateTestUser(t, s.db)

	created, err := s.db.CreateEmailSource(owner.ID, database.EmailSourceTypeSender, "delete-owner@example.com", "Delete Owner")
	require.NoError(t, err)

	t.Run("cannot delete another user's source", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/gmail/sources/"+strconv.FormatInt(created.ID, 10), nil)
		req.SetPathValue("id", strconv.FormatInt(created.ID, 10))
		req = withAuthContext(req, otherUser)
		w := httptest.NewRecorder()

		s.handleDeleteEmailSource(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		stillThere, err := s.db.GetEmailSourceByID(created.ID)
		require.NoError(t, err)
		require.NotNil(t, stillThere)
		assert.Equal(t, owner.ID, stillThere.UserID)
	})

	t.Run("owner can delete own source", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/gmail/sources/"+strconv.FormatInt(created.ID, 10), nil)
		req.SetPathValue("id", strconv.FormatInt(created.ID, 10))
		req = withAuthContext(req, owner)
		w := httptest.NewRecorder()

		s.handleDeleteEmailSource(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		deleted, err := s.db.GetEmailSourceByID(created.ID)
		require.NoError(t, err)
		assert.Nil(t, deleted)
	})
}

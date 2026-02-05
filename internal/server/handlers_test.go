package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/omriShneor/project_alfred/internal/auth"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/source"
	"github.com/omriShneor/project_alfred/internal/sse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestServer creates a minimal server for testing with just the database
func createTestServer(t *testing.T) *Server {
	t.Helper()
	db := database.NewTestDB(t)
	state := sse.NewState()

	return &Server{
		db:              db,
		onboardingState: state,
		state:           state,
	}
}

// withAuthContext adds user context to a request for testing authenticated endpoints
func withAuthContext(r *http.Request, testUser *database.TestUser) *http.Request {
	user := &auth.User{
		ID:       testUser.ID,
		GoogleID: testUser.GoogleID,
		Email:    testUser.Email,
		Name:     testUser.Name,
	}
	ctx := context.WithValue(r.Context(), auth.UserContextKey, user)
	return r.WithContext(ctx)
}

func TestHandleHealthCheck(t *testing.T) {
	t.Run("healthy with database only", func(t *testing.T) {
		s := createTestServer(t)

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		s.handleHealthCheck(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "healthy", response["status"])
		assert.Equal(t, "disconnected", response["whatsapp"])
		assert.Equal(t, "disconnected", response["telegram"])
		assert.Equal(t, "disconnected", response["gcal"])
	})
}

func TestHandleListWhatsappChannels(t *testing.T) {
	s := createTestServer(t)
	user := database.CreateTestUser(t, s.db)

	// Create some channels
	_, err := s.db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"channel1@s.whatsapp.net",
		"Channel 1",
	)
	require.NoError(t, err)

	_, err = s.db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"channel2@s.whatsapp.net",
		"Channel 2",
	)
	require.NoError(t, err)

	t.Run("list all channels", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/whatsapp/channel", nil)
		req = withAuthContext(req, user)
		w := httptest.NewRecorder()

		s.handleListWhatsappChannels(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var channels []database.SourceChannel
		err := json.Unmarshal(w.Body.Bytes(), &channels)
		require.NoError(t, err)
		assert.Len(t, channels, 2)
	})

	t.Run("filter by type", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/whatsapp/channel?type=sender", nil)
		req = withAuthContext(req, user)
		w := httptest.NewRecorder()

		s.handleListWhatsappChannels(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var channels []database.SourceChannel
		err := json.Unmarshal(w.Body.Bytes(), &channels)
		require.NoError(t, err)
		assert.Len(t, channels, 2)
	})
}

func TestHandleListWhatsappChannels_Empty(t *testing.T) {
	s := createTestServer(t)
	user := database.CreateTestUser(t, s.db)

	req := httptest.NewRequest("GET", "/api/whatsapp/channel", nil)
	req = withAuthContext(req, user)
	w := httptest.NewRecorder()

	s.handleListWhatsappChannels(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var channels []database.SourceChannel
	err := json.Unmarshal(w.Body.Bytes(), &channels)
	require.NoError(t, err)
	assert.Len(t, channels, 0)
}

func TestHandleCreateWhatsappChannel(t *testing.T) {
	t.Run("create valid channel", func(t *testing.T) {
		s := createTestServer(t)
		user := database.CreateTestUser(t, s.db)

		body := map[string]string{
			"type":       "sender",
			"identifier": "newchannel@s.whatsapp.net",
			"name":       "New Channel",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/whatsapp/channel", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req = withAuthContext(req, user)
		w := httptest.NewRecorder()

		s.handleCreateWhatsappChannel(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var channel database.SourceChannel
		err := json.Unmarshal(w.Body.Bytes(), &channel)
		require.NoError(t, err)
		assert.Equal(t, "New Channel", channel.Name)
		assert.Equal(t, "newchannel@s.whatsapp.net", channel.Identifier)
	})

	t.Run("invalid type", func(t *testing.T) {
		s := createTestServer(t)
		user := database.CreateTestUser(t, s.db)

		body := map[string]string{
			"type":       "invalid_type",
			"identifier": "test@s.whatsapp.net",
			"name":       "Test",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/whatsapp/channel", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req = withAuthContext(req, user)
		w := httptest.NewRecorder()

		s.handleCreateWhatsappChannel(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing identifier", func(t *testing.T) {
		s := createTestServer(t)
		user := database.CreateTestUser(t, s.db)

		body := map[string]string{
			"type": "sender",
			"name": "Test",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/whatsapp/channel", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req = withAuthContext(req, user)
		w := httptest.NewRecorder()

		s.handleCreateWhatsappChannel(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing name", func(t *testing.T) {
		s := createTestServer(t)
		user := database.CreateTestUser(t, s.db)

		body := map[string]string{
			"type":       "sender",
			"identifier": "test@s.whatsapp.net",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/whatsapp/channel", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req = withAuthContext(req, user)
		w := httptest.NewRecorder()

		s.handleCreateWhatsappChannel(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandleWhatsAppTopContacts(t *testing.T) {
	s := createTestServer(t)
	user := database.CreateTestUser(t, s.db)

	ch1, err := s.db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"1111111111@s.whatsapp.net",
		"Alice",
	)
	require.NoError(t, err)

	ch2, err := s.db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"2222222222@s.whatsapp.net",
		"Bob",
	)
	require.NoError(t, err)

	now := time.Now()
	_, err = s.db.StoreSourceMessage(
		source.SourceTypeWhatsApp,
		ch1.ID,
		"sender-1",
		"Sender 1",
		"One",
		"",
		now,
	)
	require.NoError(t, err)
	_, err = s.db.StoreSourceMessage(
		source.SourceTypeWhatsApp,
		ch2.ID,
		"sender-2",
		"Sender 2",
		"One",
		"",
		now,
	)
	require.NoError(t, err)
	_, err = s.db.StoreSourceMessage(
		source.SourceTypeWhatsApp,
		ch2.ID,
		"sender-2",
		"Sender 2",
		"Two",
		"",
		now,
	)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/whatsapp/top-contacts", nil)
	req = withAuthContext(req, user)
	w := httptest.NewRecorder()

	s.handleWhatsAppTopContacts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Contacts []TopContactResponse `json:"contacts"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	require.Len(t, resp.Contacts, 2)
	assert.Equal(t, ch2.Identifier, resp.Contacts[0].Identifier)
	assert.Equal(t, 2, resp.Contacts[0].MessageCount)
	assert.Equal(t, ch1.Identifier, resp.Contacts[1].Identifier)
	assert.Equal(t, 1, resp.Contacts[1].MessageCount)
}

func TestHandleWhatsAppTopContacts_EmptyHistory(t *testing.T) {
	s := createTestServer(t)
	user := database.CreateTestUser(t, s.db)

	req := httptest.NewRequest("GET", "/api/whatsapp/top-contacts", nil)
	req = withAuthContext(req, user)
	w := httptest.NewRecorder()

	s.handleWhatsAppTopContacts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Contacts []TopContactResponse `json:"contacts"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	require.Len(t, resp.Contacts, 0)
}

func TestHandleTelegramTopContacts_UserScoped(t *testing.T) {
	s := createTestServer(t)
	user1 := database.CreateTestUser(t, s.db)
	user2 := database.CreateTestUser(t, s.db)

	ch1, err := s.db.CreateSourceChannel(
		user1.ID,
		source.SourceTypeTelegram,
		source.ChannelTypeSender,
		"tg_user_1",
		"User One",
	)
	require.NoError(t, err)

	ch2, err := s.db.CreateSourceChannel(
		user2.ID,
		source.SourceTypeTelegram,
		source.ChannelTypeSender,
		"tg_user_2",
		"User Two",
	)
	require.NoError(t, err)

	now := time.Now()
	_, err = s.db.StoreSourceMessage(
		source.SourceTypeTelegram,
		ch1.ID,
		"tg_user_1",
		"User One",
		"Message 1",
		"",
		now,
	)
	require.NoError(t, err)
	_, err = s.db.StoreSourceMessage(
		source.SourceTypeTelegram,
		ch2.ID,
		"tg_user_2",
		"User Two",
		"Message 2",
		"",
		now,
	)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/telegram/top-contacts", nil)
	req = withAuthContext(req, user1)
	w := httptest.NewRecorder()

	s.handleTelegramTopContacts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Contacts []TelegramTopContactResponse `json:"contacts"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	require.Len(t, resp.Contacts, 1)
	assert.Equal(t, "tg_user_1", resp.Contacts[0].Identifier)
	assert.Equal(t, 1, resp.Contacts[0].MessageCount)
}

func TestHandleUpdateWhatsappChannel(t *testing.T) {
	s := createTestServer(t)
	user := database.CreateTestUser(t, s.db)

	// Create a channel first
	channel, err := s.db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"update@s.whatsapp.net",
		"Original Name",
	)
	require.NoError(t, err)

	t.Run("update channel successfully", func(t *testing.T) {
		body := map[string]interface{}{
			"name":    "Updated Name",
			"enabled": false,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("PUT", "/api/whatsapp/channel/"+strconv.FormatInt(channel.ID, 10), bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", strconv.FormatInt(channel.ID, 10))
		req = withAuthContext(req, user)
		w := httptest.NewRecorder()

		s.handleUpdateWhatsappChannel(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var updated database.SourceChannel
		err := json.Unmarshal(w.Body.Bytes(), &updated)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", updated.Name)
		assert.False(t, updated.Enabled)
	})

	t.Run("update non-existent channel returns not found", func(t *testing.T) {
		body := map[string]interface{}{
			"name":    "Name",
			"enabled": true,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("PUT", "/api/whatsapp/channel/999999", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", "999999")
		req = withAuthContext(req, user)
		w := httptest.NewRecorder()

		s.handleUpdateWhatsappChannel(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid channel id", func(t *testing.T) {
		body := map[string]interface{}{
			"name":    "Name",
			"enabled": true,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("PUT", "/api/whatsapp/channel/invalid", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", "invalid")
		req = withAuthContext(req, user)
		w := httptest.NewRecorder()

		s.handleUpdateWhatsappChannel(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandleDeleteWhatsappChannel(t *testing.T) {
	s := createTestServer(t)
	user := database.CreateTestUser(t, s.db)

	// Create a channel first
	channel, err := s.db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"delete@s.whatsapp.net",
		"To Delete",
	)
	require.NoError(t, err)

	t.Run("delete channel successfully", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/whatsapp/channel/"+strconv.FormatInt(channel.ID, 10), nil)
		req = withAuthContext(req, user)
		req.SetPathValue("id", strconv.FormatInt(channel.ID, 10))
		w := httptest.NewRecorder()

		s.handleDeleteWhatsappChannel(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify deleted
		deleted, _ := s.db.GetSourceChannelByID(user.ID, channel.ID)
		assert.Nil(t, deleted)
	})

	t.Run("delete non-existent channel returns 404", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/whatsapp/channel/999999", nil)
		req = withAuthContext(req, user)
		req.SetPathValue("id", "999999")
		w := httptest.NewRecorder()

		s.handleDeleteWhatsappChannel(w, req)

		// Returns 404 when channel doesn't exist or doesn't belong to user
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid channel id", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/whatsapp/channel/invalid", nil)
		req = withAuthContext(req, user)
		req.SetPathValue("id", "invalid")
		w := httptest.NewRecorder()

		s.handleDeleteWhatsappChannel(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandleListEvents(t *testing.T) {
	s := createTestServer(t)
	user := database.CreateTestUser(t, s.db)

	// Create a channel for events
	channel, err := s.db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"events@s.whatsapp.net",
		"Events Channel",
	)
	require.NoError(t, err)

	// Create some events
	event1 := &database.CalendarEvent{
		UserID:     user.ID,
		ChannelID:  channel.ID,
		CalendarID: "primary",
		Title:      "Event 1",
		StartTime:  time.Now(),
		ActionType: database.EventActionCreate,
	}
	_, err = s.db.CreatePendingEvent(event1)
	require.NoError(t, err)

	event2 := &database.CalendarEvent{
		UserID:     user.ID,
		ChannelID:  channel.ID,
		CalendarID: "primary",
		Title:      "Event 2",
		StartTime:  time.Now().Add(time.Hour),
		ActionType: database.EventActionCreate,
	}
	created2, err := s.db.CreatePendingEvent(event2)
	require.NoError(t, err)
	err = s.db.UpdateEventStatus(created2.ID, database.EventStatusSynced)
	require.NoError(t, err)

	t.Run("list all events", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/events", nil)
		req = withAuthContext(req, user)
		w := httptest.NewRecorder()

		s.handleListEvents(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var events []database.CalendarEvent
		err := json.Unmarshal(w.Body.Bytes(), &events)
		require.NoError(t, err)
		assert.Len(t, events, 2)
	})

	t.Run("filter by pending status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/events?status=pending", nil)
		req = withAuthContext(req, user)
		w := httptest.NewRecorder()

		s.handleListEvents(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var events []database.CalendarEvent
		err := json.Unmarshal(w.Body.Bytes(), &events)
		require.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, "Event 1", events[0].Title)
	})

	t.Run("filter by synced status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/events?status=synced", nil)
		req = withAuthContext(req, user)
		w := httptest.NewRecorder()

		s.handleListEvents(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var events []database.CalendarEvent
		err := json.Unmarshal(w.Body.Bytes(), &events)
		require.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, "Event 2", events[0].Title)
	})

	t.Run("filter by channel", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/events?channel_id="+strconv.FormatInt(channel.ID, 10), nil)
		req = withAuthContext(req, user)
		w := httptest.NewRecorder()

		s.handleListEvents(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var events []database.CalendarEvent
		err := json.Unmarshal(w.Body.Bytes(), &events)
		require.NoError(t, err)
		assert.Len(t, events, 2)
	})
}

func TestHandleGetEvent(t *testing.T) {
	s := createTestServer(t)
	user := database.CreateTestUser(t, s.db)

	// Create a channel and event
	channel, err := s.db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"getevent@s.whatsapp.net",
		"Get Event Channel",
	)
	require.NoError(t, err)

	event := &database.CalendarEvent{
		UserID:       user.ID,
		ChannelID:    channel.ID,
		CalendarID:   "primary",
		Title:        "Test Event",
		Description:  "Test Description",
		StartTime:    time.Now(),
		Location:     "Test Location",
		ActionType:   database.EventActionCreate,
		LLMReasoning: "Test reasoning",
	}
	created, err := s.db.CreatePendingEvent(event)
	require.NoError(t, err)

	t.Run("get existing event", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/events/"+strconv.FormatInt(created.ID, 10), nil)
		req.SetPathValue("id", strconv.FormatInt(created.ID, 10))
		req = withAuthContext(req, user)
		w := httptest.NewRecorder()

		s.handleGetEvent(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response["event"])
	})

	t.Run("get non-existent event", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/events/999999", nil)
		req.SetPathValue("id", "999999")
		req = withAuthContext(req, user)
		w := httptest.NewRecorder()

		s.handleGetEvent(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid event id", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/events/invalid", nil)
		req.SetPathValue("id", "invalid")
		req = withAuthContext(req, user)
		w := httptest.NewRecorder()

		s.handleGetEvent(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandleRejectEvent(t *testing.T) {
	s := createTestServer(t)
	user := database.CreateTestUser(t, s.db)

	// Create a channel and pending event
	channel, err := s.db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"reject@s.whatsapp.net",
		"Reject Channel",
	)
	require.NoError(t, err)

	event := &database.CalendarEvent{
		UserID:    user.ID,
		ChannelID:  channel.ID,
		CalendarID: "primary",
		Title:      "Event to Reject",
		StartTime:  time.Now(),
		ActionType: database.EventActionCreate,
	}
	created, err := s.db.CreatePendingEvent(event)
	require.NoError(t, err)

	t.Run("reject pending event", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/events/"+strconv.FormatInt(created.ID, 10)+"/reject", nil)
		req.SetPathValue("id", strconv.FormatInt(created.ID, 10))
		req = withAuthContext(req, user)
		w := httptest.NewRecorder()

		s.handleRejectEvent(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify status changed
		rejected, err := s.db.GetEventByID(created.ID)
		require.NoError(t, err)
		assert.Equal(t, database.EventStatusRejected, rejected.Status)
	})

	t.Run("reject non-existent event", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/events/999999/reject", nil)
		req.SetPathValue("id", "999999")
		w := httptest.NewRecorder()

		s.handleRejectEvent(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestRespondJSON(t *testing.T) {
	w := httptest.NewRecorder()

	data := map[string]string{"key": "value"}
	respondJSON(w, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "value", response["key"])
}

func TestRespondError(t *testing.T) {
	w := httptest.NewRecorder()

	respondError(w, http.StatusBadRequest, "test error message")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "test error message", response["error"])
}

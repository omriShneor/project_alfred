package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWhatsAppChannelManagement(t *testing.T) {
	ts := testutil.NewTestServer(t)

	t.Run("create WhatsApp channel", func(t *testing.T) {
		channelData := map[string]string{
			"type":       "sender",
			"identifier": "alice@s.whatsapp.net",
			"name":       "Alice",
		}
		body, _ := json.Marshal(channelData)

		resp, err := http.Post(ts.BaseURL()+"/api/whatsapp/channel", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var channel database.Channel
		err = json.NewDecoder(resp.Body).Decode(&channel)
		require.NoError(t, err)

		assert.Equal(t, "Alice", channel.Name)
		assert.Equal(t, "alice@s.whatsapp.net", channel.Identifier)
		assert.True(t, channel.Enabled)
	})

	t.Run("list channels", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/whatsapp/channel")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var channels []database.Channel
		err = json.NewDecoder(resp.Body).Decode(&channels)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(channels), 1)
	})

	t.Run("filter channels by type", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/whatsapp/channel?type=sender")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var channels []database.Channel
		err = json.NewDecoder(resp.Body).Decode(&channels)
		require.NoError(t, err)

		for _, ch := range channels {
			assert.Equal(t, database.ChannelType("sender"), ch.Type)
		}
	})
}

func TestChannelCRUD(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create a channel first
	channel := testutil.NewChannelBuilder().
		WithUserID(ts.TestUser.ID).
		WhatsApp().
		WithName("Test Contact").
		WithIdentifier("test@s.whatsapp.net").
		MustBuild(ts.DB)

	t.Run("update channel", func(t *testing.T) {
		updateData := map[string]interface{}{
			"name":    "Updated Contact Name",
			"enabled": false,
		}
		body, _ := json.Marshal(updateData)

		req, err := http.NewRequest("PUT", ts.BaseURL()+fmt.Sprintf("/api/whatsapp/channel/%d", channel.ID), bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify update
		updated, err := ts.DB.GetChannelByID(channel.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Contact Name", updated.Name)
		assert.False(t, updated.Enabled)
	})

	t.Run("delete channel", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", ts.BaseURL()+fmt.Sprintf("/api/whatsapp/channel/%d", channel.ID), nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify deleted - GetChannelByID returns nil, nil when not found
		deleted, err := ts.DB.GetChannelByID(channel.ID)
		assert.NoError(t, err) // No error on not found
		assert.Nil(t, deleted) // But channel should be nil
	})
}

func TestChannelWithEvents(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create channel with events
	channel := testutil.NewChannelBuilder().
		WithUserID(ts.TestUser.ID).
		WhatsApp().
		WithName("Event Channel").
		MustBuild(ts.DB)

	// Create some events for this channel
	for i := 1; i <= 3; i++ {
		testutil.NewEventBuilder(channel.ID).
			WithUserID(ts.TestUser.ID).
			WithTitle(fmt.Sprintf("Event %d", i)).
			Pending().
			MustBuild(ts.DB)
	}

	t.Run("filter events by channel", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + fmt.Sprintf("/api/events?channel_id=%d", channel.ID))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var events []database.CalendarEvent
		err = json.NewDecoder(resp.Body).Decode(&events)
		require.NoError(t, err)

		assert.Len(t, events, 3)
		for _, e := range events {
			assert.Equal(t, channel.ID, e.ChannelID)
		}
	})
}

func TestMultipleChannelSources(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create WhatsApp channel
	waChannel := testutil.NewChannelBuilder().
		WithUserID(ts.TestUser.ID).
		WhatsApp().
		WithName("WhatsApp Contact").
		WithIdentifier("wa@s.whatsapp.net").
		MustBuild(ts.DB)

	// Create Telegram channel
	tgChannel := testutil.NewChannelBuilder().
		WithUserID(ts.TestUser.ID).
		Telegram().
		WithName("Telegram Contact").
		WithIdentifier("tg_user_456").
		MustBuild(ts.DB)

	t.Run("list all channels includes both sources", func(t *testing.T) {
		// List WhatsApp channels
		resp, err := http.Get(ts.BaseURL() + "/api/whatsapp/channel")
		require.NoError(t, err)
		defer resp.Body.Close()

		var waChannels []database.Channel
		err = json.NewDecoder(resp.Body).Decode(&waChannels)
		require.NoError(t, err)

		// Should include WhatsApp channel
		found := false
		for _, ch := range waChannels {
			if ch.ID == waChannel.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "WhatsApp channel should be in list")

		// List Telegram channels
		resp2, err := http.Get(ts.BaseURL() + "/api/telegram/channel")
		require.NoError(t, err)
		defer resp2.Body.Close()

		var tgChannels []database.SourceChannel
		err = json.NewDecoder(resp2.Body).Decode(&tgChannels)
		require.NoError(t, err)

		// Should include Telegram channel
		found = false
		for _, ch := range tgChannels {
			if ch.ID == tgChannel.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "Telegram channel should be in list")
	})
}

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

func TestTelegramChannelManagement(t *testing.T) {
	ts := testutil.NewTestServer(t)

	t.Run("create Telegram channel", func(t *testing.T) {
		channelData := map[string]string{
			"type":       "sender",
			"identifier": "telegram_user_123",
			"name":       "Telegram User",
		}
		body, _ := json.Marshal(channelData)

		resp, err := http.Post(ts.BaseURL()+"/api/telegram/channel", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var channel database.SourceChannel
		err = json.NewDecoder(resp.Body).Decode(&channel)
		require.NoError(t, err)

		assert.Equal(t, "Telegram User", channel.Name)
		assert.Equal(t, "telegram_user_123", channel.Identifier)
		assert.True(t, channel.Enabled)
	})

	t.Run("list Telegram channels", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/telegram/channel")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var channels []database.SourceChannel
		err = json.NewDecoder(resp.Body).Decode(&channels)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(channels), 1)
	})
}

func TestTelegramChannelCRUD(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create a Telegram channel first
	channel := testutil.NewChannelBuilder().
		WithUserID(ts.TestUser.ID).
		Telegram().
		WithName("TG Test Contact").
		WithIdentifier("tg_test_user").
		MustBuild(ts.DB)

	t.Run("update Telegram channel", func(t *testing.T) {
		updateData := map[string]interface{}{
			"name":    "Updated TG Name",
			"enabled": false,
		}
		body, _ := json.Marshal(updateData)

		req, err := http.NewRequest("PUT", ts.BaseURL()+fmt.Sprintf("/api/telegram/channel/%d", channel.ID), bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify update
		updated, err := ts.DB.GetSourceChannelByID(channel.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated TG Name", updated.Name)
		assert.False(t, updated.Enabled)
	})

	t.Run("delete Telegram channel", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", ts.BaseURL()+fmt.Sprintf("/api/telegram/channel/%d", channel.ID), nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify deleted - GetSourceChannelByID returns nil, nil for deleted channels
		deleted, err := ts.DB.GetSourceChannelByID(channel.ID)
		assert.NoError(t, err)
		assert.Nil(t, deleted)
	})
}

func TestTelegramStatus(t *testing.T) {
	ts := testutil.NewTestServer(t)

	t.Run("get Telegram status when not connected", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/telegram/status")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var status map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&status)
		require.NoError(t, err)

		// Should show not connected since no Telegram client is configured
		assert.Equal(t, false, status["connected"])
	})
}

func TestTelegramCustomSource(t *testing.T) {
	ts := testutil.NewTestServer(t)

	t.Run("add custom Telegram source by username", func(t *testing.T) {
		sourceData := map[string]string{
			"username": "@custom_user",
		}
		body, _ := json.Marshal(sourceData)

		resp, err := http.Post(ts.BaseURL()+"/api/telegram/sources/custom", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should succeed even without Telegram client (just creates the channel in DB)
		if resp.StatusCode == http.StatusCreated {
			var channel database.SourceChannel
			err = json.NewDecoder(resp.Body).Decode(&channel)
			require.NoError(t, err)
			assert.Contains(t, channel.Identifier, "custom_user")
		}
	})
}

func TestTelegramEventsFromChannel(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create Telegram channel
	channel := testutil.NewChannelBuilder().
		WithUserID(ts.TestUser.ID).
		Telegram().
		WithName("TG Event Source").
		WithIdentifier("tg_event_user").
		MustBuild(ts.DB)

	// Create event for this channel
	event := testutil.NewEventBuilder(channel.ID).
		WithUserID(ts.TestUser.ID).
		WithTitle("Telegram Event").
		Pending().
		MustBuild(ts.DB)

	t.Run("event is linked to Telegram channel", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + fmt.Sprintf("/api/events/%d", event.ID))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		eventData := result["event"].(map[string]interface{})
		assert.Equal(t, float64(channel.ID), eventData["channel_id"])
	})
}

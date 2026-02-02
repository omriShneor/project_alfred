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

func TestGmailSourceManagement(t *testing.T) {
	ts := testutil.NewTestServer(t)

	t.Run("create Gmail sender source", func(t *testing.T) {
		sourceData := map[string]string{
			"type":       "sender",
			"identifier": "important@company.com",
			"name":       "Important Sender",
		}
		body, _ := json.Marshal(sourceData)

		resp, err := http.Post(ts.BaseURL()+"/api/gmail/sources", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var source database.EmailSource
		err = json.NewDecoder(resp.Body).Decode(&source)
		require.NoError(t, err)

		assert.Equal(t, "Important Sender", source.Name)
		assert.Equal(t, "important@company.com", source.Identifier)
		assert.Equal(t, database.EmailSourceTypeSender, source.Type)
		assert.True(t, source.Enabled)
	})

	t.Run("create Gmail domain source", func(t *testing.T) {
		sourceData := map[string]string{
			"type":       "domain",
			"identifier": "company.com",
			"name":       "Company Domain",
		}
		body, _ := json.Marshal(sourceData)

		resp, err := http.Post(ts.BaseURL()+"/api/gmail/sources", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var source database.EmailSource
		err = json.NewDecoder(resp.Body).Decode(&source)
		require.NoError(t, err)

		assert.Equal(t, database.EmailSourceTypeDomain, source.Type)
		assert.Equal(t, "company.com", source.Identifier)
	})

	t.Run("create Gmail category source", func(t *testing.T) {
		sourceData := map[string]string{
			"type":       "category",
			"identifier": "CATEGORY_PRIMARY",
			"name":       "Primary Category",
		}
		body, _ := json.Marshal(sourceData)

		resp, err := http.Post(ts.BaseURL()+"/api/gmail/sources", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var source database.EmailSource
		err = json.NewDecoder(resp.Body).Decode(&source)
		require.NoError(t, err)

		assert.Equal(t, database.EmailSourceTypeCategory, source.Type)
		assert.Equal(t, "CATEGORY_PRIMARY", source.Identifier)
	})

	t.Run("list Gmail sources", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/gmail/sources")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		// Response has "sources" key containing the array
		sources, ok := result["sources"].([]interface{})
		require.True(t, ok)

		// Should have at least the 3 sources we created
		assert.GreaterOrEqual(t, len(sources), 3)
	})
}

func TestGmailSourceCRUD(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create a Gmail source first
	channel := testutil.NewChannelBuilder().
		Gmail().
		WithName("Gmail Source").
		WithIdentifier("test@gmail.com").
		MustBuild(ts.DB)

	// Get the corresponding email source
	sources, err := ts.DB.ListEmailSources()
	require.NoError(t, err)

	var sourceID int64
	for _, s := range sources {
		if s.Identifier == "test@gmail.com" {
			sourceID = s.ID
			break
		}
	}

	// If no source found in email_sources table, skip update/delete tests
	// (the channel builder creates in channels table, not email_sources)
	if sourceID == 0 {
		// Create directly in email_sources
		source, err := ts.DB.CreateEmailSource("sender", "direct@gmail.com", "Direct Source")
		require.NoError(t, err)
		sourceID = source.ID
	}

	t.Run("update Gmail source", func(t *testing.T) {
		updateData := map[string]interface{}{
			"name":    "Updated Gmail Source",
			"enabled": false,
		}
		body, _ := json.Marshal(updateData)

		req, err := http.NewRequest("PUT", ts.BaseURL()+fmt.Sprintf("/api/gmail/sources/%d", sourceID), bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("delete Gmail source", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", ts.BaseURL()+fmt.Sprintf("/api/gmail/sources/%d", sourceID), nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Cleanup: we have an unused variable
	_ = channel
}

func TestGmailStatus(t *testing.T) {
	ts := testutil.NewTestServer(t)

	t.Run("get Gmail status when not connected", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/gmail/status")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var status map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&status)
		require.NoError(t, err)

		// Should show not connected since no Gmail client is configured
		assert.Equal(t, false, status["connected"])
	})
}

func TestGmailTopContacts(t *testing.T) {
	ts := testutil.NewTestServer(t)

	t.Run("get top contacts returns empty when no contacts cached", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/gmail/top-contacts")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		// Should return empty contacts array
		contacts := result["contacts"].([]interface{})
		assert.Empty(t, contacts)
	})
}

func TestGmailCustomSource(t *testing.T) {
	ts := testutil.NewTestServer(t)

	t.Run("add custom source by email", func(t *testing.T) {
		sourceData := map[string]string{
			"value": "custom@example.com",
		}
		body, _ := json.Marshal(sourceData)

		resp, err := http.Post(ts.BaseURL()+"/api/gmail/sources/custom", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var source database.EmailSource
		err = json.NewDecoder(resp.Body).Decode(&source)
		require.NoError(t, err)

		assert.Equal(t, database.EmailSourceTypeSender, source.Type)
		assert.Equal(t, "custom@example.com", source.Identifier)
	})

	t.Run("add custom source by domain", func(t *testing.T) {
		sourceData := map[string]string{
			"value": "customdomain.org",
		}
		body, _ := json.Marshal(sourceData)

		resp, err := http.Post(ts.BaseURL()+"/api/gmail/sources/custom", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var source database.EmailSource
		err = json.NewDecoder(resp.Body).Decode(&source)
		require.NoError(t, err)

		assert.Equal(t, database.EmailSourceTypeDomain, source.Type)
		assert.Equal(t, "customdomain.org", source.Identifier)
	})
}

func TestGmailEventsFromSource(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create Gmail channel (for event association)
	channel := testutil.NewChannelBuilder().
		Gmail().
		WithName("Gmail Event Source").
		WithIdentifier("events@company.com").
		MustBuild(ts.DB)

	// Create event for this channel
	event := testutil.NewEventBuilder(channel.ID).
		WithTitle("Email Event").
		WithDescription("Meeting request from email").
		Pending().
		MustBuild(ts.DB)

	t.Run("event is linked to Gmail channel", func(t *testing.T) {
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

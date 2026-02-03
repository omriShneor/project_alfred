package database

import (
	"testing"
	"time"

	"github.com/omriShneor/project_alfred/internal/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestChannelForMessages creates a channel for message tests
func createTestChannelForMessages(t *testing.T, db *DB, userID int64) *SourceChannel {
	t.Helper()
	channel, err := db.CreateSourceChannel(
		userID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"msg-test@s.whatsapp.net",
		"Message Test Contact",
	)
	require.NoError(t, err)
	return channel
}

func TestStoreSourceMessage(t *testing.T) {
	tests := []struct {
		name       string
		sourceType source.SourceType
		senderID   string
		senderName string
		text       string
		subject    string
	}{
		{
			name:       "store whatsapp message",
			sourceType: source.SourceTypeWhatsApp,
			senderID:   "1234567890@s.whatsapp.net",
			senderName: "John Doe",
			text:       "Hello, this is a test message!",
			subject:    "",
		},
		{
			name:       "store telegram message",
			sourceType: source.SourceTypeTelegram,
			senderID:   "telegram_user_123",
			senderName: "Jane Smith",
			text:       "Telegram message content",
			subject:    "",
		},
		{
			name:       "store email message with subject",
			sourceType: source.SourceTypeGmail,
			senderID:   "sender@example.com",
			senderName: "Email Sender",
			text:       "Email body content",
			subject:    "Important Meeting Tomorrow",
		},
		{
			name:       "store message with unicode",
			sourceType: source.SourceTypeWhatsApp,
			senderID:   "unicode@s.whatsapp.net",
			senderName: "日本語 User",
			text:       "Hello 你好 🎉 Привет",
			subject:    "",
		},
		{
			name:       "store message with empty text",
			sourceType: source.SourceTypeWhatsApp,
			senderID:   "empty@s.whatsapp.net",
			senderName: "Empty Message",
			text:       "",
			subject:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := NewTestDB(t)
			user := CreateTestUser(t, db)

			// Create appropriate channel
			channel, err := db.CreateSourceChannel(
				user.ID,
				tt.sourceType,
				source.ChannelTypeSender,
				tt.senderID,
				tt.senderName,
			)
			require.NoError(t, err)

			timestamp := time.Now().Add(-time.Hour)

			msg, err := db.StoreSourceMessage(
				tt.sourceType,
				channel.ID,
				tt.senderID,
				tt.senderName,
				tt.text,
				tt.subject,
				timestamp,
			)

			require.NoError(t, err)
			require.NotNil(t, msg)
			assert.NotZero(t, msg.ID)
			assert.Equal(t, tt.sourceType, msg.SourceType)
			assert.Equal(t, channel.ID, msg.ChannelID)
			assert.Equal(t, tt.senderID, msg.SenderID)
			assert.Equal(t, tt.senderName, msg.SenderName)
			assert.Equal(t, tt.text, msg.MessageText)
			assert.Equal(t, tt.subject, msg.Subject)
			assert.WithinDuration(t, timestamp, msg.Timestamp, time.Second)
		})
	}
}

func TestStoreSourceMessage_InvalidChannel(t *testing.T) {
	db := NewTestDB(t)

	// Try to store message for non-existent channel
	_, err := db.StoreSourceMessage(
		source.SourceTypeWhatsApp,
		999999, // Non-existent channel ID
		"sender@s.whatsapp.net",
		"Sender",
		"Message text",
		"",
		time.Now(),
	)

	assert.Error(t, err, "should fail with non-existent channel due to foreign key constraint")
}

func TestGetSourceMessageHistory(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)
	channel := createTestChannelForMessages(t, db, user.ID)

	// Store messages with different timestamps
	baseTime := time.Now()
	for i := 0; i < 5; i++ {
		timestamp := baseTime.Add(time.Duration(i) * time.Minute)
		_, err := db.StoreSourceMessage(
			source.SourceTypeWhatsApp,
			channel.ID,
			"sender@s.whatsapp.net",
			"Sender",
			"Message "+string(rune('A'+i)),
			"",
			timestamp,
		)
		require.NoError(t, err)
	}

	t.Run("get all messages in chronological order", func(t *testing.T) {
		messages, err := db.GetSourceMessageHistory(source.SourceTypeWhatsApp, channel.ID, 10)
		require.NoError(t, err)
		assert.Len(t, messages, 5)

		// Should be in chronological order (oldest first)
		for i := 0; i < len(messages)-1; i++ {
			assert.True(t, messages[i].Timestamp.Before(messages[i+1].Timestamp) ||
				messages[i].Timestamp.Equal(messages[i+1].Timestamp),
				"messages should be in chronological order")
		}

		// First message should be "Message A"
		assert.Equal(t, "Message A", messages[0].MessageText)
		// Last message should be "Message E"
		assert.Equal(t, "Message E", messages[4].MessageText)
	})

	t.Run("respect limit parameter", func(t *testing.T) {
		messages, err := db.GetSourceMessageHistory(source.SourceTypeWhatsApp, channel.ID, 3)
		require.NoError(t, err)
		assert.Len(t, messages, 3)

		// Should return the LATEST 3 messages in chronological order
		// So we get C, D, E (not A, B, C)
		assert.Equal(t, "Message C", messages[0].MessageText)
		assert.Equal(t, "Message D", messages[1].MessageText)
		assert.Equal(t, "Message E", messages[2].MessageText)
	})

	t.Run("empty history for channel with no messages", func(t *testing.T) {
		emptyChannel, err := db.CreateSourceChannel(
			user.ID,
			source.SourceTypeWhatsApp,
			source.ChannelTypeSender,
			"empty@s.whatsapp.net",
			"Empty Channel",
		)
		require.NoError(t, err)

		messages, err := db.GetSourceMessageHistory(source.SourceTypeWhatsApp, emptyChannel.ID, 10)
		require.NoError(t, err)
		assert.Len(t, messages, 0)
	})

	t.Run("filter by source type", func(t *testing.T) {
		// Create a Telegram channel and message
		tgChannel, err := db.CreateSourceChannel(
			user.ID,
			source.SourceTypeTelegram,
			source.ChannelTypeSender,
			"tg_user",
			"Telegram User",
		)
		require.NoError(t, err)

		_, err = db.StoreSourceMessage(
			source.SourceTypeTelegram,
			tgChannel.ID,
			"tg_user",
			"TG User",
			"Telegram message",
			"",
			time.Now(),
		)
		require.NoError(t, err)

		// Getting WhatsApp history should not include Telegram messages
		waMessages, err := db.GetSourceMessageHistory(source.SourceTypeWhatsApp, channel.ID, 10)
		require.NoError(t, err)
		for _, msg := range waMessages {
			assert.Equal(t, source.SourceTypeWhatsApp, msg.SourceType)
		}
	})
}

func TestPruneSourceMessages(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)
	channel := createTestChannelForMessages(t, db, user.ID)

	// Store 10 messages
	baseTime := time.Now()
	for i := 0; i < 10; i++ {
		timestamp := baseTime.Add(time.Duration(i) * time.Minute)
		_, err := db.StoreSourceMessage(
			source.SourceTypeWhatsApp,
			channel.ID,
			"sender@s.whatsapp.net",
			"Sender",
			"Message "+string(rune('A'+i)),
			"",
			timestamp,
		)
		require.NoError(t, err)
	}

	// Verify 10 messages exist
	count, err := db.CountSourceMessages(source.SourceTypeWhatsApp, channel.ID)
	require.NoError(t, err)
	assert.Equal(t, 10, count)

	t.Run("prune to keep only 5 newest messages", func(t *testing.T) {
		err := db.PruneSourceMessages(source.SourceTypeWhatsApp, channel.ID, 5)
		require.NoError(t, err)

		count, err := db.CountSourceMessages(source.SourceTypeWhatsApp, channel.ID)
		require.NoError(t, err)
		assert.Equal(t, 5, count)

		// Verify the newest 5 messages remain (F, G, H, I, J)
		messages, err := db.GetSourceMessageHistory(source.SourceTypeWhatsApp, channel.ID, 10)
		require.NoError(t, err)
		assert.Len(t, messages, 5)

		// In chronological order: F, G, H, I, J
		assert.Equal(t, "Message F", messages[0].MessageText)
		assert.Equal(t, "Message J", messages[4].MessageText)
	})

	t.Run("prune when already at or below limit", func(t *testing.T) {
		// Should be a no-op
		err := db.PruneSourceMessages(source.SourceTypeWhatsApp, channel.ID, 10)
		require.NoError(t, err)

		count, err := db.CountSourceMessages(source.SourceTypeWhatsApp, channel.ID)
		require.NoError(t, err)
		assert.Equal(t, 5, count) // Still 5 from previous test
	})

	t.Run("prune does not affect other source types", func(t *testing.T) {
		// Create Telegram channel with messages
		tgChannel, err := db.CreateSourceChannel(
			user.ID,
			source.SourceTypeTelegram,
			source.ChannelTypeSender,
			"tg_prune_test",
			"TG Prune Test",
		)
		require.NoError(t, err)

		for i := 0; i < 5; i++ {
			_, err := db.StoreSourceMessage(
				source.SourceTypeTelegram,
				tgChannel.ID,
				"tg_prune_test",
				"TG User",
				"TG Message",
				"",
				time.Now().Add(time.Duration(i)*time.Minute),
			)
			require.NoError(t, err)
		}

		// Prune WhatsApp messages
		err = db.PruneSourceMessages(source.SourceTypeWhatsApp, channel.ID, 3)
		require.NoError(t, err)

		// Telegram messages should be unaffected
		tgCount, err := db.CountSourceMessages(source.SourceTypeTelegram, tgChannel.ID)
		require.NoError(t, err)
		assert.Equal(t, 5, tgCount)
	})
}

func TestGetSourceMessageByID(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)
	channel := createTestChannelForMessages(t, db, user.ID)

	// Store a message
	stored, err := db.StoreSourceMessage(
		source.SourceTypeWhatsApp,
		channel.ID,
		"sender@s.whatsapp.net",
		"Sender Name",
		"Test message text",
		"",
		time.Now(),
	)
	require.NoError(t, err)

	t.Run("get existing message", func(t *testing.T) {
		msg, err := db.GetSourceMessageByID(stored.ID)
		require.NoError(t, err)
		require.NotNil(t, msg)
		assert.Equal(t, stored.ID, msg.ID)
		assert.Equal(t, "Test message text", msg.MessageText)
		assert.Equal(t, "Sender Name", msg.SenderName)
	})

	t.Run("get non-existent message returns nil", func(t *testing.T) {
		msg, err := db.GetSourceMessageByID(999999)
		require.NoError(t, err)
		assert.Nil(t, msg)
	})
}

func TestCountSourceMessages(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)
	channel := createTestChannelForMessages(t, db, user.ID)

	t.Run("zero messages initially", func(t *testing.T) {
		count, err := db.CountSourceMessages(source.SourceTypeWhatsApp, channel.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	// Add some messages
	for i := 0; i < 7; i++ {
		_, err := db.StoreSourceMessage(
			source.SourceTypeWhatsApp,
			channel.ID,
			"sender@s.whatsapp.net",
			"Sender",
			"Message",
			"",
			time.Now(),
		)
		require.NoError(t, err)
	}

	t.Run("counts messages correctly", func(t *testing.T) {
		count, err := db.CountSourceMessages(source.SourceTypeWhatsApp, channel.ID)
		require.NoError(t, err)
		assert.Equal(t, 7, count)
	})

	t.Run("counts only for specified source type", func(t *testing.T) {
		// Create Telegram channel with messages
		tgChannel, err := db.CreateSourceChannel(
			user.ID,
			source.SourceTypeTelegram,
			source.ChannelTypeSender,
			"tg_count_test",
			"TG Count",
		)
		require.NoError(t, err)

		_, err = db.StoreSourceMessage(
			source.SourceTypeTelegram,
			tgChannel.ID,
			"tg_count_test",
			"TG User",
			"TG Message",
			"",
			time.Now(),
		)
		require.NoError(t, err)

		// Count should still be 7 for WhatsApp
		waCount, err := db.CountSourceMessages(source.SourceTypeWhatsApp, channel.ID)
		require.NoError(t, err)
		assert.Equal(t, 7, waCount)

		// Telegram should be 1
		tgCount, err := db.CountSourceMessages(source.SourceTypeTelegram, tgChannel.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, tgCount)
	})
}

func TestSourceMessageToHistoryMessage(t *testing.T) {
	// Test the ToHistoryMessage conversion method
	msg := &SourceMessage{
		ID:          42,
		SourceType:  source.SourceTypeGmail,
		ChannelID:   100,
		SenderID:    "sender@example.com",
		SenderName:  "Sender Name",
		MessageText: "Message content",
		Subject:     "Email Subject",
		Timestamp:   time.Now(),
	}

	converted := msg.ToHistoryMessage()

	assert.Equal(t, int64(42), converted.ID)
	assert.Equal(t, "sender@example.com", converted.SenderID)
	assert.Equal(t, "Sender Name", converted.SenderName)
	assert.Equal(t, "Message content", converted.Text)
	assert.Equal(t, "Email Subject", converted.Subject)
	assert.Equal(t, msg.Timestamp, converted.Timestamp)
}

func TestGetAllSourceMessages(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)

	// Create multiple channels
	ch1, err := db.CreateSourceChannel(user.ID, source.SourceTypeWhatsApp, source.ChannelTypeSender, "wa1@s.whatsapp.net", "WA 1")
	require.NoError(t, err)
	ch2, err := db.CreateSourceChannel(user.ID, source.SourceTypeWhatsApp, source.ChannelTypeSender, "wa2@s.whatsapp.net", "WA 2")
	require.NoError(t, err)

	// Store messages in different channels
	for i := 0; i < 3; i++ {
		_, err := db.StoreSourceMessage(source.SourceTypeWhatsApp, ch1.ID, "wa1@s.whatsapp.net", "WA 1", "Channel 1 msg", "", time.Now().Add(time.Duration(i)*time.Minute))
		require.NoError(t, err)
		_, err = db.StoreSourceMessage(source.SourceTypeWhatsApp, ch2.ID, "wa2@s.whatsapp.net", "WA 2", "Channel 2 msg", "", time.Now().Add(time.Duration(i)*time.Minute))
		require.NoError(t, err)
	}

	t.Run("get all messages across channels", func(t *testing.T) {
		messages, err := db.GetAllSourceMessages(source.SourceTypeWhatsApp, 10)
		require.NoError(t, err)
		assert.Len(t, messages, 6) // 3 from each channel
	})

	t.Run("respects limit", func(t *testing.T) {
		messages, err := db.GetAllSourceMessages(source.SourceTypeWhatsApp, 3)
		require.NoError(t, err)
		assert.Len(t, messages, 3)
	})

	t.Run("filter by source type", func(t *testing.T) {
		// Create Telegram channel with message
		tgChannel, err := db.CreateSourceChannel(user.ID, source.SourceTypeTelegram, source.ChannelTypeSender, "tg_all", "TG All")
		require.NoError(t, err)
		_, err = db.StoreSourceMessage(source.SourceTypeTelegram, tgChannel.ID, "tg_all", "TG", "TG msg", "", time.Now())
		require.NoError(t, err)

		// WhatsApp should still return only WhatsApp messages
		waMessages, err := db.GetAllSourceMessages(source.SourceTypeWhatsApp, 10)
		require.NoError(t, err)
		for _, msg := range waMessages {
			assert.Equal(t, source.SourceTypeWhatsApp, msg.SourceType)
		}
	})
}

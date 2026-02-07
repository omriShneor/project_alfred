package database

import (
	"testing"

	"github.com/omriShneor/project_alfred/internal/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateSourceChannel(t *testing.T) {
	tests := []struct {
		name        string
		sourceType  source.SourceType
		channelType source.ChannelType
		identifier  string
		channelName string
	}{
		{
			name:        "create whatsapp sender channel",
			sourceType:  source.SourceTypeWhatsApp,
			channelType: source.ChannelTypeSender,
			identifier:  "1234567890@s.whatsapp.net",
			channelName: "WhatsApp Contact",
		},
		{
			name:        "create telegram sender channel",
			sourceType:  source.SourceTypeTelegram,
			channelType: source.ChannelTypeSender,
			identifier:  "telegram_user_123",
			channelName: "Telegram Contact",
		},
		{
			name:        "create gmail sender channel",
			sourceType:  source.SourceTypeGmail,
			channelType: source.ChannelTypeSender,
			identifier:  "test@example.com",
			channelName: "Email Contact",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := NewTestDB(t)
			user := CreateTestUser(t, db)

			channel, err := db.CreateSourceChannel(
				user.ID,
				tt.sourceType,
				tt.channelType,
				tt.identifier,
				tt.channelName,
			)

			require.NoError(t, err)
			require.NotNil(t, channel)
			assert.NotZero(t, channel.ID)
			assert.Equal(t, user.ID, channel.UserID)
			assert.Equal(t, tt.sourceType, channel.SourceType)
			assert.Equal(t, tt.channelType, channel.Type)
			assert.Equal(t, tt.identifier, channel.Identifier)
			assert.Equal(t, tt.channelName, channel.Name)
			assert.True(t, channel.Enabled, "new channels should be enabled by default")
		})
	}
}

func TestCreateSourceChannel_DuplicateIdentifier(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)

	// Create first channel
	_, err := db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"duplicate@s.whatsapp.net",
		"First Channel",
	)
	require.NoError(t, err)

	// Try to create channel with same identifier and source type
	_, err = db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"duplicate@s.whatsapp.net",
		"Duplicate Channel",
	)
	assert.Error(t, err, "should fail on duplicate identifier for same source type")
}

func TestCreateSourceChannel_DifferentSourceTypes(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)

	// Create WhatsApp channel
	ch1, err := db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"wa_user@s.whatsapp.net",
		"WhatsApp Channel",
	)
	require.NoError(t, err)

	// Different identifier for different source type
	ch2, err := db.CreateSourceChannel(
		user.ID,
		source.SourceTypeGmail,
		source.ChannelTypeSender,
		"email_user@example.com",
		"Gmail Channel",
	)
	require.NoError(t, err)

	assert.NotEqual(t, ch1.ID, ch2.ID)
	assert.Equal(t, source.SourceTypeWhatsApp, ch1.SourceType)
	assert.Equal(t, source.SourceTypeGmail, ch2.SourceType)
}

func TestCreateSourceChannel_SameIdentifierDifferentUsers(t *testing.T) {
	db := NewTestDB(t)
	user1 := CreateTestUser(t, db)
	user2 := CreateTestUser(t, db)

	identifier := "shared@s.whatsapp.net"

	_, err := db.CreateSourceChannel(
		user1.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		identifier,
		"User 1 Channel",
	)
	require.NoError(t, err)

	_, err = db.CreateSourceChannel(
		user2.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		identifier,
		"User 2 Channel",
	)
	require.NoError(t, err)
}

func TestEnsureManualReminderChannel(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)

	first, err := db.EnsureManualReminderChannel(user.ID)
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Equal(t, user.ID, first.UserID)
	assert.Equal(t, source.SourceType("manual"), first.SourceType)
	assert.Equal(t, source.ChannelTypeSender, first.Type)
	assert.Equal(t, "manual:todo", first.Identifier)
	assert.Equal(t, "My Tasks", first.Name)

	second, err := db.EnsureManualReminderChannel(user.ID)
	require.NoError(t, err)
	require.NotNil(t, second)
	assert.Equal(t, first.ID, second.ID, "channel should be reused")
}

func TestGetSourceChannelByID(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)

	// Create a channel
	created, err := db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"test@s.whatsapp.net",
		"Test Channel",
	)
	require.NoError(t, err)

	t.Run("get existing channel", func(t *testing.T) {
		channel, err := db.GetSourceChannelByID(user.ID, created.ID)
		require.NoError(t, err)
		require.NotNil(t, channel)
		assert.Equal(t, created.ID, channel.ID)
		assert.Equal(t, "Test Channel", channel.Name)
		assert.Equal(t, source.SourceTypeWhatsApp, channel.SourceType)
	})

	t.Run("get non-existent channel returns nil", func(t *testing.T) {
		channel, err := db.GetSourceChannelByID(user.ID, 999999)
		require.NoError(t, err)
		assert.Nil(t, channel)
	})
}

func TestGetSourceChannelByIdentifier(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)

	// Create channels
	waChannel, err := db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"1234567890@s.whatsapp.net",
		"WA Contact",
	)
	require.NoError(t, err)

	gmailChannel, err := db.CreateSourceChannel(
		user.ID,
		source.SourceTypeGmail,
		source.ChannelTypeSender,
		"test@example.com",
		"Email Contact",
	)
	require.NoError(t, err)

	t.Run("find whatsapp channel", func(t *testing.T) {
		channel, err := db.GetSourceChannelByIdentifier(
			user.ID,
			source.SourceTypeWhatsApp,
			"1234567890@s.whatsapp.net",
		)
		require.NoError(t, err)
		require.NotNil(t, channel)
		assert.Equal(t, waChannel.ID, channel.ID)
	})

	t.Run("find gmail channel", func(t *testing.T) {
		channel, err := db.GetSourceChannelByIdentifier(
			user.ID,
			source.SourceTypeGmail,
			"test@example.com",
		)
		require.NoError(t, err)
		require.NotNil(t, channel)
		assert.Equal(t, gmailChannel.ID, channel.ID)
	})

	t.Run("not found returns nil", func(t *testing.T) {
		channel, err := db.GetSourceChannelByIdentifier(
			user.ID,
			source.SourceTypeWhatsApp,
			"nonexistent@s.whatsapp.net",
		)
		require.NoError(t, err)
		assert.Nil(t, channel)
	})

	t.Run("wrong source type returns nil", func(t *testing.T) {
		// The identifier exists but for a different source type
		channel, err := db.GetSourceChannelByIdentifier(
			user.ID,
			source.SourceTypeTelegram,
			"1234567890@s.whatsapp.net",
		)
		require.NoError(t, err)
		assert.Nil(t, channel)
	})
}

func TestListSourceChannels(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)

	// Create multiple channels for different sources
	_, err := db.CreateSourceChannel(user.ID, source.SourceTypeWhatsApp, source.ChannelTypeSender, "wa1@s.whatsapp.net", "WA 1")
	require.NoError(t, err)
	_, err = db.CreateSourceChannel(user.ID, source.SourceTypeWhatsApp, source.ChannelTypeSender, "wa2@s.whatsapp.net", "WA 2")
	require.NoError(t, err)
	_, err = db.CreateSourceChannel(user.ID, source.SourceTypeTelegram, source.ChannelTypeSender, "tg1", "TG 1")
	require.NoError(t, err)
	_, err = db.CreateSourceChannel(user.ID, source.SourceTypeGmail, source.ChannelTypeSender, "email@test.com", "Gmail 1")
	require.NoError(t, err)

	t.Run("list whatsapp channels", func(t *testing.T) {
		channels, err := db.ListSourceChannels(user.ID, source.SourceTypeWhatsApp)
		require.NoError(t, err)
		assert.Len(t, channels, 2)

		for _, ch := range channels {
			assert.Equal(t, source.SourceTypeWhatsApp, ch.SourceType)
		}
	})

	t.Run("list telegram channels", func(t *testing.T) {
		channels, err := db.ListSourceChannels(user.ID, source.SourceTypeTelegram)
		require.NoError(t, err)
		assert.Len(t, channels, 1)
		assert.Equal(t, "TG 1", channels[0].Name)
	})

	t.Run("list gmail channels", func(t *testing.T) {
		channels, err := db.ListSourceChannels(user.ID, source.SourceTypeGmail)
		require.NoError(t, err)
		assert.Len(t, channels, 1)
	})

	t.Run("empty list for source with no channels", func(t *testing.T) {
		db2 := NewTestDB(t)
		user2 := CreateTestUser(t, db2)
		channels, err := db2.ListSourceChannels(user2.ID, source.SourceTypeWhatsApp)
		require.NoError(t, err)
		assert.Len(t, channels, 0)
	})
}

func TestUpdateSourceChannel(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)

	channel, err := db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"update@s.whatsapp.net",
		"Original Name",
	)
	require.NoError(t, err)
	assert.True(t, channel.Enabled)

	t.Run("update name", func(t *testing.T) {
		err := db.UpdateSourceChannel(user.ID, channel.ID, "Updated Name", true)
		require.NoError(t, err)

		updated, err := db.GetSourceChannelByID(user.ID, channel.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", updated.Name)
		assert.True(t, updated.Enabled)
	})

	t.Run("disable channel", func(t *testing.T) {
		err := db.UpdateSourceChannel(user.ID, channel.ID, "Updated Name", false)
		require.NoError(t, err)

		updated, err := db.GetSourceChannelByID(user.ID, channel.ID)
		require.NoError(t, err)
		assert.False(t, updated.Enabled)
	})

	t.Run("re-enable channel", func(t *testing.T) {
		err := db.UpdateSourceChannel(user.ID, channel.ID, "Re-enabled", true)
		require.NoError(t, err)

		updated, err := db.GetSourceChannelByID(user.ID, channel.ID)
		require.NoError(t, err)
		assert.Equal(t, "Re-enabled", updated.Name)
		assert.True(t, updated.Enabled)
	})
}

func TestDeleteSourceChannel(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)

	channel, err := db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"delete@s.whatsapp.net",
		"To Delete",
	)
	require.NoError(t, err)

	t.Run("delete existing channel", func(t *testing.T) {
		err := db.DeleteSourceChannel(user.ID, channel.ID)
		require.NoError(t, err)

		deleted, err := db.GetSourceChannelByID(user.ID, channel.ID)
		require.NoError(t, err)
		assert.Nil(t, deleted)
	})

	t.Run("delete non-existent channel (no error)", func(t *testing.T) {
		err := db.DeleteSourceChannel(user.ID, 999999)
		require.Error(t, err)
	})
}

func TestIsSourceChannelTracked(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)

	// Create an enabled channel
	enabledCh, err := db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"enabled@s.whatsapp.net",
		"Enabled Channel",
	)
	require.NoError(t, err)

	// Create a disabled channel
	disabledCh, err := db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"disabled@s.whatsapp.net",
		"Disabled Channel",
	)
	require.NoError(t, err)
	err = db.UpdateSourceChannel(user.ID, disabledCh.ID, disabledCh.Name, false)
	require.NoError(t, err)

	t.Run("tracked and enabled", func(t *testing.T) {
		isTracked, channelID, channelType, err := db.IsSourceChannelTracked(
			user.ID,
			source.SourceTypeWhatsApp,
			"enabled@s.whatsapp.net",
		)
		require.NoError(t, err)
		assert.True(t, isTracked)
		assert.Equal(t, enabledCh.ID, channelID)
		assert.Equal(t, source.ChannelTypeSender, channelType)
	})

	t.Run("tracked but disabled returns not tracked", func(t *testing.T) {
		isTracked, channelID, channelType, err := db.IsSourceChannelTracked(
			user.ID,
			source.SourceTypeWhatsApp,
			"disabled@s.whatsapp.net",
		)
		require.NoError(t, err)
		assert.False(t, isTracked)
		assert.Zero(t, channelID)
		assert.Empty(t, channelType)
	})

	t.Run("not tracked at all", func(t *testing.T) {
		isTracked, channelID, channelType, err := db.IsSourceChannelTracked(
			user.ID,
			source.SourceTypeWhatsApp,
			"nonexistent@s.whatsapp.net",
		)
		require.NoError(t, err)
		assert.False(t, isTracked)
		assert.Zero(t, channelID)
		assert.Empty(t, channelType)
	})

	t.Run("wrong source type", func(t *testing.T) {
		isTracked, _, _, err := db.IsSourceChannelTracked(
			user.ID,
			source.SourceTypeTelegram, // Wrong source
			"enabled@s.whatsapp.net",
		)
		require.NoError(t, err)
		assert.False(t, isTracked)
	})
}

func TestSourceChannelToSourceChannel(t *testing.T) {
	// Test the ToSourceChannel conversion method
	sc := &SourceChannel{
		ID:         42,
		SourceType: source.SourceTypeWhatsApp,
		Type:       source.ChannelTypeSender,
		Identifier: "test@s.whatsapp.net",
		Name:       "Test",
		Enabled:    true,
	}

	converted := sc.ToSourceChannel()

	assert.Equal(t, int64(42), converted.ID)
	assert.Equal(t, source.SourceTypeWhatsApp, converted.SourceType)
	assert.Equal(t, source.ChannelTypeSender, converted.Type)
	assert.Equal(t, "test@s.whatsapp.net", converted.Identifier)
	assert.Equal(t, "Test", converted.Name)
	assert.True(t, converted.Enabled)
}

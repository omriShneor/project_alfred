package source

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSourceTypeConstants(t *testing.T) {
	// Verify source type constants are correct
	assert.Equal(t, SourceType("whatsapp"), SourceTypeWhatsApp)
	assert.Equal(t, SourceType("telegram"), SourceTypeTelegram)
	assert.Equal(t, SourceType("gmail"), SourceTypeGmail)
}

func TestChannelTypeConstants(t *testing.T) {
	// Verify channel type constants are correct
	assert.Equal(t, ChannelType("sender"), ChannelTypeSender)
	assert.Equal(t, ChannelType("domain"), ChannelTypeDomain)
	assert.Equal(t, ChannelType("category"), ChannelTypeCategory)
}

func TestMessageToHistoryMessage(t *testing.T) {
	now := time.Now()
	msg := &Message{
		SourceType: SourceTypeWhatsApp,
		SourceID:   100,
		Identifier: "1234567890@s.whatsapp.net",
		SenderID:   "sender123",
		SenderName: "Test Sender",
		Text:       "Hello, world!",
		Subject:    "",
		Timestamp:  now,
	}

	channelID := int64(42)
	historyMsg := msg.ToHistoryMessage(channelID)

	assert.Equal(t, SourceTypeWhatsApp, historyMsg.SourceType)
	assert.Equal(t, channelID, historyMsg.ChannelID)
	assert.Equal(t, "sender123", historyMsg.SenderID)
	assert.Equal(t, "Test Sender", historyMsg.SenderName)
	assert.Equal(t, "Hello, world!", historyMsg.Text)
	assert.Empty(t, historyMsg.Subject)
	assert.Equal(t, now, historyMsg.Timestamp)
}

func TestMessageToHistoryMessage_WithSubject(t *testing.T) {
	now := time.Now()
	msg := &Message{
		SourceType: SourceTypeGmail,
		SourceID:   200,
		Identifier: "sender@example.com",
		SenderID:   "email_sender",
		SenderName: "Email Sender",
		Text:       "Email body content",
		Subject:    "Important Email Subject",
		Timestamp:  now,
	}

	channelID := int64(99)
	historyMsg := msg.ToHistoryMessage(channelID)

	assert.Equal(t, SourceTypeGmail, historyMsg.SourceType)
	assert.Equal(t, channelID, historyMsg.ChannelID)
	assert.Equal(t, "email_sender", historyMsg.SenderID)
	assert.Equal(t, "Email Sender", historyMsg.SenderName)
	assert.Equal(t, "Email body content", historyMsg.Text)
	assert.Equal(t, "Important Email Subject", historyMsg.Subject)
	assert.Equal(t, now, historyMsg.Timestamp)
}

func TestMessage_Fields(t *testing.T) {
	now := time.Now()
	msg := Message{
		SourceType: SourceTypeTelegram,
		SourceID:   300,
		Identifier: "telegram_user_123",
		SenderID:   "tg_sender",
		SenderName: "Telegram User",
		Text:       "Telegram message",
		Subject:    "",
		Timestamp:  now,
	}

	assert.Equal(t, SourceTypeTelegram, msg.SourceType)
	assert.Equal(t, int64(300), msg.SourceID)
	assert.Equal(t, "telegram_user_123", msg.Identifier)
	assert.Equal(t, "tg_sender", msg.SenderID)
	assert.Equal(t, "Telegram User", msg.SenderName)
	assert.Equal(t, "Telegram message", msg.Text)
	assert.Empty(t, msg.Subject)
	assert.Equal(t, now, msg.Timestamp)
}

func TestChannel_Fields(t *testing.T) {
	now := time.Now()
	ch := Channel{
		ID:         42,
		SourceType: SourceTypeWhatsApp,
		Type:       ChannelTypeSender,
		Identifier: "contact@s.whatsapp.net",
		Name:       "Test Contact",
		Enabled:    true,
		CreatedAt:  now,
	}

	assert.Equal(t, int64(42), ch.ID)
	assert.Equal(t, SourceTypeWhatsApp, ch.SourceType)
	assert.Equal(t, ChannelTypeSender, ch.Type)
	assert.Equal(t, "contact@s.whatsapp.net", ch.Identifier)
	assert.Equal(t, "Test Contact", ch.Name)
	assert.True(t, ch.Enabled)
	assert.Equal(t, now, ch.CreatedAt)
}

func TestDiscoverableChannel_Fields(t *testing.T) {
	channelID := int64(100)
	enabled := true

	dc := DiscoverableChannel{
		SourceType: SourceTypeGmail,
		Type:       ChannelTypeDomain,
		Identifier: "example.com",
		Name:       "Example Domain",
		IsTracked:  true,
		ChannelID:  &channelID,
		Enabled:    &enabled,
	}

	assert.Equal(t, SourceTypeGmail, dc.SourceType)
	assert.Equal(t, ChannelTypeDomain, dc.Type)
	assert.Equal(t, "example.com", dc.Identifier)
	assert.Equal(t, "Example Domain", dc.Name)
	assert.True(t, dc.IsTracked)
	assert.NotNil(t, dc.ChannelID)
	assert.Equal(t, int64(100), *dc.ChannelID)
	assert.NotNil(t, dc.Enabled)
	assert.True(t, *dc.Enabled)
}

func TestDiscoverableChannel_Untracked(t *testing.T) {
	dc := DiscoverableChannel{
		SourceType: SourceTypeWhatsApp,
		Type:       ChannelTypeSender,
		Identifier: "untracked@s.whatsapp.net",
		Name:       "Untracked Contact",
		IsTracked:  false,
		ChannelID:  nil,
		Enabled:    nil,
	}

	assert.False(t, dc.IsTracked)
	assert.Nil(t, dc.ChannelID)
	assert.Nil(t, dc.Enabled)
}

func TestHistoryMessage_Fields(t *testing.T) {
	now := time.Now()
	hm := HistoryMessage{
		ID:         1,
		SourceType: SourceTypeTelegram,
		ChannelID:  50,
		SenderID:   "tg_user",
		SenderName: "Telegram User",
		Text:       "History message text",
		Subject:    "",
		Timestamp:  now,
	}

	assert.Equal(t, int64(1), hm.ID)
	assert.Equal(t, SourceTypeTelegram, hm.SourceType)
	assert.Equal(t, int64(50), hm.ChannelID)
	assert.Equal(t, "tg_user", hm.SenderID)
	assert.Equal(t, "Telegram User", hm.SenderName)
	assert.Equal(t, "History message text", hm.Text)
	assert.Empty(t, hm.Subject)
	assert.Equal(t, now, hm.Timestamp)
}

func TestSourceType_StringComparison(t *testing.T) {
	// Test that SourceType can be compared as strings
	var st SourceType = "whatsapp"
	assert.Equal(t, SourceTypeWhatsApp, st)

	st = SourceType("telegram")
	assert.Equal(t, SourceTypeTelegram, st)

	st = SourceType("gmail")
	assert.Equal(t, SourceTypeGmail, st)
}

func TestChannelType_StringComparison(t *testing.T) {
	// Test that ChannelType can be compared as strings
	var ct ChannelType = "sender"
	assert.Equal(t, ChannelTypeSender, ct)

	ct = ChannelType("domain")
	assert.Equal(t, ChannelTypeDomain, ct)

	ct = ChannelType("category")
	assert.Equal(t, ChannelTypeCategory, ct)
}

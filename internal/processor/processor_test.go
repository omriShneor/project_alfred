package processor

import (
	"testing"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/source"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	db := database.NewTestDB(t)
	msgChan := make(chan source.Message)

	t.Run("with valid history size", func(t *testing.T) {
		p := New(db, nil, nil, msgChan, 50, nil)
		assert.NotNil(t, p)
		assert.Equal(t, 50, p.historySize)
	})

	t.Run("with zero history size uses default", func(t *testing.T) {
		p := New(db, nil, nil, msgChan, 0, nil)
		assert.NotNil(t, p)
		assert.Equal(t, defaultHistorySize, p.historySize)
	})

	t.Run("with negative history size uses default", func(t *testing.T) {
		p := New(db, nil, nil, msgChan, -10, nil)
		assert.NotNil(t, p)
		assert.Equal(t, defaultHistorySize, p.historySize)
	})
}

func TestStart_WithoutClaudeClient(t *testing.T) {
	db := database.NewTestDB(t)
	msgChan := make(chan source.Message)

	p := New(db, nil, nil, msgChan, 25, nil) // nil claude client

	err := p.Start()
	assert.NoError(t, err) // Should not error, just disable processor
}

func TestStop(t *testing.T) {
	db := database.NewTestDB(t)
	msgChan := make(chan source.Message)

	p := New(db, nil, nil, msgChan, 25, nil)

	// Just test that Stop doesn't panic
	p.Stop()
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "under limit",
			input:    "short",
			maxLen:   10,
			expected: "short",
		},
		{
			name:     "at limit",
			input:    "exact",
			maxLen:   5,
			expected: "exact",
		},
		{
			name:     "over limit",
			input:    "this is a longer string",
			maxLen:   10,
			expected: "this is a ...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertToMessageRecords(t *testing.T) {
	now := time.Now()
	messages := []database.SourceMessage{
		{
			ID:          1,
			SourceType:  source.SourceTypeWhatsApp,
			ChannelID:   100,
			SenderID:    "sender1@s.whatsapp.net",
			SenderName:  "Sender One",
			MessageText: "First message",
			Timestamp:   now.Add(-time.Hour),
			CreatedAt:   now,
		},
		{
			ID:          2,
			SourceType:  source.SourceTypeWhatsApp,
			ChannelID:   100,
			SenderID:    "sender2@s.whatsapp.net",
			SenderName:  "Sender Two",
			MessageText: "Second message",
			Timestamp:   now,
			CreatedAt:   now,
		},
	}

	records := convertToMessageRecords(messages)

	assert.Len(t, records, 2)

	assert.Equal(t, int64(1), records[0].ID)
	assert.Equal(t, int64(100), records[0].ChannelID)
	assert.Equal(t, "sender1@s.whatsapp.net", records[0].SenderJID)
	assert.Equal(t, "Sender One", records[0].SenderName)
	assert.Equal(t, "First message", records[0].MessageText)

	assert.Equal(t, int64(2), records[1].ID)
	assert.Equal(t, "Sender Two", records[1].SenderName)
	assert.Equal(t, "Second message", records[1].MessageText)
}

func TestConvertSourceMessageToRecord(t *testing.T) {
	now := time.Now()
	msg := &database.SourceMessage{
		ID:          42,
		SourceType:  source.SourceTypeTelegram,
		ChannelID:   200,
		SenderID:    "tg_user_123",
		SenderName:  "Telegram User",
		MessageText: "Hello from Telegram",
		Subject:     "",
		Timestamp:   now,
		CreatedAt:   now,
	}

	record := convertSourceMessageToRecord(msg)

	assert.Equal(t, int64(42), record.ID)
	assert.Equal(t, int64(200), record.ChannelID)
	assert.Equal(t, "tg_user_123", record.SenderJID)
	assert.Equal(t, "Telegram User", record.SenderName)
	assert.Equal(t, "Hello from Telegram", record.MessageText)
	assert.Equal(t, now, record.Timestamp)
}

func TestConvertToMessageRecords_Empty(t *testing.T) {
	records := convertToMessageRecords(nil)
	assert.Len(t, records, 0)

	records = convertToMessageRecords([]database.SourceMessage{})
	assert.Len(t, records, 0)
}

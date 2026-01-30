package source

import "time"

// SourceType identifies the message source
type SourceType string

const (
	SourceTypeWhatsApp SourceType = "whatsapp"
	SourceTypeTelegram SourceType = "telegram"
	SourceTypeGmail    SourceType = "gmail"
)

// ChannelType identifies the type of channel within a source
type ChannelType string

const (
	// WhatsApp/Telegram channel types
	ChannelTypeSender  ChannelType = "sender"
	ChannelTypeGroup   ChannelType = "group"
	ChannelTypeChannel ChannelType = "channel" // Telegram broadcast channels

	// Gmail channel types
	ChannelTypeDomain   ChannelType = "domain"
	ChannelTypeCategory ChannelType = "category"
)

// Message represents a message from any source (WhatsApp, Telegram, Gmail)
type Message struct {
	SourceType  SourceType
	SourceID    int64  // Channel/source database ID
	Identifier  string // WhatsApp JID / Telegram chat ID / email address
	SenderID    string
	SenderName  string
	Text        string
	Subject     string // For emails
	Timestamp   time.Time
	IsGroup     bool
	CalendarID  string // Target calendar for events
}

// Channel represents a tracked source (contact, group, email sender)
type Channel struct {
	ID         int64
	SourceType SourceType
	Type       ChannelType
	Identifier string
	Name       string
	CalendarID string
	Enabled    bool
	CreatedAt  time.Time
}

// DiscoverableChannel represents an available but not-yet-tracked source
type DiscoverableChannel struct {
	SourceType SourceType
	Type       ChannelType
	Identifier string
	Name       string
	IsTracked  bool
	ChannelID  *int64
	Enabled    *bool
}

// HistoryMessage represents a stored message in the history table
type HistoryMessage struct {
	ID         int64
	SourceType SourceType
	ChannelID  int64
	SenderID   string
	SenderName string
	Text       string
	Subject    string // For emails
	Timestamp  time.Time
}

// ToHistoryMessage converts a Message to a HistoryMessage for storage
func (m *Message) ToHistoryMessage(channelID int64) HistoryMessage {
	return HistoryMessage{
		SourceType: m.SourceType,
		ChannelID:  channelID,
		SenderID:   m.SenderID,
		SenderName: m.SenderName,
		Text:       m.Text,
		Subject:    m.Subject,
		Timestamp:  m.Timestamp,
	}
}

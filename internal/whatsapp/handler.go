package whatsapp

import (
	"fmt"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/source"
	"github.com/omriShneor/project_alfred/internal/sse"
	"go.mau.fi/whatsmeow/types/events"
)

// FilteredMessage represents a message from a tracked sender/group
// Deprecated: Use source.Message instead
type FilteredMessage = source.Message

type Handler struct {
	db               *database.DB
	debugAllMessages bool
	messageChan      chan source.Message
	state            *sse.State
}

func NewHandler(db *database.DB, debugAllMessages bool, state *sse.State) *Handler {
	return &Handler{
		db:               db,
		debugAllMessages: debugAllMessages,
		messageChan:      make(chan source.Message, 100),
		state:            state,
	}
}

func (h *Handler) MessageChan() <-chan source.Message {
	return h.messageChan
}

func (h *Handler) HandleEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		h.handleMessage(v)
	}
}

func (h *Handler) handleMessage(msg *events.Message) {
	text := extractText(msg)
	if text == "" {
		return
	}

	// Only process direct messages (contacts), skip groups
	if msg.Info.IsGroup {
		return
	}

	sender := msg.Info.Sender
	identifier := sender.User

	var sourceID int64
	var tracked bool
	var err error
	var calendarID string

	if h.debugAllMessages {
		tracked = true
		identifier = "debug"
	} else {
		var channelType source.ChannelType
		tracked, sourceID, channelType, err = h.db.IsSourceChannelTracked(source.SourceTypeWhatsApp, identifier)
		if err != nil {
			fmt.Printf("Error checking channel: %v\n", err)
			return
		}
		_ = channelType // channelType used for logging if needed

		// Get calendar ID from channel
		if tracked {
			channel, err := h.db.GetSourceChannelByID(sourceID)
			if err == nil && channel != nil {
				calendarID = channel.CalendarID
			}
		}
	}

	if !tracked {
		return
	}

	// Log to stdout
	fmt.Printf("[WhatsApp DM: %s] %s\n", sender.User, text)

	// Send to channel for assistant processing
	select {
	case h.messageChan <- source.Message{
		SourceType: source.SourceTypeWhatsApp,
		SourceID:   sourceID,
		Identifier: identifier,
		SenderID:   sender.String(),
		SenderName: sender.User,
		Text:       text,
		Timestamp:  msg.Info.Timestamp,
		CalendarID: calendarID,
	}:
	default:
		fmt.Println("Warning: message channel full, dropping message")
	}
}

func extractText(msg *events.Message) string {
	m := msg.Message

	if m.GetConversation() != "" {
		return m.GetConversation()
	}

	if ext := m.GetExtendedTextMessage(); ext != nil {
		return ext.GetText()
	}

	if img := m.GetImageMessage(); img != nil && img.GetCaption() != "" {
		return "[Image] " + img.GetCaption()
	}

	if vid := m.GetVideoMessage(); vid != nil && vid.GetCaption() != "" {
		return "[Video] " + vid.GetCaption()
	}

	if doc := m.GetDocumentMessage(); doc != nil {
		return "[Document] " + doc.GetFileName()
	}

	return ""
}

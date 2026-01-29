package whatsapp

import (
	"fmt"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/sse"
	"go.mau.fi/whatsmeow/types/events"
)

// FilteredMessage represents a message from a tracked sender/group
type FilteredMessage struct {
	SourceType string // "sender" or "group"
	SourceID   int64
	SenderJID  string
	SenderName string
	Text       string
	IsGroup    bool
	Timestamp  time.Time
}

type Handler struct {
	db               *database.DB
	debugAllMessages bool
	messageChan      chan FilteredMessage
	state            *sse.State
}

func NewHandler(db *database.DB, debugAllMessages bool, state *sse.State) *Handler {
	return &Handler{
		db:               db,
		debugAllMessages: debugAllMessages,
		messageChan:      make(chan FilteredMessage, 100),
		state:            state,
	}
}

func (h *Handler) MessageChan() <-chan FilteredMessage {
	return h.messageChan
}

func (h *Handler) HandleEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		h.handleMessage(v)
	case *events.PairSuccess:
		fmt.Printf("WhatsApp paired successfully! JID: %s\n", v.ID)
		// PairSuccess is followed by a websocket reconnect, wait for Connected
	case *events.Connected:
		fmt.Println("WhatsApp connected!")
		if h.state != nil {
			h.state.SetWhatsAppStatus("connected")
		}
	}
}

func (h *Handler) handleMessage(msg *events.Message) {
	text := extractText(msg)
	if text == "" {
		return
	}

	sender := msg.Info.Sender
	chat := msg.Info.Chat
	isGroup := msg.Info.IsGroup

	var sourceType string
	var sourceID int64
	var tracked bool
	var err error

	if h.debugAllMessages {
		tracked = true
		sourceType = "debug"
	} else {
		var identifier string
		if isGroup {
			identifier = chat.String()
		} else {
			identifier = sender.User
		}

		var channelType database.ChannelType
		tracked, sourceID, channelType, err = h.db.IsChannelTracked(identifier)
		if err != nil {
			fmt.Printf("Error checking channel: %v\n", err)
			return
		}
		sourceType = string(channelType)
	}

	if !tracked {
		return
	}

	// Log to stdout (not DB)
	if isGroup {
		fmt.Printf("[GROUP: %s] %s: %s\n", chat.String(), sender.User, text)
	} else {
		fmt.Printf("[DM: %s] %s\n", sender.User, text)
	}

	// Send to channel for assistant processing
	select {
	case h.messageChan <- FilteredMessage{
		SourceType: sourceType,
		SourceID:   sourceID,
		SenderJID:  sender.String(),
		SenderName: sender.User,
		Text:       text,
		IsGroup:    isGroup,
		Timestamp:  msg.Info.Timestamp,
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

package telegram

import (
	"fmt"
	"time"

	"github.com/gotd/td/tg"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/source"
	"github.com/omriShneor/project_alfred/internal/sse"
)

// Handler processes incoming Telegram messages (contacts only)
type Handler struct {
	UserID           int64 // User who owns this handler (for multi-user support)
	db               *database.DB
	debugAllMessages bool
	messageChan      chan source.Message
	state            *sse.State
	users            map[int64]*tg.User // Cache of user info
}

// NewHandler creates a handler for a specific user (multi-user mode)
func NewHandler(userID int64, db *database.DB) *Handler {
	return &Handler{
		UserID:           userID,
		db:               db,
		debugAllMessages: false,
		messageChan:      make(chan source.Message, 100),
		state:            nil,
		users:            make(map[int64]*tg.User),
	}
}

// SetMessageChannel allows ClientManager to override the message channel
// with a shared channel for multi-user support
func (h *Handler) SetMessageChannel(ch chan source.Message) {
	h.messageChan = ch
}

// MessageChan returns the channel for receiving filtered messages
func (h *Handler) MessageChan() <-chan source.Message {
	return h.messageChan
}

// HandleUpdate processes a Telegram update
func (h *Handler) HandleUpdate(update tg.UpdatesClass) {
	switch u := update.(type) {
	case *tg.Updates:
		h.cacheEntities(u.Users, u.Chats)
		for _, upd := range u.Updates {
			h.handleSingleUpdate(upd)
		}
	case *tg.UpdatesCombined:
		h.cacheEntities(u.Users, u.Chats)
		for _, upd := range u.Updates {
			h.handleSingleUpdate(upd)
		}
	case *tg.UpdateShort:
		h.handleSingleUpdate(u.Update)
	case *tg.UpdateShortMessage:
		h.handleShortMessage(u)
	case *tg.UpdateShortChatMessage:
		// Group messages not supported - only contacts
		return
	}
}

// cacheEntities caches user information
func (h *Handler) cacheEntities(users []tg.UserClass, chats []tg.ChatClass) {
	for _, u := range users {
		if user, ok := u.(*tg.User); ok {
			h.users[user.ID] = user
		}
	}
	// Note: chats/channels not cached as we only support contacts
	_ = chats
}

// handleSingleUpdate processes a single update
func (h *Handler) handleSingleUpdate(update tg.UpdateClass) {
	switch msg := update.(type) {
	case *tg.UpdateNewMessage:
		h.handleNewMessage(msg.Message)
	case *tg.UpdateNewChannelMessage:
		h.handleNewMessage(msg.Message)
	}
}

// handleNewMessage processes a new message (contacts only)
func (h *Handler) handleNewMessage(msg tg.MessageClass) {
	message, ok := msg.(*tg.Message)
	if !ok {
		return
	}

	text := message.Message
	if text == "" {
		return
	}

	// Only process direct messages from users (contacts), skip groups/channels
	peer, ok := message.PeerID.(*tg.PeerUser)
	if !ok {
		return
	}

	chatIdentifier := fmt.Sprintf("%d", peer.UserID)
	var senderID string
	var senderName string

	if user, ok := h.users[peer.UserID]; ok {
		senderName = getUserName(user)
		senderID = fmt.Sprintf("%d", user.ID)
	} else {
		senderName = fmt.Sprintf("User %d", peer.UserID)
		senderID = chatIdentifier
	}

	// Check if tracked
	var sourceID int64
	var tracked bool

	if h.debugAllMessages {
		tracked = true
	} else {
		var channelType source.ChannelType
		var err error
		tracked, sourceID, channelType, err = h.db.IsSourceChannelTracked(h.UserID, source.SourceTypeTelegram, chatIdentifier)
		if err != nil {
			fmt.Printf("Telegram: Error checking channel: %v\n", err)
			return
		}
		_ = channelType
	}

	if !tracked {
		return
	}

	// Log message
	fmt.Printf("[Telegram DM: %s] %s\n", senderName, truncateText(text, 100))

	// Send to processor (blocking for reliability).
	h.messageChan <- source.Message{
		UserID:     h.UserID,
		SourceType: source.SourceTypeTelegram,
		SourceID:   sourceID,
		Identifier: chatIdentifier,
		SenderID:   senderID,
		SenderName: senderName,
		Text:       text,
		Timestamp:  time.Unix(int64(message.Date), 0),
	}
}

// handleShortMessage processes a short direct message update
func (h *Handler) handleShortMessage(msg *tg.UpdateShortMessage) {
	if msg.Message == "" {
		return
	}

	chatIdentifier := fmt.Sprintf("%d", msg.UserID)
	senderID := chatIdentifier
	senderName := fmt.Sprintf("User %d", msg.UserID)

	if user, ok := h.users[msg.UserID]; ok {
		senderName = getUserName(user)
	}

	// Check if tracked
	var sourceID int64
	var tracked bool

	if h.debugAllMessages {
		tracked = true
	} else {
		var err error
		tracked, sourceID, _, err = h.db.IsSourceChannelTracked(h.UserID, source.SourceTypeTelegram, chatIdentifier)
		if err != nil {
			fmt.Printf("Telegram: Error checking channel: %v\n", err)
			return
		}
	}

	if !tracked {
		return
	}

	fmt.Printf("[Telegram DM: %s] %s\n", senderName, truncateText(msg.Message, 100))

	h.messageChan <- source.Message{
		UserID:     h.UserID,
		SourceType: source.SourceTypeTelegram,
		SourceID:   sourceID,
		Identifier: chatIdentifier,
		SenderID:   senderID,
		SenderName: senderName,
		Text:       msg.Message,
		Timestamp:  time.Unix(int64(msg.Date), 0),
	}
}

// getUserName returns a display name for a user
func getUserName(user *tg.User) string {
	if user.FirstName != "" {
		if user.LastName != "" {
			return user.FirstName + " " + user.LastName
		}
		return user.FirstName
	}
	if user.Username != "" {
		return "@" + user.Username
	}
	return fmt.Sprintf("User %d", user.ID)
}

// truncateText shortens text for logging
func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

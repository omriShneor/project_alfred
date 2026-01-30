package telegram

import (
	"fmt"
	"time"

	"github.com/gotd/td/tg"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/source"
	"github.com/omriShneor/project_alfred/internal/sse"
)

// Handler processes incoming Telegram messages
type Handler struct {
	db               *database.DB
	debugAllMessages bool
	messageChan      chan source.Message
	state            *sse.State
	users            map[int64]*tg.User  // Cache of user info
	chats            map[int64]*tg.Chat  // Cache of chat info
	channels         map[int64]*tg.Channel // Cache of channel info
}

// NewHandler creates a new Telegram message handler
func NewHandler(db *database.DB, debugAllMessages bool, state *sse.State) *Handler {
	return &Handler{
		db:               db,
		debugAllMessages: debugAllMessages,
		messageChan:      make(chan source.Message, 100),
		state:            state,
		users:            make(map[int64]*tg.User),
		chats:            make(map[int64]*tg.Chat),
		channels:         make(map[int64]*tg.Channel),
	}
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
		h.handleShortChatMessage(u)
	}
}

// cacheEntities caches user and chat information
func (h *Handler) cacheEntities(users []tg.UserClass, chats []tg.ChatClass) {
	for _, u := range users {
		if user, ok := u.(*tg.User); ok {
			h.users[user.ID] = user
		}
	}
	for _, c := range chats {
		switch chat := c.(type) {
		case *tg.Chat:
			h.chats[chat.ID] = chat
		case *tg.Channel:
			h.channels[chat.ID] = chat
		}
	}
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

// handleNewMessage processes a new message
func (h *Handler) handleNewMessage(msg tg.MessageClass) {
	message, ok := msg.(*tg.Message)
	if !ok {
		return
	}

	text := message.Message
	if text == "" {
		return
	}

	// Determine sender and chat info
	var senderID string
	var senderName string
	var chatIdentifier string
	var isGroup bool

	// Get peer information
	switch peer := message.PeerID.(type) {
	case *tg.PeerUser:
		// Direct message
		chatIdentifier = fmt.Sprintf("%d", peer.UserID)
		isGroup = false
		if user, ok := h.users[peer.UserID]; ok {
			senderName = getUserName(user)
			senderID = fmt.Sprintf("%d", user.ID)
		} else {
			senderName = fmt.Sprintf("User %d", peer.UserID)
			senderID = chatIdentifier
		}
	case *tg.PeerChat:
		// Group chat
		chatIdentifier = fmt.Sprintf("chat_%d", peer.ChatID)
		isGroup = true
		if chat, ok := h.chats[peer.ChatID]; ok {
			senderName = chat.Title
		} else {
			senderName = fmt.Sprintf("Chat %d", peer.ChatID)
		}
		// Get actual sender from FromID
		if fromID, ok := message.FromID.(*tg.PeerUser); ok {
			senderID = fmt.Sprintf("%d", fromID.UserID)
			if user, ok := h.users[fromID.UserID]; ok {
				senderName = getUserName(user)
			}
		}
	case *tg.PeerChannel:
		// Channel or supergroup
		chatIdentifier = fmt.Sprintf("channel_%d", peer.ChannelID)
		isGroup = true
		if channel, ok := h.channels[peer.ChannelID]; ok {
			senderName = channel.Title
		} else {
			senderName = fmt.Sprintf("Channel %d", peer.ChannelID)
		}
		// Get actual sender from FromID
		if fromID, ok := message.FromID.(*tg.PeerUser); ok {
			senderID = fmt.Sprintf("%d", fromID.UserID)
			if user, ok := h.users[fromID.UserID]; ok {
				senderName = getUserName(user)
			}
		}
	default:
		return
	}

	// Check if tracked
	var sourceID int64
	var tracked bool
	var calendarID string

	if h.debugAllMessages {
		tracked = true
	} else {
		var channelType source.ChannelType
		var err error
		tracked, sourceID, channelType, err = h.db.IsSourceChannelTracked(source.SourceTypeTelegram, chatIdentifier)
		if err != nil {
			fmt.Printf("Telegram: Error checking channel: %v\n", err)
			return
		}
		_ = channelType

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

	// Log message
	if isGroup {
		fmt.Printf("[Telegram GROUP: %s] %s: %s\n", chatIdentifier, senderName, truncateText(text, 100))
	} else {
		fmt.Printf("[Telegram DM: %s] %s\n", senderName, truncateText(text, 100))
	}

	// Send to processor
	select {
	case h.messageChan <- source.Message{
		SourceType: source.SourceTypeTelegram,
		SourceID:   sourceID,
		Identifier: chatIdentifier,
		SenderID:   senderID,
		SenderName: senderName,
		Text:       text,
		IsGroup:    isGroup,
		Timestamp:  time.Unix(int64(message.Date), 0),
		CalendarID: calendarID,
	}:
	default:
		fmt.Println("Telegram: message channel full, dropping message")
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
	var calendarID string

	if h.debugAllMessages {
		tracked = true
	} else {
		var err error
		tracked, sourceID, _, err = h.db.IsSourceChannelTracked(source.SourceTypeTelegram, chatIdentifier)
		if err != nil {
			fmt.Printf("Telegram: Error checking channel: %v\n", err)
			return
		}

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

	fmt.Printf("[Telegram DM: %s] %s\n", senderName, truncateText(msg.Message, 100))

	select {
	case h.messageChan <- source.Message{
		SourceType: source.SourceTypeTelegram,
		SourceID:   sourceID,
		Identifier: chatIdentifier,
		SenderID:   senderID,
		SenderName: senderName,
		Text:       msg.Message,
		IsGroup:    false,
		Timestamp:  time.Unix(int64(msg.Date), 0),
		CalendarID: calendarID,
	}:
	default:
		fmt.Println("Telegram: message channel full, dropping message")
	}
}

// handleShortChatMessage processes a short group message update
func (h *Handler) handleShortChatMessage(msg *tg.UpdateShortChatMessage) {
	if msg.Message == "" {
		return
	}

	chatIdentifier := fmt.Sprintf("chat_%d", msg.ChatID)
	senderID := fmt.Sprintf("%d", msg.FromID)
	senderName := fmt.Sprintf("User %d", msg.FromID)

	if user, ok := h.users[msg.FromID]; ok {
		senderName = getUserName(user)
	}

	// Check if tracked
	var sourceID int64
	var tracked bool
	var calendarID string

	if h.debugAllMessages {
		tracked = true
	} else {
		var err error
		tracked, sourceID, _, err = h.db.IsSourceChannelTracked(source.SourceTypeTelegram, chatIdentifier)
		if err != nil {
			fmt.Printf("Telegram: Error checking channel: %v\n", err)
			return
		}

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

	fmt.Printf("[Telegram GROUP: %s] %s: %s\n", chatIdentifier, senderName, truncateText(msg.Message, 100))

	select {
	case h.messageChan <- source.Message{
		SourceType: source.SourceTypeTelegram,
		SourceID:   sourceID,
		Identifier: chatIdentifier,
		SenderID:   senderID,
		SenderName: senderName,
		Text:       msg.Message,
		IsGroup:    true,
		Timestamp:  time.Unix(int64(msg.Date), 0),
		CalendarID: calendarID,
	}:
	default:
		fmt.Println("Telegram: message channel full, dropping message")
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

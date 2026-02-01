package whatsapp

import (
	"context"
	"fmt"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/source"
	"github.com/omriShneor/project_alfred/internal/sse"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
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
	wClient          *whatsmeow.Client // For ParseWebMessage in history sync
}

func NewHandler(db *database.DB, debugAllMessages bool, state *sse.State) *Handler {
	return &Handler{
		db:               db,
		debugAllMessages: debugAllMessages,
		messageChan:      make(chan source.Message, 100),
		state:            state,
		wClient:          nil,
	}
}

// SetClient sets the WhatsApp client reference (needed for history sync)
func (h *Handler) SetClient(client *whatsmeow.Client) {
	h.wClient = client
}

func (h *Handler) MessageChan() <-chan source.Message {
	return h.messageChan
}

func (h *Handler) HandleEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		h.handleMessage(v)
	case *events.HistorySync:
		go h.handleHistorySync(v) // Run async to not block event handler
	case *events.AppStateSyncComplete:
		h.handleAppStateSyncComplete(v) // Fallback for contact name sync
	}
}

// handleAppStateSyncComplete handles contact sync events as fallback/additional sync
// From GitHub issue #583:
//   - critical_block: PushNames of recently messaged users
//   - critical_unblock_low: Full contact list
func (h *Handler) handleAppStateSyncComplete(evt *events.AppStateSyncComplete) {
	switch evt.Name {
	case appstate.WAPatchCriticalBlock:
		fmt.Println("AppStateSyncComplete: (critical_block)")
		go h.refreshTopContactNames()

	case appstate.WAPatchCriticalUnblockLow:
		fmt.Println("AppStateSyncComplete: (critical_unblock_low)")
		go h.refreshAllContactNames()
	}
}

func (h *Handler) forceSyncContactsAndRefresh() {
	if h.wClient == nil {
		fmt.Println("ForceSyncContacts: WhatsApp client not available")
		return
	}

	fmt.Println("ForceSyncContacts: Attempting to sync contact list via FetchAppState")

	err := h.wClient.FetchAppState(context.Background(), appstate.WAPatchCriticalUnblockLow, true, false)
	if err != nil {
		fmt.Printf("ForceSyncContacts: FetchAppState failed: %v (will rely on events)\n", err)
	} else {
		fmt.Println("ForceSyncContacts: FetchAppState succeeded")
	}

	// Small delay to let the store populate
	time.Sleep(500 * time.Millisecond)

	// Phase 1: Refresh top senders first (for fast Add Contact modal)
	h.refreshTopContactNames()

	// Phase 2: Refresh all remaining contacts
	h.refreshAllContactNames()
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

// HistorySync handling - populates message_history from WhatsApp's history sync

const maxHistoryMessagesPerContact = 25

// senderInfo tracks accurate message stats for a sender during HistorySync
type senderInfo struct {
	identifier    string
	jid           types.JID
	messageCount  int
	lastMessageAt *time.Time
}

// handleHistorySync processes the HistorySync event from WhatsApp
// This runs in a goroutine to not block the event handler
func (h *Handler) handleHistorySync(evt *events.HistorySync) {
	if h.wClient == nil {
		fmt.Println("HistorySync: WhatsApp client not set, skipping")
		return
	}

	conversations := evt.Data.GetConversations()
	fmt.Printf("HistorySync: Processing %d conversations\n", len(conversations))

	// Track ACCURATE message counts per sender (not limited to 25)
	senderStats := make(map[string]*senderInfo)

	processedContacts := 0
	for _, conv := range conversations {
		chatJID, err := types.ParseJID(conv.GetID())
		if err != nil {
			fmt.Printf("HistorySync: Failed to parse JID %s: %v\n", conv.GetID(), err)
			continue
		}

		// Only process direct messages (contacts), skip groups
		if chatJID.Server != "s.whatsapp.net" {
			continue
		}

		messages := conv.GetMessages()
		if len(messages) == 0 {
			continue
		}

		identifier := chatJID.User

		if _, exists := senderStats[identifier]; !exists {
			senderStats[identifier] = &senderInfo{
				identifier:   identifier,
				jid:          chatJID,
				messageCount: 0,
			}
		}
		senderStats[identifier].messageCount += len(messages)

		if len(messages) > 0 {
			lastMsg := messages[0]
			if parsedEvt, err := h.wClient.ParseWebMessage(chatJID, lastMsg.GetMessage()); err == nil {
				ts := parsedEvt.Info.Timestamp
				// Only update if this is more recent
				if senderStats[identifier].lastMessageAt == nil || ts.After(*senderStats[identifier].lastMessageAt) {
					senderStats[identifier].lastMessageAt = &ts
				}
			}
		}

		if h.processConversationHistory(chatJID, messages) {
			processedContacts++
		}
	}

	fmt.Printf("HistorySync: Found %d unique senders\n", len(senderStats))

	statsUpdated := 0
	for identifier, info := range senderStats {
		channel, err := h.db.GetSourceChannelByIdentifier(source.SourceTypeWhatsApp, identifier)
		if err != nil || channel == nil {
			continue
		}

		if err := h.db.UpdateChannelStats(channel.ID, info.messageCount, info.lastMessageAt); err != nil {
			fmt.Printf("HistorySync: Failed to update stats for %s: %v\n", identifier, err)
			continue
		}
		statsUpdated++
	}

	fmt.Printf("HistorySync: Completed - processed %d contacts, updated stats for %d\n", processedContacts, statsUpdated)

	// Force sync contacts and refresh names immediately
	// This uses FetchAppState to get contacts ASAP, with AppStateSyncComplete as fallback
	go h.forceSyncContactsAndRefresh()
}

// processConversationHistory stores messages from a single conversation
func (h *Handler) processConversationHistory(chatJID types.JID, messages []*waProto.HistorySyncMsg) bool {
	identifier := chatJID.User

	// Get or create channel (disabled by default for history sync contacts)
	channel, err := h.getOrCreateHistoryChannel(identifier, chatJID)
	if err != nil {
		fmt.Printf("HistorySync: Failed to get/create channel for %s: %v\n", identifier, err)
		return false
	}

	// Process messages (limit to maxHistoryMessagesPerContact)
	processed := 0
	for _, historyMsg := range messages {
		if processed >= maxHistoryMessagesPerContact {
			break
		}

		evt, err := h.wClient.ParseWebMessage(chatJID, historyMsg.GetMessage())
		if err != nil {
			continue // Skip messages that can't be parsed
		}

		text := extractText(evt)
		if text == "" {
			continue
		}

		// Get sender info
		senderID := evt.Info.Sender.String()
		senderName := evt.Info.Sender.User
		if evt.Info.PushName != "" {
			senderName = evt.Info.PushName
		}

		// Store message
		_, err = h.db.StoreSourceMessage(
			source.SourceTypeWhatsApp,
			channel.ID,
			senderID,
			senderName,
			text,
			"", // no subject for WhatsApp
			evt.Info.Timestamp,
		)
		if err != nil {
			// Log but continue - might be duplicate
			continue
		}

		processed++
	}

	// Prune to keep only last N messages
	if processed > 0 {
		if err := h.db.PruneSourceMessages(source.SourceTypeWhatsApp, channel.ID, maxHistoryMessagesPerContact); err != nil {
			fmt.Printf("HistorySync: Failed to prune messages for %s: %v\n", identifier, err)
		}
		fmt.Printf("HistorySync: Stored %d messages for %s\n", processed, identifier)
	}

	return processed > 0
}

// getOrCreateHistoryChannel gets an existing channel or creates a new disabled one
func (h *Handler) getOrCreateHistoryChannel(identifier string, jid types.JID) (*database.SourceChannel, error) {
	// Check if channel exists
	channel, err := h.db.GetSourceChannelByIdentifier(source.SourceTypeWhatsApp, identifier)
	if err != nil {
		return nil, err
	}

	if channel != nil {
		return channel, nil
	}

	// Get contact name from store if available
	name := identifier
	if h.wClient != nil && h.wClient.Store != nil && h.wClient.Store.Contacts != nil {
		contact, err := h.wClient.Store.Contacts.GetContact(context.Background(), jid)
		if err == nil {
			if contact.FullName != "" {
				name = contact.FullName
			} else if contact.PushName != "" {
				name = contact.PushName
			}
		}
	}

	// Create new channel (disabled by default - user must enable to track events)
	channel, err = h.db.CreateSourceChannel(
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		identifier,
		name,
	)
	if err != nil {
		return nil, err
	}

	// Disable the channel - it's just for discovery, not event tracking
	if err := h.db.UpdateSourceChannel(channel.ID, channel.Name, false); err != nil {
		fmt.Printf("HistorySync: Warning - failed to disable new channel: %v\n", err)
	}

	return channel, nil
}

// Contact name refresh after HistorySync

// refreshTopContactNames updates names for top 8 contacts by ACTUAL message count
// This runs immediately after HistorySync to ensure the Add Source modal shows names ASAP
func (h *Handler) refreshTopContactNames() {
	if h.wClient == nil || h.wClient.Store == nil || h.wClient.Store.Contacts == nil {
		return
	}

	// Get all contacts from WhatsApp store (batch lookup)
	allContacts, err := h.wClient.Store.Contacts.GetAllContacts(context.Background())
	if err != nil {
		fmt.Printf("RefreshTopNames: Failed to get contacts: %v\n", err)
		return
	}

	// Get top 8 contacts by ACTUAL message count (from channels.total_message_count)
	topChannels, err := h.db.GetTopChannelsByMessageCount(source.SourceTypeWhatsApp, 8)
	if err != nil {
		fmt.Printf("RefreshTopNames: Failed to get top channels: %v\n", err)
		return
	}

	updated := 0
	for _, channel := range topChannels {
		// Skip if already has a real name (name != identifier)
		if channel.Name != channel.Identifier {
			continue
		}

		// Look up in contacts map
		jid, err := types.ParseJID(channel.Identifier + "@s.whatsapp.net")
		if err != nil {
			continue
		}

		if contact, ok := allContacts[jid]; ok {
			name := contact.FullName
			if name == "" {
				name = contact.PushName
			}
			if name != "" && name != channel.Identifier {
				if err := h.db.UpdateSourceChannel(channel.ID, name, channel.Enabled); err != nil {
					continue
				}
				updated++
			}
		}
	}

	if updated > 0 {
		fmt.Printf("RefreshTopNames: Updated %d top contact names\n", updated)
	}
}

// refreshAllContactNames updates names for all contacts missing names
// This runs in background after top contacts are done
func (h *Handler) refreshAllContactNames() {
	if h.wClient == nil || h.wClient.Store == nil || h.wClient.Store.Contacts == nil {
		return
	}

	// Get all contacts from WhatsApp store (batch lookup)
	allContacts, err := h.wClient.Store.Contacts.GetAllContacts(context.Background())
	if err != nil {
		fmt.Printf("RefreshAllNames: Failed to get contacts: %v\n", err)
		return
	}

	// Get all WhatsApp channels
	channels, err := h.db.ListSourceChannels(source.SourceTypeWhatsApp)
	if err != nil {
		fmt.Printf("RefreshAllNames: Failed to list channels: %v\n", err)
		return
	}

	updated := 0
	for _, channel := range channels {
		// Skip if already has a real name
		if channel.Name != channel.Identifier {
			continue
		}

		// Look up in contacts map
		jid, err := types.ParseJID(channel.Identifier + "@s.whatsapp.net")
		if err != nil {
			continue
		}

		if contact, ok := allContacts[jid]; ok {
			name := contact.FullName
			if name == "" {
				name = contact.PushName
			}
			if name != "" && name != channel.Identifier {
				if err := h.db.UpdateSourceChannel(channel.ID, name, channel.Enabled); err != nil {
					continue
				}
				updated++
			}
		}
	}

	if updated > 0 {
		fmt.Printf("RefreshAllNames: Updated %d contact names\n", updated)
	}
}

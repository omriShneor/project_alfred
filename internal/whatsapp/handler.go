package whatsapp

import (
	"context"
	"fmt"
	"sync"
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

// HistorySyncBackfillHook is invoked after HistorySync has stored messages for enabled channels.
type HistorySyncBackfillHook func(userID int64, channel *database.SourceChannel)

type Handler struct {
	UserID           int64 // User who owns this handler (for multi-user support)
	db               *database.DB
	debugAllMessages bool
	messageChan      chan source.Message
	state            *sse.State
	wClient          *whatsmeow.Client // For ParseWebMessage in history sync

	historySyncMu               sync.Mutex
	historySyncBackfillHook     HistorySyncBackfillHook
	historySyncPendingBackfill  map[int64]*database.SourceChannel
	historySyncBackfillDebounce *time.Timer
}

func NewHandler(userID int64, db *database.DB, debugAllMessages bool, state *sse.State) *Handler {
	return &Handler{
		UserID:                     userID,
		db:                         db,
		debugAllMessages:           debugAllMessages,
		messageChan:                make(chan source.Message, 100),
		state:                      state,
		wClient:                    nil,
		historySyncPendingBackfill: make(map[int64]*database.SourceChannel),
	}
}

// SetClient sets the WhatsApp client reference (needed for history sync)
func (h *Handler) SetClient(client *whatsmeow.Client) {
	h.wClient = client
}

// SetMessageChannel allows ClientManager to override the message channel
// with a shared channel for multi-user support
func (h *Handler) SetMessageChannel(ch chan source.Message) {
	h.messageChan = ch
}

// SetHistorySyncBackfillHook configures an optional callback that runs backfill
// after HistorySync has persisted messages for enabled channels.
func (h *Handler) SetHistorySyncBackfillHook(hook HistorySyncBackfillHook) {
	h.historySyncMu.Lock()
	defer h.historySyncMu.Unlock()
	h.historySyncBackfillHook = hook
}

func (h *Handler) MessageChan() <-chan source.Message {
	return h.messageChan
}

func (h *Handler) HandleEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Connected:
		h.handleConnected(v)
	case *events.Disconnected:
		h.handleDisconnected(v)
	case *events.LoggedOut:
		h.handleLoggedOut(v)
	case *events.Message:
		h.handleMessage(v)
	case *events.HistorySync:
		go h.handleHistorySync(v) // Run async to not block event handler
	case *events.AppStateSyncComplete:
		h.handleAppStateSyncComplete(v) // Fallback for contact name sync
	}
}

func (h *Handler) handleConnected(_ *events.Connected) {
	if h.db == nil || h.UserID == 0 {
		return
	}

	deviceJID := ""
	if h.wClient != nil && h.wClient.Store != nil && h.wClient.Store.ID != nil {
		deviceJID = h.wClient.Store.ID.String()
	}

	if err := h.db.SaveWhatsAppSession(h.UserID, "", deviceJID, true); err != nil {
		fmt.Printf("WhatsApp: failed to save connected session for user %d: %v\n", h.UserID, err)
	}
}

func (h *Handler) handleDisconnected(_ *events.Disconnected) {
	if h.db == nil || h.UserID == 0 {
		return
	}

	if err := h.db.UpdateWhatsAppConnected(h.UserID, false); err != nil {
		fmt.Printf("WhatsApp: failed to mark disconnected session for user %d: %v\n", h.UserID, err)
	}
}

func (h *Handler) handleLoggedOut(_ *events.LoggedOut) {
	if h.db == nil || h.UserID == 0 {
		return
	}

	if err := h.db.UpdateWhatsAppConnected(h.UserID, false); err != nil {
		fmt.Printf("WhatsApp: failed to mark logged-out session for user %d: %v\n", h.UserID, err)
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
		tracked, sourceID, channelType, err = h.db.IsSourceChannelTracked(h.UserID, source.SourceTypeWhatsApp, identifier)
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

	// Send to channel for assistant processing (blocking for reliability).
	h.messageChan <- source.Message{
		UserID:     h.UserID,
		SourceType: source.SourceTypeWhatsApp,
		SourceID:   sourceID,
		Identifier: identifier,
		SenderID:   sender.String(),
		SenderName: sender.User,
		Text:       text,
		Timestamp:  msg.Info.Timestamp,
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

var historySyncBackfillDebounce = 2 * time.Second

// senderInfo tracks accurate message stats for a sender during HistorySync
type senderInfo struct {
	identifier    string
	jid           types.JID
	messageCount  int
	lastMessageAt *time.Time
}

type historyConversationWork struct {
	identifier string
	chatJID    types.JID
	messages   []*waProto.HistorySyncMsg
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
	workItems := make([]historyConversationWork, 0, len(conversations))

	// Phase 1: fast metadata pass.
	// Build sender stats across all conversations so top contacts can be shown
	// while message history is still being processed.
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

		workItems = append(workItems, historyConversationWork{
			identifier: identifier,
			chatJID:    chatJID,
			messages:   messages,
		})
	}

	fmt.Printf("HistorySync: Found %d unique senders\n", len(senderStats))

	// Prime channels + top-contact stats from metadata before message processing.
	// This is intentionally best-effort/inaccurate while sync is still running.
	channelsByIdentifier := make(map[string]*database.SourceChannel, len(senderStats))
	statsUpdated := 0
	for identifier, info := range senderStats {
		channel, err := h.getOrCreateHistoryChannel(identifier, info.jid)
		if err != nil {
			fmt.Printf("HistorySync: Failed to get/create channel for %s: %v\n", identifier, err)
			continue
		}
		channelsByIdentifier[identifier] = channel

		if err := h.db.UpdateChannelStats(channel.ID, info.messageCount, info.lastMessageAt); err != nil {
			fmt.Printf("HistorySync: Failed to update in-progress stats for %s: %v\n", identifier, err)
			continue
		}
		statsUpdated++
	}

	fmt.Printf("HistorySync: Primed top-contact stats for %d senders\n", statsUpdated)

	// Phase 2: store message history.
	processedContacts := 0
	processedChannels := make(map[int64]*database.SourceChannel)
	for _, item := range workItems {
		channel := channelsByIdentifier[item.identifier]
		if channel == nil {
			fallbackChannel, err := h.getOrCreateHistoryChannel(item.identifier, item.chatJID)
			if err != nil {
				fmt.Printf("HistorySync: Failed to resolve channel during message processing for %s: %v\n", item.identifier, err)
				continue
			}
			channel = fallbackChannel
			channelsByIdentifier[item.identifier] = channel
		}

		if h.processConversationHistory(channel, item.chatJID, item.messages) {
			processedContacts++
			processedChannels[channel.ID] = channel
		}
	}

	// Finalize top-contact stats after phase 2 so ranking reflects the full HistorySync snapshot.
	// This guarantees we end in a consistent "most accurate available" state.
	finalizedStats := 0
	for identifier, info := range senderStats {
		channel := channelsByIdentifier[identifier]
		if channel == nil {
			resolved, err := h.db.GetSourceChannelByIdentifier(h.UserID, source.SourceTypeWhatsApp, identifier)
			if err != nil || resolved == nil {
				continue
			}
			channel = resolved
			channelsByIdentifier[identifier] = channel
		}

		if err := h.db.UpdateChannelStats(channel.ID, info.messageCount, info.lastMessageAt); err != nil {
			fmt.Printf("HistorySync: Failed to finalize stats for %s: %v\n", identifier, err)
			continue
		}
		finalizedStats++
	}

	fmt.Printf(
		"HistorySync: Completed - processed %d contacts, top stats primed for %d, finalized for %d\n",
		processedContacts,
		statsUpdated,
		finalizedStats,
	)

	h.queueHistorySyncBackfill(processedChannels)

	// Force sync contacts and refresh names immediately
	// This uses FetchAppState to get contacts ASAP, with AppStateSyncComplete as fallback
	go h.forceSyncContactsAndRefresh()
}

func (h *Handler) queueHistorySyncBackfill(processedChannels map[int64]*database.SourceChannel) {
	h.historySyncMu.Lock()
	defer h.historySyncMu.Unlock()

	if h.historySyncBackfillHook == nil {
		return
	}

	for channelID, channel := range processedChannels {
		if channel == nil || !channel.Enabled {
			continue
		}
		h.historySyncPendingBackfill[channelID] = channel
	}

	if len(h.historySyncPendingBackfill) == 0 {
		return
	}

	if h.historySyncBackfillDebounce != nil {
		h.historySyncBackfillDebounce.Stop()
	}

	h.historySyncBackfillDebounce = time.AfterFunc(historySyncBackfillDebounce, h.flushHistorySyncBackfill)
}

func (h *Handler) flushHistorySyncBackfill() {
	historySyncHook, channels := func() (HistorySyncBackfillHook, []*database.SourceChannel) {
		h.historySyncMu.Lock()
		defer h.historySyncMu.Unlock()

		hook := h.historySyncBackfillHook
		if hook == nil || len(h.historySyncPendingBackfill) == 0 {
			return nil, nil
		}

		channels := make([]*database.SourceChannel, 0, len(h.historySyncPendingBackfill))
		for _, channel := range h.historySyncPendingBackfill {
			channels = append(channels, channel)
		}
		h.historySyncPendingBackfill = make(map[int64]*database.SourceChannel)
		h.historySyncBackfillDebounce = nil
		return hook, channels
	}()

	if historySyncHook == nil || len(channels) == 0 {
		return
	}

	triggered := 0
	for _, channel := range channels {
		if channel == nil || !channel.Enabled {
			continue
		}
		historySyncHook(h.UserID, channel)
		triggered++
	}

	if triggered > 0 {
		fmt.Printf("HistorySync: Triggered post-sync backfill for %d enabled channels\n", triggered)
	}
}

// processConversationHistory stores messages from a single conversation
func (h *Handler) processConversationHistory(channel *database.SourceChannel, chatJID types.JID, messages []*waProto.HistorySyncMsg) bool {
	identifier := chatJID.User

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
		if err := h.db.PruneSourceMessages(h.UserID, source.SourceTypeWhatsApp, channel.ID, maxHistoryMessagesPerContact); err != nil {
			fmt.Printf("HistorySync: Failed to prune messages for %s: %v\n", identifier, err)
		}
		fmt.Printf("HistorySync: Stored %d messages for %s\n", processed, identifier)
	}

	return processed > 0
}

// getOrCreateHistoryChannel gets an existing channel or creates a new disabled one
func (h *Handler) getOrCreateHistoryChannel(identifier string, jid types.JID) (*database.SourceChannel, error) {
	// Check if channel exists
	channel, err := h.db.GetSourceChannelByIdentifier(h.UserID, source.SourceTypeWhatsApp, identifier)
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
		h.UserID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		identifier,
		name,
	)
	if err != nil {
		return nil, err
	}

	// Disable the channel - it's just for discovery, not event tracking
	if err := h.db.UpdateSourceChannel(h.UserID, channel.ID, channel.Name, false); err != nil {
		fmt.Printf("HistorySync: Warning - failed to disable new channel: %v\n", err)
	}

	return channel, nil
}

// Contact name refresh after HistorySync

// refreshTopContactNames updates names for top 8 contacts by message history
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

	// Get top 8 contacts from message history (fast, user-scoped)
	topContacts, err := h.db.GetTopContactsBySourceTypeForUser(h.UserID, source.SourceTypeWhatsApp, 8)
	if err != nil {
		fmt.Printf("RefreshTopNames: Failed to get top contacts: %v\n", err)
		return
	}

	updated := 0
	for _, contact := range topContacts {
		// Skip if already has a real name (name != identifier)
		if contact.Name != contact.Identifier {
			continue
		}

		// Look up in contacts map
		jid, err := types.ParseJID(contact.Identifier + "@s.whatsapp.net")
		if err != nil {
			continue
		}

		if waContact, ok := allContacts[jid]; ok {
			name := waContact.FullName
			if name == "" {
				name = waContact.PushName
			}
			if name != "" && name != contact.Identifier {
				if err := h.db.UpdateSourceChannel(h.UserID, contact.ChannelID, name, contact.IsTracked); err != nil {
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
	channels, err := h.db.ListSourceChannels(h.UserID, source.SourceTypeWhatsApp)
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
				if err := h.db.UpdateSourceChannel(h.UserID, channel.ID, name, channel.Enabled); err != nil {
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

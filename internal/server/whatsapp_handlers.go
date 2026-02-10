package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/source"
	"go.mau.fi/whatsmeow/types"
)

// WhatsApp Top Contacts API

// TopContactResponse represents a top contact for the Add Source modal
type TopContactResponse struct {
	Identifier     string `json:"identifier"`
	Name           string `json:"name"`
	PushName       string `json:"push_name,omitempty"`
	SecondaryLabel string `json:"secondary_label"` // Pre-formatted: "+1234567890"
	MessageCount   int    `json:"message_count"`
	IsTracked      bool   `json:"is_tracked"`
	ChannelID      *int64 `json:"channel_id,omitempty"`
	Type           string `json:"type"`
}

func formatWhatsAppTopContactsFromHistory(stats []database.TopContactStats) []TopContactResponse {
	contacts := make([]TopContactResponse, len(stats))
	for i, c := range stats {
		phone := strings.TrimSuffix(c.Identifier, "@s.whatsapp.net")
		contacts[i] = TopContactResponse{
			Identifier:     c.Identifier,
			Name:           c.Name,
			SecondaryLabel: "+" + phone,
			MessageCount:   c.MessageCount,
			IsTracked:      c.IsTracked,
			Type:           c.Type,
		}
		if c.IsTracked {
			contacts[i].ChannelID = &c.ChannelID
		}
	}
	return contacts
}

func formatWhatsAppTopContactsFromChannelStats(channels []*database.SourceChannel) []TopContactResponse {
	contacts := make([]TopContactResponse, len(channels))
	for i, channel := range channels {
		phone := strings.TrimSuffix(channel.Identifier, "@s.whatsapp.net")
		contacts[i] = TopContactResponse{
			Identifier:     channel.Identifier,
			Name:           channel.Name,
			SecondaryLabel: "+" + phone,
			MessageCount:   channel.TotalMessageCount,
			IsTracked:      channel.Enabled,
			Type:           string(channel.Type),
		}
		if channel.Enabled {
			contacts[i].ChannelID = &channel.ID
		}
	}
	return contacts
}

func (s *Server) handleWhatsAppTopContacts(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	const topContactsLimit = 8

	// Prefer channel-level stats first: these are updated as HistorySync progresses and
	// can return a partial "best effort" list before message_history catch-up completes.
	topChannels, err := s.db.GetTopChannelsByMessageCount(userID, source.SourceTypeWhatsApp, topContactsLimit)
	if err == nil && len(topChannels) > 0 {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"contacts": formatWhatsAppTopContactsFromChannelStats(topChannels),
		})
		return
	}

	historyStats, err := s.db.GetTopContactsBySourceTypeForUser(userID, source.SourceTypeWhatsApp, topContactsLimit)
	if err == nil && len(historyStats) > 0 {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"contacts": formatWhatsAppTopContactsFromHistory(historyStats),
		})
		return
	}

	// If no stats available yet, wait briefly for HistorySync to complete before falling back
	if s.clientManager != nil {
		waClient, err := s.clientManager.GetWhatsAppClient(userID)
		if err == nil && waClient.WAClient != nil && waClient.IsLoggedIn() {
			timeout := time.After(6 * time.Second)
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()

			timedOut := false
			for !timedOut {
				select {
				case <-timeout:
					timedOut = true
				case <-ticker.C:
					topChannels, err = s.db.GetTopChannelsByMessageCount(userID, source.SourceTypeWhatsApp, topContactsLimit)
					if err == nil && len(topChannels) > 0 {
						respondJSON(w, http.StatusOK, map[string]interface{}{
							"contacts": formatWhatsAppTopContactsFromChannelStats(topChannels),
						})
						return
					}

					historyStats, err = s.db.GetTopContactsBySourceTypeForUser(userID, source.SourceTypeWhatsApp, topContactsLimit)
					if err == nil && len(historyStats) > 0 {
						respondJSON(w, http.StatusOK, map[string]interface{}{
							"contacts": formatWhatsAppTopContactsFromHistory(historyStats),
						})
						return
					}
				}
			}

			// HistorySync didn't complete in time; return empty for now.
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"contacts": []TopContactResponse{},
	})
}

func (s *Server) handleWhatsAppContactSearch(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if len(query) < 2 {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"contacts": []TopContactResponse{},
		})
		return
	}

	if s.clientManager == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"contacts": []TopContactResponse{},
		})
		return
	}

	waClient, err := s.clientManager.GetWhatsAppClient(userID)
	if err != nil || waClient == nil || waClient.WAClient == nil || waClient.WAClient.Store == nil || waClient.WAClient.Store.Contacts == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"contacts": []TopContactResponse{},
		})
		return
	}

	allContacts, err := waClient.WAClient.Store.Contacts.GetAllContacts(r.Context())
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"contacts": []TopContactResponse{},
		})
		return
	}

	queryLower := strings.ToLower(query)
	var results []TopContactResponse
	for jid, contact := range allContacts {
		if jid.Server != "s.whatsapp.net" {
			continue
		}

		full := strings.TrimSpace(contact.FullName)
		push := strings.TrimSpace(contact.PushName)

		matched := false
		if full != "" && strings.Contains(strings.ToLower(full), queryLower) {
			matched = true
		} else if push != "" && strings.Contains(strings.ToLower(push), queryLower) {
			matched = true
		}

		if !matched {
			continue
		}

		name := full
		if name == "" {
			name = push
		}
		if name == "" {
			name = jid.User
		}

		identifier := jid.User
		phone := identifier
		isTracked := false
		var channelID *int64

		if tracked, id, _, _ := s.db.IsSourceChannelTracked(userID, source.SourceTypeWhatsApp, identifier); tracked {
			isTracked = true
			channelID = &id
		} else {
			legacyIdentifier := identifier + "@s.whatsapp.net"
			if tracked, id, _, _ := s.db.IsSourceChannelTracked(userID, source.SourceTypeWhatsApp, legacyIdentifier); tracked {
				isTracked = true
				channelID = &id
			}
		}

		results = append(results, TopContactResponse{
			Identifier:     identifier,
			Name:           name,
			PushName:       push,
			SecondaryLabel: "+" + phone,
			IsTracked:      isTracked,
			ChannelID:      channelID,
			Type:           "sender",
		})

		if len(results) >= 10 {
			break
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"contacts": results,
	})
}

func (s *Server) handleWhatsAppCustomSource(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req struct {
		Name        string `json:"name"`
		PhoneNumber string `json:"phone_number"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	nameInput := strings.TrimSpace(req.Name)
	phoneInput := strings.TrimSpace(req.PhoneNumber)

	if nameInput == "" && phoneInput == "" {
		respondError(w, http.StatusBadRequest, "name or phone_number is required")
		return
	}

	// Name-based contact resolution (preferred)
	if nameInput != "" {
		if s.clientManager == nil {
			respondError(w, http.StatusBadRequest, "WhatsApp contacts not available")
			return
		}

		waClient, err := s.clientManager.GetWhatsAppClient(userID)
		if err != nil || waClient == nil || waClient.WAClient == nil || waClient.WAClient.Store == nil || waClient.WAClient.Store.Contacts == nil {
			respondError(w, http.StatusBadRequest, "WhatsApp contacts not available")
			return
		}

		allContacts, err := waClient.WAClient.Store.Contacts.GetAllContacts(r.Context())
		if err != nil {
			respondError(w, http.StatusBadRequest, "Failed to load WhatsApp contacts")
			return
		}

		query := strings.ToLower(nameInput)
		var matchJID types.JID
		var matchName string
		matches := 0
		for jid, contact := range allContacts {
			full := strings.TrimSpace(contact.FullName)
			push := strings.TrimSpace(contact.PushName)

			matched := false
			if full != "" && strings.Contains(strings.ToLower(full), query) {
				matched = true
			} else if push != "" && strings.Contains(strings.ToLower(push), query) {
				matched = true
			}

			if matched {
				matches++
				if matches == 1 {
					matchJID = jid
					if full != "" {
						matchName = full
					} else if push != "" {
						matchName = push
					} else {
						matchName = nameInput
					}
				}
				if matches > 1 {
					break
				}
			}
		}

		if matches == 0 {
			respondError(w, http.StatusBadRequest, "No matching WhatsApp contact found")
			return
		}
		if matches > 1 {
			respondError(w, http.StatusConflict, "Multiple contacts match that name. Please be more specific.")
			return
		}

		identifier := matchJID.User
		if identifier == "" {
			respondError(w, http.StatusBadRequest, "Invalid contact identifier")
			return
		}

		legacyIdentifier := identifier + "@s.whatsapp.net"
		existing, err := s.db.GetSourceChannelByIdentifier(userID, source.SourceTypeWhatsApp, identifier)
		if err == nil && existing == nil {
			existing, err = s.db.GetSourceChannelByIdentifier(userID, source.SourceTypeWhatsApp, legacyIdentifier)
		}
		if err == nil && existing != nil {
			respondError(w, http.StatusConflict, "This contact is already being tracked")
			return
		}

		channel, err := s.db.CreateSourceChannel(
			userID,
			source.SourceTypeWhatsApp,
			source.ChannelTypeSender,
			identifier,
			matchName,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		s.startChannelBackfill(userID, channel)

		respondJSON(w, http.StatusCreated, channel)
		return
	}

	// Phone-number-based fallback (legacy)
	phone := phoneInput
	for _, char := range []string{" ", "-", "(", ")", "+"} {
		phone = replaceAll(phone, char, "")
	}

	// Basic validation: should be 7-15 digits
	if len(phone) < 7 || len(phone) > 15 {
		respondError(w, http.StatusBadRequest, "Invalid phone number format")
		return
	}
	for _, c := range phone {
		if c < '0' || c > '9' {
			respondError(w, http.StatusBadRequest, "Phone number must contain only digits")
			return
		}
	}

	// Canonical identifier (digits only)
	identifier := phone
	legacyIdentifier := phone + "@s.whatsapp.net"

	// Check if already tracked
	existing, err := s.db.GetSourceChannelByIdentifier(userID, source.SourceTypeWhatsApp, identifier)
	if err == nil && existing == nil {
		existing, err = s.db.GetSourceChannelByIdentifier(userID, source.SourceTypeWhatsApp, legacyIdentifier)
	}
	if err == nil && existing != nil {
		respondError(w, http.StatusConflict, "This phone number is already being tracked")
		return
	}

	name := req.PhoneNumber
	if s.clientManager != nil {
		if waClient, err := s.clientManager.GetWhatsAppClient(userID); err == nil &&
			waClient != nil && waClient.WAClient != nil && waClient.WAClient.Store != nil && waClient.WAClient.Store.Contacts != nil {
			if jid, err := types.ParseJID(legacyIdentifier); err == nil {
				if contact, err := waClient.WAClient.Store.Contacts.GetContact(r.Context(), jid); err == nil {
					if contact.FullName != "" {
						name = contact.FullName
					} else if contact.PushName != "" {
						name = contact.PushName
					}
				}
			}
		}
	}

	// Create the channel
	channel, err := s.db.CreateSourceChannel(
		userID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		identifier,
		name,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.startChannelBackfill(userID, channel)

	respondJSON(w, http.StatusCreated, channel)
}

func (s *Server) handleListWhatsappChannels(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Note: type filter removed - only contacts (sender type) are supported now
	channels, err := s.db.ListSourceChannels(userID, source.SourceTypeWhatsApp)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pushNameByIdentifier := map[string]string{}
	if s.clientManager != nil {
		if waClient, err := s.clientManager.GetWhatsAppClient(userID); err == nil &&
			waClient != nil && waClient.WAClient != nil && waClient.WAClient.Store != nil && waClient.WAClient.Store.Contacts != nil {
			if allContacts, err := waClient.WAClient.Store.Contacts.GetAllContacts(r.Context()); err == nil {
				for jid, contact := range allContacts {
					if jid.Server != "s.whatsapp.net" {
						continue
					}
					push := strings.TrimSpace(contact.PushName)
					if push == "" {
						continue
					}
					identifier := jid.User
					pushNameByIdentifier[identifier] = push
					pushNameByIdentifier[identifier+"@s.whatsapp.net"] = push
				}
			}
		}
	}

	type whatsappChannelResponse struct {
		*database.SourceChannel
		PushName string `json:"push_name,omitempty"`
	}

	response := make([]whatsappChannelResponse, 0, len(channels))
	for _, channel := range channels {
		response = append(response, whatsappChannelResponse{
			SourceChannel: channel,
			PushName:      pushNameByIdentifier[channel.Identifier],
		})
	}

	respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleCreateWhatsappChannel(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req struct {
		Type       string `json:"type"`
		Identifier string `json:"identifier"`
		Name       string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Type == "" || req.Identifier == "" || req.Name == "" {
		respondError(w, http.StatusBadRequest, "type, identifier and name are required")
		return
	}

	if req.Type != "sender" {
		respondError(w, http.StatusBadRequest, "type must be 'sender' (contacts only)")
		return
	}

	// Check if channel already exists (may have been created by history sync)
	existingChannel, err := s.db.GetSourceChannelByIdentifier(userID, source.SourceTypeWhatsApp, req.Identifier)
	if err == nil && existingChannel != nil {
		wasDisabled := !existingChannel.Enabled
		// Channel exists - update it (enable it, update name)
		if err := s.db.UpdateSourceChannel(userID, existingChannel.ID, req.Name, true); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to enable channel: %v", err))
			return
		}
		// Return updated channel
		existingChannel.Name = req.Name
		existingChannel.Enabled = true
		if wasDisabled {
			s.startChannelBackfill(userID, existingChannel)
		}
		respondJSON(w, http.StatusOK, existingChannel)
		return
	}

	// Channel doesn't exist - create a new one
	channel, err := s.db.CreateSourceChannel(
		userID,
		source.SourceTypeWhatsApp,
		source.ChannelType(req.Type),
		req.Identifier,
		req.Name,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create channel: %v", err))
		return
	}

	s.startChannelBackfill(userID, channel)

	respondJSON(w, http.StatusCreated, channel)
}

func (s *Server) handleUpdateWhatsappChannel(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := s.db.UpdateSourceChannel(userID, id, req.Name, req.Enabled); err != nil {
		if err.Error() == "channel not found" {
			respondError(w, http.StatusNotFound, "channel not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	channel, _ := s.db.GetSourceChannelByID(userID, id)
	respondJSON(w, http.StatusOK, channel)
}

func (s *Server) handleDeleteWhatsappChannel(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	channel, err := s.db.GetSourceChannelByID(userID, id)
	if err != nil || channel == nil {
		respondError(w, http.StatusNotFound, "channel not found")
		return
	}

	if err := s.db.UpdateSourceChannel(userID, id, channel.Name, false); err != nil {
		if err.Error() == "channel not found" {
			respondError(w, http.StatusNotFound, "channel not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Disable legacy/alternate identifier if present to avoid accidental re-tracking.
	if channel != nil {
		identifier := channel.Identifier
		legacyIdentifier := identifier + "@s.whatsapp.net"
		if strings.HasSuffix(identifier, "@s.whatsapp.net") {
			legacyIdentifier = strings.TrimSuffix(identifier, "@s.whatsapp.net")
		}
		if legacyIdentifier != identifier {
			legacyChannel, legacyErr := s.db.GetSourceChannelByIdentifier(userID, source.SourceTypeWhatsApp, legacyIdentifier)
			if legacyErr == nil && legacyChannel != nil {
				_ = s.db.UpdateSourceChannel(userID, legacyChannel.ID, legacyChannel.Name, false)
			}
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "disabled"})
}

// WhatsApp API
func (s *Server) handleWhatsAppReconnect(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if s.clientManager == nil {
		respondError(w, http.StatusServiceUnavailable, "Client manager not initialized")
		return
	}

	if s.onboardingState == nil {
		respondError(w, http.StatusServiceUnavailable, "Onboarding state not initialized")
		return
	}

	// Get per-user WhatsApp client
	waClient, err := s.clientManager.GetWhatsAppClient(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get WhatsApp client: %v", err))
		return
	}

	// Trigger reconnect - use background context since request context will be cancelled
	go waClient.Reconnect(context.Background(), s.onboardingState)

	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "reconnecting",
		"message": "Reconnection initiated, new QR code will be generated",
	})
}

// handleWhatsAppStatus returns the WhatsApp connection status
func (s *Server) handleWhatsAppStatus(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	status := map[string]interface{}{
		"connected": false,
		"message":   "Not connected",
	}

	if s.clientManager == nil {
		status["message"] = "Client manager not initialized"
		respondJSON(w, http.StatusOK, status)
		return
	}

	// Read-only status path: do not create clients.
	if waClient, ok := s.clientManager.PeekWhatsAppClient(userID); ok && waClient.IsLoggedIn() {
		status["connected"] = true
		status["message"] = "Connected"
		respondJSON(w, http.StatusOK, status)
		return
	}

	// Fall back to stored session metadata when client is not in memory.
	if waSession, err := s.db.GetWhatsAppSession(userID); err == nil && waSession != nil && waSession.Connected {
		status["connected"] = true
		status["message"] = "Connected"
	} else {
		status["message"] = "Not authenticated"
	}

	respondJSON(w, http.StatusOK, status)
}

// handleWhatsAppPair generates a pairing code for phone-number-based WhatsApp linking
func (s *Server) handleWhatsAppPair(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if s.clientManager == nil {
		respondError(w, http.StatusServiceUnavailable, "Client manager not initialized")
		return
	}

	var req struct {
		PhoneNumber string `json:"phone_number"` // e.g., "+1234567890" or "1234567890"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.PhoneNumber == "" {
		respondError(w, http.StatusBadRequest, "phone_number is required")
		return
	}

	// Remove leading '+' if present
	phone := req.PhoneNumber
	if len(phone) > 0 && phone[0] == '+' {
		phone = phone[1:]
	}

	// Persist the latest pairing phone immediately; connection/device details are updated on WA events.
	_ = s.db.SaveWhatsAppSession(userID, phone, "", false)

	// Update onboarding state
	if s.onboardingState != nil {
		s.onboardingState.SetWhatsAppStatus("pairing")
	}

	// Get per-user WhatsApp client
	waClient, err := s.clientManager.GetWhatsAppClient(userID)
	if err != nil {
		if s.onboardingState != nil {
			s.onboardingState.SetWhatsAppError(fmt.Sprintf("Failed to get WhatsApp client: %v", err))
		}
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get WhatsApp client: %v", err))
		return
	}

	// Generate pairing code
	code, err := waClient.PairWithPhone(r.Context(), phone, s.onboardingState)
	if err != nil {
		if s.onboardingState != nil {
			s.onboardingState.SetWhatsAppError(fmt.Sprintf("Failed to generate pairing code: %v", err))
		}
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to generate pairing code: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"code":    code,
		"message": "Enter this code in WhatsApp > Linked Devices > Link with phone number",
	})
}

// handleWhatsAppDisconnect logs out from WhatsApp and clears the session
func (s *Server) handleWhatsAppDisconnect(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if s.clientManager == nil {
		respondError(w, http.StatusServiceUnavailable, "Client manager not initialized")
		return
	}

	// Logout via ClientManager (handles session cleanup)
	if err := s.clientManager.LogoutWhatsApp(userID); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to logout: %v", err))
		return
	}

	if s.onboardingState != nil {
		s.onboardingState.SetWhatsAppStatus("disconnected")
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "disconnected",
		"message": "WhatsApp logged out and session cleared",
	})
}

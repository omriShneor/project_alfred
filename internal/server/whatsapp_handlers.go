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
	"github.com/omriShneor/project_alfred/internal/whatsapp"
)

// WhatsApp Top Contacts API

// TopContactResponse represents a top contact for the Add Source modal
type TopContactResponse struct {
	Identifier     string `json:"identifier"`
	Name           string `json:"name"`
	SecondaryLabel string `json:"secondary_label"` // Pre-formatted: "+1234567890"
	MessageCount   int    `json:"message_count"`
	IsTracked      bool   `json:"is_tracked"`
	ChannelID      *int64 `json:"channel_id,omitempty"`
	Type           string `json:"type"`
}

func (s *Server) handleWhatsAppTopContacts(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	response := []TopContactResponse{}

	channels, err := s.db.GetTopChannelsByMessageCount(userID, source.SourceTypeWhatsApp, 8)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// If no stats available yet, wait briefly for HistorySync to complete before falling back
	if len(channels) == 0 && s.clientManager != nil {
		waClient, err := s.clientManager.GetWhatsAppClient(userID)
		if err == nil && waClient.WAClient != nil && waClient.IsLoggedIn() {
			timeout := time.After(25 * time.Second)
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-timeout:
					goto fallbackToDiscovery
				case <-ticker.C:
					channels, err = s.db.GetTopChannelsByMessageCount(userID, source.SourceTypeWhatsApp, 8)
					if err == nil && len(channels) > 0 {
						response = make([]TopContactResponse, len(channels))
						for i, c := range channels {
							// Format phone number
							phone := strings.TrimSuffix(c.Identifier, "@s.whatsapp.net")
							response[i] = TopContactResponse{
								Identifier:     c.Identifier,
								Name:           c.Name,
								SecondaryLabel: "+" + phone,
								MessageCount:   c.TotalMessageCount, // Accurate count from HistorySync
								IsTracked:      c.Enabled,
								Type:           string(c.Type),
							}
							if c.Enabled {
								response[i].ChannelID = &c.ID
							}
						}
					}
				}
			}

		fallbackToDiscovery:
			// HistorySync didn't complete in time, fall back to discoverable channels
			discoverableChannels, err := waClient.GetDiscoverableChannels()
			if err == nil && len(discoverableChannels) > 0 {
				// Filter to only contacts (not groups) and limit to 8
				var contactChannels []whatsapp.DiscoverableChannel
				for _, ch := range discoverableChannels {
					if ch.Type == "sender" {
						contactChannels = append(contactChannels, ch)
					}
				}
				if len(contactChannels) > 8 {
					contactChannels = contactChannels[:8]
				}
				// Convert to response format
				response := make([]TopContactResponse, len(contactChannels))
				for i, ch := range contactChannels {
					// Check if already tracked (user-scoped query for multi-user support)
					existingChannel, _ := s.db.GetSourceChannelByIdentifier(userID, source.SourceTypeWhatsApp, ch.Identifier)
					// Format phone number as secondary label
					phone := strings.TrimSuffix(ch.Identifier, "@s.whatsapp.net")
					response[i] = TopContactResponse{
						Identifier:     ch.Identifier,
						Name:           ch.Name,
						SecondaryLabel: "+" + phone,
						MessageCount:   0, // No message count from discovery
						IsTracked:      existingChannel != nil && existingChannel.Enabled,
						Type:           ch.Type,
					}
					if existingChannel != nil && existingChannel.Enabled {
						response[i].ChannelID = &existingChannel.ID
					}
				}
				respondJSON(w, http.StatusOK, map[string]interface{}{
					"contacts": response,
				})
				return
			}
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"contacts": response,
	})
}

func (s *Server) handleWhatsAppCustomSource(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PhoneNumber string `json:"phone_number"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.PhoneNumber == "" {
		respondError(w, http.StatusBadRequest, "phone_number is required")
		return
	}

	// Normalize phone number - remove spaces, dashes, parentheses
	phone := req.PhoneNumber
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

	// Create identifier in WhatsApp format
	identifier := phone + "@s.whatsapp.net"

	// Check if already tracked
	existing, err := s.db.GetChannelByIdentifier(identifier)
	if err == nil && existing != nil {
		respondError(w, http.StatusConflict, "This phone number is already being tracked")
		return
	}

	// Create the channel
	channel, err := s.db.CreateChannel(database.ChannelTypeSender, identifier, req.PhoneNumber)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

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
	respondJSON(w, http.StatusOK, channels)
}

func (s *Server) handleCreateWhatsappChannel(w http.ResponseWriter, r *http.Request) {
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
	existingChannel, err := s.db.GetChannelByIdentifier(req.Identifier)
	if err == nil && existingChannel != nil {
		// Channel exists - update it (enable it, update name)
		if err := s.db.UpdateChannel(existingChannel.ID, req.Name, true); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to enable channel: %v", err))
			return
		}
		// Return updated channel
		existingChannel.Name = req.Name
		existingChannel.Enabled = true
		respondJSON(w, http.StatusOK, existingChannel)
		return
	}

	// Channel doesn't exist - create a new one
	channel, err := s.db.CreateChannel(database.ChannelType(req.Type), req.Identifier, req.Name)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create channel: %v", err))
		return
	}

	respondJSON(w, http.StatusCreated, channel)
}

func (s *Server) handleUpdateWhatsappChannel(w http.ResponseWriter, r *http.Request) {
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

	if err := s.db.UpdateChannel(id, req.Name, req.Enabled); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	channel, _ := s.db.GetChannelByID(id)
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

	if err := s.db.DeleteChannel(userID, id); err != nil {
		if err.Error() == "channel not found" {
			respondError(w, http.StatusNotFound, "channel not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
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

	// Get per-user WhatsApp client
	waClient, err := s.clientManager.GetWhatsAppClient(userID)
	if err != nil {
		status["message"] = fmt.Sprintf("Failed to get client: %v", err)
		respondJSON(w, http.StatusOK, status)
		return
	}

	if waClient.IsLoggedIn() {
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

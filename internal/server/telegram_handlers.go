package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/omriShneor/project_alfred/internal/source"
)

// TelegramStatusResponse represents the Telegram connection status
type TelegramStatusResponse struct {
	Connected bool   `json:"connected"`
	Message   string `json:"message,omitempty"`
}

// handleTelegramStatus returns the current Telegram connection status
func (s *Server) handleTelegramStatus(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if s.clientManager == nil {
		respondJSON(w, http.StatusOK, TelegramStatusResponse{
			Connected: false,
			Message:   "Client manager not configured",
		})
		return
	}

	// Read-only status path: do not create clients.
	if tgClient, ok := s.clientManager.PeekTelegramClient(userID); ok {
		respondJSON(w, http.StatusOK, TelegramStatusResponse{
			Connected: tgClient.IsConnected(),
			Message:   "",
		})
		return
	}

	// Fall back to stored session metadata when client is not in memory.
	if tgSession, err := s.db.GetTelegramSession(userID); err == nil && tgSession != nil && tgSession.Connected {
		respondJSON(w, http.StatusOK, TelegramStatusResponse{
			Connected: true,
			Message:   "",
		})
		return
	}

	respondJSON(w, http.StatusOK, TelegramStatusResponse{
		Connected: false,
		Message:   "Not connected",
	})
}

// TelegramSendCodeRequest represents a request to send verification code
type TelegramSendCodeRequest struct {
	PhoneNumber string `json:"phone_number"`
}

// handleTelegramSendCode sends a verification code to the given phone number
func (s *Server) handleTelegramSendCode(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if s.clientManager == nil {
		respondError(w, http.StatusServiceUnavailable, "Client manager not configured")
		return
	}

	var req TelegramSendCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.PhoneNumber == "" {
		respondError(w, http.StatusBadRequest, "Phone number is required")
		return
	}

	// Get per-user Telegram client
	tgClient, err := s.clientManager.GetTelegramClient(userID)
	if err != nil {
		s.state.SetTelegramError(fmt.Sprintf("Failed to get Telegram client: %v", err))
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get Telegram client: %v", err))
		return
	}

	if err := tgClient.SendCode(r.Context(), req.PhoneNumber); err != nil {
		s.state.SetTelegramError(err.Error())
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to send code: %v", err))
		return
	}

	s.state.SetTelegramStatus("code_sent")
	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Verification code sent",
	})
}

// TelegramVerifyCodeRequest represents a request to verify the code
type TelegramVerifyCodeRequest struct {
	PhoneNumber string `json:"phone_number"`
	Code        string `json:"code"`
}

// handleTelegramVerifyCode verifies the code and completes authentication
func (s *Server) handleTelegramVerifyCode(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if s.clientManager == nil {
		respondError(w, http.StatusServiceUnavailable, "Client manager not configured")
		return
	}

	var req TelegramVerifyCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Code == "" {
		respondError(w, http.StatusBadRequest, "Verification code is required")
		return
	}

	// Get per-user Telegram client
	tgClient, err := s.clientManager.GetTelegramClient(userID)
	if err != nil {
		s.state.SetTelegramError(fmt.Sprintf("Failed to get Telegram client: %v", err))
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get Telegram client: %v", err))
		return
	}

	if err := tgClient.VerifyCode(r.Context(), req.Code); err != nil {
		s.state.SetTelegramError(err.Error())
		respondError(w, http.StatusUnauthorized, fmt.Sprintf("Failed to verify code: %v", err))
		return
	}

	s.state.SetTelegramStatus("connected")
	respondJSON(w, http.StatusOK, TelegramStatusResponse{
		Connected: true,
		Message:   "Successfully authenticated",
	})
}

// handleTelegramDisconnect disconnects the Telegram client
func (s *Server) handleTelegramDisconnect(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if s.clientManager == nil {
		respondError(w, http.StatusServiceUnavailable, "Client manager not configured")
		return
	}

	// Logout via ClientManager (handles session cleanup)
	if err := s.clientManager.LogoutTelegram(userID); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to disconnect: %v", err))
		return
	}

	s.state.SetTelegramStatus("pending")

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Telegram disconnected",
	})
}

// handleTelegramReconnect attempts to reconnect the Telegram client
func (s *Server) handleTelegramReconnect(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if s.clientManager == nil {
		respondError(w, http.StatusServiceUnavailable, "Client manager not configured")
		return
	}

	// Get per-user Telegram client
	tgClient, err := s.clientManager.GetTelegramClient(userID)
	if err != nil {
		s.state.SetTelegramError(fmt.Sprintf("Failed to get Telegram client: %v", err))
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get Telegram client: %v", err))
		return
	}

	if err := tgClient.Connect(); err != nil {
		s.state.SetTelegramError(err.Error())
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to reconnect: %v", err))
		return
	}

	if tgClient.IsConnected() {
		s.state.SetTelegramStatus("connected")
		respondJSON(w, http.StatusOK, TelegramStatusResponse{
			Connected: true,
			Message:   "Reconnected successfully",
		})
	} else {
		s.state.SetTelegramStatus("pending")
		respondJSON(w, http.StatusOK, TelegramStatusResponse{
			Connected: false,
			Message:   "Connection started but not authenticated",
		})
	}
}

// handleDiscoverTelegramChannels lists available Telegram chats
func (s *Server) handleDiscoverTelegramChannels(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	if s.clientManager == nil {
		respondError(w, http.StatusServiceUnavailable, "Client manager not configured")
		return
	}

	// Get per-user Telegram client
	tgClient, err := s.clientManager.GetTelegramClient(userID)
	if err != nil {
		respondError(w, http.StatusServiceUnavailable, fmt.Sprintf("Failed to get Telegram client: %v", err))
		return
	}

	if !tgClient.IsConnected() {
		respondError(w, http.StatusServiceUnavailable, "Telegram not connected")
		return
	}

	channels, err := tgClient.GetDiscoverableChannels(r.Context(), userID, s.db)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to discover channels: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, channels)
}

// handleListTelegramChannels lists tracked Telegram channels
func (s *Server) handleListTelegramChannels(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	channels, err := s.db.ListSourceChannels(userID, source.SourceTypeTelegram)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list channels: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, channels)
}

// TelegramCreateChannelRequest represents a request to create a Telegram channel
type TelegramCreateChannelRequest struct {
	Type       string `json:"type"` // "contact", "group", "channel"
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
}

// handleCreateTelegramChannel adds a Telegram channel to track
func (s *Server) handleCreateTelegramChannel(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var req TelegramCreateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Identifier == "" || req.Name == "" {
		respondError(w, http.StatusBadRequest, "Identifier and name are required")
		return
	}

	// Only contacts (sender type) are supported
	if req.Type != "" && req.Type != "contact" && req.Type != "sender" {
		respondError(w, http.StatusBadRequest, "Only contacts are supported (type must be 'contact' or 'sender')")
		return
	}
	channelType := source.ChannelTypeSender

	existingChannel, err := s.db.GetSourceChannelByIdentifier(userID, source.SourceTypeTelegram, req.Identifier)
	if err == nil && existingChannel != nil {
		wasDisabled := !existingChannel.Enabled
		if err := s.db.UpdateSourceChannel(userID, existingChannel.ID, req.Name, true); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to enable channel: %v", err))
			return
		}
		existingChannel.Name = req.Name
		existingChannel.Enabled = true
		if wasDisabled {
			s.startChannelBackfill(userID, existingChannel)
		}
		respondJSON(w, http.StatusOK, existingChannel)
		return
	}

	channel, err := s.db.CreateSourceChannel(userID, source.SourceTypeTelegram, channelType, req.Identifier, req.Name)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create channel: %v", err))
		return
	}

	s.startChannelBackfill(userID, channel)

	respondJSON(w, http.StatusCreated, channel)
}

// TelegramUpdateChannelRequest represents a request to update a Telegram channel
type TelegramUpdateChannelRequest struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// handleUpdateTelegramChannel updates a tracked Telegram channel
func (s *Server) handleUpdateTelegramChannel(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid channel ID")
		return
	}

	var req TelegramUpdateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := s.db.UpdateSourceChannel(userID, id, req.Name, req.Enabled); err != nil {
		if err.Error() == "channel not found" {
			respondError(w, http.StatusNotFound, "channel not found")
			return
		}
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update channel: %v", err))
		return
	}

	channel, _ := s.db.GetSourceChannelByID(userID, id)
	respondJSON(w, http.StatusOK, channel)
}

// handleDeleteTelegramChannel removes a tracked Telegram channel
func (s *Server) handleDeleteTelegramChannel(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid channel ID")
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
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to disable channel: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Channel disabled",
	})
}

// Telegram Top Contacts API

// TelegramTopContactResponse represents a top contact for the Add Source modal
type TelegramTopContactResponse struct {
	Identifier     string `json:"identifier"`
	Name           string `json:"name"`
	SecondaryLabel string `json:"secondary_label"` // Pre-formatted: "@username" or ""
	MessageCount   int    `json:"message_count"`
	IsTracked      bool   `json:"is_tracked"`
	ChannelID      *int64 `json:"channel_id,omitempty"`
	Type           string `json:"type"`
}

// handleTelegramTopContacts returns top contacts based on message history
func (s *Server) handleTelegramTopContacts(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Get top contacts from message history
	contacts, err := s.db.GetTopContactsBySourceTypeForUser(userID, source.SourceTypeTelegram, 8)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// If no message history, fall back to discoverable channels (recent chats from Telegram)
	if len(contacts) == 0 && s.clientManager != nil {
		// Get per-user Telegram client
		tgClient, err := s.clientManager.GetTelegramClient(userID)
		if err == nil && tgClient.IsConnected() {
			discoverableChannels, err := tgClient.GetDiscoverableChannels(r.Context(), userID, s.db)
			if err == nil && len(discoverableChannels) > 0 {
				// Filter to contacts only and limit to 8
				var response []TelegramTopContactResponse
				for _, ch := range discoverableChannels {
					if ch.Type == "contact" {
						resp := TelegramTopContactResponse{
							Identifier:     ch.Identifier,
							Name:           ch.Name,
							SecondaryLabel: ch.SecondaryLabel, // Already formatted in groups.go
							MessageCount:   0,                 // No message count from discovery
							IsTracked:      ch.IsTracked,
							Type:           ch.Type,
						}
						if ch.IsTracked && ch.ChannelID != nil {
							resp.ChannelID = ch.ChannelID
						}
						response = append(response, resp)
						if len(response) >= 8 {
							break
						}
					}
				}
				respondJSON(w, http.StatusOK, map[string]interface{}{
					"contacts": response,
				})
				return
			}
		}
	}

	// Convert to response format
	response := make([]TelegramTopContactResponse, len(contacts))
	for i, c := range contacts {
		response[i] = TelegramTopContactResponse{
			Identifier:     c.Identifier,
			Name:           c.Name,
			SecondaryLabel: "", // Username not available from message history
			MessageCount:   c.MessageCount,
			IsTracked:      c.IsTracked,
			Type:           c.Type,
		}
		if c.IsTracked {
			response[i].ChannelID = &c.ChannelID
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"contacts": response,
	})
}

func (s *Server) handleTelegramContactSearch(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if len(query) < 2 {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"contacts": []TelegramTopContactResponse{},
		})
		return
	}

	if s.clientManager == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"contacts": []TelegramTopContactResponse{},
		})
		return
	}

	tgClient, err := s.clientManager.GetTelegramClient(userID)
	if err != nil || !tgClient.IsConnected() {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"contacts": []TelegramTopContactResponse{},
		})
		return
	}

	allChannels, err := tgClient.GetDiscoverableChannels(r.Context(), userID, s.db)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"contacts": []TelegramTopContactResponse{},
		})
		return
	}

	queryLower := strings.ToLower(query)
	var results []TelegramTopContactResponse
	for _, ch := range allChannels {
		nameMatch := strings.Contains(strings.ToLower(ch.Name), queryLower)
		usernameMatch := ch.SecondaryLabel != "" && strings.Contains(strings.ToLower(ch.SecondaryLabel), queryLower)

		if !nameMatch && !usernameMatch {
			continue
		}

		resp := TelegramTopContactResponse{
			Identifier:     ch.Identifier,
			Name:           ch.Name,
			SecondaryLabel: ch.SecondaryLabel,
			IsTracked:      ch.IsTracked,
			Type:           ch.Type,
		}
		if ch.IsTracked && ch.ChannelID != nil {
			resp.ChannelID = ch.ChannelID
		}
		results = append(results, resp)

		if len(results) >= 10 {
			break
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"contacts": results,
	})
}

// TelegramCustomSourceRequest represents a request to add a custom Telegram source
type TelegramCustomSourceRequest struct {
	Username string `json:"username"`
}

// handleTelegramCustomSource creates a Telegram channel from a username
func (s *Server) handleTelegramCustomSource(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req TelegramCustomSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Username == "" {
		respondError(w, http.StatusBadRequest, "username is required")
		return
	}

	// Normalize username - remove @ prefix if present
	username := req.Username
	if len(username) > 0 && username[0] == '@' {
		username = username[1:]
	}

	// Basic validation: 5-32 characters, alphanumeric and underscore, starts with letter
	if len(username) < 5 || len(username) > 32 {
		respondError(w, http.StatusBadRequest, "Username must be 5-32 characters")
		return
	}
	if username[0] < 'a' || (username[0] > 'z' && username[0] < 'A') || username[0] > 'Z' {
		respondError(w, http.StatusBadRequest, "Username must start with a letter")
		return
	}

	if s.clientManager == nil {
		respondError(w, http.StatusServiceUnavailable, "Telegram client not available")
		return
	}
	tgClient, err := s.clientManager.GetTelegramClient(userID)
	if err != nil || !tgClient.IsConnected() {
		respondError(w, http.StatusBadRequest, "Telegram not connected")
		return
	}

	resolvedID, displayName, resolvedUsername, err := tgClient.ResolveUsername(r.Context(), username)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Use numeric Telegram user ID as identifier for consistency.
	identifier := fmt.Sprintf("%d", resolvedID)
	if displayName == "" {
		displayName = "@" + resolvedUsername
	}

	// Check if already tracked
	existing, err := s.db.GetSourceChannelByIdentifier(userID, source.SourceTypeTelegram, identifier)
	if err == nil && existing != nil {
		respondError(w, http.StatusConflict, "This username is already being tracked")
		return
	}

	// Create the channel as a contact type
	channel, err := s.db.CreateSourceChannel(userID, source.SourceTypeTelegram, source.ChannelTypeSender, identifier, displayName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create channel: %v", err))
		return
	}

	s.startChannelBackfill(userID, channel)

	respondJSON(w, http.StatusCreated, channel)
}

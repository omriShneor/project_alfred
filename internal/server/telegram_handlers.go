package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/omriShneor/project_alfred/internal/source"
)

// TelegramStatusResponse represents the Telegram connection status
type TelegramStatusResponse struct {
	Connected bool   `json:"connected"`
	Message   string `json:"message,omitempty"`
}

// handleTelegramStatus returns the current Telegram connection status
func (s *Server) handleTelegramStatus(w http.ResponseWriter, r *http.Request) {
	if s.tgClient == nil {
		respondJSON(w, http.StatusOK, TelegramStatusResponse{
			Connected: false,
			Message:   "Telegram not configured",
		})
		return
	}

	respondJSON(w, http.StatusOK, TelegramStatusResponse{
		Connected: s.tgClient.IsConnected(),
		Message:   "",
	})
}

// TelegramSendCodeRequest represents a request to send verification code
type TelegramSendCodeRequest struct {
	PhoneNumber string `json:"phone_number"`
}

// handleTelegramSendCode sends a verification code to the given phone number
func (s *Server) handleTelegramSendCode(w http.ResponseWriter, r *http.Request) {
	if s.tgClient == nil {
		respondError(w, http.StatusServiceUnavailable, "Telegram not configured")
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

	if err := s.tgClient.SendCode(r.Context(), req.PhoneNumber); err != nil {
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
	if s.tgClient == nil {
		respondError(w, http.StatusServiceUnavailable, "Telegram not configured")
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

	if err := s.tgClient.VerifyCode(r.Context(), req.Code); err != nil {
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
	if s.tgClient == nil {
		respondError(w, http.StatusServiceUnavailable, "Telegram not configured")
		return
	}

	s.tgClient.Disconnect()
	s.state.SetTelegramStatus("pending")

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Telegram disconnected",
	})
}

// handleTelegramReconnect attempts to reconnect the Telegram client
func (s *Server) handleTelegramReconnect(w http.ResponseWriter, r *http.Request) {
	if s.tgClient == nil {
		respondError(w, http.StatusServiceUnavailable, "Telegram not configured")
		return
	}

	if err := s.tgClient.Connect(); err != nil {
		s.state.SetTelegramError(err.Error())
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to reconnect: %v", err))
		return
	}

	if s.tgClient.IsConnected() {
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
	if s.tgClient == nil {
		respondError(w, http.StatusServiceUnavailable, "Telegram not configured")
		return
	}

	if !s.tgClient.IsConnected() {
		respondError(w, http.StatusServiceUnavailable, "Telegram not connected")
		return
	}

	channels, err := s.tgClient.GetDiscoverableChannels(r.Context(), s.db)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to discover channels: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, channels)
}

// handleListTelegramChannels lists tracked Telegram channels
func (s *Server) handleListTelegramChannels(w http.ResponseWriter, r *http.Request) {
	channels, err := s.db.ListSourceChannels(source.SourceTypeTelegram)
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

	channel, err := s.db.CreateSourceChannel(source.SourceTypeTelegram, channelType, req.Identifier, req.Name)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create channel: %v", err))
		return
	}

	respondJSON(w, http.StatusCreated, channel)
}

// TelegramUpdateChannelRequest represents a request to update a Telegram channel
type TelegramUpdateChannelRequest struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// handleUpdateTelegramChannel updates a tracked Telegram channel
func (s *Server) handleUpdateTelegramChannel(w http.ResponseWriter, r *http.Request) {
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

	if err := s.db.UpdateSourceChannel(id, req.Name, req.Enabled); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update channel: %v", err))
		return
	}

	channel, _ := s.db.GetSourceChannelByID(id)
	respondJSON(w, http.StatusOK, channel)
}

// handleDeleteTelegramChannel removes a tracked Telegram channel
func (s *Server) handleDeleteTelegramChannel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid channel ID")
		return
	}

	if err := s.db.DeleteSourceChannel(id); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete channel: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Channel deleted",
	})
}

// Telegram Top Contacts API

// TelegramTopContactResponse represents a top contact for the Add Source modal
type TelegramTopContactResponse struct {
	Identifier   string `json:"identifier"`
	Name         string `json:"name"`
	MessageCount int    `json:"message_count"`
	IsTracked    bool   `json:"is_tracked"`
	ChannelID    *int64 `json:"channel_id,omitempty"`
	Type         string `json:"type"`
}

// handleTelegramTopContacts returns top contacts based on message history
func (s *Server) handleTelegramTopContacts(w http.ResponseWriter, r *http.Request) {
	if s.tgClient == nil || !s.tgClient.IsConnected() {
		// Return empty contacts if Telegram not connected
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"contacts": []TelegramTopContactResponse{},
		})
		return
	}

	// Get top contacts from message history
	contacts, err := s.db.GetTopContactsBySourceType("telegram", 8)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// If no message history, fall back to discoverable channels (recent chats from Telegram)
	if len(contacts) == 0 {
		discoverableChannels, err := s.tgClient.GetDiscoverableChannels(r.Context(), s.db)
		if err == nil && len(discoverableChannels) > 0 {
			// Filter to contacts only and limit to 8
			var response []TelegramTopContactResponse
			for _, ch := range discoverableChannels {
				if ch.Type == "contact" {
					resp := TelegramTopContactResponse{
						Identifier:   ch.Identifier,
						Name:         ch.Name,
						MessageCount: 0, // No message count from discovery
						IsTracked:    ch.IsTracked,
						Type:         ch.Type,
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

	// Convert to response format
	response := make([]TelegramTopContactResponse, len(contacts))
	for i, c := range contacts {
		response[i] = TelegramTopContactResponse{
			Identifier:   c.Identifier,
			Name:         c.Name,
			MessageCount: c.MessageCount,
			IsTracked:    c.IsTracked,
			Type:         c.Type,
		}
		if c.IsTracked {
			response[i].ChannelID = &c.ChannelID
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"contacts": response,
	})
}

// TelegramCustomSourceRequest represents a request to add a custom Telegram source
type TelegramCustomSourceRequest struct {
	Username string `json:"username"`
}

// handleTelegramCustomSource creates a Telegram channel from a username
func (s *Server) handleTelegramCustomSource(w http.ResponseWriter, r *http.Request) {
	if s.tgClient == nil || !s.tgClient.IsConnected() {
		respondError(w, http.StatusServiceUnavailable, "Telegram not connected")
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

	// Use username as identifier (Telegram-style)
	identifier := username

	// Check if already tracked
	existing, err := s.db.GetSourceChannelByIdentifier(source.SourceTypeTelegram, identifier)
	if err == nil && existing != nil {
		respondError(w, http.StatusConflict, "This username is already being tracked")
		return
	}

	// Create the channel as a contact type
	channel, err := s.db.CreateSourceChannel(source.SourceTypeTelegram, source.ChannelTypeSender, identifier, "@"+username)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create channel: %v", err))
		return
	}

	respondJSON(w, http.StatusCreated, channel)
}

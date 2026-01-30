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
	Type       string `json:"type"`        // "contact", "group", "channel"
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
	CalendarID string `json:"calendar_id"`
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

	// Map channel type
	var channelType source.ChannelType
	switch req.Type {
	case "contact":
		channelType = source.ChannelTypeSender
	case "group":
		channelType = source.ChannelTypeGroup
	case "channel":
		channelType = source.ChannelTypeChannel
	default:
		channelType = source.ChannelTypeSender
	}

	channel, err := s.db.CreateSourceChannel(source.SourceTypeTelegram, channelType, req.Identifier, req.Name, req.CalendarID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create channel: %v", err))
		return
	}

	respondJSON(w, http.StatusCreated, channel)
}

// TelegramUpdateChannelRequest represents a request to update a Telegram channel
type TelegramUpdateChannelRequest struct {
	Name       string `json:"name"`
	CalendarID string `json:"calendar_id"`
	Enabled    bool   `json:"enabled"`
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

	if err := s.db.UpdateSourceChannel(id, req.Name, req.CalendarID, req.Enabled); err != nil {
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

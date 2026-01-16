package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/omriShneor/project_alfred/internal/database"
)

// Dashboard

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	channels, _ := s.db.ListChannels()
	pendingEvents, _ := s.db.ListPendingEvents()
	recentEvents, _ := s.db.ListRecentCalendarEvents(10)

	senderCount := 0
	groupCount := 0
	for _, ch := range channels {
		if ch.Type == "sender" {
			senderCount++
		} else if ch.Type == "group" {
			groupCount++
		}
	}

	data := map[string]interface{}{
		"tracked_channels": len(channels),
		"tracked_senders":  senderCount,
		"tracked_groups":   groupCount,
		"pending_events":   len(pendingEvents),
		"recent_events":    len(recentEvents),
	}

	respondJSON(w, http.StatusOK, data)
}

// WhatsApp Discovery

func (s *Server) handleWhatsAppContacts(w http.ResponseWriter, r *http.Request) {
	if s.waClient == nil || s.waClient.WAClient == nil {
		respondError(w, http.StatusServiceUnavailable, "WhatsApp not connected")
		return
	}

	contacts, err := s.waClient.GetRecentContacts(15)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, contacts)
}

func (s *Server) handleWhatsAppGroups(w http.ResponseWriter, r *http.Request) {
	if s.waClient == nil || s.waClient.WAClient == nil {
		respondError(w, http.StatusServiceUnavailable, "WhatsApp not connected")
		return
	}

	groups, err := s.waClient.GetGroups(15)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, groups)
}

// Events API

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.db.ListAllEvents(50)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, events)
}

func (s *Server) handleEventMessages(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	messages, err := s.db.GetMessagesByPendingEvent(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, messages)
}

func (s *Server) handleConfirmEvent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// TODO: Actually create the Google Calendar event
	if err := s.db.UpdatePendingEventStatus(id, "confirmed"); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "confirmed"})
}

func (s *Server) handleRejectEvent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := s.db.UpdatePendingEventStatus(id, "rejected"); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

// Channels API (Consolidated)

func (s *Server) handleListChannels(w http.ResponseWriter, r *http.Request) {
	typeFilter := r.URL.Query().Get("type")

	var channels []*database.Channel
	var err error

	if typeFilter != "" {
		channels, err = s.db.ListChannelsByType(database.ChannelType(typeFilter))
	} else {
		channels, err = s.db.ListChannels()
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, channels)
}

func (s *Server) handleCreateChannel(w http.ResponseWriter, r *http.Request) {
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

	if req.Type != "sender" && req.Type != "group" {
		respondError(w, http.StatusBadRequest, "type must be 'sender' or 'group'")
		return
	}

	channel, err := s.db.CreateChannel(database.ChannelType(req.Type), req.Identifier, req.Name)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, channel)
}

func (s *Server) handleUpdateChannel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req struct {
		Name       string `json:"name"`
		CalendarID string `json:"calendar_id"`
		Enabled    bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := s.db.UpdateChannel(id, req.Name, req.CalendarID, req.Enabled); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	channel, _ := s.db.GetChannelByID(id)
	respondJSON(w, http.StatusOK, channel)
}

func (s *Server) handleDeleteChannel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := s.db.DeleteChannel(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Printf("Error encoding JSON response: %v\n", err)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

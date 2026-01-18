package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/whatsapp"
)

// Dashboard

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	channels, _ := s.db.ListChannels()

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
	}

	respondJSON(w, http.StatusOK, data)
}

// Discovery Page

func (s *Server) handleDiscoveryPage(w http.ResponseWriter, r *http.Request) {
	html, err := staticFiles.ReadFile("static/discovery.html")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load discovery page")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(html)
}

// WhatsApp Discovery API

func (s *Server) handleDiscoverChannels(w http.ResponseWriter, r *http.Request) {
	if s.waClient == nil || s.waClient.WAClient == nil {
		respondError(w, http.StatusServiceUnavailable, "WhatsApp not connected")
		return
	}

	// Get all discoverable channels from WhatsApp
	channels, err := s.waClient.GetDiscoverableChannels()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Enrich with tracking status from database
	for i := range channels {
		channel, err := s.db.GetChannelByIdentifier(channels[i].Identifier)
		if err == nil && channel != nil {
			channels[i].IsTracked = true
			channels[i].ChannelID = &channel.ID
			channels[i].Enabled = &channel.Enabled
		}
	}

	// Optional: filter by type if query parameter is provided
	typeFilter := r.URL.Query().Get("type")
	if typeFilter != "" && (typeFilter == "sender" || typeFilter == "group") {
		filtered := make([]whatsapp.DiscoverableChannel, 0)
		for _, ch := range channels {
			if ch.Type == typeFilter {
				filtered = append(filtered, ch)
			}
		}
		channels = filtered
	}

	respondJSON(w, http.StatusOK, channels)
}

// Channels API

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

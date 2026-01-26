package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/gmail"
)

// Gmail Status API

func (s *Server) handleGmailStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"connected":  false,
		"enabled":    false,
		"message":    "Gmail not configured",
		"has_scopes": false,
	}

	// Check if Gmail client is available and authenticated
	if s.gmailClient != nil && s.gmailClient.IsAuthenticated() {
		status["connected"] = true
		status["has_scopes"] = true
		status["message"] = "Connected"
	} else if s.gcalClient != nil && s.gcalClient.IsAuthenticated() {
		// GCal is connected but Gmail might need re-authorization for new scopes
		status["connected"] = false
		status["has_scopes"] = false
		status["message"] = "Gmail requires re-authorization. Please reconnect Google account to grant Gmail access."
	}

	// Get settings
	settings, err := s.db.GetGmailSettings()
	if err == nil && settings != nil {
		status["enabled"] = settings.Enabled
		status["poll_interval_minutes"] = settings.PollIntervalMinutes
		if settings.LastPollAt != nil {
			status["last_poll_at"] = settings.LastPollAt
		}
	}

	respondJSON(w, http.StatusOK, status)
}

func (s *Server) handleGmailSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.db.GetGmailSettings()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, settings)
}

func (s *Server) handleUpdateGmailSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled             bool `json:"enabled"`
		PollIntervalMinutes int  `json:"poll_interval_minutes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Validate poll interval
	if req.PollIntervalMinutes < 1 {
		req.PollIntervalMinutes = 5
	}
	if req.PollIntervalMinutes > 60 {
		req.PollIntervalMinutes = 60
	}

	if err := s.db.UpdateGmailSettings(req.Enabled, req.PollIntervalMinutes); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	settings, _ := s.db.GetGmailSettings()
	respondJSON(w, http.StatusOK, settings)
}

// Gmail Discovery API

func (s *Server) handleDiscoverGmailCategories(w http.ResponseWriter, r *http.Request) {
	if s.gmailClient == nil || !s.gmailClient.IsAuthenticated() {
		respondError(w, http.StatusServiceUnavailable, "Gmail not connected")
		return
	}

	categories, err := s.gmailClient.DiscoverCategories()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Enrich with tracking status
	for i := range categories {
		source, err := s.db.GetEmailSourceByIdentifier(database.EmailSourceTypeCategory, categories[i].ID)
		if err == nil && source != nil {
			// Add tracking info (would need to extend the type, for now we use a map)
		}
	}

	respondJSON(w, http.StatusOK, categories)
}

func (s *Server) handleDiscoverGmailSenders(w http.ResponseWriter, r *http.Request) {
	if s.gmailClient == nil || !s.gmailClient.IsAuthenticated() {
		respondError(w, http.StatusServiceUnavailable, "Gmail not connected")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	senders, err := s.gmailClient.DiscoverSenders(limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, senders)
}

func (s *Server) handleDiscoverGmailDomains(w http.ResponseWriter, r *http.Request) {
	if s.gmailClient == nil || !s.gmailClient.IsAuthenticated() {
		respondError(w, http.StatusServiceUnavailable, "Gmail not connected")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	domains, err := s.gmailClient.DiscoverDomains(limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, domains)
}

// Email Sources API (similar to WhatsApp channels)

func (s *Server) handleListEmailSources(w http.ResponseWriter, r *http.Request) {
	typeFilter := r.URL.Query().Get("type")

	var sources []*database.EmailSource
	var err error

	if typeFilter != "" {
		sources, err = s.db.ListEmailSourcesByType(database.EmailSourceType(typeFilter))
	} else {
		sources, err = s.db.ListEmailSources()
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert to response format that includes tracking info for discovery
	type SourceResponse struct {
		*database.EmailSource
		IsTracked bool `json:"is_tracked"`
	}

	response := make([]SourceResponse, len(sources))
	for i, s := range sources {
		response[i] = SourceResponse{
			EmailSource: s,
			IsTracked:   true, // All sources in DB are tracked
		}
	}

	respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleCreateEmailSource(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type       string `json:"type"`       // "category", "sender", "domain"
		Identifier string `json:"identifier"` // e.g., "CATEGORY_PRIMARY", "user@example.com", "example.com"
		Name       string `json:"name"`       // Display name
		CalendarID string `json:"calendar_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Type == "" || req.Identifier == "" || req.Name == "" {
		respondError(w, http.StatusBadRequest, "type, identifier, and name are required")
		return
	}

	// Validate type
	sourceType := database.EmailSourceType(req.Type)
	if sourceType != database.EmailSourceTypeCategory &&
		sourceType != database.EmailSourceTypeSender &&
		sourceType != database.EmailSourceTypeDomain {
		respondError(w, http.StatusBadRequest, "type must be 'category', 'sender', or 'domain'")
		return
	}

	// Check if already exists
	existing, _ := s.db.GetEmailSourceByIdentifier(sourceType, req.Identifier)
	if existing != nil {
		respondError(w, http.StatusConflict, "email source already exists")
		return
	}

	source, err := s.db.CreateEmailSource(sourceType, req.Identifier, req.Name)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Update calendar ID if provided
	if req.CalendarID != "" {
		s.db.UpdateEmailSource(source.ID, source.Name, req.CalendarID, source.Enabled)
		source, _ = s.db.GetEmailSourceByID(source.ID)
	}

	respondJSON(w, http.StatusCreated, source)
}

func (s *Server) handleGetEmailSource(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	source, err := s.db.GetEmailSourceByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if source == nil {
		respondError(w, http.StatusNotFound, "email source not found")
		return
	}

	respondJSON(w, http.StatusOK, source)
}

func (s *Server) handleUpdateEmailSource(w http.ResponseWriter, r *http.Request) {
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

	if err := s.db.UpdateEmailSource(id, req.Name, req.CalendarID, req.Enabled); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	source, _ := s.db.GetEmailSourceByID(id)
	respondJSON(w, http.StatusOK, source)
}

func (s *Server) handleDeleteEmailSource(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := s.db.DeleteEmailSource(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// SetGmailClient sets the Gmail client after OAuth authentication
func (s *Server) SetGmailClient(client *gmail.Client) {
	s.gmailClient = client
}

package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/omriShneor/project_alfred/internal/database"
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

	respondJSON(w, http.StatusOK, map[string]interface{}{"sources": response})
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

// Top Contacts API - returns cached top contacts for quick discovery

type topContactResponse struct {
	Email      string `json:"email"`
	Name       string `json:"name"`
	EmailCount int    `json:"email_count"`
	IsTracked  bool   `json:"is_tracked"`
	SourceID   int64  `json:"source_id,omitempty"`
}

func (s *Server) handleGetTopContacts(w http.ResponseWriter, r *http.Request) {
	contacts, err := s.db.GetTopContacts(8)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get tracked sender sources to mark which contacts are already tracked
	senderSources, _ := s.db.ListEmailSourcesByType(database.EmailSourceTypeSender)
	trackedEmails := make(map[string]int64)
	for _, src := range senderSources {
		trackedEmails[src.Identifier] = src.ID
	}

	response := make([]topContactResponse, len(contacts))
	for i, c := range contacts {
		response[i] = topContactResponse{
			Email:      c.Email,
			Name:       c.Name,
			EmailCount: c.EmailCount,
			IsTracked:  trackedEmails[c.Email] > 0,
			SourceID:   trackedEmails[c.Email],
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"contacts": response})
}

// Custom Source API - validates and creates a custom email or domain source

func (s *Server) handleAddCustomSource(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Value      string `json:"value"`       // Email address or domain
		CalendarID string `json:"calendar_id"` // Target calendar
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Value == "" {
		respondError(w, http.StatusBadRequest, "value is required")
		return
	}

	// Determine type and validate
	var sourceType database.EmailSourceType
	var identifier, name string

	// Check if it's an email address
	if isValidEmail(req.Value) {
		sourceType = database.EmailSourceTypeSender
		identifier = strings.ToLower(req.Value)
		name = identifier
	} else if isValidDomain(req.Value) {
		sourceType = database.EmailSourceTypeDomain
		// Remove @ prefix if present
		identifier = strings.ToLower(strings.TrimPrefix(req.Value, "@"))
		name = identifier
	} else {
		respondError(w, http.StatusBadRequest, "Invalid email address or domain. Use format: user@domain.com or domain.com")
		return
	}

	// Check if already exists
	existing, _ := s.db.GetEmailSourceByIdentifier(sourceType, identifier)
	if existing != nil {
		respondError(w, http.StatusConflict, "Already tracking this source")
		return
	}

	// Create source
	source, err := s.db.CreateEmailSource(sourceType, identifier, name)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Update calendar ID if provided
	calendarID := req.CalendarID
	if calendarID == "" {
		calendarID = "primary"
	}
	s.db.UpdateEmailSource(source.ID, source.Name, calendarID, source.Enabled)
	source, _ = s.db.GetEmailSourceByID(source.ID)

	respondJSON(w, http.StatusCreated, source)
}

// isValidEmail checks if a string is a valid email address
func isValidEmail(s string) bool {
	// Basic email validation
	if s == "" || !strings.Contains(s, "@") || !strings.Contains(s, ".") {
		return false
	}
	parts := strings.Split(s, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	// Check domain part has a dot
	if !strings.Contains(parts[1], ".") {
		return false
	}
	return true
}

// isValidDomain checks if a string is a valid domain name
func isValidDomain(s string) bool {
	s = strings.TrimPrefix(s, "@")
	if s == "" || strings.Contains(s, " ") || strings.Contains(s, "@") {
		return false
	}
	// Must have at least one dot
	if !strings.Contains(s, ".") {
		return false
	}
	// Basic check for valid domain characters
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '-') {
			return false
		}
	}
	return true
}


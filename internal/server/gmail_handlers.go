package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
)

// Gmail Status API

func (s *Server) handleGmailStatus(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	status := map[string]interface{}{
		"connected":  false,
		"enabled":    false,
		"message":    "Gmail not configured",
		"has_scopes": false,
	}

	// Check if user has Gmail scope granted
	hasGmailScope := false
	if s.authService != nil && userID != 0 {
		hasGmailScope, _ = s.authService.HasGmailScope(userID)
	}

	status["has_scopes"] = hasGmailScope

	if !hasGmailScope {
		// User hasn't granted Gmail access yet
		status["message"] = "Gmail access not authorized. Please connect Gmail to scan emails."
	} else {
		// User has scope - in multi-user mode, we consider this "connected"
		// Per-user Gmail clients are created on-demand by the worker
		status["connected"] = true
		status["message"] = "Connected"
	}

	// Get settings
	settings, err := s.db.GetGmailSettings(userID)
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
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	typeFilter := r.URL.Query().Get("type")

	var sources []*database.EmailSource

	if typeFilter != "" {
		sources, err = s.db.ListEmailSourcesByType(userID, database.EmailSourceType(typeFilter))
	} else {
		sources, err = s.db.ListEmailSources(userID)
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
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if s.authService == nil {
		respondError(w, http.StatusForbidden, "gmail access not authorized")
		return
	}
	hasGmailScope, _ := s.authService.HasGmailScope(userID)
	if !hasGmailScope {
		respondError(w, http.StatusForbidden, "gmail access not authorized")
		return
	}

	var req struct {
		Type       string `json:"type"`       // "category", "sender", "domain"
		Identifier string `json:"identifier"` // e.g., "CATEGORY_PRIMARY", "user@example.com", "example.com"
		Name       string `json:"name"`       // Display name
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

	identifier := req.Identifier
	name := req.Name
	if sourceType == database.EmailSourceTypeSender {
		identifier = strings.ToLower(strings.TrimSpace(identifier))
		if identifier != "" {
			name = identifier
		}
	} else if sourceType == database.EmailSourceTypeDomain {
		identifier = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(identifier, "@")))
		if identifier != "" {
			name = identifier
		}
	}

	// Check if already exists
	existing, _ := s.db.GetEmailSourceByIdentifier(userID, sourceType, identifier)
	if existing != nil {
		respondError(w, http.StatusConflict, "email source already exists")
		return
	}

	source, err := s.db.CreateEmailSource(userID, sourceType, identifier, name)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.startEmailSourceBackfill(userID, source)

	respondJSON(w, http.StatusCreated, source)
}

func (s *Server) handleUpdateEmailSource(w http.ResponseWriter, r *http.Request) {
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

	if err := s.db.UpdateEmailSourceForUser(userID, id, req.Name, req.Enabled); err != nil {
		if errors.Is(err, database.ErrEmailSourceNotFound) {
			respondError(w, http.StatusNotFound, "source not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	source, err := s.db.GetEmailSourceByIDForUser(userID, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if source == nil {
		respondError(w, http.StatusNotFound, "source not found")
		return
	}

	respondJSON(w, http.StatusOK, source)
}

func (s *Server) handleDeleteEmailSource(w http.ResponseWriter, r *http.Request) {
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

	if err := s.db.DeleteEmailSourceForUser(userID, id); err != nil {
		if errors.Is(err, database.ErrEmailSourceNotFound) {
			respondError(w, http.StatusNotFound, "source not found")
			return
		}
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
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	contacts, err := s.db.GetTopContacts(userID, 8)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if len(contacts) == 0 {
		if s.userServiceManager != nil {
			worker := s.userServiceManager.GetGmailWorkerForUser(userID)
			if worker != nil {
				worker.RefreshContacts()
				time.Sleep(2 * time.Second)
				contacts, _ = s.db.GetTopContacts(userID, 8)
			}
		}
	}

	// Get tracked sender sources to mark which contacts are already tracked
	senderSources, _ := s.db.ListEmailSourcesByType(userID, database.EmailSourceTypeSender)
	trackedEmails := make(map[string]int64)
	for _, src := range senderSources {
		// Only include enabled sources
		if src.Enabled {
			trackedEmails[src.Identifier] = src.ID
		}
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

// Contact Search API - searches all cached contacts by name or email

func (s *Server) handleGmailContactSearch(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if len(query) < 2 {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"contacts": []topContactResponse{},
		})
		return
	}

	contacts, err := s.db.SearchGoogleContacts(userID, query, 10)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get tracked sender sources to mark which contacts are already tracked
	senderSources, _ := s.db.ListEmailSourcesByType(userID, database.EmailSourceTypeSender)
	trackedEmails := make(map[string]int64)
	for _, src := range senderSources {
		if src.Enabled {
			trackedEmails[src.Identifier] = src.ID
		}
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

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"contacts": response,
	})
}

// Custom Source API - validates and creates a custom email or domain source

func (s *Server) handleAddCustomSource(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if s.authService == nil {
		respondError(w, http.StatusForbidden, "gmail access not authorized")
		return
	}
	hasGmailScope, _ := s.authService.HasGmailScope(userID)
	if !hasGmailScope {
		respondError(w, http.StatusForbidden, "gmail access not authorized")
		return
	}

	var req struct {
		Value string `json:"value"` // Email address or domain
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
	existing, _ := s.db.GetEmailSourceByIdentifier(userID, sourceType, identifier)
	if existing != nil {
		respondError(w, http.StatusConflict, "Already tracking this source")
		return
	}

	// Create source
	source, err := s.db.CreateEmailSource(userID, sourceType, identifier, name)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.startEmailSourceBackfill(userID, source)

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

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/omriShneor/project_alfred/internal/auth"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/gcal"
	"github.com/omriShneor/project_alfred/internal/source"
	"github.com/omriShneor/project_alfred/internal/whatsapp"
)

// getUserID extracts the authenticated user's ID from the request context.
// Returns 0 if no user is authenticated (for development/testing).
func getUserID(r *http.Request) int64 {
	return auth.GetUserID(r.Context())
}

// Health Check

func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	if err := s.db.Ping(); err != nil {
		respondError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}

	// Build status response
	status := map[string]interface{}{
		"status":   "healthy",
		"whatsapp": "disconnected",
		"telegram": "disconnected",
		"gcal":     "disconnected",
	}

	if s.waClient != nil && s.waClient.IsLoggedIn() {
		status["whatsapp"] = "connected"
	}

	if s.tgClient != nil && s.tgClient.IsConnected() {
		status["telegram"] = "connected"
	}

	if s.credentialsFile != "" {
		status["gcal"] = "configured"
	}

	respondJSON(w, http.StatusOK, status)
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
	if s.waClient == nil || s.waClient.WAClient == nil {
		// Return empty contacts if WhatsApp not connected
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"contacts": []TopContactResponse{},
		})
		return
	}

	// Get top contacts by ACTUAL message count (from channels.total_message_count)
	channels, err := s.db.GetTopChannelsByMessageCount(source.SourceTypeWhatsApp, 8)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// If no stats available yet, fall back to discoverable channels (recent chats from WhatsApp)
	if len(channels) == 0 {
		discoverableChannels, err := s.waClient.GetDiscoverableChannels()
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
				// Check if already tracked
				existingChannel, _ := s.db.GetChannelByIdentifier(ch.Identifier)
				// Format phone number as secondary label
				phone := strings.TrimSuffix(ch.Identifier, "@s.whatsapp.net")
				response[i] = TopContactResponse{
					Identifier:     ch.Identifier,
					Name:           ch.Name,
					SecondaryLabel: "+" + phone,
					MessageCount:   0, // No message count from discovery
					IsTracked:      existingChannel != nil,
					Type:           ch.Type,
				}
				if existingChannel != nil {
					response[i].ChannelID = &existingChannel.ID
				}
			}
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"contacts": response,
			})
			return
		}
	}

	// Convert to response format using accurate message counts
	response := make([]TopContactResponse, len(channels))
	for i, c := range channels {
		// Format phone number as secondary label
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

// Helper function
func replaceAll(s, old, new string) string {
	result := s
	for i := 0; i < len(result); i++ {
		if i+len(old) <= len(result) && result[i:i+len(old)] == old {
			result = result[:i] + new + result[i+len(old):]
		}
	}
	return result
}

// Channels API

func (s *Server) handleListChannels(w http.ResponseWriter, r *http.Request) {
	// Note: type filter removed - only contacts (sender type) are supported now
	channels, err := s.db.ListChannels()
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

func (s *Server) handleUpdateChannel(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) handleDeleteChannel(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
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

// Google Calendar API

// handleClearGoogleTokens clears Google tokens for debugging (development only)
func (s *Server) handleClearGoogleTokens(w http.ResponseWriter, r *http.Request) {
	_, err := s.db.Exec("DELETE FROM google_tokens")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to clear tokens: "+err.Error())
		return
	}
	fmt.Printf("[Clear Tokens] Cleared all Google tokens\n")
	respondJSON(w, http.StatusOK, map[string]string{"status": "tokens_cleared"})
}

func (s *Server) handleGCalStatus(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	fmt.Printf("[GCal Status] Checking status for user %d\n", userID)

	status := map[string]interface{}{
		"connected":  false,
		"message":    "Not configured",
		"has_scopes": false,
	}

	if s.credentialsFile == "" {
		status["message"] = "Google Calendar not configured. Check credentials.json."
		respondJSON(w, http.StatusOK, status)
		return
	}

	if userID == 0 {
		status["message"] = "Authentication required"
		respondJSON(w, http.StatusOK, status)
		return
	}

	// Get per-user gcal client
	userGCalClient := s.getGCalClientForUser(userID)
	fmt.Printf("[GCal Status] Client retrieved: %v\n", userGCalClient != nil)

	// Check if client is authenticated (has token)
	if userGCalClient != nil {
		isAuth := userGCalClient.IsAuthenticated()
		fmt.Printf("[GCal Status] IsAuthenticated: %v\n", isAuth)
		if isAuth {
			status["connected"] = true
			status["message"] = "Connected"
			status["has_scopes"] = true
		} else {
			status["message"] = "Calendar access not authorized. Please connect Google Calendar."
		}
	} else {
		fmt.Printf("[GCal Status] Client is nil\n")
		status["message"] = "Calendar access not authorized. Please connect Google Calendar."
	}

	respondJSON(w, http.StatusOK, status)
}

func (s *Server) handleGCalListCalendars(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	userGCalClient := s.getGCalClientForUser(userID)

	if userGCalClient == nil || !userGCalClient.IsAuthenticated() {
		// Return empty array instead of error - allows UI to gracefully handle missing GCal
		respondJSON(w, http.StatusOK, []interface{}{})
		return
	}

	calendars, err := userGCalClient.ListCalendars()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, calendars)
}

func (s *Server) handleListTodayEvents(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	userGCalClient := s.getGCalClientForUser(userID)

	if userGCalClient == nil || !userGCalClient.IsAuthenticated() {
		respondError(w, http.StatusServiceUnavailable, "Google Calendar not connected")
		return
	}

	// Use primary calendar by default, or allow override via query param
	calendarID := r.URL.Query().Get("calendar_id")
	if calendarID == "" {
		calendarID = "primary"
	}

	events, err := userGCalClient.ListTodayEvents(calendarID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, events)
}

// TodayEventResponse represents a unified event format for Today's Schedule
type TodayEventResponse struct {
	ID          string `json:"id"`
	Summary     string `json:"summary"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location,omitempty"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	AllDay      bool   `json:"all_day"`
	CalendarID  string `json:"calendar_id"`
	Source      string `json:"source"` // "alfred", "google", "outlook"
}

// handleListMergedTodayEvents returns merged events from Alfred Calendar + external calendars
// This is the primary endpoint for Today's Schedule
func (s *Server) handleListMergedTodayEvents(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var events []TodayEventResponse

	// Get feature settings to check which calendars are enabled
	settings, err := s.db.GetFeatureSettings(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Track Google Event IDs to avoid duplicates
	seenGoogleIDs := make(map[string]bool)

	// 1. Always get Alfred Calendar events (local database)
	alfredEvents, err := s.db.GetTodayEvents(userID)
	if err != nil {
		// Log error but don't fail - Alfred events are best-effort
		fmt.Printf("Warning: failed to get Alfred events: %v\n", err)
		alfredEvents = nil
	}

	for _, e := range alfredEvents {
		endTime := e.StartTime.Add(1 * time.Hour) // Default 1 hour duration
		if e.EndTime != nil {
			endTime = *e.EndTime
		}

		event := TodayEventResponse{
			ID:          fmt.Sprintf("alfred-%d", e.ID),
			Summary:     e.Title,
			Description: e.Description,
			Location:    e.Location,
			StartTime:   e.StartTime.Format(time.RFC3339),
			EndTime:     endTime.Format(time.RFC3339),
			AllDay:      false,
			CalendarID:  "alfred",
			Source:      "alfred",
		}
		events = append(events, event)

		// Track if this event is synced to Google
		if e.GoogleEventID != nil {
			seenGoogleIDs[*e.GoogleEventID] = true
		}
	}

	// 2. Get Google Calendar events if enabled and connected
	userGCalClient := s.getGCalClientForUser(userID)
	if settings.GoogleCalendarEnabled && userGCalClient != nil && userGCalClient.IsAuthenticated() {
		gcalEvents, err := userGCalClient.ListTodayEvents("primary")
		if err != nil {
			// Log error but don't fail - Google events are best-effort
			fmt.Printf("Warning: failed to get Google Calendar events: %v\n", err)
		} else {
			for _, ge := range gcalEvents {
				// Skip if already seen (synced from Alfred)
				if seenGoogleIDs[ge.ID] {
					continue
				}

				event := TodayEventResponse{
					ID:          ge.ID,
					Summary:     ge.Summary,
					Description: ge.Description,
					Location:    ge.Location,
					StartTime:   ge.StartTime.Format(time.RFC3339),
					EndTime:     ge.EndTime.Format(time.RFC3339),
					AllDay:      ge.AllDay,
					CalendarID:  ge.CalendarID,
					Source:      "google",
				}
				events = append(events, event)
			}
		}
	}

	// 3. Sort events by start time using standard library
	sort.Slice(events, func(i, j int) bool {
		return events[i].StartTime < events[j].StartTime
	})

	respondJSON(w, http.StatusOK, events)
}

func (s *Server) handleGCalConnect(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == 0 {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	if s.credentialsFile == "" {
		respondError(w, http.StatusServiceUnavailable, "Google Calendar not configured. Check credentials.json.")
		return
	}

	// Get or create per-user gcal client (needed for auth URL generation)
	userGCalClient := s.getGCalClientForUser(userID)
	if userGCalClient == nil {
		respondError(w, http.StatusInternalServerError, "Failed to create Google Calendar client")
		return
	}

	// Check if a custom redirect URI is provided (for mobile deep links)
	var req struct {
		RedirectURI string `json:"redirect_uri"` // e.g., "alfred://oauth/callback"
	}
	// Try to decode, but don't fail if no body (for backward compatibility)
	_ = json.NewDecoder(r.Body).Decode(&req)

	// If a custom redirect URI is provided, use it and return the auth URL
	// The mobile app will handle the callback via deep link and call /api/gcal/callback
	if req.RedirectURI != "" {
		authURL := userGCalClient.GetAuthURLWithRedirect(req.RedirectURI)
		respondJSON(w, http.StatusOK, map[string]string{
			"auth_url":     authURL,
			"redirect_uri": req.RedirectURI,
			"message":      "Open this URL to authorize Google Calendar access. After authorization, the app will handle the callback.",
		})
		return
	}

	// Return the auth URL for the frontend to open (browser flow)
	authURL := userGCalClient.GetAuthURL()

	respondJSON(w, http.StatusOK, map[string]string{
		"auth_url": authURL,
		"message":  "Open this URL to authorize Google Calendar access",
	})
}

// handleOAuthCallback handles the OAuth callback from Google (browser redirect)
// This endpoint receives the callback from Google after user authorizes.
// It redirects back to the mobile app via deep link with the authorization code.
// The app then sends the code to /api/gcal/callback for exchange.
func (s *Server) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		respondError(w, http.StatusBadRequest, "No authorization code received")
		return
	}

	// Redirect back to the app using deep link with the code
	// The app will receive this and call /api/gcal/callback to exchange the code
	deepLink := fmt.Sprintf("alfred://oauth/callback?code=%s", code)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>Connecting...</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #f5f5f5; }
        .container { text-align: center; padding: 40px; background: white; border-radius: 16px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #27ae60; margin-bottom: 16px; }
        p { color: #666; margin-bottom: 24px; }
    </style>
    <script>
        // Redirect to app with authorization code
        window.location.href = '%s';
    </script>
</head>
<body>
    <div class="container">
        <h1>Authorization Complete!</h1>
        <p>Redirecting to Alfred...</p>
        <p><a href="%s">Tap here if not redirected</a></p>
    </div>
</body>
</html>`, deepLink, deepLink)
}

// handleGCalExchangeCode handles OAuth code exchange from mobile apps
// Mobile apps receive the code via deep link and send it to this endpoint
func (s *Server) handleGCalExchangeCode(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	fmt.Printf("[OAuth Exchange] Request received for user %d\n", userID)

	if userID == 0 {
		fmt.Printf("[OAuth Exchange] ERROR: No user ID found\n")
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	if s.credentialsFile == "" {
		fmt.Printf("[OAuth Exchange] ERROR: Google Calendar not configured\n")
		respondError(w, http.StatusServiceUnavailable, "Google Calendar not configured")
		return
	}

	var req struct {
		Code string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("[OAuth Exchange] ERROR: Invalid JSON: %v\n", err)
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	codePreview := req.Code
	if len(codePreview) > 10 {
		codePreview = codePreview[:10] + "..."
	}
	fmt.Printf("[OAuth Exchange] Code (first 10 chars): %s\n", codePreview)

	if req.Code == "" {
		fmt.Printf("[OAuth Exchange] ERROR: Code is empty\n")
		respondError(w, http.StatusBadRequest, "code is required")
		return
	}

	// Create per-user gcal client for token exchange (saves token to database with userID)
	userGCalClient, err := gcal.NewClientForUser(userID, s.credentialsFile, s.db)
	if err != nil {
		fmt.Printf("[OAuth Exchange] ERROR: Failed to create gcal client: %v\n", err)
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create gcal client: %v", err))
		return
	}

	// Exchange the code using default HTTPS callback
	fmt.Printf("[OAuth Exchange] Exchanging code for user %d using default HTTPS callback...\n", userID)
	if err := userGCalClient.ExchangeCode(context.Background(), req.Code); err != nil {
		fmt.Printf("[OAuth Exchange] ERROR: Failed to exchange code: %v\n", err)
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to exchange code: %v", err))
		return
	}

	fmt.Printf("[OAuth Exchange] SUCCESS: Token saved for user %d\n", userID)

	// Update onboarding state
	if s.onboardingState != nil {
		s.onboardingState.SetGCalStatus("connected")
	}

	fmt.Printf("Google Calendar connected successfully for user %d via mobile!\n", userID)

	// Re-initialize Gmail client with the new token
	if err := s.initializeGmailClient(); err != nil {
		fmt.Printf("Warning: Failed to initialize Gmail client: %v\n", err)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"connected": true,
		"message":   "Google Calendar connected successfully",
	})
}

// handleGCalDisconnect disconnects Google Calendar and clears token
func (s *Server) handleGCalDisconnect(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == 0 {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	userGCalClient := s.getGCalClientForUser(userID)
	if userGCalClient == nil {
		respondError(w, http.StatusServiceUnavailable, "Google Calendar not configured")
		return
	}

	if err := userGCalClient.Disconnect(); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if s.onboardingState != nil {
		s.onboardingState.SetGCalStatus("disconnected")
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "disconnected",
		"message": "Google Calendar disconnected",
	})
}

// Events API

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	statusFilter := r.URL.Query().Get("status")
	channelIDStr := r.URL.Query().Get("channel_id")

	var status *database.EventStatus
	if statusFilter != "" {
		s := database.EventStatus(statusFilter)
		status = &s
	}

	var channelID *int64
	if channelIDStr != "" {
		id, err := strconv.ParseInt(channelIDStr, 10, 64)
		if err == nil {
			channelID = &id
		}
	}

	events, err := s.db.ListEvents(userID, status, channelID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, events)
}

func (s *Server) handleGetEvent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	event, err := s.db.GetEventByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "event not found")
		return
	}

	// Include message context if available
	response := map[string]interface{}{
		"event": event,
	}

	if event.OriginalMsgID != nil {
		msg, err := s.db.GetMessageByID(*event.OriginalMsgID)
		if err == nil {
			response["trigger_message"] = msg
		}
	}

	respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleConfirmEvent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	event, err := s.db.GetEventByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "event not found")
		return
	}

	if event.Status != database.EventStatusPending {
		respondError(w, http.StatusBadRequest, "event is not pending")
		return
	}

	// Check if sync is enabled and Google Calendar is connected
	userID := getUserID(r)
	gcalSettings, _ := s.db.GetGCalSettings(userID)
	userGCalClient := s.getGCalClientForUser(userID)
	shouldSync := gcalSettings != nil && gcalSettings.SyncEnabled && userGCalClient != nil && userGCalClient.IsAuthenticated()

	// If not syncing to Google Calendar, just confirm the event locally
	if !shouldSync {
		var newStatus database.EventStatus
		if event.ActionType == database.EventActionDelete {
			newStatus = database.EventStatusDeleted
		} else {
			newStatus = database.EventStatusConfirmed
		}

		if err := s.db.UpdateEventStatus(id, newStatus); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to confirm event: %v", err))
			return
		}

		updatedEvent, _ := s.db.GetEventByID(id)
		respondJSON(w, http.StatusOK, updatedEvent)
		return
	}

	// Sync to Google Calendar
	var googleEventID string
	switch event.ActionType {
	case database.EventActionCreate:
		// Create event in Google Calendar
		endTime := event.StartTime.Add(1 * 60 * 60 * 1000000000) // 1 hour default
		if event.EndTime != nil {
			endTime = *event.EndTime
		}

		// Extract attendee emails
		attendeeEmails := make([]string, len(event.Attendees))
		for i, a := range event.Attendees {
			attendeeEmails[i] = a.Email
		}

		googleEventID, err = userGCalClient.CreateEvent(event.CalendarID, gcal.EventInput{
			Summary:     event.Title,
			Description: event.Description,
			Location:    event.Location,
			StartTime:   event.StartTime,
			EndTime:     endTime,
			Attendees:   attendeeEmails,
		})
		if err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create calendar event: %v", err))
			return
		}

		// Update database with Google event ID
		if err := s.db.UpdateEventGoogleID(id, googleEventID); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update event: %v", err))
			return
		}

	case database.EventActionUpdate:
		// If no Google event ID, just confirm locally (event was created before sync was enabled)
		if event.GoogleEventID == nil {
			if err := s.db.UpdateEventStatus(id, database.EventStatusConfirmed); err != nil {
				respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to confirm event: %v", err))
				return
			}
			break
		}

		endTime := event.StartTime.Add(1 * 60 * 60 * 1000000000)
		if event.EndTime != nil {
			endTime = *event.EndTime
		}

		// Extract attendee emails
		updateAttendeeEmails := make([]string, len(event.Attendees))
		for i, a := range event.Attendees {
			updateAttendeeEmails[i] = a.Email
		}

		err = userGCalClient.UpdateEvent(event.CalendarID, *event.GoogleEventID, gcal.EventInput{
			Summary:     event.Title,
			Description: event.Description,
			Location:    event.Location,
			StartTime:   event.StartTime,
			EndTime:     endTime,
			Attendees:   updateAttendeeEmails,
		})
		if err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update calendar event: %v", err))
			return
		}

		if err := s.db.UpdateEventStatus(id, database.EventStatusSynced); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update event status: %v", err))
			return
		}

	case database.EventActionDelete:
		// If no Google event ID, just mark as deleted locally
		if event.GoogleEventID == nil {
			if err := s.db.UpdateEventStatus(id, database.EventStatusDeleted); err != nil {
				respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete event: %v", err))
				return
			}
			break
		}

		err = userGCalClient.DeleteEvent(event.CalendarID, *event.GoogleEventID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete calendar event: %v", err))
			return
		}

		if err := s.db.UpdateEventStatus(id, database.EventStatusDeleted); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update event status: %v", err))
			return
		}
	}

	// Get updated event
	updatedEvent, _ := s.db.GetEventByID(id)
	respondJSON(w, http.StatusOK, updatedEvent)
}

func (s *Server) handleUpdateEvent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	event, err := s.db.GetEventByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "event not found")
		return
	}

	if event.Status != database.EventStatusPending {
		respondError(w, http.StatusBadRequest, "can only edit pending events")
		return
	}

	var req struct {
		Title       string  `json:"title"`
		Description string  `json:"description"`
		StartTime   string  `json:"start_time"`
		EndTime     *string `json:"end_time"`
		Location    string  `json:"location"`
		Attendees   []struct {
			Email       string `json:"email"`
			DisplayName string `json:"display_name"`
		} `json:"attendees"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Title == "" {
		respondError(w, http.StatusBadRequest, "title is required")
		return
	}

	if req.StartTime == "" {
		respondError(w, http.StatusBadRequest, "start_time is required")
		return
	}

	// Parse start time
	startTime, err := parseEventTime(req.StartTime)
	if err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid start_time format: %v", err))
		return
	}

	// Parse end time if provided
	var endTime *time.Time
	if req.EndTime != nil && *req.EndTime != "" {
		et, err := parseEventTime(*req.EndTime)
		if err != nil {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid end_time format: %v", err))
			return
		}
		endTime = &et
	}

	if err := s.db.UpdatePendingEvent(id, req.Title, req.Description, startTime, endTime, req.Location); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Update attendees
	attendees := make([]database.Attendee, len(req.Attendees))
	for i, a := range req.Attendees {
		attendees[i] = database.Attendee{
			Email:       a.Email,
			DisplayName: a.DisplayName,
		}
	}
	if err := s.db.SetEventAttendees(id, attendees); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update attendees: %v", err))
		return
	}

	updatedEvent, _ := s.db.GetEventByID(id)
	respondJSON(w, http.StatusOK, updatedEvent)
}

// parseEventTime parses a time string in various formats
func parseEventTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	}

	for _, format := range formats {
		if t, err := time.ParseInLocation(format, s, time.Local); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unrecognized time format")
}

func (s *Server) handleRejectEvent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	event, err := s.db.GetEventByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "event not found")
		return
	}

	if event.Status != database.EventStatusPending {
		respondError(w, http.StatusBadRequest, "event is not pending")
		return
	}

	if err := s.db.UpdateEventStatus(id, database.EventStatusRejected); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

func (s *Server) handleGetChannelHistory(w http.ResponseWriter, r *http.Request) {
	channelID, err := strconv.ParseInt(r.PathValue("channelId"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 25
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	messages, err := s.db.GetMessageHistory(channelID, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, messages)
}

// WhatsApp API

func (s *Server) handleWhatsAppReconnect(w http.ResponseWriter, r *http.Request) {
	if s.waClient == nil {
		respondError(w, http.StatusServiceUnavailable, "WhatsApp client not initialized")
		return
	}

	if s.onboardingState == nil {
		respondError(w, http.StatusServiceUnavailable, "Onboarding state not initialized")
		return
	}

	// Trigger reconnect - use background context since request context will be cancelled
	go s.waClient.Reconnect(context.Background(), s.onboardingState)

	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "reconnecting",
		"message": "Reconnection initiated, new QR code will be generated",
	})
}

// handleWhatsAppStatus returns the WhatsApp connection status
func (s *Server) handleWhatsAppStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"connected": false,
		"message":   "Not connected",
	}

	if s.waClient != nil && s.waClient.IsLoggedIn() {
		status["connected"] = true
		status["message"] = "Connected"
	} else if s.waClient != nil {
		status["message"] = "Not authenticated"
	} else {
		status["message"] = "WhatsApp client not initialized"
	}

	respondJSON(w, http.StatusOK, status)
}

// handleWhatsAppPair generates a pairing code for phone-number-based WhatsApp linking
func (s *Server) handleWhatsAppPair(w http.ResponseWriter, r *http.Request) {
	if s.waClient == nil {
		respondError(w, http.StatusServiceUnavailable, "WhatsApp client not initialized")
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

	// Generate pairing code
	code, err := s.waClient.PairWithPhone(r.Context(), phone, s.onboardingState)
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
	if s.waClient == nil {
		respondError(w, http.StatusServiceUnavailable, "WhatsApp client not initialized")
		return
	}

	if err := s.waClient.Logout(); err != nil {
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

// Onboarding API Handlers

func (s *Server) handleOnboardingStatus(w http.ResponseWriter, r *http.Request) {
	if s.onboardingState == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"whatsapp": map[string]string{"status": "unknown"},
			"gcal":     map[string]interface{}{"status": "unknown", "configured": false},
			"complete": false,
		})
		return
	}

	respondJSON(w, http.StatusOK, s.onboardingState.GetStatus())
}

func (s *Server) handleOnboardingSSE(w http.ResponseWriter, r *http.Request) {
	if s.onboardingState == nil {
		respondError(w, http.StatusServiceUnavailable, "Onboarding not initialized")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Streaming not supported")
		return
	}

	// Subscribe to updates
	updates := s.onboardingState.Subscribe()
	defer s.onboardingState.Unsubscribe(updates)

	// Send initial status
	statusJSON := s.onboardingState.GetStatusJSON()
	fmt.Fprintf(w, "event: status\ndata: %s\n\n", statusJSON)
	flusher.Flush()

	// Stream updates
	for {
		select {
		case update, ok := <-updates:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", update.Type, update.Data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// Notification Preferences API

func (s *Server) handleGetNotificationPrefs(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	prefs, err := s.db.GetUserNotificationPrefs(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Determine push availability
	pushAvailable := false
	if s.notifyService != nil {
		pushAvailable = s.notifyService.IsPushAvailable()
	}

	// Include server-side availability info
	response := map[string]interface{}{
		"preferences": prefs,
		"available": map[string]bool{
			"email":   s.resendAPIKey != "",
			"push":    pushAvailable,
			"sms":     false,
			"webhook": false,
		},
	}
	respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleUpdateEmailPrefs(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		Enabled bool   `json:"enabled"`
		Address string `json:"address"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Enabled && req.Address == "" {
		respondError(w, http.StatusBadRequest, "email address required when enabling notifications")
		return
	}

	if err := s.db.UpdateEmailPrefs(userID, req.Enabled, req.Address); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	prefs, _ := s.db.GetUserNotificationPrefs(userID)
	respondJSON(w, http.StatusOK, prefs)
}

// handleRegisterPushToken stores the Expo push token from mobile app
func (s *Server) handleRegisterPushToken(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Token == "" {
		respondError(w, http.StatusBadRequest, "token is required")
		return
	}

	if err := s.db.UpdatePushToken(userID, req.Token); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	fmt.Printf("Push token registered: %s...\n", req.Token[:min(20, len(req.Token))])
	respondJSON(w, http.StatusOK, map[string]string{"status": "registered"})
}

// handleUpdatePushPrefs enables/disables push notifications
func (s *Server) handleUpdatePushPrefs(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := s.db.UpdatePushPrefs(userID, req.Enabled); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	prefs, _ := s.db.GetUserNotificationPrefs(userID)
	respondJSON(w, http.StatusOK, prefs)
}

// Google Calendar Settings API

// handleGetGCalSettings returns the Google Calendar settings for the current user
func (s *Server) handleGetGCalSettings(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	settings, err := s.db.GetGCalSettings(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, settings)
}

// handleUpdateGCalSettings updates the Google Calendar settings for the current user
func (s *Server) handleUpdateGCalSettings(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var req struct {
		SyncEnabled          bool   `json:"sync_enabled"`
		SelectedCalendarID   string `json:"selected_calendar_id"`
		SelectedCalendarName string `json:"selected_calendar_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Default to "primary" if no calendar selected
	if req.SelectedCalendarID == "" {
		req.SelectedCalendarID = "primary"
	}
	if req.SelectedCalendarName == "" {
		req.SelectedCalendarName = "Primary"
	}

	if err := s.db.UpdateGCalSettings(userID, req.SyncEnabled, req.SelectedCalendarID, req.SelectedCalendarName); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	settings, _ := s.db.GetGCalSettings(userID)
	respondJSON(w, http.StatusOK, settings)
}

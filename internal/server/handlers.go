package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/gcal"
	"github.com/omriShneor/project_alfred/internal/whatsapp"
)

// serveStaticFile serves a static file from filesystem (dev mode) or embedded (production)
func (s *Server) serveStaticFile(w http.ResponseWriter, filename string) {
	var html []byte
	var err error

	if s.devMode {
		// In dev mode, read from filesystem for hot reloading
		path := filepath.Join("internal", "server", "static", filename)
		html, err = os.ReadFile(path)
	} else {
		// In production, use embedded files
		html, err = staticFiles.ReadFile("static/" + filename)
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to load %s", filename))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(html)
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
		"gcal":     "disconnected",
	}

	if s.waClient != nil && s.waClient.IsLoggedIn() {
		status["whatsapp"] = "connected"
	}

	if s.gcalClient != nil && s.gcalClient.IsAuthenticated() {
		status["gcal"] = "connected"
	}

	respondJSON(w, http.StatusOK, status)
}

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

// Admin Page

func (s *Server) handleAdminPage(w http.ResponseWriter, r *http.Request) {
	s.serveStaticFile(w, "admin.html")
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

// Google Calendar API

func (s *Server) handleGCalStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"connected": false,
		"message":   "Not configured",
	}

	if s.gcalClient == nil {
		status["message"] = "Google Calendar client not initialized. Check credentials.json."
		respondJSON(w, http.StatusOK, status)
		return
	}

	if s.gcalClient.IsAuthenticated() {
		status["connected"] = true
		status["message"] = "Connected"
	} else {
		status["message"] = "Not authenticated. Click Connect to authorize."
	}

	respondJSON(w, http.StatusOK, status)
}

func (s *Server) handleGCalListCalendars(w http.ResponseWriter, r *http.Request) {
	if s.gcalClient == nil || !s.gcalClient.IsAuthenticated() {
		respondError(w, http.StatusServiceUnavailable, "Google Calendar not connected")
		return
	}

	calendars, err := s.gcalClient.ListCalendars()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, calendars)
}

func (s *Server) handleGCalConnect(w http.ResponseWriter, r *http.Request) {
	if s.gcalClient == nil {
		respondError(w, http.StatusServiceUnavailable, "Google Calendar not configured. Check credentials.json.")
		return
	}

	// Check if we're using main server callback (production) or separate server (local)
	baseURL := os.Getenv("ALFRED_BASE_URL")
	if baseURL != "" {
		// Production mode: use main server's /oauth/callback endpoint
		s.oauthCodeChan = make(chan string, 1)

		// Listen for the OAuth code in a goroutine
		go func() {
			select {
			case code := <-s.oauthCodeChan:
				if err := s.gcalClient.ExchangeCode(context.Background(), code); err != nil {
					fmt.Printf("Failed to exchange OAuth code: %v\n", err)
					if s.onboardingState != nil {
						s.onboardingState.SetGCalError(fmt.Sprintf("Failed to authenticate: %v", err))
					}
					return
				}
				fmt.Println("Google Calendar connected successfully!")
				if s.onboardingState != nil {
					s.onboardingState.SetGCalStatus("connected")
				}
			case <-time.After(5 * time.Minute):
				fmt.Println("OAuth timeout - no callback received")
				if s.onboardingState != nil {
					s.onboardingState.SetGCalError("Authorization timeout. Please try again.")
				}
			}
		}()
	} else {
		// Local mode: start separate callback server
		redirectURL := fmt.Sprintf("http://localhost:%d/onboarding", s.port)
		codeChan, errChan := s.gcalClient.StartCallbackServer(context.Background(), redirectURL)

		go func() {
			select {
			case code := <-codeChan:
				if err := s.gcalClient.ExchangeCode(context.Background(), code); err != nil {
					fmt.Printf("Failed to exchange OAuth code: %v\n", err)
					if s.onboardingState != nil {
						s.onboardingState.SetGCalError(fmt.Sprintf("Failed to authenticate: %v", err))
					}
					return
				}
				fmt.Println("Google Calendar connected successfully!")
				if s.onboardingState != nil {
					s.onboardingState.SetGCalStatus("connected")
				}
			case err := <-errChan:
				fmt.Printf("OAuth callback error: %v\n", err)
				if s.onboardingState != nil {
					s.onboardingState.SetGCalError(fmt.Sprintf("Authorization failed: %v", err))
				}
			}
		}()
	}

	// Return the auth URL for the frontend to open
	authURL := s.gcalClient.GetAuthURL()

	respondJSON(w, http.StatusOK, map[string]string{
		"auth_url": authURL,
		"message":  "Open this URL to authorize Google Calendar access",
	})
}

// handleOAuthCallback handles the OAuth callback from Google (used in production)
func (s *Server) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		respondError(w, http.StatusBadRequest, "No authorization code received")
		return
	}

	// Send code to waiting goroutine
	if s.oauthCodeChan != nil {
		select {
		case s.oauthCodeChan <- code:
			// Code sent successfully
		default:
			// Channel full or closed, try direct exchange
			if err := s.gcalClient.ExchangeCode(context.Background(), code); err != nil {
				respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to exchange code: %v", err))
				return
			}
			if s.onboardingState != nil {
				s.onboardingState.SetGCalStatus("connected")
			}
		}
	} else {
		// No waiting goroutine, do direct exchange
		if s.gcalClient != nil {
			if err := s.gcalClient.ExchangeCode(context.Background(), code); err != nil {
				respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to exchange code: %v", err))
				return
			}
			if s.onboardingState != nil {
				s.onboardingState.SetGCalStatus("connected")
			}
		}
	}

	// Redirect to onboarding page
	baseURL := os.Getenv("ALFRED_BASE_URL")
	if baseURL != "" {
		http.Redirect(w, r, baseURL+"/onboarding", http.StatusFound)
	} else {
		http.Redirect(w, r, fmt.Sprintf("http://localhost:%d/onboarding", s.port), http.StatusFound)
	}
}

// Events Page

func (s *Server) handleEventsPage(w http.ResponseWriter, r *http.Request) {
	s.serveStaticFile(w, "events.html")
}

// Events API

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
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

	events, err := s.db.ListEvents(status, channelID)
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

	// Check if Google Calendar is connected
	if s.gcalClient == nil || !s.gcalClient.IsAuthenticated() {
		respondError(w, http.StatusServiceUnavailable, "Google Calendar not connected")
		return
	}

	// Execute the action based on event type
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

		googleEventID, err = s.gcalClient.CreateEvent(event.CalendarID, gcal.EventInput{
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
		if event.GoogleEventID == nil {
			respondError(w, http.StatusBadRequest, "no Google event ID to update")
			return
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

		err = s.gcalClient.UpdateEvent(event.CalendarID, *event.GoogleEventID, gcal.EventInput{
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
		if event.GoogleEventID == nil {
			respondError(w, http.StatusBadRequest, "no Google event ID to delete")
			return
		}

		err = s.gcalClient.DeleteEvent(event.CalendarID, *event.GoogleEventID)
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

// Onboarding Handlers

func (s *Server) handleOnboardingPage(w http.ResponseWriter, r *http.Request) {
	// If onboarding is complete, redirect to admin
	if s.onboardingState != nil && s.onboardingState.IsComplete() {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}

	s.serveStaticFile(w, "onboarding.html")
}

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

// Settings Page

func (s *Server) handleSettingsPage(w http.ResponseWriter, r *http.Request) {
	s.serveStaticFile(w, "settings.html")
}

// Notification Preferences API

func (s *Server) handleGetNotificationPrefs(w http.ResponseWriter, r *http.Request) {
	prefs, err := s.db.GetUserNotificationPrefs()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Include server-side availability info
	response := map[string]interface{}{
		"preferences": prefs,
		"available": map[string]bool{
			"email":   s.resendAPIKey != "",
			"push":    false,
			"sms":     false,
			"webhook": false,
		},
	}
	respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleUpdateEmailPrefs(w http.ResponseWriter, r *http.Request) {
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

	if err := s.db.UpdateEmailPrefs(req.Enabled, req.Address); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	prefs, _ := s.db.GetUserNotificationPrefs()
	respondJSON(w, http.StatusOK, prefs)
}

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
)

// Google Calendar API
func (s *Server) handleGCalStatus(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

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

	// Check if client is authenticated (has token) AND has Calendar scopes
	if userGCalClient != nil {
		isAuth := userGCalClient.IsAuthenticated()

		if isAuth {
			// Check if token has Calendar scopes (not just ProfileScopes)
			tokenInfo, err := s.db.GetGoogleTokenInfo(userID)
			if err != nil {
				status["message"] = "Error checking token scopes"
			} else if tokenInfo != nil && tokenInfo.HasToken {
				hasCalendarScope := false
				for _, scope := range tokenInfo.Scopes {
					if scope == "https://www.googleapis.com/auth/calendar" {
						hasCalendarScope = true
						break
					}
				}

				if hasCalendarScope {
					status["connected"] = true
					status["message"] = "Connected"
					status["has_scopes"] = true
				} else {
					status["message"] = "Calendar access not authorized. Please connect Google Calendar."
				}
			} else {
				status["message"] = "Calendar access not authorized. Please connect Google Calendar."
			}
		} else {
			status["message"] = "Calendar access not authorized. Please connect Google Calendar."
		}
	} else {
		status["message"] = "Calendar access not authorized. Please connect Google Calendar."
	}

	respondJSON(w, http.StatusOK, status)
}

func (s *Server) handleGCalListCalendars(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
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
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
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
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var events []TodayEventResponse

	// Track Google Event IDs to avoid duplicates
	seenGoogleIDs := make(map[string]bool)

	// 1. Always get Alfred Calendar events (local database)
	alfredEvents, err := s.db.GetTodayEvents(userID)
	if err != nil {
		// Ignore error - Alfred events are best-effort
		alfredEvents = nil
	}

	for _, e := range alfredEvents {
		endTime := e.StartTime.Add(1 * time.Hour) // Default 1 hour duration
		if e.EndTime != nil {
			endTime = *e.EndTime
		}

		eventSource := "alfred"
		switch e.ChannelSourceType {
		case "google_calendar":
			eventSource = "google"
		case "outlook_calendar":
			eventSource = "outlook"
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
			Source:      eventSource,
		}
		events = append(events, event)

		// Track if this event is synced to Google
		if e.GoogleEventID != nil {
			seenGoogleIDs[*e.GoogleEventID] = true
		}
	}

	// 2. Get Google Calendar events when the account is connected.
	// Prefer query param, otherwise user-selected calendar, then primary.
	selectedCalendarID := r.URL.Query().Get("calendar_id")
	if selectedCalendarID == "" {
		gcalSettings, err := s.db.GetGCalSettings(userID)
		if err == nil && gcalSettings.SelectedCalendarID != "" {
			selectedCalendarID = gcalSettings.SelectedCalendarID
		}
	}
	if selectedCalendarID == "" {
		selectedCalendarID = "primary"
	}

	userGCalClient := s.getGCalClientForUser(userID)
	if userGCalClient != nil && userGCalClient.IsAuthenticated() {
		gcalEvents, err := userGCalClient.ListTodayEvents(selectedCalendarID)
		if err != nil && selectedCalendarID != "primary" {
			// Fall back to primary if selected calendar is no longer accessible.
			gcalEvents, err = userGCalClient.ListTodayEvents("primary")
		}
		if err == nil {
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

// handleGCalDisconnect disconnects Google services (Calendar, Gmail, or both)
// Accepts optional "scope" parameter in request body to selectively disconnect
func (s *Server) handleGCalDisconnect(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if userID == 0 {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Parse request body for scope parameter
	var req struct {
		Scope string `json:"scope"` // "gmail", "calendar", or empty for all
	}

	// Try to parse JSON body, but don't fail if it's empty (backwards compatibility)
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	// If no specific scope provided, disconnect everything (backwards compatible)
	if req.Scope == "" {
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

		if s.userServiceManager != nil {
			s.userServiceManager.StopGCalWorkerForUser(userID)
			s.userServiceManager.StopGmailWorkerForUser(userID)
		}
		_ = s.db.SetGmailEnabled(userID, false)

		respondJSON(w, http.StatusOK, map[string]string{
			"status": "disconnected",
			"scope":  "all",
		})
		return
	}

	// Selective scope removal
	if req.Scope != "gmail" && req.Scope != "calendar" {
		respondError(w, http.StatusBadRequest, "scope must be 'gmail' or 'calendar'")
		return
	}

	if err := s.db.RemoveGoogleTokenScope(userID, req.Scope); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Clear client cache to force reinitialization with new scopes
	if req.Scope == "calendar" {
		if s.userServiceManager != nil {
			s.userServiceManager.StopGCalWorkerForUser(userID)
		}
		if s.onboardingState != nil {
			s.onboardingState.SetGCalStatus("disconnected")
		}
	}
	if req.Scope == "gmail" {
		if s.userServiceManager != nil {
			s.userServiceManager.StopGmailWorkerForUser(userID)
		}
		_ = s.db.SetGmailEnabled(userID, false)
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "disconnected",
		"scope":  req.Scope,
	})
}

func (s *Server) handleGetGCalSettings(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	settings, err := s.db.GetGCalSettings(userID)
	if err != nil {
		// Return default settings if not found
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"sync_enabled":           false,
			"selected_calendar_id":   "",
			"selected_calendar_name": "",
		})
		return
	}

	respondJSON(w, http.StatusOK, settings)
}

func (s *Server) handleUpdateGCalSettings(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req struct {
		SyncEnabled          bool   `json:"sync_enabled"`
		SelectedCalendarID   string `json:"selected_calendar_id"`
		SelectedCalendarName string `json:"selected_calendar_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := s.db.UpdateGCalSettings(userID, req.SyncEnabled, req.SelectedCalendarID, req.SelectedCalendarName); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update settings: %v", err))
		return
	}

	settings, _ := s.db.GetGCalSettings(userID)
	respondJSON(w, http.StatusOK, settings)
}

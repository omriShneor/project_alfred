package server

import (
	"encoding/json"
	"net/http"
)

// FeaturesResponse represents the response for GET /api/features
type FeaturesResponse struct {
	SmartCalendar SmartCalendarFeature `json:"smart_calendar"`
}

// SmartCalendarFeature represents the Smart Calendar feature settings
type SmartCalendarFeature struct {
	Enabled       bool                   `json:"enabled"`
	SetupComplete bool                   `json:"setup_complete"`
	Inputs        SmartCalendarInputs    `json:"inputs"`
	Calendars     SmartCalendarCalendars `json:"calendars"`
}

// SmartCalendarInputs represents the input settings for Smart Calendar
type SmartCalendarInputs struct {
	WhatsApp IntegrationStatus `json:"whatsapp"`
	Email    IntegrationStatus `json:"email"`
	SMS      IntegrationStatus `json:"sms"`
}

// SmartCalendarCalendars represents the calendar settings for Smart Calendar
type SmartCalendarCalendars struct {
	Alfred         IntegrationStatus `json:"alfred"` // Local Alfred calendar (always available)
	GoogleCalendar IntegrationStatus `json:"google_calendar"`
	Outlook        IntegrationStatus `json:"outlook"`
}

// IntegrationStatus represents the status of an integration
type IntegrationStatus struct {
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"` // "pending", "connecting", "available", "error"
}

// handleGetFeatures returns the current feature settings with integration status
func (s *Server) handleGetFeatures(w http.ResponseWriter, r *http.Request) {
	settings, err := s.db.GetFeatureSettings()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Determine integration statuses based on actual client connections
	whatsappStatus := "pending"
	if s.waClient != nil && s.waClient.IsLoggedIn() {
		whatsappStatus = "available"
	}

	googleCalendarStatus := "pending"
	if s.gcalClient != nil && s.gcalClient.IsAuthenticated() {
		googleCalendarStatus = "available"
	}

	emailStatus := "pending"
	if s.gmailClient != nil && s.gmailClient.IsAuthenticated() {
		emailStatus = "available"
	}

	response := FeaturesResponse{
		SmartCalendar: SmartCalendarFeature{
			Enabled:       settings.SmartCalendarEnabled,
			SetupComplete: settings.SmartCalendarSetupComplete,
			Inputs: SmartCalendarInputs{
				WhatsApp: IntegrationStatus{
					Enabled: settings.WhatsAppInputEnabled,
					Status:  whatsappStatus,
				},
				Email: IntegrationStatus{
					Enabled: settings.EmailInputEnabled,
					Status:  emailStatus,
				},
				SMS: IntegrationStatus{
					Enabled: settings.SMSInputEnabled,
					Status:  "pending", // SMS not implemented yet
				},
			},
			Calendars: SmartCalendarCalendars{
				Alfred: IntegrationStatus{
					Enabled: settings.AlfredCalendarEnabled,
					Status:  "available", // Alfred calendar is always available (local storage)
				},
				GoogleCalendar: IntegrationStatus{
					Enabled: settings.GoogleCalendarEnabled,
					Status:  googleCalendarStatus,
				},
				Outlook: IntegrationStatus{
					Enabled: settings.OutlookCalendarEnabled,
					Status:  "pending", // Outlook not implemented yet
				},
			},
		},
	}

	respondJSON(w, http.StatusOK, response)
}

// UpdateSmartCalendarRequest represents the request body for PUT /api/features/smart-calendar
type UpdateSmartCalendarRequest struct {
	Enabled       *bool `json:"enabled,omitempty"`
	SetupComplete *bool `json:"setup_complete,omitempty"`
	Inputs        *struct {
		WhatsApp bool `json:"whatsapp"`
		Email    bool `json:"email"`
		SMS      bool `json:"sms"`
	} `json:"inputs,omitempty"`
	Calendars *struct {
		Alfred         bool `json:"alfred"`
		GoogleCalendar bool `json:"google_calendar"`
		Outlook        bool `json:"outlook"`
	} `json:"calendars,omitempty"`
}

// handleUpdateSmartCalendar updates the Smart Calendar feature settings
func (s *Server) handleUpdateSmartCalendar(w http.ResponseWriter, r *http.Request) {
	var req UpdateSmartCalendarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Get current settings
	settings, err := s.db.GetFeatureSettings()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Update only provided fields
	if req.Enabled != nil {
		settings.SmartCalendarEnabled = *req.Enabled
	}
	if req.SetupComplete != nil {
		settings.SmartCalendarSetupComplete = *req.SetupComplete
	}
	if req.Inputs != nil {
		settings.WhatsAppInputEnabled = req.Inputs.WhatsApp
		settings.EmailInputEnabled = req.Inputs.Email
		settings.SMSInputEnabled = req.Inputs.SMS
	}
	if req.Calendars != nil {
		settings.AlfredCalendarEnabled = req.Calendars.Alfred
		settings.GoogleCalendarEnabled = req.Calendars.GoogleCalendar
		settings.OutlookCalendarEnabled = req.Calendars.Outlook
	}

	// Save updated settings
	if err := s.db.UpdateFeatureSettings(settings); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return updated features
	s.handleGetFeatures(w, r)
}

// handleGetSmartCalendarStatus returns the detailed status of Smart Calendar integrations
func (s *Server) handleGetSmartCalendarStatus(w http.ResponseWriter, r *http.Request) {
	settings, err := s.db.GetFeatureSettings()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Build detailed status for each enabled integration
	status := map[string]interface{}{
		"enabled":        settings.SmartCalendarEnabled,
		"setup_complete": settings.SmartCalendarSetupComplete,
		"integrations":   []map[string]interface{}{},
	}

	integrations := []map[string]interface{}{}

	// Alfred Calendar status (always available - local storage)
	if settings.AlfredCalendarEnabled {
		integrations = append(integrations, map[string]interface{}{
			"type":    "calendar",
			"name":    "Alfred Calendar",
			"status":  "available",
			"message": "Local storage",
		})
	}

	// WhatsApp status
	if settings.WhatsAppInputEnabled {
		waStatus := "pending"
		waMessage := "Not connected"
		if s.waClient != nil && s.waClient.IsLoggedIn() {
			waStatus = "available"
			waMessage = "Connected"
		}
		integrations = append(integrations, map[string]interface{}{
			"type":    "input",
			"name":    "WhatsApp",
			"status":  waStatus,
			"message": waMessage,
		})
	}

	// Google Calendar status
	if settings.GoogleCalendarEnabled {
		gcalStatus := "pending"
		gcalMessage := "Not connected"
		if s.gcalClient != nil && s.gcalClient.IsAuthenticated() {
			gcalStatus = "available"
			gcalMessage = "Connected"
		}
		integrations = append(integrations, map[string]interface{}{
			"type":    "calendar",
			"name":    "Google Calendar",
			"status":  gcalStatus,
			"message": gcalMessage,
		})
	}

	// Email (Gmail) status
	if settings.EmailInputEnabled {
		emailStatus := "pending"
		emailMessage := "Not connected"
		if s.gmailClient != nil && s.gmailClient.IsAuthenticated() {
			emailStatus = "available"
			emailMessage = "Connected"
		}
		integrations = append(integrations, map[string]interface{}{
			"type":    "input",
			"name":    "Gmail",
			"status":  emailStatus,
			"message": emailMessage,
		})
	}

	status["integrations"] = integrations

	// Check if all enabled integrations are available
	allAvailable := true
	for _, integration := range integrations {
		if integration["status"] != "available" {
			allAvailable = false
			break
		}
	}
	status["all_available"] = allAvailable

	respondJSON(w, http.StatusOK, status)
}

// ---- Simplified App Status API (new navigation flow) ----

// AppStatusResponse represents the simplified app status for the new UI flow
type AppStatusResponse struct {
	OnboardingComplete bool             `json:"onboarding_complete"`
	WhatsApp           ConnectionStatus `json:"whatsapp"`
	Gmail              ConnectionStatus `json:"gmail"`
	GoogleCalendar     ConnectionStatus `json:"google_calendar"`
}

// ConnectionStatus represents the connection status of an integration
type ConnectionStatus struct {
	Enabled   bool `json:"enabled"`
	Connected bool `json:"connected"`
}

// handleGetAppStatus returns the simplified app status
func (s *Server) handleGetAppStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.db.GetAppStatus()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Check actual connection status
	whatsappConnected := s.waClient != nil && s.waClient.IsLoggedIn()
	gmailConnected := s.gmailClient != nil && s.gmailClient.IsAuthenticated()
	googleCalConnected := s.gcalClient != nil && s.gcalClient.IsAuthenticated()

	response := AppStatusResponse{
		OnboardingComplete: status.OnboardingComplete,
		WhatsApp: ConnectionStatus{
			Enabled:   status.WhatsAppEnabled,
			Connected: whatsappConnected,
		},
		Gmail: ConnectionStatus{
			Enabled:   status.GmailEnabled,
			Connected: gmailConnected,
		},
		GoogleCalendar: ConnectionStatus{
			Enabled:   status.GoogleCalEnabled,
			Connected: googleCalConnected,
		},
	}

	respondJSON(w, http.StatusOK, response)
}

// CompleteOnboardingRequest represents the request body for POST /api/onboarding/complete
type CompleteOnboardingRequest struct {
	WhatsAppEnabled bool `json:"whatsapp_enabled"`
	GmailEnabled    bool `json:"gmail_enabled"`
}

// handleCompleteOnboarding marks the onboarding as complete
func (s *Server) handleCompleteOnboarding(w http.ResponseWriter, r *http.Request) {
	var req CompleteOnboardingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// At least one input must be enabled
	if !req.WhatsAppEnabled && !req.GmailEnabled {
		respondError(w, http.StatusBadRequest, "at least one input (WhatsApp or Gmail) must be enabled")
		return
	}

	if err := s.db.CompleteOnboarding(req.WhatsAppEnabled, req.GmailEnabled); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return updated status
	s.handleGetAppStatus(w, r)
}

// handleResetOnboarding resets the onboarding status (for testing)
func (s *Server) handleResetOnboarding(w http.ResponseWriter, r *http.Request) {
	if err := s.db.ResetOnboarding(); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return updated status
	s.handleGetAppStatus(w, r)
}

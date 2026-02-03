package server

import (
	"encoding/json"
	"net/http"
)

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
	userID := getUserID(r)

	// Check actual connection status (available regardless of user login)
	whatsappConnected := s.waClient != nil && s.waClient.IsLoggedIn()
	gmailConnected := s.gmailClient != nil && s.gmailClient.IsAuthenticated()
	userGCalClient := s.getGCalClientForUser(userID)
	googleCalConnected := userGCalClient != nil && userGCalClient.IsAuthenticated()

	// If no user is logged in, return default status
	if userID == 0 {
		response := AppStatusResponse{
			OnboardingComplete: false,
			WhatsApp: ConnectionStatus{
				Enabled:   false,
				Connected: whatsappConnected,
			},
			Gmail: ConnectionStatus{
				Enabled:   false,
				Connected: gmailConnected,
			},
			GoogleCalendar: ConnectionStatus{
				Enabled:   false,
				Connected: googleCalConnected,
			},
		}
		respondJSON(w, http.StatusOK, response)
		return
	}

	status, err := s.db.GetAppStatus(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

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
	TelegramEnabled bool `json:"telegram_enabled"`
	GmailEnabled    bool `json:"gmail_enabled"`
}

// handleCompleteOnboarding marks the onboarding as complete
func (s *Server) handleCompleteOnboarding(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var req CompleteOnboardingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// At least one input must be enabled
	if !req.WhatsAppEnabled && !req.TelegramEnabled && !req.GmailEnabled {
		respondError(w, http.StatusBadRequest, "at least one input (WhatsApp, Telegram, or Gmail) must be enabled")
		return
	}

	if err := s.db.CompleteOnboarding(userID, req.WhatsAppEnabled, req.TelegramEnabled, req.GmailEnabled); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Start user services now that onboarding is complete
	if s.userServiceManager != nil {
		go s.userServiceManager.StartServicesForUser(userID)
	}

	// Return updated status
	s.handleGetAppStatus(w, r)
}

// handleResetOnboarding resets the onboarding status (for testing)
func (s *Server) handleResetOnboarding(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	if err := s.db.ResetOnboarding(userID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return updated status
	s.handleGetAppStatus(w, r)
}

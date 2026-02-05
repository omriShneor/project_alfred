package server

import (
	"encoding/json"
	"fmt"
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
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Check actual connection status
	// For WhatsApp/Telegram in multi-user mode, we check if ClientManager is available
	whatsappConnected := false
	if s.clientManager != nil && userID > 0 {
		if waClient, err := s.clientManager.GetWhatsAppClient(userID); err == nil {
			whatsappConnected = waClient.IsLoggedIn()
		}
	}

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

	// Start services on app load if we have cached auth/sessions (faster home screen)
	if s.userServiceManager != nil && !s.userServiceManager.IsRunningForUser(userID) {
		shouldStart := false

		// Gmail scope cached
		if s.authService != nil {
			hasGmailScope, _ := s.authService.HasGmailScope(userID)
			if hasGmailScope {
				shouldStart = true
			}
		}

		// WhatsApp / Telegram sessions cached
		if !shouldStart {
			if waSession, _ := s.db.GetWhatsAppSession(userID); waSession != nil && waSession.Connected {
				shouldStart = true
			}
		}
		if !shouldStart {
			if tgSession, _ := s.db.GetTelegramSession(userID); tgSession != nil && tgSession.Connected {
				shouldStart = true
			}
		}

		// Enabled integrations (onboarding already completed)
		if !shouldStart && (status.WhatsAppEnabled || status.TelegramEnabled || status.GmailEnabled) {
			shouldStart = true
		}

		if shouldStart {
			go s.userServiceManager.StartServicesForUser(userID)
		}
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
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

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
	if req.GmailEnabled {
		_ = s.db.SetGmailEnabled(userID, true)
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
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Reset all client sessions for this user (full logout with session deletion)
	if s.clientManager != nil {
		if err := s.clientManager.ResetUserSessions(userID); err != nil {
			fmt.Printf("Warning: Failed to reset user sessions: %v\n", err)
		}
	}

	// Stop user services (Gmail workers, etc.)
	if s.userServiceManager != nil {
		s.userServiceManager.StopServicesForUser(userID)
	}

	// Reset database state (deletes Google tokens, WhatsApp/Telegram session records)
	if err := s.db.ResetOnboarding(userID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Delete all user sessions (forces re-login)
	if s.authService != nil {
		if err := s.authService.DeleteAllUserSessions(userID); err != nil {
			fmt.Printf("Warning: Failed to delete user sessions: %v\n", err)
		}
	}

	// Reset SSE state
	if s.state != nil {
		s.state.SetTelegramStatus("pending")
		s.state.SetWhatsAppStatus("pending")
		s.state.SetGCalStatus("pending")
	}

	// Return updated status
	s.handleGetAppStatus(w, r)
}

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/omriShneor/project_alfred/internal/auth"
	"github.com/omriShneor/project_alfred/internal/timeutil"
)

// getUserID extracts the authenticated user's ID from the request context.
func getUserID(r *http.Request) (int64, error) {
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

	// Note: Health check now just reports that ClientManager is available
	// Individual user connection status should be checked via /api/whatsapp/status and /api/telegram/status
	if s.clientManager != nil {
		status["whatsapp"] = "available"
		status["telegram"] = "available"
	}

	if s.credentialsFile != "" {
		status["gcal"] = "configured"
	}

	respondJSON(w, http.StatusOK, status)
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

// Helper functions
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// parseEventTime parses a time string with user timezone fallback.
func parseEventTime(s, timezone string) (time.Time, bool, error) {
	return timeutil.ParseDateTime(s, timezone)
}

func (s *Server) getUserTimezone(userID int64) string {
	tz, err := s.db.GetUserTimezone(userID)
	if err != nil || tz == "" {
		return "UTC"
	}
	return tz
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
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
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
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
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
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
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

	respondJSON(w, http.StatusOK, map[string]string{"status": "registered"})
}

// handleUpdatePushPrefs enables/disables push notifications
func (s *Server) handleUpdatePushPrefs(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
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

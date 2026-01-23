package sse

import (
	"context"
	"encoding/json"
	"sync"
)

// State manages the onboarding state for SSE streaming
type State struct {
	mu sync.RWMutex

	WhatsAppStatus string // "checking", "needs_qr", "waiting", "connected", "error"
	CurrentQR      string // Base64 data URL
	WhatsAppError  string

	GCalStatus     string // "not_configured", "needs_auth", "waiting", "connected", "error"
	GCalConfigured bool
	GCalError      string

	Complete bool

	subscribers map[chan Update]struct{}
	completeCh  chan struct{}
}

// Update represents an SSE update event
type Update struct {
	Type string `json:"type"` // "whatsapp_status", "qr", "gcal_status", "complete"
	Data string `json:"data"`
}

// StatusResponse is the JSON response for /api/onboarding/status
type StatusResponse struct {
	WhatsApp WhatsAppStatusResponse `json:"whatsapp"`
	GCal     GCalStatusResponse     `json:"gcal"`
	Complete bool                   `json:"complete"`
}

// WhatsAppStatusResponse contains WhatsApp status details
type WhatsAppStatusResponse struct {
	Status string `json:"status"`
	QRCode string `json:"qr_code,omitempty"`
	Error  string `json:"error,omitempty"`
}

// GCalStatusResponse contains Google Calendar status details
type GCalStatusResponse struct {
	Status     string `json:"status"`
	Configured bool   `json:"configured"`
	Error      string `json:"error,omitempty"`
}

// NewState creates a new onboarding state
func NewState() *State {
	return &State{
		WhatsAppStatus: "checking",
		GCalStatus:     "checking",
		subscribers:    make(map[chan Update]struct{}),
		completeCh:     make(chan struct{}),
	}
}

// Subscribe creates a new channel for receiving updates
func (s *State) Subscribe() chan Update {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan Update, 10)
	s.subscribers[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a subscriber channel
func (s *State) Unsubscribe(ch chan Update) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.subscribers, ch)
	close(ch)
}

// broadcast sends an update to all subscribers
func (s *State) broadcast(update Update) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for ch := range s.subscribers {
		select {
		case ch <- update:
		default:
			// Channel full, skip
		}
	}
}

// SetWhatsAppStatus updates the WhatsApp status and broadcasts
func (s *State) SetWhatsAppStatus(status string) {
	s.mu.Lock()
	s.WhatsAppStatus = status
	if status != "error" {
		s.WhatsAppError = "" // Clear error when status changes to non-error
	}
	s.mu.Unlock()

	s.broadcast(Update{Type: "whatsapp_status", Data: status})
	s.checkComplete()
}

// SetWhatsAppError sets an error for WhatsApp
func (s *State) SetWhatsAppError(err string) {
	s.mu.Lock()
	s.WhatsAppStatus = "error"
	s.WhatsAppError = err
	s.mu.Unlock()

	s.broadcast(Update{Type: "whatsapp_status", Data: "error"})
}

// SetQR updates the QR code and broadcasts
func (s *State) SetQR(dataURL string) {
	s.mu.Lock()
	s.CurrentQR = dataURL
	s.WhatsAppStatus = "waiting"
	s.mu.Unlock()

	s.broadcast(Update{Type: "qr", Data: dataURL})
}

// SetGCalStatus updates the Google Calendar status and broadcasts
func (s *State) SetGCalStatus(status string) {
	s.mu.Lock()
	s.GCalStatus = status
	if status != "error" {
		s.GCalError = "" // Clear error when status changes to non-error
	}
	s.mu.Unlock()

	s.broadcast(Update{Type: "gcal_status", Data: status})
	s.checkComplete()
}

// SetGCalConfigured sets whether Google Calendar credentials are available
func (s *State) SetGCalConfigured(configured bool) {
	s.mu.Lock()
	s.GCalConfigured = configured
	if !configured {
		s.GCalStatus = "not_configured"
	}
	s.mu.Unlock()
}

// SetGCalError sets an error for Google Calendar
func (s *State) SetGCalError(err string) {
	s.mu.Lock()
	s.GCalStatus = "error"
	s.GCalError = err
	s.mu.Unlock()

	s.broadcast(Update{Type: "gcal_status", Data: "error"})
}

// checkComplete checks if all integrations are connected and marks complete
func (s *State) checkComplete() {
	s.mu.RLock()
	waConnected := s.WhatsAppStatus == "connected"
	gcalConnected := s.GCalStatus == "connected"
	alreadyComplete := s.Complete
	s.mu.RUnlock()

	if waConnected && gcalConnected && !alreadyComplete {
		s.MarkComplete()
	}
}

// MarkComplete marks onboarding as complete
func (s *State) MarkComplete() {
	s.mu.Lock()
	if s.Complete {
		s.mu.Unlock()
		return
	}
	s.Complete = true
	s.mu.Unlock()

	s.broadcast(Update{Type: "complete", Data: "{}"})

	// Signal completion to waiters
	close(s.completeCh)
}

// IsComplete returns whether onboarding is complete
func (s *State) IsComplete() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Complete
}

// WaitForCompletion blocks until onboarding is complete or context is cancelled
func (s *State) WaitForCompletion(ctx context.Context) error {
	select {
	case <-s.completeCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetStatus returns the current status as a JSON-serializable struct
func (s *State) GetStatus() StatusResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return StatusResponse{
		WhatsApp: WhatsAppStatusResponse{
			Status: s.WhatsAppStatus,
			QRCode: s.CurrentQR,
			Error:  s.WhatsAppError,
		},
		GCal: GCalStatusResponse{
			Status:     s.GCalStatus,
			Configured: s.GCalConfigured,
			Error:      s.GCalError,
		},
		Complete: s.Complete,
	}
}

// GetStatusJSON returns the current status as JSON
func (s *State) GetStatusJSON() string {
	status := s.GetStatus()
	data, _ := json.Marshal(status)
	return string(data)
}

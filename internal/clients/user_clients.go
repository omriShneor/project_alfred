package clients

import (
	"sync"

	"github.com/omriShneor/project_alfred/internal/gcal"
	"github.com/omriShneor/project_alfred/internal/gmail"
	"github.com/omriShneor/project_alfred/internal/telegram"
	"github.com/omriShneor/project_alfred/internal/whatsapp"
)

// UserClients holds all client instances for a single user
type UserClients struct {
	userID int64
	mu     sync.RWMutex

	// Messaging clients
	WhatsApp        *whatsapp.Client
	WhatsAppHandler *whatsapp.Handler
	Telegram        *telegram.Client
	TelegramHandler *telegram.Handler

	// Google clients
	GCal  *gcal.Client
	Gmail *gmail.Client
}

// UserID returns the user ID for this client container
func (c *UserClients) UserID() int64 {
	return c.userID
}

// IsWhatsAppConnected returns whether WhatsApp is connected
func (c *UserClients) IsWhatsAppConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.WhatsApp != nil && c.WhatsApp.IsLoggedIn()
}

// IsTelegramConnected returns whether Telegram is connected
func (c *UserClients) IsTelegramConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Telegram != nil && c.Telegram.IsConnected()
}

// IsGCalAuthenticated returns whether Google Calendar is authenticated
func (c *UserClients) IsGCalAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.GCal != nil && c.GCal.IsAuthenticated()
}

// IsGmailAuthenticated returns whether Gmail is authenticated
func (c *UserClients) IsGmailAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Gmail != nil && c.Gmail.IsAuthenticated()
}

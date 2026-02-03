package clients

import (
	"fmt"
	"sync"

	"github.com/omriShneor/project_alfred/internal/config"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/gcal"
	"github.com/omriShneor/project_alfred/internal/gmail"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/source"
	"github.com/omriShneor/project_alfred/internal/sse"
	"github.com/omriShneor/project_alfred/internal/telegram"
	"github.com/omriShneor/project_alfred/internal/whatsapp"
)

// Manager manages per-user client instances for WhatsApp, Telegram, Gmail, and GCal.
// For the current 1-10 user scale, all clients are kept in memory.
type Manager struct {
	db            *database.DB
	cfg           *config.Config
	notifyService *notify.Service
	state         *sse.State

	clients map[int64]*UserClients
	mu      sync.RWMutex
}

// ManagerConfig holds configuration for the client manager
type ManagerConfig struct {
	DB            *database.DB
	Config        *config.Config
	NotifyService *notify.Service
	State         *sse.State
}

// NewManager creates a new client manager
func NewManager(cfg ManagerConfig) *Manager {
	return &Manager{
		db:            cfg.DB,
		cfg:           cfg.Config,
		notifyService: cfg.NotifyService,
		state:         cfg.State,
		clients:       make(map[int64]*UserClients),
	}
}

// GetClients returns the client container for a user, creating it if necessary
func (m *Manager) GetClients(userID int64) *UserClients {
	m.mu.RLock()
	clients, exists := m.clients[userID]
	m.mu.RUnlock()

	if exists {
		return clients
	}

	// Create new client container
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if clients, exists = m.clients[userID]; exists {
		return clients
	}

	clients = &UserClients{
		userID: userID,
	}
	m.clients[userID] = clients
	return clients
}

// GetOrCreateWhatsApp returns the WhatsApp client for a user, creating if necessary
func (m *Manager) GetOrCreateWhatsApp(userID int64) (*whatsapp.Client, error) {
	clients := m.GetClients(userID)

	clients.mu.Lock()
	defer clients.mu.Unlock()

	if clients.WhatsApp != nil {
		return clients.WhatsApp, nil
	}

	// Create per-user handler with user's message channel
	handler := whatsapp.NewHandlerForUser(userID, m.db, m.cfg.DebugAllMessages, m.state)

	// Create client with per-user DB file
	dbPath := m.getWhatsAppDBPath(userID)
	client, err := whatsapp.NewClient(handler, dbPath, m.notifyService)
	if err != nil {
		return nil, fmt.Errorf("failed to create WhatsApp client for user %d: %w", userID, err)
	}
	client.UserID = userID

	clients.WhatsApp = client
	clients.WhatsAppHandler = handler

	return client, nil
}

// GetOrCreateTelegram returns the Telegram client for a user, creating if necessary
func (m *Manager) GetOrCreateTelegram(userID int64) (*telegram.Client, error) {
	clients := m.GetClients(userID)

	clients.mu.Lock()
	defer clients.mu.Unlock()

	if clients.Telegram != nil {
		return clients.Telegram, nil
	}

	if m.cfg.TelegramAPIID == 0 || m.cfg.TelegramAPIHash == "" {
		return nil, fmt.Errorf("Telegram API credentials not configured")
	}

	// Create per-user handler
	handler := telegram.NewHandlerForUser(userID, m.db)

	// Create client with per-user session file
	sessionPath := m.getTelegramSessionPath(userID)
	client, err := telegram.NewClient(telegram.ClientConfig{
		APIID:       m.cfg.TelegramAPIID,
		APIHash:     m.cfg.TelegramAPIHash,
		SessionPath: sessionPath,
		Handler:     handler,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram client for user %d: %w", userID, err)
	}

	clients.Telegram = client
	clients.TelegramHandler = handler

	return client, nil
}

// GetOrCreateGCal returns the Google Calendar client for a user
// Note: GCal uses shared OAuth config, tokens are per-user from database
func (m *Manager) GetOrCreateGCal(userID int64) (*gcal.Client, error) {
	clients := m.GetClients(userID)

	clients.mu.Lock()
	defer clients.mu.Unlock()

	if clients.GCal != nil {
		return clients.GCal, nil
	}

	// For now, use the shared credentials file
	// TODO: Load per-user tokens from database
	client, err := gcal.NewClientForUser(userID, m.cfg.GoogleCredentialsFile, m.db)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCal client for user %d: %w", userID, err)
	}

	clients.GCal = client

	return client, nil
}

// GetOrCreateGmail returns the Gmail client for a user
func (m *Manager) GetOrCreateGmail(userID int64) (*gmail.Client, error) {
	clients := m.GetClients(userID)

	clients.mu.Lock()
	defer clients.mu.Unlock()

	if clients.Gmail != nil {
		return clients.Gmail, nil
	}

	// Gmail client requires GCal OAuth config/token
	gcalClient, err := m.GetOrCreateGCal(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get GCal client for Gmail: %w", err)
	}

	if !gcalClient.IsAuthenticated() {
		return nil, fmt.Errorf("Google account not authenticated")
	}

	oauthConfig := gcalClient.GetOAuthConfig()
	oauthToken := gcalClient.GetToken()
	if oauthConfig == nil || oauthToken == nil {
		return nil, fmt.Errorf("OAuth config or token not available")
	}

	gmailClient, err := gmail.NewClient(oauthConfig, oauthToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail client for user %d: %w", userID, err)
	}

	clients.Gmail = gmailClient

	return gmailClient, nil
}

// DisconnectAll disconnects all clients for a user
func (m *Manager) DisconnectAll(userID int64) {
	m.mu.Lock()
	clients, exists := m.clients[userID]
	if exists {
		delete(m.clients, userID)
	}
	m.mu.Unlock()

	if !exists || clients == nil {
		return
	}

	clients.mu.Lock()
	defer clients.mu.Unlock()

	if clients.WhatsApp != nil {
		clients.WhatsApp.Disconnect()
		clients.WhatsApp = nil
	}

	if clients.Telegram != nil {
		clients.Telegram.Disconnect()
		clients.Telegram = nil
	}

	// GCal and Gmail don't need explicit disconnect
	clients.GCal = nil
	clients.Gmail = nil
}

// GetMessageChannel returns the message channel for a user's WhatsApp handler
func (m *Manager) GetMessageChannel(userID int64) <-chan source.Message {
	clients := m.GetClients(userID)

	clients.mu.RLock()
	defer clients.mu.RUnlock()

	if clients.WhatsAppHandler != nil {
		return clients.WhatsAppHandler.MessageChan()
	}
	return nil
}

// getWhatsAppDBPath returns the per-user WhatsApp database path
func (m *Manager) getWhatsAppDBPath(userID int64) string {
	// Use data directory with per-user suffix
	basePath := m.cfg.WhatsAppDBPath
	if basePath == "" {
		basePath = "./whatsapp.db"
	}
	// For user 1 (legacy), use the original path for backwards compatibility
	if userID == 1 {
		return basePath
	}
	return fmt.Sprintf("%s.user_%d", basePath, userID)
}

// getTelegramSessionPath returns the per-user Telegram session path
func (m *Manager) getTelegramSessionPath(userID int64) string {
	basePath := m.cfg.TelegramDBPath
	if basePath == "" {
		basePath = "./telegram.db"
	}
	// For user 1 (legacy), use the original path for backwards compatibility
	if userID == 1 {
		return basePath
	}
	return fmt.Sprintf("%s.user_%d", basePath, userID)
}

// RestoreAllSessions attempts to restore sessions for all users with stored credentials
func (m *Manager) RestoreAllSessions() {
	// TODO: Implement session restoration from database
	// For now, this is a placeholder for Phase 3 completion
	fmt.Println("ClientManager: Session restoration not yet implemented")
}

package clients

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/source"
	"github.com/omriShneor/project_alfred/internal/sse"
	"github.com/omriShneor/project_alfred/internal/telegram"
	"github.com/omriShneor/project_alfred/internal/whatsapp"
)

// ClientManager manages per-user WhatsApp and Telegram client instances
type ClientManager struct {
	db              *database.DB
	cfg             *ManagerConfig
	notifyService   *notify.Service
	onboardingState *sse.State

	// Shared message channel (all users' messages tagged with UserID)
	msgChan chan source.Message

	// Per-user client instances
	mu              sync.RWMutex
	whatsappClients map[int64]*whatsapp.Client
	telegramClients map[int64]*telegram.Client
}

// ManagerConfig holds configuration for the ClientManager
type ManagerConfig struct {
	// Base paths for session storage
	WhatsAppDBBasePath string // e.g., "/data/whatsapp.db" or "./whatsapp.db"
	TelegramDBBasePath string // e.g., "/data/telegram.db" or "./telegram.db"

	// Telegram API credentials
	TelegramAPIID   int
	TelegramAPIHash string

	// Feature flags
	DebugAllMessages bool
}

// NewClientManager creates a new client manager
func NewClientManager(db *database.DB, cfg *ManagerConfig, notifyService *notify.Service, state *sse.State) *ClientManager {
	return &ClientManager{
		db:              db,
		cfg:             cfg,
		notifyService:   notifyService,
		onboardingState: state,
		msgChan:         make(chan source.Message, 1000), // Large buffer for multi-user
		whatsappClients: make(map[int64]*whatsapp.Client),
		telegramClients: make(map[int64]*telegram.Client),
	}
}

// MessageChan returns the shared message channel
func (m *ClientManager) MessageChan() <-chan source.Message {
	return m.msgChan
}

// ==================== WhatsApp Client Management ====================

// GetWhatsAppClient returns an existing WhatsApp client for the user or creates a new one
func (m *ClientManager) GetWhatsAppClient(userID int64) (*whatsapp.Client, error) {
	m.mu.RLock()
	client, exists := m.whatsappClients[userID]
	m.mu.RUnlock()

	if exists {
		return client, nil
	}

	return m.CreateWhatsAppClient(userID)
}

// CreateWhatsAppClient creates a new WhatsApp client for the user
func (m *ClientManager) CreateWhatsAppClient(userID int64) (*whatsapp.Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check again inside lock to prevent race condition
	if client, exists := m.whatsappClients[userID]; exists {
		return client, nil
	}

	dbPath := m.getUserWhatsAppDBPath(userID)
	fmt.Printf("ClientManager: Creating WhatsApp client for user %d with session path: %s\n", userID, dbPath)

	// Create handler for this user first
	handler := whatsapp.NewHandlerForUser(userID, m.db, m.cfg.DebugAllMessages, m.onboardingState)

	// Override handler's message channel with shared channel
	// This ensures all users' messages go to the same channel with UserID tags
	handler.SetMessageChannel(m.msgChan)

	// Create client with handler
	client, err := whatsapp.NewClient(handler, dbPath, m.notifyService)
	if err != nil {
		return nil, fmt.Errorf("failed to create WhatsApp client for user %d: %w", userID, err)
	}

	// Set UserID on client
	client.SetUserID(userID)

	m.whatsappClients[userID] = client
	fmt.Printf("ClientManager: WhatsApp client created for user %d\n", userID)

	return client, nil
}

// DestroyWhatsAppClient disconnects and removes the WhatsApp client for a user
// but preserves the session file for reconnection
func (m *ClientManager) DestroyWhatsAppClient(userID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.whatsappClients[userID]
	if !exists {
		return nil // Already destroyed
	}

	fmt.Printf("ClientManager: Destroying WhatsApp client for user %d\n", userID)

	// Disconnect but don't delete session
	if client.IsLoggedIn() {
		client.Disconnect()
	}

	delete(m.whatsappClients, userID)
	fmt.Printf("ClientManager: WhatsApp client destroyed for user %d (session preserved)\n", userID)

	return nil
}

// LogoutWhatsApp performs a full logout for WhatsApp, deleting the session
func (m *ClientManager) LogoutWhatsApp(userID int64) error {
	m.mu.Lock()
	client, exists := m.whatsappClients[userID]
	m.mu.Unlock()

	if !exists {
		// No client in memory, but session file might exist - delete it
		sessionPath := m.getUserWhatsAppDBPath(userID)
		if err := os.Remove(sessionPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete WhatsApp session file: %w", err)
		}
		fmt.Printf("ClientManager: WhatsApp session file deleted for user %d\n", userID)
		return nil
	}

	fmt.Printf("ClientManager: Logging out WhatsApp for user %d\n", userID)

	// Perform protocol-level logout
	if client.IsLoggedIn() {
		if err := client.Logout(); err != nil {
			fmt.Printf("Warning: WhatsApp logout failed for user %d: %v\n", userID, err)
		}
	}

	// Delete session file
	sessionPath := m.getUserWhatsAppDBPath(userID)
	if err := os.Remove(sessionPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete WhatsApp session file: %w", err)
	}

	// Remove from memory
	m.mu.Lock()
	delete(m.whatsappClients, userID)
	m.mu.Unlock()

	fmt.Printf("ClientManager: WhatsApp fully logged out for user %d\n", userID)
	return nil
}

// getUserWhatsAppDBPath generates the per-user WhatsApp session path
func (m *ClientManager) getUserWhatsAppDBPath(userID int64) string {
	return fmt.Sprintf("%s.user_%d", m.cfg.WhatsAppDBBasePath, userID)
}

// ==================== Telegram Client Management ====================

// GetTelegramClient returns an existing Telegram client for the user or creates a new one
func (m *ClientManager) GetTelegramClient(userID int64) (*telegram.Client, error) {
	m.mu.RLock()
	client, exists := m.telegramClients[userID]
	m.mu.RUnlock()

	if exists {
		return client, nil
	}

	return m.CreateTelegramClient(userID)
}

// CreateTelegramClient creates a new Telegram client for the user
func (m *ClientManager) CreateTelegramClient(userID int64) (*telegram.Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check again inside lock to prevent race condition
	if client, exists := m.telegramClients[userID]; exists {
		return client, nil
	}

	sessionPath := m.getUserTelegramSessionPath(userID)
	fmt.Printf("ClientManager: Creating Telegram client for user %d with session path: %s\n", userID, sessionPath)

	// Create handler for this user first
	handler := telegram.NewHandlerForUser(userID, m.db)

	// Override handler's message channel with shared channel
	handler.SetMessageChannel(m.msgChan)

	// Create client with handler using ClientConfig
	client, err := telegram.NewClient(telegram.ClientConfig{
		APIID:       m.cfg.TelegramAPIID,
		APIHash:     m.cfg.TelegramAPIHash,
		SessionPath: sessionPath,
		Handler:     handler,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram client for user %d: %w", userID, err)
	}

	// Set UserID on client (this sets it on the handler)
	client.SetUserID(userID)

	m.telegramClients[userID] = client
	fmt.Printf("ClientManager: Telegram client created for user %d\n", userID)

	return client, nil
}

// DestroyTelegramClient disconnects and removes the Telegram client for a user
// but preserves the session file for reconnection
func (m *ClientManager) DestroyTelegramClient(userID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.telegramClients[userID]
	if !exists {
		return nil // Already destroyed
	}

	fmt.Printf("ClientManager: Destroying Telegram client for user %d\n", userID)

	// Disconnect but don't delete session
	if client.IsConnected() {
		client.Disconnect()
	}

	delete(m.telegramClients, userID)
	fmt.Printf("ClientManager: Telegram client destroyed for user %d (session preserved)\n", userID)

	return nil
}

// LogoutTelegram performs a full logout for Telegram, deleting the session
func (m *ClientManager) LogoutTelegram(userID int64) error {
	m.mu.Lock()
	client, exists := m.telegramClients[userID]
	m.mu.Unlock()

	if !exists {
		// No client in memory, but session file might exist - delete it
		sessionPath := m.getUserTelegramSessionPath(userID)
		if err := os.Remove(sessionPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete Telegram session file: %w", err)
		}
		fmt.Printf("ClientManager: Telegram session file deleted for user %d\n", userID)
		return nil
	}

	fmt.Printf("ClientManager: Logging out Telegram for user %d\n", userID)

	// Disconnect
	if client.IsConnected() {
		client.Disconnect()
	}

	// Delete session file
	if err := client.DeleteSession(); err != nil {
		fmt.Printf("Warning: Failed to delete Telegram session file for user %d: %v\n", userID, err)
	}

	// Remove from memory
	m.mu.Lock()
	delete(m.telegramClients, userID)
	m.mu.Unlock()

	fmt.Printf("ClientManager: Telegram fully logged out for user %d\n", userID)
	return nil
}

// getUserTelegramSessionPath generates the per-user Telegram session path
func (m *ClientManager) getUserTelegramSessionPath(userID int64) string {
	return fmt.Sprintf("%s.user_%d", m.cfg.TelegramDBBasePath, userID)
}

// ==================== Lifecycle Management ====================

// CleanupUser destroys all clients for a user (called on logout)
// Preserves session files for reconnection
func (m *ClientManager) CleanupUser(userID int64) error {
	fmt.Printf("ClientManager: Cleaning up all clients for user %d\n", userID)

	var errs []error

	if err := m.DestroyWhatsAppClient(userID); err != nil {
		errs = append(errs, fmt.Errorf("WhatsApp cleanup failed: %w", err))
	}

	if err := m.DestroyTelegramClient(userID); err != nil {
		errs = append(errs, fmt.Errorf("Telegram cleanup failed: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors for user %d: %v", userID, errs)
	}

	fmt.Printf("ClientManager: All clients cleaned up for user %d\n", userID)
	return nil
}

// RestoreUserSessions restores sessions for all users on server startup
func (m *ClientManager) RestoreUserSessions(ctx context.Context) error {
	fmt.Println("ClientManager: Restoring user sessions...")

	// Get all users from database
	users, err := m.db.GetAllUsers()
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	for _, user := range users {
		// Check if user has onboarding complete
		status, err := m.db.GetAppStatus(user.ID)
		if err != nil {
			fmt.Printf("Warning: Failed to get status for user %d: %v\n", user.ID, err)
			continue
		}

		if !status.OnboardingComplete {
			fmt.Printf("ClientManager: Skipping user %d (onboarding incomplete)\n", user.ID)
			continue
		}

		// Restore WhatsApp session if enabled
		if status.WhatsAppEnabled {
			sessionPath := m.getUserWhatsAppDBPath(user.ID)
			if _, err := os.Stat(sessionPath); err == nil {
				fmt.Printf("ClientManager: Restoring WhatsApp session for user %d\n", user.ID)
				if _, err := m.CreateWhatsAppClient(user.ID); err != nil {
					fmt.Printf("Warning: Failed to restore WhatsApp for user %d: %v\n", user.ID, err)
				}
			}
		}

		// Restore Telegram session if enabled
		if status.TelegramEnabled {
			sessionPath := m.getUserTelegramSessionPath(user.ID)
			if _, err := os.Stat(sessionPath); err == nil {
				fmt.Printf("ClientManager: Restoring Telegram session for user %d\n", user.ID)
				if _, err := m.CreateTelegramClient(user.ID); err != nil {
					fmt.Printf("Warning: Failed to restore Telegram for user %d: %v\n", user.ID, err)
				}
			}
		}
	}

	fmt.Println("ClientManager: Session restoration complete")
	return nil
}

// CleanupLegacySessions deletes old single-user session files (one-time migration)
// This method should be removed after the migration is complete
func (m *ClientManager) CleanupLegacySessions() {
	fmt.Println("ClientManager: Cleaning up legacy session files...")

	legacyFiles := []string{
		m.cfg.WhatsAppDBBasePath, // e.g., "./whatsapp.db"
		m.cfg.TelegramDBBasePath, // e.g., "./telegram.db"
	}

	for _, file := range legacyFiles {
		if _, err := os.Stat(file); err == nil {
			if err := os.Remove(file); err != nil {
				fmt.Printf("Warning: Failed to delete legacy session file %s: %v\n", file, err)
			} else {
				fmt.Printf("ClientManager: Deleted legacy session file: %s\n", file)
			}
		}
	}

	fmt.Println("ClientManager: Legacy session cleanup complete")
}

// Shutdown gracefully shuts down all clients
func (m *ClientManager) Shutdown(ctx context.Context) error {
	fmt.Println("ClientManager: Shutting down all clients...")

	m.mu.Lock()
	defer m.mu.Unlock()

	// Disconnect all WhatsApp clients
	for userID, client := range m.whatsappClients {
		fmt.Printf("ClientManager: Disconnecting WhatsApp for user %d\n", userID)
		if client.IsLoggedIn() {
			client.Disconnect()
		}
	}

	// Disconnect all Telegram clients
	for userID, client := range m.telegramClients {
		fmt.Printf("ClientManager: Disconnecting Telegram for user %d\n", userID)
		if client.IsConnected() {
			client.Disconnect()
		}
	}

	// Clear maps
	m.whatsappClients = make(map[int64]*whatsapp.Client)
	m.telegramClients = make(map[int64]*telegram.Client)

	// Close message channel
	close(m.msgChan)

	fmt.Println("ClientManager: Shutdown complete")
	return nil
}

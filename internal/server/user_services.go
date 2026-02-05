package server

import (
	"fmt"
	"sync"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/clients"
	"github.com/omriShneor/project_alfred/internal/config"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/gcal"
	"github.com/omriShneor/project_alfred/internal/gmail"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/processor"
)

// UserServices holds the active services for a single user
type UserServices struct {
	UserID      int64
	GmailWorker *gmail.Worker
	Processor   *processor.Processor
	running     bool
}

// UserServiceManager handles per-user service lifecycle
type UserServiceManager struct {
	db               *database.DB
	cfg              *config.Config
	credentialsFile  string // Path to Google OAuth credentials file (for per-user gcal clients)
	notifyService    *notify.Service
	eventAnalyzer    agent.EventAnalyzer
	reminderAnalyzer agent.ReminderAnalyzer

	// ClientManager for per-user WhatsApp/Telegram clients
	clientManager *clients.ClientManager

	// Active services per user
	mu           sync.RWMutex
	userServices map[int64]*UserServices
}

// UserServiceManagerConfig holds configuration for creating a UserServiceManager
type UserServiceManagerConfig struct {
	DB               *database.DB
	Config           *config.Config
	CredentialsFile  string // Path to Google OAuth credentials file
	NotifyService    *notify.Service
	EventAnalyzer    agent.EventAnalyzer
	ReminderAnalyzer agent.ReminderAnalyzer
	ClientManager    *clients.ClientManager
}

// NewUserServiceManager creates a new UserServiceManager
func NewUserServiceManager(cfg UserServiceManagerConfig) *UserServiceManager {
	return &UserServiceManager{
		db:               cfg.DB,
		cfg:              cfg.Config,
		credentialsFile:  cfg.CredentialsFile,
		notifyService:    cfg.NotifyService,
		eventAnalyzer:    cfg.EventAnalyzer,
		reminderAnalyzer: cfg.ReminderAnalyzer,
		clientManager:    cfg.ClientManager,
		userServices:     make(map[int64]*UserServices),
	}
}


// StartServicesForUser initializes and starts services for a specific user
func (m *UserServiceManager) StartServicesForUser(userID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if services already running for this user
	if existing, ok := m.userServices[userID]; ok && existing.running {
		fmt.Printf("Services already running for user %d\n", userID)
		return nil
	}

	fmt.Printf("Starting services for user %d\n", userID)

	services := &UserServices{
		UserID: userID,
	}

	// Note: Per-user WhatsApp/Telegram clients are managed by ClientManager
	// They are created on-demand when handlers need them
	// No need to set userID on handlers - already done during client creation

	// Start Gmail worker if user has authenticated Google Calendar (for Gmail access)
	gmailWorker, err := m.createGmailWorker(userID)
	if err != nil {
		fmt.Printf("  - Gmail worker failed to start: %v\n", err)
	} else if gmailWorker != nil {
		services.GmailWorker = gmailWorker
		fmt.Printf("  - Gmail worker started\n")
	}

	// Start processor
	if m.clientManager != nil && m.eventAnalyzer != nil {
		// Get message channel from ClientManager
		msgChan := m.clientManager.MessageChan()
		proc := processor.New(
			m.db,
			m.eventAnalyzer,
			m.reminderAnalyzer,
			msgChan,
			m.cfg.MessageHistorySize,
			m.notifyService,
		)
		if err := proc.Start(); err != nil {
			fmt.Printf("  - Processor failed to start: %v\n", err)
		} else {
			services.Processor = proc
			fmt.Printf("  - Processor started\n")
		}
	}

	services.running = true
	m.userServices[userID] = services

	fmt.Printf("Services started for user %d\n", userID)
	return nil
}

// StopServicesForUser stops all services for a specific user
func (m *UserServiceManager) StopServicesForUser(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	services, ok := m.userServices[userID]
	if !ok || !services.running {
		return
	}

	fmt.Printf("Stopping services for user %d\n", userID)

	if services.GmailWorker != nil {
		services.GmailWorker.Stop()
	}

	if services.Processor != nil {
		services.Processor.Stop()
	}

	// Cleanup WhatsApp/Telegram clients for this user
	if m.clientManager != nil {
		if err := m.clientManager.CleanupUser(userID); err != nil {
			fmt.Printf("  - Warning: Failed to cleanup clients for user %d: %v\n", userID, err)
		} else {
			fmt.Printf("  - Clients cleaned up for user %d\n", userID)
		}
	}

	services.running = false
	delete(m.userServices, userID)
}

// StopAllServices stops services for all users (called on shutdown)
func (m *UserServiceManager) StopAllServices() {
	m.mu.Lock()
	userIDs := make([]int64, 0, len(m.userServices))
	for userID := range m.userServices {
		userIDs = append(userIDs, userID)
	}
	m.mu.Unlock()

	for _, userID := range userIDs {
		m.StopServicesForUser(userID)
	}
}

// IsRunningForUser checks if services are running for a user
func (m *UserServiceManager) IsRunningForUser(userID int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	services, ok := m.userServices[userID]
	return ok && services.running
}

// GetGmailWorkerForUser retrieves the Gmail worker for a specific user
func (m *UserServiceManager) GetGmailWorkerForUser(userID int64) *gmail.Worker {
	m.mu.RLock()
	defer m.mu.RUnlock()

	services, ok := m.userServices[userID]
	if !ok || !services.running {
		return nil
	}

	return services.GmailWorker
}

// createGmailWorker creates and starts a Gmail worker for a user
func (m *UserServiceManager) createGmailWorker(userID int64) (*gmail.Worker, error) {
	// Create per-user gcal client to get OAuth token for Gmail
	if m.credentialsFile == "" || userID == 0 {
		return nil, nil
	}

	userGCalClient, err := gcal.NewClientForUser(userID, m.credentialsFile, m.db)
	if err != nil || userGCalClient == nil || !userGCalClient.IsAuthenticated() {
		return nil, nil
	}

	oauthConfig := userGCalClient.GetOAuthConfig()
	oauthToken := userGCalClient.GetToken()
	if oauthConfig == nil || oauthToken == nil {
		return nil, nil
	}

	gmailClient, err := gmail.NewClient(oauthConfig, oauthToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail client: %w", err)
	}

	if !gmailClient.IsAuthenticated() {
		return nil, nil
	}

	emailProc := processor.NewEmailProcessor(m.db, m.eventAnalyzer, m.reminderAnalyzer, m.notifyService)

	pollInterval := 1 // Default 1 minute
	maxEmails := 10   // Default 10
	if m.cfg != nil {
		if m.cfg.GmailPollInterval > 0 {
			pollInterval = m.cfg.GmailPollInterval
		}
		if m.cfg.GmailMaxEmails > 0 {
			maxEmails = m.cfg.GmailMaxEmails
		}
	}

	worker := gmail.NewWorker(gmailClient, m.db, emailProc, gmail.WorkerConfig{
		UserID:              userID,
		PollIntervalMinutes: pollInterval,
		MaxEmailsPerPoll:    maxEmails,
	})

	if err := worker.Start(); err != nil {
		return nil, fmt.Errorf("failed to start Gmail worker: %w", err)
	}

	return worker, nil
}

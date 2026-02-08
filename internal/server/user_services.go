package server

import (
	"fmt"
	"sync"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/auth"
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
	GCalWorker  *gcal.Worker
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

	// Global processor (single instance for all users)
	globalProcessor *processor.Processor
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

// StartGlobalProcessor starts a single shared processor for all users.
// This must only be started once to avoid multiple consumers on the shared channel.
func (m *UserServiceManager) StartGlobalProcessor() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.globalProcessor != nil {
		return nil
	}

	if m.clientManager == nil {
		return nil
	}

	if m.eventAnalyzer == nil && m.reminderAnalyzer == nil {
		return nil
	}

	msgChan := m.clientManager.MessageChan()
	historySize := 0
	if m.cfg != nil {
		historySize = m.cfg.MessageHistorySize
	}
	proc := processor.New(
		m.db,
		m.eventAnalyzer,
		m.reminderAnalyzer,
		msgChan,
		historySize,
		m.notifyService,
	)
	if err := proc.Start(); err != nil {
		return err
	}

	m.globalProcessor = proc
	fmt.Println("Global processor started")
	return nil
}

// StopGlobalProcessor stops the shared processor if running.
func (m *UserServiceManager) StopGlobalProcessor() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.globalProcessor != nil {
		m.globalProcessor.Stop()
		m.globalProcessor = nil
	}
}

// GlobalProcessorRunning returns true if the shared processor is running.
func (m *UserServiceManager) GlobalProcessorRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.globalProcessor != nil
}

// StartServicesForUser initializes and starts services for a specific user
func (m *UserServiceManager) StartServicesForUser(userID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if services already running for this user
	if existing, ok := m.userServices[userID]; ok && existing.running {
		// If Google Calendar worker isn't running yet, try to start it now.
		if existing.GCalWorker == nil {
			gcalWorker, err := m.createGCalWorker(userID)
			if err != nil {
				fmt.Printf("  - Google Calendar worker failed to start: %v\n", err)
			} else if gcalWorker != nil {
				existing.GCalWorker = gcalWorker
				fmt.Printf("  - Google Calendar worker started\n")
			}
		}

		// If Gmail worker isn't running yet, try to start it now.
		if existing.GmailWorker == nil {
			gmailWorker, err := m.createGmailWorker(userID)
			if err != nil {
				fmt.Printf("  - Gmail worker failed to start: %v\n", err)
			} else if gmailWorker != nil {
				existing.GmailWorker = gmailWorker
				fmt.Printf("  - Gmail worker started\n")
			}
		}
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

	// Start Google Calendar sync worker if user has authenticated Calendar scope
	gcalWorker, err := m.createGCalWorker(userID)
	if err != nil {
		fmt.Printf("  - Google Calendar worker failed to start: %v\n", err)
	} else if gcalWorker != nil {
		services.GCalWorker = gcalWorker
		fmt.Printf("  - Google Calendar worker started\n")
	}

	// Start Gmail worker if user has authenticated Google Calendar (for Gmail access)
	gmailWorker, err := m.createGmailWorker(userID)
	if err != nil {
		fmt.Printf("  - Gmail worker failed to start: %v\n", err)
	} else if gmailWorker != nil {
		services.GmailWorker = gmailWorker
		fmt.Printf("  - Gmail worker started\n")
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

	if services.GCalWorker != nil {
		services.GCalWorker.Stop()
	}

	if services.GmailWorker != nil {
		services.GmailWorker.Stop()
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

	m.StopGlobalProcessor()
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

// StopGmailWorkerForUser stops and removes the Gmail worker for a specific user.
func (m *UserServiceManager) StopGmailWorkerForUser(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	services, ok := m.userServices[userID]
	if !ok {
		return
	}

	if services.GmailWorker != nil {
		services.GmailWorker.Stop()
	}

	services.GmailWorker = nil
	if services.GCalWorker == nil {
		services.running = false
		delete(m.userServices, userID)
	}
}

// StopGCalWorkerForUser stops and removes the Google Calendar worker for a specific user.
func (m *UserServiceManager) StopGCalWorkerForUser(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	services, ok := m.userServices[userID]
	if !ok {
		return
	}

	if services.GCalWorker != nil {
		services.GCalWorker.Stop()
	}

	services.GCalWorker = nil
	if services.GmailWorker == nil {
		services.running = false
		delete(m.userServices, userID)
	}
}

// StartServicesForEligibleUsers starts services for any user with valid auth/sessions.
// This is used on server startup to keep background processing always-on.
func (m *UserServiceManager) StartServicesForEligibleUsers() {
	if m.db == nil {
		return
	}

	userIDs := make(map[int64]struct{})

	if waUsers, err := m.db.ListUsersWithWhatsAppSession(); err == nil {
		for _, userID := range waUsers {
			userIDs[userID] = struct{}{}
		}
	}

	if tgUsers, err := m.db.ListUsersWithTelegramSession(); err == nil {
		for _, userID := range tgUsers {
			userIDs[userID] = struct{}{}
		}
	}

	if tokenUsers, err := m.db.ListUsersWithGoogleToken(); err == nil {
		for _, userID := range tokenUsers {
			info, err := m.db.GetGoogleTokenInfo(userID)
			if err != nil || info == nil || !info.HasToken {
				continue
			}
			if hasScope(info.Scopes, auth.CalendarScopes[0]) {
				userIDs[userID] = struct{}{}
			}
			if hasScope(info.Scopes, auth.GmailScopes[0]) {
				userIDs[userID] = struct{}{}
				_ = m.db.SetGmailEnabled(userID, true)
			}
		}
	}

	for userID := range userIDs {
		if err := m.StartServicesForUser(userID); err != nil {
			fmt.Printf("Warning: failed to start services for user %d: %v\n", userID, err)
		}
	}
}

func hasScope(scopes []string, scope string) bool {
	for _, s := range scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// createGmailWorker creates and starts a Gmail worker for a user
func (m *UserServiceManager) createGmailWorker(userID int64) (*gmail.Worker, error) {
	// Create per-user gcal client to get OAuth token for Gmail
	if m.credentialsFile == "" || userID == 0 {
		return nil, nil
	}

	// Require Gmail scope before starting a Gmail worker
	tokenInfo, err := m.db.GetGoogleTokenInfo(userID)
	if err != nil || tokenInfo == nil || !tokenInfo.HasToken {
		return nil, nil
	}
	if !hasScope(tokenInfo.Scopes, auth.GmailScopes[0]) {
		return nil, nil
	}

	_ = m.db.SetGmailEnabled(userID, true)

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

// createGCalWorker creates and starts a Google Calendar worker for a user.
// The worker periodically syncs Google-side edits/deletes back into Alfred's DB.
func (m *UserServiceManager) createGCalWorker(userID int64) (*gcal.Worker, error) {
	if m.credentialsFile == "" || userID == 0 {
		return nil, nil
	}

	// Require Calendar scope before starting a Calendar worker.
	tokenInfo, err := m.db.GetGoogleTokenInfo(userID)
	if err != nil || tokenInfo == nil || !tokenInfo.HasToken {
		return nil, nil
	}
	if !hasScope(tokenInfo.Scopes, auth.CalendarScopes[0]) {
		return nil, nil
	}

	userGCalClient, err := gcal.NewClientForUser(userID, m.credentialsFile, m.db)
	if err != nil || userGCalClient == nil || !userGCalClient.IsAuthenticated() {
		return nil, nil
	}

	pollInterval := 1 // Default 1 minute
	if m.cfg != nil && m.cfg.GCalPollInterval > 0 {
		pollInterval = m.cfg.GCalPollInterval
	}

	worker := gcal.NewWorker(userGCalClient, m.db, gcal.WorkerConfig{
		UserID:              userID,
		PollIntervalMinutes: pollInterval,
	})

	if err := worker.Start(); err != nil {
		return nil, fmt.Errorf("failed to start Google Calendar worker: %w", err)
	}

	// Trigger an immediate reconciliation pass on startup.
	worker.PollNow()

	return worker, nil
}

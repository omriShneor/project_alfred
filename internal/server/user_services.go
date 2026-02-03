package server

import (
	"fmt"
	"sync"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/config"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/gcal"
	"github.com/omriShneor/project_alfred/internal/gmail"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/processor"
	"github.com/omriShneor/project_alfred/internal/source"
	"github.com/omriShneor/project_alfred/internal/telegram"
	"github.com/omriShneor/project_alfred/internal/whatsapp"
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
	notifyService    *notify.Service
	eventAnalyzer    agent.EventAnalyzer
	reminderAnalyzer agent.ReminderAnalyzer

	// Shared clients (currently single-instance, will be per-user later)
	gcalClient *gcal.Client
	waClient   *whatsapp.Client
	tgClient   *telegram.Client

	// Message channel for processor
	msgChan <-chan source.Message

	// Active services per user
	mu           sync.RWMutex
	userServices map[int64]*UserServices
}

// UserServiceManagerConfig holds configuration for creating a UserServiceManager
type UserServiceManagerConfig struct {
	DB               *database.DB
	Config           *config.Config
	NotifyService    *notify.Service
	EventAnalyzer    agent.EventAnalyzer
	ReminderAnalyzer agent.ReminderAnalyzer
	GCalClient       *gcal.Client
	WAClient         *whatsapp.Client
	TGClient         *telegram.Client
	MsgChan          <-chan source.Message
}

// NewUserServiceManager creates a new UserServiceManager
func NewUserServiceManager(cfg UserServiceManagerConfig) *UserServiceManager {
	return &UserServiceManager{
		db:               cfg.DB,
		cfg:              cfg.Config,
		notifyService:    cfg.NotifyService,
		eventAnalyzer:    cfg.EventAnalyzer,
		reminderAnalyzer: cfg.ReminderAnalyzer,
		gcalClient:       cfg.GCalClient,
		waClient:         cfg.WAClient,
		tgClient:         cfg.TGClient,
		msgChan:          cfg.MsgChan,
		userServices:     make(map[int64]*UserServices),
	}
}

// SetClients updates the shared clients (used after OAuth completion)
func (m *UserServiceManager) SetClients(gcalClient *gcal.Client, waClient *whatsapp.Client, tgClient *telegram.Client, msgChan <-chan source.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gcalClient = gcalClient
	m.waClient = waClient
	m.tgClient = tgClient
	m.msgChan = msgChan
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

	// Set userID on WhatsApp handler
	if m.waClient != nil {
		m.waClient.SetUserID(userID)
		fmt.Printf("  - WhatsApp handler userID set to %d\n", userID)
	}

	// Set userID on Telegram handler
	if m.tgClient != nil {
		m.tgClient.SetUserID(userID)
		fmt.Printf("  - Telegram handler userID set to %d\n", userID)
	}

	// Start Gmail worker if Gmail client is authenticated
	if m.gcalClient != nil && m.gcalClient.IsAuthenticated() {
		gmailWorker, err := m.createGmailWorker(userID)
		if err != nil {
			fmt.Printf("  - Gmail worker failed to start: %v\n", err)
		} else if gmailWorker != nil {
			services.GmailWorker = gmailWorker
			fmt.Printf("  - Gmail worker started\n")
		}
	}

	// Start processor
	if m.msgChan != nil && m.eventAnalyzer != nil {
		proc := processor.New(
			m.db,
			m.gcalClient,
			m.eventAnalyzer,
			m.reminderAnalyzer,
			m.msgChan,
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

// createGmailWorker creates and starts a Gmail worker for a user
func (m *UserServiceManager) createGmailWorker(userID int64) (*gmail.Worker, error) {
	if m.gcalClient == nil || !m.gcalClient.IsAuthenticated() {
		return nil, nil
	}

	oauthConfig := m.gcalClient.GetOAuthConfig()
	oauthToken := m.gcalClient.GetToken()
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

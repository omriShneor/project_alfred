package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/gcal"
	"github.com/omriShneor/project_alfred/internal/gmail"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/processor"
	"github.com/omriShneor/project_alfred/internal/sse"
	"github.com/omriShneor/project_alfred/internal/telegram"
	"github.com/omriShneor/project_alfred/internal/whatsapp"
)

type Server struct {
	db              *database.DB
	waClient        *whatsapp.Client
	tgClient        *telegram.Client
	gcalClient      *gcal.Client
	gmailClient     *gmail.Client
	gmailWorker     *gmail.Worker
	onboardingState *sse.State
	state           *sse.State // Alias for onboardingState (for consistency)
	notifyService   *notify.Service
	analyzer        agent.Analyzer
	httpSrv         *http.Server
	port            int
	resendAPIKey    string      // For checking email availability
	oauthCodeChan   chan string // Channel to receive OAuth code from callback
	// Gmail worker config
	gmailPollInterval int
	gmailMaxEmails    int
}

// ServerConfig holds configuration for initial server creation (onboarding-capable)
type ServerConfig struct {
	DB              *database.DB
	OnboardingState *sse.State
	Port            int
	ResendAPIKey    string
}

// ClientsConfig holds configuration for completing initialization after onboarding
type ClientsConfig struct {
	WAClient      *whatsapp.Client
	TGClient      *telegram.Client
	GCalClient    *gcal.Client
	GmailClient   *gmail.Client
	GmailWorker   *gmail.Worker
	NotifyService *notify.Service
	Analyzer      agent.Analyzer
	// Gmail worker config
	GmailPollInterval int
	GmailMaxEmails    int
}

func New(cfg ServerConfig) *Server {
	s := &Server{
		db:              cfg.DB,
		onboardingState: cfg.OnboardingState,
		state:           cfg.OnboardingState, // Alias for consistency
		port:            cfg.Port,
		resendAPIKey:    cfg.ResendAPIKey,
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	s.httpSrv = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      s.corsMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// InitializeClients completes server initialization after onboarding
func (s *Server) InitializeClients(cfg ClientsConfig) {
	s.waClient = cfg.WAClient
	s.tgClient = cfg.TGClient
	s.gcalClient = cfg.GCalClient
	s.gmailClient = cfg.GmailClient
	s.gmailWorker = cfg.GmailWorker
	s.notifyService = cfg.NotifyService
	s.analyzer = cfg.Analyzer
	s.gmailPollInterval = cfg.GmailPollInterval
	s.gmailMaxEmails = cfg.GmailMaxEmails
}

// SetGCalClient sets the gcal client (used during onboarding before full initialization)
func (s *Server) SetGCalClient(client *gcal.Client) {
	s.gcalClient = client
}

// SetWAClient sets the WhatsApp client (used during onboarding before full initialization)
func (s *Server) SetWAClient(client *whatsapp.Client) {
	s.waClient = client
}

// SetTGClient sets the Telegram client (used during onboarding before full initialization)
func (s *Server) SetTGClient(client *telegram.Client) {
	s.tgClient = client
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Health check
	mux.HandleFunc("GET /health", s.handleHealthCheck)

	// Onboarding API
	mux.HandleFunc("GET /api/onboarding/status", s.handleOnboardingStatus)
	mux.HandleFunc("GET /api/onboarding/stream", s.handleOnboardingSSE)

	// WhatsApp API
	mux.HandleFunc("GET /api/whatsapp/status", s.handleWhatsAppStatus)
	mux.HandleFunc("POST /api/whatsapp/pair", s.handleWhatsAppPair)
	mux.HandleFunc("POST /api/whatsapp/reconnect", s.handleWhatsAppReconnect)
	mux.HandleFunc("POST /api/whatsapp/disconnect", s.handleWhatsAppDisconnect)
	mux.HandleFunc("GET /api/whatsapp/top-contacts", s.handleWhatsAppTopContacts)
	mux.HandleFunc("POST /api/whatsapp/sources/custom", s.handleWhatsAppCustomSource)

	// Telegram API
	mux.HandleFunc("GET /api/telegram/status", s.handleTelegramStatus)
	mux.HandleFunc("POST /api/telegram/send-code", s.handleTelegramSendCode)
	mux.HandleFunc("POST /api/telegram/verify-code", s.handleTelegramVerifyCode)
	mux.HandleFunc("POST /api/telegram/disconnect", s.handleTelegramDisconnect)
	mux.HandleFunc("POST /api/telegram/reconnect", s.handleTelegramReconnect)
	mux.HandleFunc("GET /api/telegram/discovery/channels", s.handleDiscoverTelegramChannels)
	mux.HandleFunc("GET /api/telegram/channel", s.handleListTelegramChannels)
	mux.HandleFunc("POST /api/telegram/channel", s.handleCreateTelegramChannel)
	mux.HandleFunc("PUT /api/telegram/channel/{id}", s.handleUpdateTelegramChannel)
	mux.HandleFunc("DELETE /api/telegram/channel/{id}", s.handleDeleteTelegramChannel)
	mux.HandleFunc("GET /api/telegram/top-contacts", s.handleTelegramTopContacts)
	mux.HandleFunc("POST /api/telegram/sources/custom", s.handleTelegramCustomSource)

	// Discovery API
	mux.HandleFunc("GET /api/discovery/channels", s.handleDiscoverChannels)

	//Whatsapp Channel Registry API
	mux.HandleFunc("GET /api/channel", s.handleListChannels)
	mux.HandleFunc("POST /api/channel", s.handleCreateChannel)
	mux.HandleFunc("PUT /api/channel/{id}", s.handleUpdateChannel)
	mux.HandleFunc("DELETE /api/channel/{id}", s.handleDeleteChannel)

	// Google Calendar API
	mux.HandleFunc("GET /api/gcal/status", s.handleGCalStatus)
	mux.HandleFunc("GET /api/gcal/calendars", s.handleGCalListCalendars)
	mux.HandleFunc("GET /api/gcal/settings", s.handleGetGCalSettings)
	mux.HandleFunc("PUT /api/gcal/settings", s.handleUpdateGCalSettings)
	mux.HandleFunc("GET /api/gcal/events/today", s.handleListTodayEvents)
	mux.HandleFunc("POST /api/gcal/connect", s.handleGCalConnect)
	mux.HandleFunc("POST /api/gcal/callback", s.handleGCalExchangeCode)
	mux.HandleFunc("POST /api/gcal/disconnect", s.handleGCalDisconnect)
	mux.HandleFunc("GET /oauth/callback", s.handleOAuthCallback)

	// Events API
	mux.HandleFunc("GET /api/events", s.handleListEvents)
	mux.HandleFunc("GET /api/events/today", s.handleListMergedTodayEvents) // Today's Schedule (Alfred + external)
	mux.HandleFunc("GET /api/events/{id}", s.handleGetEvent)
	mux.HandleFunc("PUT /api/events/{id}", s.handleUpdateEvent)
	mux.HandleFunc("POST /api/events/{id}/confirm", s.handleConfirmEvent)
	mux.HandleFunc("POST /api/events/{id}/reject", s.handleRejectEvent)
	mux.HandleFunc("GET /api/events/channel/{channelId}/history", s.handleGetChannelHistory)

	// Notification Preferences API
	mux.HandleFunc("GET /api/notifications/preferences", s.handleGetNotificationPrefs)
	mux.HandleFunc("PUT /api/notifications/email", s.handleUpdateEmailPrefs)
	mux.HandleFunc("POST /api/notifications/push/register", s.handleRegisterPushToken)
	mux.HandleFunc("PUT /api/notifications/push", s.handleUpdatePushPrefs)

	// Gmail Top Contacts API (cached, fast discovery)
	mux.HandleFunc("GET /api/gmail/top-contacts", s.handleGetTopContacts)
	mux.HandleFunc("POST /api/gmail/sources/custom", s.handleAddCustomSource)

	// Gmail Sources API
	mux.HandleFunc("GET /api/gmail/status", s.handleGmailStatus)
	mux.HandleFunc("GET /api/gmail/sources", s.handleListEmailSources)
	mux.HandleFunc("POST /api/gmail/sources", s.handleCreateEmailSource)
	mux.HandleFunc("PUT /api/gmail/sources/{id}", s.handleUpdateEmailSource)
	mux.HandleFunc("DELETE /api/gmail/sources/{id}", s.handleDeleteEmailSource)

	// App Status API (new simplified flow)
	mux.HandleFunc("GET /api/app/status", s.handleGetAppStatus)
	mux.HandleFunc("POST /api/onboarding/complete", s.handleCompleteOnboarding)
	mux.HandleFunc("POST /api/onboarding/reset", s.handleResetOnboarding)
}

func (s *Server) Start() error {
	fmt.Printf("Starting HTTP server on http://localhost:%d\n", s.port)
	return s.httpSrv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

// Handler returns the server's HTTP handler for testing purposes
func (s *Server) Handler() http.Handler {
	return s.httpSrv.Handler
}

// corsMiddleware adds CORS headers to allow mobile app requests
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// initializeGmailClient creates and initializes the Gmail client after OAuth authentication.
// This should be called after gcalClient.ExchangeCode() succeeds.
func (s *Server) initializeGmailClient() error {
	if s.gcalClient == nil || !s.gcalClient.IsAuthenticated() {
		return fmt.Errorf("Google Calendar client not authenticated")
	}

	// Stop existing Gmail worker if running
	if s.gmailWorker != nil {
		s.gmailWorker.Stop()
		s.gmailWorker = nil
	}

	oauthConfig := s.gcalClient.GetOAuthConfig()
	oauthToken := s.gcalClient.GetToken()
	if oauthConfig == nil || oauthToken == nil {
		return fmt.Errorf("OAuth config or token not available")
	}

	gmailClient, err := gmail.NewClient(oauthConfig, oauthToken)
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %w", err)
	}

	if !gmailClient.IsAuthenticated() {
		return fmt.Errorf("Gmail client created but not authenticated")
	}

	s.gmailClient = gmailClient
	fmt.Println("Gmail client initialized after OAuth")

	// Create and start Gmail worker if we have the required dependencies
	if s.db != nil && s.analyzer != nil && s.notifyService != nil {
		emailProc := processor.NewEmailProcessor(s.db, s.analyzer, s.notifyService)
		pollInterval := s.gmailPollInterval
		if pollInterval <= 0 {
			pollInterval = 1 // Default to 1 minute
		}
		maxEmails := s.gmailMaxEmails
		if maxEmails <= 0 {
			maxEmails = 10 // Default to 10
		}
		s.gmailWorker = gmail.NewWorker(gmailClient, s.db, emailProc, gmail.WorkerConfig{
			PollIntervalMinutes: pollInterval,
			MaxEmailsPerPoll:    maxEmails,
		})
		if err := s.gmailWorker.Start(); err != nil {
			fmt.Printf("Warning: Gmail worker failed to start: %v\n", err)
		} else {
			fmt.Println("Gmail worker started")
			// Force refresh top contacts on re-authentication
			s.gmailWorker.RefreshTopContactsNow()
		}
	}

	return nil
}

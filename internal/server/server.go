package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/auth"
	"github.com/omriShneor/project_alfred/internal/clients"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/gcal"
	"github.com/omriShneor/project_alfred/internal/gmail"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/sse"
)

type Server struct {
	db               *database.DB
	clientManager    *clients.ClientManager
	gmailClient      *gmail.Client
	gmailWorker      *gmail.Worker
	onboardingState  *sse.State
	state            *sse.State // Alias for onboardingState (for consistency)
	notifyService    *notify.Service
	eventAnalyzer    agent.EventAnalyzer
	reminderAnalyzer agent.ReminderAnalyzer
	httpSrv         *http.Server
	port            int
	resendAPIKey    string // For checking email availability
	credentialsFile string // Path to Google OAuth credentials file (for per-user gcal clients)
	devMode         bool   // Enable development features
	// Authentication
	authService    *auth.Service
	authMiddleware *auth.Middleware
	// Per-user service management
	userServiceManager *UserServiceManager
}

// ServerConfig holds configuration for initial server creation (onboarding-capable)
type ServerConfig struct {
	DB              *database.DB
	OnboardingState *sse.State
	Port            int
	ResendAPIKey    string
	DevMode         bool // Enable development features (e.g., unauthenticated reset)
	// Auth configuration (optional - auth disabled if not provided)
	CredentialsFile string // Path to Google OAuth credentials file
	CredentialsJSON string // Google OAuth credentials as JSON string
}

// ClientsConfig holds configuration for completing initialization after onboarding
type ClientsConfig struct {
	GmailClient      *gmail.Client
	GmailWorker      *gmail.Worker
	NotifyService    *notify.Service
	EventAnalyzer    agent.EventAnalyzer
	ReminderAnalyzer agent.ReminderAnalyzer
}

func New(cfg ServerConfig) *Server {
	s := &Server{
		db:              cfg.DB,
		onboardingState: cfg.OnboardingState,
		state:           cfg.OnboardingState, // Alias for consistency
		port:            cfg.Port,
		resendAPIKey:    cfg.ResendAPIKey,
		credentialsFile: cfg.CredentialsFile,
		devMode:         cfg.DevMode,
	}

	if cfg.DevMode {
		fmt.Println("Development mode enabled - some endpoints will bypass authentication")
	}

	// Initialize authentication if credentials are available
	authCfg := AuthConfig{
		CredentialsFile: cfg.CredentialsFile,
		CredentialsJSON: cfg.CredentialsJSON,
	}
	if err := s.initAuth(authCfg); err != nil {
		// Auth initialization is optional - log warning but continue
		fmt.Printf("Warning: authentication not configured: %v\n", err)
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
	s.gmailClient = cfg.GmailClient
	s.gmailWorker = cfg.GmailWorker
	s.notifyService = cfg.NotifyService
	s.eventAnalyzer = cfg.EventAnalyzer
	s.reminderAnalyzer = cfg.ReminderAnalyzer
}

// SetClientManager sets the ClientManager for per-user WhatsApp/Telegram clients
func (s *Server) SetClientManager(mgr *clients.ClientManager) {
	s.clientManager = mgr
}

// GetClientManager returns the ClientManager
func (s *Server) GetClientManager() *clients.ClientManager {
	return s.clientManager
}

// getGCalClientForUser creates or retrieves a GCal client for a specific user.
// Returns nil if credentials are not configured or userID is 0.
func (s *Server) getGCalClientForUser(userID int64) *gcal.Client {
	if userID == 0 || s.credentialsFile == "" {
		return nil
	}
	client, err := gcal.NewClientForUser(userID, s.credentialsFile, s.db)
	if err != nil {
		fmt.Printf("Warning: failed to create gcal client for user %d: %v\n", userID, err)
		return nil
	}
	return client
}

// SetUserServiceManager sets the user service manager
func (s *Server) SetUserServiceManager(mgr *UserServiceManager) {
	s.userServiceManager = mgr
}

// GetUserServiceManager returns the user service manager
func (s *Server) GetUserServiceManager() *UserServiceManager {
	return s.userServiceManager
}

// requireAuth wraps a handler to require authentication
func (s *Server) requireAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.authMiddleware == nil {
			// Auth not configured, allow through (development mode)
			handler(w, r)
			return
		}
		s.authMiddleware.RequireAuth(http.HandlerFunc(handler)).ServeHTTP(w, r)
	}
}

// requireAuthUnlessDevMode wraps a handler to require auth in production but allow in dev mode
func (s *Server) requireAuthUnlessDevMode(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// In dev mode, bypass auth and inject a default test user
		if s.devMode {
			fmt.Printf("Dev mode enabled - bypassing auth for %s\n", r.URL.Path)
			// Inject a default user with ID 1 for dev mode
			user := &auth.User{
				ID:    1,
				Email: "dev@localhost",
				Name:  "Dev User",
			}
			ctx := auth.SetUserInContext(r.Context(), user)
			handler(w, r.WithContext(ctx))
			return
		}
		fmt.Printf("Dev mode disabled - requiring auth for %s\n", r.URL.Path)
		// Otherwise use normal auth
		s.requireAuth(handler)(w, r)
	}
}

// optionalAuth wraps a handler to optionally populate user context if authenticated
func (s *Server) optionalAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.authMiddleware == nil {
			// Auth not configured, allow through
			handler(w, r)
			return
		}
		s.authMiddleware.OptionalAuth(http.HandlerFunc(handler)).ServeHTTP(w, r)
	}
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	// ============================================
	// PUBLIC ROUTES (no authentication required)
	// ============================================

	// Health check
	mux.HandleFunc("GET /health", s.handleHealthCheck)

	// Authentication API (must be public for login flow)
	mux.HandleFunc("POST /api/auth/google/login", s.handleAuthGoogleLogin)
	mux.HandleFunc("POST /api/auth/google/callback", s.handleAuthGoogleCallback)
	mux.HandleFunc("POST /api/auth/google/logout", s.handleAuthLogout)
	mux.HandleFunc("GET /api/auth/me", s.handleAuthMe)

	// Incremental authorization (requires auth - user must be logged in to add scopes)
	mux.HandleFunc("POST /api/auth/google/add-scopes", s.requireAuth(s.handleRequestAdditionalScopes))
	mux.HandleFunc("POST /api/auth/google/add-scopes/callback", s.requireAuth(s.handleAddScopesCallback))

	// Onboarding API (public for initial app load)
	mux.HandleFunc("GET /api/onboarding/status", s.handleOnboardingStatus)
	mux.HandleFunc("GET /api/onboarding/stream", s.handleOnboardingSSE)

	// OAuth callback for auth flow (browser redirect from Google, redirects to mobile deep link)
	mux.HandleFunc("GET /api/auth/callback", s.handleAuthOAuthCallback)

	// ============================================
	// OPTIONAL AUTH ROUTES (work for both authenticated and anonymous)
	// ============================================

	// App status works for both (returns defaults for anonymous)
	mux.HandleFunc("GET /api/app/status", s.optionalAuth(s.handleGetAppStatus))

	// ============================================
	// PROTECTED ROUTES (require authentication)
	// ============================================

	// WhatsApp API
	mux.HandleFunc("GET /api/whatsapp/status", s.requireAuth(s.handleWhatsAppStatus))
	mux.HandleFunc("POST /api/whatsapp/pair", s.requireAuth(s.handleWhatsAppPair))
	mux.HandleFunc("POST /api/whatsapp/reconnect", s.requireAuth(s.handleWhatsAppReconnect))
	mux.HandleFunc("POST /api/whatsapp/disconnect", s.requireAuth(s.handleWhatsAppDisconnect))
	mux.HandleFunc("GET /api/whatsapp/top-contacts", s.requireAuth(s.handleWhatsAppTopContacts))
	mux.HandleFunc("POST /api/whatsapp/sources/custom", s.requireAuth(s.handleWhatsAppCustomSource))

	// Telegram API
	mux.HandleFunc("GET /api/telegram/status", s.requireAuth(s.handleTelegramStatus))
	mux.HandleFunc("POST /api/telegram/send-code", s.requireAuth(s.handleTelegramSendCode))
	mux.HandleFunc("POST /api/telegram/verify-code", s.requireAuth(s.handleTelegramVerifyCode))
	mux.HandleFunc("POST /api/telegram/disconnect", s.requireAuth(s.handleTelegramDisconnect))
	mux.HandleFunc("POST /api/telegram/reconnect", s.requireAuth(s.handleTelegramReconnect))
	mux.HandleFunc("GET /api/telegram/discovery/channels", s.requireAuth(s.handleDiscoverTelegramChannels))
	mux.HandleFunc("GET /api/telegram/channel", s.requireAuth(s.handleListTelegramChannels))
	mux.HandleFunc("POST /api/telegram/channel", s.requireAuth(s.handleCreateTelegramChannel))
	mux.HandleFunc("PUT /api/telegram/channel/{id}", s.requireAuth(s.handleUpdateTelegramChannel))
	mux.HandleFunc("DELETE /api/telegram/channel/{id}", s.requireAuth(s.handleDeleteTelegramChannel))
	mux.HandleFunc("GET /api/telegram/top-contacts", s.requireAuth(s.handleTelegramTopContacts))
	mux.HandleFunc("POST /api/telegram/sources/custom", s.requireAuth(s.handleTelegramCustomSource))

	// Discovery API
	mux.HandleFunc("GET /api/discovery/channels", s.requireAuth(s.handleDiscoverChannels))

	// WhatsApp Channel Registry API
	mux.HandleFunc("GET /api/channel", s.requireAuth(s.handleListChannels))
	mux.HandleFunc("POST /api/channel", s.requireAuth(s.handleCreateChannel))
	mux.HandleFunc("PUT /api/channel/{id}", s.requireAuth(s.handleUpdateChannel))
	mux.HandleFunc("DELETE /api/channel/{id}", s.requireAuth(s.handleDeleteChannel))

	// Google Calendar API
	mux.HandleFunc("GET /api/gcal/status", s.requireAuth(s.handleGCalStatus))
	mux.HandleFunc("GET /api/gcal/calendars", s.requireAuth(s.handleGCalListCalendars))
	mux.HandleFunc("GET /api/gcal/settings", s.requireAuth(s.handleGetGCalSettings))
	mux.HandleFunc("PUT /api/gcal/settings", s.requireAuth(s.handleUpdateGCalSettings))
	mux.HandleFunc("GET /api/gcal/events/today", s.requireAuth(s.handleListTodayEvents))
	mux.HandleFunc("POST /api/gcal/disconnect", s.requireAuth(s.handleGCalDisconnect))

	// Events API
	mux.HandleFunc("GET /api/events", s.requireAuth(s.handleListEvents))
	mux.HandleFunc("GET /api/events/today", s.requireAuth(s.handleListMergedTodayEvents))
	mux.HandleFunc("GET /api/events/{id}", s.requireAuth(s.handleGetEvent))
	mux.HandleFunc("PUT /api/events/{id}", s.requireAuth(s.handleUpdateEvent))
	mux.HandleFunc("POST /api/events/{id}/confirm", s.requireAuth(s.handleConfirmEvent))
	mux.HandleFunc("POST /api/events/{id}/reject", s.requireAuth(s.handleRejectEvent))
	mux.HandleFunc("GET /api/events/channel/{channelId}/history", s.requireAuth(s.handleGetChannelHistory))

	// Reminders API
	mux.HandleFunc("GET /api/reminders", s.requireAuth(s.handleListReminders))
	mux.HandleFunc("GET /api/reminders/{id}", s.requireAuth(s.handleGetReminder))
	mux.HandleFunc("PUT /api/reminders/{id}", s.requireAuth(s.handleUpdateReminder))
	mux.HandleFunc("POST /api/reminders/{id}/confirm", s.requireAuth(s.handleConfirmReminder))
	mux.HandleFunc("POST /api/reminders/{id}/reject", s.requireAuth(s.handleRejectReminder))
	mux.HandleFunc("POST /api/reminders/{id}/complete", s.requireAuth(s.handleCompleteReminder))
	mux.HandleFunc("POST /api/reminders/{id}/dismiss", s.requireAuth(s.handleDismissReminder))

	// Notification Preferences API
	mux.HandleFunc("GET /api/notifications/preferences", s.requireAuth(s.handleGetNotificationPrefs))
	mux.HandleFunc("PUT /api/notifications/email", s.requireAuth(s.handleUpdateEmailPrefs))
	mux.HandleFunc("POST /api/notifications/push/register", s.requireAuth(s.handleRegisterPushToken))
	mux.HandleFunc("PUT /api/notifications/push", s.requireAuth(s.handleUpdatePushPrefs))

	// Gmail Top Contacts API
	mux.HandleFunc("GET /api/gmail/top-contacts", s.requireAuth(s.handleGetTopContacts))
	mux.HandleFunc("POST /api/gmail/sources/custom", s.requireAuth(s.handleAddCustomSource))

	// Gmail Sources API
	mux.HandleFunc("GET /api/gmail/status", s.requireAuth(s.handleGmailStatus))
	mux.HandleFunc("GET /api/gmail/sources", s.requireAuth(s.handleListEmailSources))
	mux.HandleFunc("POST /api/gmail/sources", s.requireAuth(s.handleCreateEmailSource))
	mux.HandleFunc("PUT /api/gmail/sources/{id}", s.requireAuth(s.handleUpdateEmailSource))
	mux.HandleFunc("DELETE /api/gmail/sources/{id}", s.requireAuth(s.handleDeleteEmailSource))

	// Onboarding completion (requires auth - user must be logged in)
	mux.HandleFunc("POST /api/onboarding/complete", s.requireAuth(s.handleCompleteOnboarding))
	// Reset endpoint - requires auth in production, but allows unauthenticated access in dev mode
	mux.HandleFunc("POST /api/onboarding/reset", s.requireAuthUnlessDevMode(s.handleResetOnboarding))
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

package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/gcal"
	"github.com/omriShneor/project_alfred/internal/gmail"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/sse"
	"github.com/omriShneor/project_alfred/internal/whatsapp"
)

type Server struct {
	db              *database.DB
	waClient        *whatsapp.Client
	gcalClient      *gcal.Client
	gmailClient     *gmail.Client
	onboardingState *sse.State
	notifyService   *notify.Service
	httpSrv         *http.Server
	port            int
	resendAPIKey    string      // For checking email availability
	oauthCodeChan   chan string // Channel to receive OAuth code from callback
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
	GCalClient    *gcal.Client
	GmailClient   *gmail.Client
	NotifyService *notify.Service
}

func New(cfg ServerConfig) *Server {
	s := &Server{
		db:              cfg.DB,
		onboardingState: cfg.OnboardingState,
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
	s.gcalClient = cfg.GCalClient
	s.gmailClient = cfg.GmailClient
	s.notifyService = cfg.NotifyService
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

	// Discovery API
	mux.HandleFunc("GET /api/discovery/channels", s.handleDiscoverChannels)

	// Channel Registry API
	mux.HandleFunc("GET /api/channel", s.handleListChannels)
	mux.HandleFunc("POST /api/channel", s.handleCreateChannel)
	mux.HandleFunc("PUT /api/channel/{id}", s.handleUpdateChannel)
	mux.HandleFunc("DELETE /api/channel/{id}", s.handleDeleteChannel)

	// Google Calendar API
	mux.HandleFunc("GET /api/gcal/status", s.handleGCalStatus)
	mux.HandleFunc("GET /api/gcal/calendars", s.handleGCalListCalendars)
	mux.HandleFunc("GET /api/gcal/events/today", s.handleListTodayEvents)
	mux.HandleFunc("POST /api/gcal/connect", s.handleGCalConnect)
	mux.HandleFunc("POST /api/gcal/callback", s.handleGCalExchangeCode)
	mux.HandleFunc("GET /oauth/callback", s.handleOAuthCallback)

	// Events API
	mux.HandleFunc("GET /api/events", s.handleListEvents)
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

	// Gmail API
	mux.HandleFunc("GET /api/gmail/status", s.handleGmailStatus)
	mux.HandleFunc("GET /api/gmail/settings", s.handleGmailSettings)
	mux.HandleFunc("PUT /api/gmail/settings", s.handleUpdateGmailSettings)

	// Gmail Discovery API
	mux.HandleFunc("GET /api/gmail/discover/categories", s.handleDiscoverGmailCategories)
	mux.HandleFunc("GET /api/gmail/discover/senders", s.handleDiscoverGmailSenders)
	mux.HandleFunc("GET /api/gmail/discover/domains", s.handleDiscoverGmailDomains)

	// Email Sources API (similar to WhatsApp channels)
	mux.HandleFunc("GET /api/gmail/sources", s.handleListEmailSources)
	mux.HandleFunc("POST /api/gmail/sources", s.handleCreateEmailSource)
	mux.HandleFunc("GET /api/gmail/sources/{id}", s.handleGetEmailSource)
	mux.HandleFunc("PUT /api/gmail/sources/{id}", s.handleUpdateEmailSource)
	mux.HandleFunc("DELETE /api/gmail/sources/{id}", s.handleDeleteEmailSource)
}

func (s *Server) Start() error {
	fmt.Printf("Starting HTTP server on http://localhost:%d\n", s.port)
	return s.httpSrv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
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

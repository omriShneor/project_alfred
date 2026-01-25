package server

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/gcal"
	"github.com/omriShneor/project_alfred/internal/sse"
	"github.com/omriShneor/project_alfred/internal/whatsapp"
)

//go:embed static/index.html static/settings.html
var staticFiles embed.FS

type Server struct {
	db              *database.DB
	waClient        *whatsapp.Client
	gcalClient      *gcal.Client
	onboardingState *sse.State
	httpSrv         *http.Server
	port            int
	devMode         bool
	resendAPIKey    string      // For checking email availability
	oauthCodeChan   chan string // Channel to receive OAuth code from callback
}

func New(db *database.DB, waClient *whatsapp.Client, gcalClient *gcal.Client, port int, onboardingState *sse.State, devMode bool, resendAPIKey string) *Server {
	s := &Server{
		db:              db,
		waClient:        waClient,
		gcalClient:      gcalClient,
		onboardingState: onboardingState,
		port:            port,
		devMode:         devMode,
		resendAPIKey:    resendAPIKey,
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	s.httpSrv = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      s.corsMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Health check
	mux.HandleFunc("GET /health", s.handleHealthCheck)

	// Main page (Channels + Events)
	mux.HandleFunc("GET /", s.handleMainPage)

	// Onboarding API (used by settings page for SSE)
	mux.HandleFunc("GET /api/onboarding/status", s.handleOnboardingStatus)
	mux.HandleFunc("GET /api/onboarding/stream", s.handleOnboardingSSE)

	// WhatsApp API
	mux.HandleFunc("POST /api/whatsapp/reconnect", s.handleWhatsAppReconnect)

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
	mux.HandleFunc("POST /api/gcal/connect", s.handleGCalConnect)
	mux.HandleFunc("GET /oauth/callback", s.handleOAuthCallback)

	// Events API
	mux.HandleFunc("GET /api/events", s.handleListEvents)
	mux.HandleFunc("GET /api/events/{id}", s.handleGetEvent)
	mux.HandleFunc("PUT /api/events/{id}", s.handleUpdateEvent)
	mux.HandleFunc("POST /api/events/{id}/confirm", s.handleConfirmEvent)
	mux.HandleFunc("POST /api/events/{id}/reject", s.handleRejectEvent)
	mux.HandleFunc("GET /api/events/channel/{channelId}/history", s.handleGetChannelHistory)

	// Settings Page
	mux.HandleFunc("GET /settings", s.handleSettingsPage)

	// Notification Preferences API
	mux.HandleFunc("GET /api/notifications/preferences", s.handleGetNotificationPrefs)
	mux.HandleFunc("PUT /api/notifications/email", s.handleUpdateEmailPrefs)
}

func (s *Server) Start() error {
	fmt.Printf("Starting HTTP server on http://localhost:%d\n", s.port)
	return s.httpSrv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

// SetClients updates the WhatsApp and GCal clients after onboarding completes
func (s *Server) SetClients(waClient *whatsapp.Client, gcalClient *gcal.Client) {
	s.waClient = waClient
	s.gcalClient = gcalClient
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

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

//go:embed static/admin.html static/events.html static/onboarding.html
var staticFiles embed.FS

type Server struct {
	db              *database.DB
	waClient        *whatsapp.Client
	gcalClient      *gcal.Client
	onboardingState *sse.State
	httpSrv         *http.Server
	port            int
	devMode         bool
}

func New(db *database.DB, waClient *whatsapp.Client, gcalClient *gcal.Client, port int, onboardingState *sse.State, devMode bool) *Server {
	s := &Server{
		db:              db,
		waClient:        waClient,
		gcalClient:      gcalClient,
		onboardingState: onboardingState,
		port:            port,
		devMode:         devMode,
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	s.httpSrv = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Dashboard
	mux.HandleFunc("GET /", s.handleDashboard)

	// Admin Page
	mux.HandleFunc("GET /admin", s.handleAdminPage)

	// Onboarding
	mux.HandleFunc("GET /onboarding", s.handleOnboardingPage)
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

	// Events Page (separate from Admin)
	mux.HandleFunc("GET /events", s.handleEventsPage)

	// Events API
	mux.HandleFunc("GET /api/events", s.handleListEvents)
	mux.HandleFunc("GET /api/events/{id}", s.handleGetEvent)
	mux.HandleFunc("PUT /api/events/{id}", s.handleUpdateEvent)
	mux.HandleFunc("POST /api/events/{id}/confirm", s.handleConfirmEvent)
	mux.HandleFunc("POST /api/events/{id}/reject", s.handleRejectEvent)
	mux.HandleFunc("GET /api/events/channel/{channelId}/history", s.handleGetChannelHistory)
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

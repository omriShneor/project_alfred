package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/whatsapp"
)

type Server struct {
	db       *database.DB
	waClient *whatsapp.Client
	httpSrv  *http.Server
	port     int
}

func New(db *database.DB, waClient *whatsapp.Client, port int) *Server {
	s := &Server{
		db:       db,
		waClient: waClient,
		port:     port,
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

	// Channel Registery API
	mux.HandleFunc("GET /api/channel", s.handleListChannels)
	mux.HandleFunc("POST /api/channel", s.handleCreateChannel)
	mux.HandleFunc("PUT /api/channel/{id}", s.handleUpdateChannel)
	mux.HandleFunc("DELETE /api/channel/{id}", s.handleDeleteChannel)

	// Events API
	mux.HandleFunc("GET /api/events", s.handleListEvents)
	mux.HandleFunc("GET /api/events/{id}/messages", s.handleEventMessages)
	mux.HandleFunc("POST /api/events/{id}/confirm", s.handleConfirmEvent)
	mux.HandleFunc("POST /api/events/{id}/reject", s.handleRejectEvent)
}

func (s *Server) Start() error {
	fmt.Printf("Starting HTTP server on http://localhost:%d\n", s.port)
	return s.httpSrv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

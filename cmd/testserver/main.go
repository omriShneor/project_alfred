// Package main provides a test server for E2E testing the mobile app.
// This server runs with in-memory SQLite and real Claude API for testing agent accuracy.
// External services (GCal, WhatsApp, Telegram, Gmail) are mocked.
//
// Usage:
//
//	ANTHROPIC_API_KEY=sk-... go run cmd/testserver/main.go
//
// The server exposes additional test control endpoints:
//   - POST /api/test/reset - Reset all data
//   - POST /api/test/inject-message - Inject a message for event detection
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/agent/event"
	"github.com/omriShneor/project_alfred/internal/config"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/processor"
	"github.com/omriShneor/project_alfred/internal/server"
	"github.com/omriShneor/project_alfred/internal/source"
	"github.com/omriShneor/project_alfred/internal/sse"
)

func main() {
	fmt.Println("Starting Project Alfred Test Server...")
	fmt.Println("This server uses in-memory SQLite and real Claude API for E2E testing.")

	// Load config
	cfg := config.LoadFromEnv()

	// Check for required env vars
	if cfg.AnthropicAPIKey == "" {
		fmt.Println("Warning: ANTHROPIC_API_KEY not set. Event detection will not work.")
	}

	// Create in-memory database
	db, err := database.New(":memory:")
	if err != nil {
		fmt.Printf("Failed to create database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println("In-memory database initialized")

	// Create SSE state for onboarding
	state := sse.NewState()

	// Create notify service (with push notifier)
	pushNotifier := notify.NewExpoPushNotifier()
	notifyService := notify.NewService(db, nil, pushNotifier)
	fmt.Println("Push notification service configured")

	// Create event analyzer (uses real Claude API if ANTHROPIC_API_KEY is set)
	var eventAnalyzer agent.EventAnalyzer
	if cfg.AnthropicAPIKey != "" {
		eventAnalyzer = event.NewAgent(event.Config{
			APIKey:      cfg.AnthropicAPIKey,
			Model:       cfg.ClaudeModel,
			Temperature: cfg.ClaudeTemperature,
		})
		fmt.Println("Claude API configured for event detection")
	}

	// Create message channel for processor
	msgChan := make(chan source.Message, 100)

	// Create message processor
	var messageProcessor *processor.Processor
	if eventAnalyzer != nil {
		messageProcessor = processor.New(db, eventAnalyzer, nil, msgChan, cfg.MessageHistorySize, notifyService)
		if err := messageProcessor.Start(); err != nil {
			fmt.Printf("Warning: processor failed to start: %v\n", err)
		} else {
			fmt.Println("Message processor initialized")
		}
	}

	// Create server
	serverCfg := server.ServerConfig{
		DB:              db,
		OnboardingState: state,
		Port:            cfg.HTTPPort,
	}
	srv := server.New(serverCfg)

	// Initialize clients with mock services
	clientsCfg := server.ClientsConfig{
		NotifyService: notifyService,
		EventAnalyzer: eventAnalyzer,
	}
	srv.InitializeClients(clientsCfg)

	// Create test control mux
	testMux := http.NewServeMux()

	// Wrap the server handler with test endpoints
	mainHandler := srv.Handler()
	testMux.HandleFunc("/api/test/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Reset database by recreating tables
		fmt.Println("Resetting test database...")

		// Reset onboarding for default test user (ID=1)
		if err := db.ResetOnboarding(1); err != nil {
			http.Error(w, fmt.Sprintf("Failed to reset onboarding: %v", err), http.StatusInternalServerError)
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{"status": "reset"})
	})

	testMux.HandleFunc("/api/test/inject-message", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if messageProcessor == nil {
			http.Error(w, "Message processor not configured (missing ANTHROPIC_API_KEY)", http.StatusServiceUnavailable)
			return
		}

		var req struct {
			ChannelID  int64  `json:"channel_id"`
			SenderID   string `json:"sender_id"`
			SenderName string `json:"sender_name"`
			Text       string `json:"text"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Create a message and send to processor via channel
		msg := source.Message{
			SourceType: source.SourceTypeWhatsApp,
			SourceID:   req.ChannelID,
			SenderID:   req.SenderID,
			SenderName: req.SenderName,
			Text:       req.Text,
			Timestamp:  time.Now(),
		}

		// Send message to processor channel
		fmt.Printf("Injecting message for channel %d: %s\n", req.ChannelID, req.Text)
		select {
		case msgChan <- msg:
			respondJSON(w, http.StatusOK, map[string]string{"status": "injected"})
		default:
			http.Error(w, "Message channel full", http.StatusServiceUnavailable)
		}
	})

	testMux.HandleFunc("/api/test/create-channel", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			UserID      int64  `json:"user_id"`
			SourceType  string `json:"source_type"`
			ChannelType string `json:"channel_type"`
			Identifier  string `json:"identifier"`
			Name        string `json:"name"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Default values
		if req.UserID == 0 {
			req.UserID = 1 // Default test user ID
		}
		if req.SourceType == "" {
			req.SourceType = "whatsapp"
		}
		if req.ChannelType == "" {
			req.ChannelType = "sender"
		}

		channel, err := db.CreateSourceChannel(
			req.UserID,
			source.SourceType(req.SourceType),
			source.ChannelType(req.ChannelType),
			req.Identifier,
			req.Name,
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create channel: %v", err), http.StatusInternalServerError)
			return
		}

		respondJSON(w, http.StatusCreated, channel)
	})

	// Fallback to main handler
	testMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		mainHandler.ServeHTTP(w, r)
	})

	// Create HTTP server with CORS
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      corsMiddleware(testMux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		fmt.Printf("\nTest Server running on http://localhost:%d\n", cfg.HTTPPort)
		fmt.Println("\nTest endpoints:")
		fmt.Println("  POST /api/test/reset         - Reset all data")
		fmt.Println("  POST /api/test/inject-message - Inject message for event detection")
		fmt.Println("  POST /api/test/create-channel - Create a test channel")
		fmt.Println("\nPress Ctrl+C to stop")

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nShutting down test server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if messageProcessor != nil {
		messageProcessor.Stop()
	}

	if err := httpServer.Shutdown(ctx); err != nil {
		fmt.Printf("Shutdown error: %v\n", err)
	}

	fmt.Println("Test server stopped")
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

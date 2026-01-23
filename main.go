package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/omriShneor/project_alfred/internal/claude"
	"github.com/omriShneor/project_alfred/internal/config"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/gcal"
	"github.com/omriShneor/project_alfred/internal/onboarding"
	"github.com/omriShneor/project_alfred/internal/processor"
	"github.com/omriShneor/project_alfred/internal/server"
	"github.com/omriShneor/project_alfred/internal/sse"
	"github.com/omriShneor/project_alfred/internal/whatsapp"
)

func main() {
	cfg := config.LoadFromEnv()

	db, err := database.New(cfg.DBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx := context.Background()

	// Create SSE state for onboarding
	state := sse.NewState()

	srv := server.New(db, nil, nil, cfg.HTTPPort, state, cfg.DevMode)
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
		}
	}()

	clients, err := onboarding.Initialize(ctx, db, cfg, state, func(waClient *whatsapp.Client, gcalClient *gcal.Client) {
		// Set clients on server immediately so they're available during onboarding
		srv.SetClients(waClient, gcalClient)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Initialization failed: %v\n", err)
		os.Exit(1)
	}

	var claudeClient *claude.Client
	if cfg.AnthropicAPIKey != "" {
		claudeClient = claude.NewClient(cfg.AnthropicAPIKey, cfg.ClaudeModel, cfg.ClaudeTemperature)
		fmt.Println("Claude API configured for event detection")
	} else {
		fmt.Println("Warning: ANTHROPIC_API_KEY not set, event detection disabled")
	}

	proc := processor.New(db, clients.GCalClient, claudeClient, clients.MsgChan, cfg.MessageHistorySize)
	if err := proc.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Event processor failed to start: %v\n", err)
	}

	waitForShutdown(proc, srv, clients.WAClient)
}

func waitForShutdown(proc *processor.Processor, srv *server.Server, waClient *whatsapp.Client) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc.Stop()
	srv.Shutdown(ctx)
	waClient.Disconnect()
}

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
	"github.com/omriShneor/project_alfred/internal/gmail"
	"github.com/omriShneor/project_alfred/internal/notify"
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

	srv := server.New(db, nil, nil, cfg.HTTPPort, state, cfg.ResendAPIKey, nil)
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

	// Initialize email notifier (server-side config only)
	var emailNotifier notify.Notifier
	if cfg.ResendAPIKey != "" {
		emailNotifier = notify.NewResendNotifier(
			cfg.ResendAPIKey,
			cfg.EmailFrom,
			fmt.Sprintf("http://localhost:%d", cfg.HTTPPort),
		)
		if emailNotifier != nil && emailNotifier.IsConfigured() {
			fmt.Println("Email notification service configured (Resend)")
		}
	}

	// Initialize push notifier (always available - Expo doesn't require server credentials)
	pushNotifier := notify.NewExpoPushNotifier()
	fmt.Println("Push notification service configured (Expo)")

	// Create notification service
	notifyService := notify.NewService(db, emailNotifier, pushNotifier)

	// Set notify service on server for API handlers
	srv.SetNotifyService(notifyService)

	proc := processor.New(db, clients.GCalClient, claudeClient, clients.MsgChan, cfg.MessageHistorySize, notifyService)
	if err := proc.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Event processor failed to start: %v\n", err)
	}

	// Initialize Gmail client and worker
	var gmailWorker *gmail.Worker
	if clients.GCalClient != nil && clients.GCalClient.IsAuthenticated() {
		oauthConfig := clients.GCalClient.GetOAuthConfig()
		oauthToken := clients.GCalClient.GetToken()
		if oauthConfig != nil && oauthToken != nil {
			gmailClient, err := gmail.NewClient(oauthConfig, oauthToken)
			if err != nil {
				fmt.Printf("Warning: Failed to create Gmail client: %v\n", err)
			} else if gmailClient.IsAuthenticated() {
				fmt.Println("Gmail client initialized")
				srv.SetGmailClient(gmailClient)

				// Create email processor for Gmail worker
				emailProc := processor.NewEmailProcessor(db, claudeClient, notifyService)

				// Create and start Gmail worker
				gmailWorker = gmail.NewWorker(gmailClient, db, emailProc, gmail.WorkerConfig{
					PollIntervalMinutes: cfg.GmailPollInterval,
					MaxEmailsPerPoll:    cfg.GmailMaxEmails,
				})
				if err := gmailWorker.Start(); err != nil {
					fmt.Printf("Warning: Gmail worker failed to start: %v\n", err)
				}
			} else {
				fmt.Println("Gmail client created but not authenticated (may need re-authorization for Gmail scope)")
			}
		}
	}

	waitForShutdown(proc, srv, clients.WAClient, gmailWorker)
}

func waitForShutdown(proc *processor.Processor, srv *server.Server, waClient *whatsapp.Client, gmailWorker *gmail.Worker) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc.Stop()
	if gmailWorker != nil {
		gmailWorker.Stop()
	}
	srv.Shutdown(ctx)
	waClient.Disconnect()
}

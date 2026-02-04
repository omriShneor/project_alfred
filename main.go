package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/agent/event"
	"github.com/omriShneor/project_alfred/internal/agent/reminder"
	"github.com/omriShneor/project_alfred/internal/config"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/onboarding"
	"github.com/omriShneor/project_alfred/internal/server"
	"github.com/omriShneor/project_alfred/internal/sse"
	"github.com/omriShneor/project_alfred/internal/telegram"
	"github.com/omriShneor/project_alfred/internal/whatsapp"
)

func main() {
	cfg := config.LoadFromEnv()

	// Phase 1: Core infrastructure
	db, err := initDatabase(cfg)
	if err != nil {
		fatal("creating database", err)
	}
	defer db.Close()

	state := sse.NewState()

	notifyService := initNotifyService(db, cfg)

	srv := server.New(server.ServerConfig{
		DB:              db,
		OnboardingState: state,
		Port:            cfg.HTTPPort,
		ResendAPIKey:    cfg.ResendAPIKey,
		DevMode:         cfg.DevMode,
		CredentialsFile: cfg.GoogleCredentialsFile,
		CredentialsJSON: cfg.GoogleCredentialsJSON,
	})
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
		}
	}()

	ctx := context.Background()
	clients, err := onboarding.Initialize(ctx, db, cfg, state, srv, notifyService)
	if err != nil {
		fatal("initialization", err)
	}

	eventAnalyzer := initEventAnalyzer(cfg)
	reminderAnalyzer := initReminderAnalyzer(cfg)
	tgClient := initTelegram(db, cfg, state)

	// Initialize clients on server (but don't start workers yet - they start after user login)
	srv.InitializeClients(server.ClientsConfig{
		WAClient:         clients.WAClient,
		TGClient:         tgClient,
		GmailClient:      nil, // Created per-user after login
		GmailWorker:      nil, // Created per-user after login
		NotifyService:    notifyService,
		EventAnalyzer:    eventAnalyzer,
		ReminderAnalyzer: reminderAnalyzer,
	})

	// Create UserServiceManager for per-user service lifecycle
	userServiceManager := server.NewUserServiceManager(server.UserServiceManagerConfig{
		DB:               db,
		Config:           cfg,
		CredentialsFile:  cfg.GoogleCredentialsFile,
		NotifyService:    notifyService,
		EventAnalyzer:    eventAnalyzer,
		ReminderAnalyzer: reminderAnalyzer,
		WAClient:         clients.WAClient,
		TGClient:         tgClient,
		MsgChan:          clients.MsgChan,
	})
	srv.SetUserServiceManager(userServiceManager)

	// NOTE: Workers and processors are NOT started here.
	// They will be started by UserServiceManager after user login + onboarding.

	waitForShutdown(srv, clients.WAClient, tgClient, userServiceManager)
}

func initDatabase(cfg *config.Config) (*database.DB, error) {
	return database.New(cfg.DBPath)
}

func initEventAnalyzer(cfg *config.Config) agent.EventAnalyzer {
	if cfg.AnthropicAPIKey == "" {
		fmt.Println("Warning: ANTHROPIC_API_KEY not set, event detection disabled")
		return nil
	}
	eventAgent := event.NewAgent(event.Config{
		APIKey:      cfg.AnthropicAPIKey,
		Model:       cfg.ClaudeModel,
		Temperature: cfg.ClaudeTemperature,
	})
	fmt.Println("Event agent configured (tool-calling mode)")
	return eventAgent
}

func initReminderAnalyzer(cfg *config.Config) agent.ReminderAnalyzer {
	if cfg.AnthropicAPIKey == "" {
		fmt.Println("Warning: ANTHROPIC_API_KEY not set, reminder detection disabled")
		return nil
	}
	reminderAgent := reminder.NewAgent(reminder.Config{
		APIKey:      cfg.AnthropicAPIKey,
		Model:       cfg.ClaudeModel,
		Temperature: cfg.ClaudeTemperature,
	})
	fmt.Println("Reminder agent configured (tool-calling mode)")
	return reminderAgent
}

func initNotifyService(db *database.DB, cfg *config.Config) *notify.Service {
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

	pushNotifier := notify.NewExpoPushNotifier()
	fmt.Println("Push notification service configured (Expo)")

	return notify.NewService(db, emailNotifier, pushNotifier)
}

func initTelegram(db *database.DB, cfg *config.Config, state *sse.State) *telegram.Client {
	if cfg.TelegramAPIID == 0 || cfg.TelegramAPIHash == "" {
		fmt.Println("Telegram: Not configured (ALFRED_TELEGRAM_API_ID and ALFRED_TELEGRAM_API_HASH required)")
		return nil
	}

	handler := telegram.NewHandler(db, cfg.DebugAllMessages, state)

	tgClient, err := telegram.NewClient(telegram.ClientConfig{
		APIID:       cfg.TelegramAPIID,
		APIHash:     cfg.TelegramAPIHash,
		SessionPath: cfg.TelegramDBPath,
		Handler:     handler,
	})
	if err != nil {
		fmt.Printf("Warning: Failed to create Telegram client: %v\n", err)
		return nil
	}

	fmt.Println("Telegram client initialized")

	// Auto-connect to restore session if exists
	if err := tgClient.Connect(); err != nil {
		fmt.Printf("Warning: Failed to auto-connect Telegram: %v\n", err)
	} else if tgClient.IsConnected() {
		fmt.Println("Telegram: Restored session - already authenticated")
		state.SetTelegramStatus("connected")
	} else {
		fmt.Println("Telegram: Connected but not authenticated")
	}

	return tgClient
}

func fatal(context string, err error) {
	fmt.Fprintf(os.Stderr, "Error %s: %v\n", context, err)
	os.Exit(1)
}

func waitForShutdown(srv *server.Server, waClient *whatsapp.Client, tgClient *telegram.Client, userServiceManager *server.UserServiceManager) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Stop all user services (processors, workers)
	if userServiceManager != nil {
		userServiceManager.StopAllServices()
	}
	if tgClient != nil {
		tgClient.Disconnect()
	}
	srv.Shutdown(ctx)
	if waClient != nil {
		waClient.Disconnect()
	}
}

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
	"github.com/omriShneor/project_alfred/internal/clients"
	"github.com/omriShneor/project_alfred/internal/config"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/server"
	"github.com/omriShneor/project_alfred/internal/sse"
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

	// Create ClientManager for per-user WhatsApp/Telegram clients
	clientManager := clients.NewClientManager(db, &clients.ManagerConfig{
		WhatsAppDBBasePath: cfg.WhatsAppDBPath,
		TelegramDBBasePath: cfg.TelegramDBPath,
		TelegramAPIID:      cfg.TelegramAPIID,
		TelegramAPIHash:    cfg.TelegramAPIHash,
		DebugAllMessages:   cfg.DebugAllMessages,
	}, notifyService, state)

	// Clean up legacy session files (one-time migration)
	clientManager.CleanupLegacySessions()

	// Create dev user if in dev mode (for unauthenticated testing)
	if cfg.DevMode {
		if err := ensureDevUser(db); err != nil {
			fmt.Printf("Warning: Failed to create dev user: %v\n", err)
		}
	}

	// Set ClientManager on server
	srv.SetClientManager(clientManager)

	eventAnalyzer := initEventAnalyzer(cfg)
	reminderAnalyzer := initReminderAnalyzer(cfg)

	// Create UserServiceManager for per-user service lifecycle
	userServiceManager := server.NewUserServiceManager(server.UserServiceManagerConfig{
		DB:               db,
		Config:           cfg,
		CredentialsFile:  cfg.GoogleCredentialsFile,
		NotifyService:    notifyService,
		EventAnalyzer:    eventAnalyzer,
		ReminderAnalyzer: reminderAnalyzer,
		ClientManager:    clientManager,
	})
	srv.SetUserServiceManager(userServiceManager)

	// Restore sessions for users who were previously connected
	if err := clientManager.RestoreUserSessions(ctx); err != nil {
		fmt.Printf("Warning: Failed to restore some user sessions: %v\n", err)
	}

	// NOTE: Workers and processors are NOT started here.
	// They will be started by UserServiceManager after user login + onboarding.

	waitForShutdown(srv, clientManager, userServiceManager)
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


// ensureDevUser creates the dev user (ID 1) if it doesn't exist
// This runs on every startup in dev mode to ensure the user exists
func ensureDevUser(db *database.DB) error {
	const (
		devUserID    = 1
		devEmail     = "omrishneor@gmail.com"
		devGoogleID  = "117916007686632359623"
		devName      = "Omri Shneor"
	)

	// Check if user 1 already exists
	var existingEmail string
	err := db.QueryRow(`SELECT email FROM users WHERE id = ?`, devUserID).Scan(&existingEmail)

	if err == nil {
		// User exists
		if existingEmail == devEmail {
			fmt.Printf("✅ Dev user already exists (ID: %d, Email: %s)\n", devUserID, devEmail)
			return nil
		}
		// User 1 exists but with different email - this is unexpected
		fmt.Printf("⚠️  Warning: User ID 1 exists with different email (%s), not creating dev user\n", existingEmail)
		return nil
	}

	// User doesn't exist - create it
	_, err = db.Exec(`
		INSERT INTO users (id, google_id, email, name, created_at, updated_at, last_login_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, devUserID, devGoogleID, devEmail, devName)

	if err != nil {
		return fmt.Errorf("failed to create dev user: %w", err)
	}

	fmt.Printf("✅ Dev user created (ID: %d, Email: %s)\n", devUserID, devEmail)
	return nil
}

func fatal(context string, err error) {
	fmt.Fprintf(os.Stderr, "Error %s: %v\n", context, err)
	os.Exit(1)
}

func waitForShutdown(srv *server.Server, clientManager *clients.ClientManager, userServiceManager *server.UserServiceManager) {
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

	// Shutdown all clients gracefully
	if clientManager != nil {
		clientManager.Shutdown(ctx)
	}

	srv.Shutdown(ctx)
}

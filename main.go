package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/omriShneor/project_alfred/internal/config"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/onboarding"
	"github.com/omriShneor/project_alfred/internal/server"
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

	handler := whatsapp.NewHandler(db, cfg.DebugAllMessages)

	waClient, err := whatsapp.NewClient(handler)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating WhatsApp client: %v\n", err)
		os.Exit(1)
	}

	if err := onboarding.RunOnboarding(db, waClient, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Onboarding failed: %v\n", err)
		os.Exit(1)
	}

	printStartupInfo(cfg)

	srv := server.New(db, waClient, cfg.HTTPPort)
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
		}
	}()

	// TODO: Start assistant goroutine (Phase 3)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "HTTP server shutdown error: %v\n", err)
	} else {
		fmt.Println("HTTP server stopped")
	}

	waClient.Disconnect()
	fmt.Println("WhatsApp disconnected")
	fmt.Println("Goodbye!")
}

func printStartupInfo(cfg *config.Config) {
	if cfg.DebugAllMessages {
		fmt.Println("Debug mode: printing ALL messages")
	}

	fmt.Printf("Admin UI available at http://localhost:%d\n", cfg.HTTPPort)
	fmt.Println("Press Ctrl+C to exit")
}

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

	if err := waClient.Connect(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to WhatsApp: %v\n", err)
		os.Exit(1)
	}

	printStartupInfo(db, cfg)

	// Start HTTP server
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

	fmt.Println("\nShutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	srv.Shutdown(ctx)
	waClient.Disconnect()
}

func printStartupInfo(db *database.DB, cfg *config.Config) {
	if cfg.DebugAllMessages {
		fmt.Println("Debug mode: printing ALL messages")
	} else {
		channels, _ := db.ListEnabledChannels()

		senderCount := 0
		groupCount := 0
		for _, ch := range channels {
			if ch.Type == "sender" {
				senderCount++
			} else if ch.Type == "group" {
				groupCount++
			}
		}

		if len(channels) > 0 {
			fmt.Printf("Listening for messages from %d senders and %d groups (%d channels total)...\n", senderCount, groupCount, len(channels))
		} else {
			fmt.Println("Warning: No tracked channels configured.")
		}
	}

	fmt.Printf("Admin UI available at http://localhost:%d\n", cfg.HTTPPort)
	fmt.Println("Press Ctrl+C to exit\n")
}

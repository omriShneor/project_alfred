package onboarding

import (
	"context"
	"fmt"
	"os"

	"github.com/omriShneor/project_alfred/internal/config"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/whatsapp"
)

// RunOnboarding orchestrates the complete onboarding process
func RunOnboarding(db *database.DB, waClient *whatsapp.Client, cfg *config.Config) error {
	fmt.Println("=== WhatsApp Connection Setup ===")

	// Step 1: Connect to WhatsApp (handles QR code if needed)
	if err := waClient.Connect(context.Background()); err != nil {
		return fmt.Errorf("failed to connect to WhatsApp: %w", err)
	}

	// Step 2: Display tracked channels
	displayTrackedChannels(db, cfg.HTTPPort)

	fmt.Println("✓ Onboarding complete. Starting application...")
	return nil
}

// displayTrackedChannels shows all enabled channels in basic format
func displayTrackedChannels(db *database.DB, httpPort int) {
	channels, err := db.ListEnabledChannels()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading channels: %v\n", err)
		return
	}

	if len(channels) == 0 {
		fmt.Println("⚠️  No tracked channels configured.")
		fmt.Printf("   Use the API at http://localhost:%d to add channels", httpPort)
		return
	}

	fmt.Println("=== Tracked Channels ===")
	for _, ch := range channels {
		// Basic format: [TYPE] Name
		typeLabel := "SENDER"
		if ch.Type == "group" {
			typeLabel = "GROUP "
		}
		fmt.Printf("  [%s] %s\n", typeLabel, ch.Name)
	}
	fmt.Printf("\nTotal: %d channel(s)\n", len(channels))
}

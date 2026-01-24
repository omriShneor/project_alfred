package onboarding

import (
	"context"
	"fmt"

	"github.com/omriShneor/project_alfred/internal/config"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/gcal"
	"github.com/omriShneor/project_alfred/internal/sse"
	"github.com/omriShneor/project_alfred/internal/whatsapp"
)

// ClientsReadyCallback is called when clients are created but before onboarding completes
type ClientsReadyCallback func(waClient *whatsapp.Client, gcalClient *gcal.Client)

// Initialize creates WhatsApp and GCal clients, runs onboarding if needed, returns Clients
func Initialize(ctx context.Context, db *database.DB, cfg *config.Config, state *sse.State, onClientsReady ClientsReadyCallback) (*Clients, error) {
	// 1. Create WhatsApp handler and client
	handler := whatsapp.NewHandler(db, cfg.DebugAllMessages)
	waClient, err := whatsapp.NewClient(handler, cfg.WhatsAppDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create WhatsApp client: %w", err)
	}

	// 2. Create Google Calendar client (uses embedded credentials)
	gcalClient, err := gcal.NewClient(cfg.GoogleCredentialsFile, cfg.GoogleTokenFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create Google Calendar client: %w", err)
	}

	// 3. Notify that clients are ready (so server can use them during onboarding)
	if onClientsReady != nil {
		onClientsReady(waClient, gcalClient)
	}

	// 4. Run onboarding if needed
	if NeedsSetup(waClient, gcalClient) {
		fmt.Printf("\n=== Setup Required ===\n")
		fmt.Printf("Visit http://localhost:%d/onboarding to complete setup\n\n", cfg.HTTPPort)

		RunWeb(ctx, state, waClient, gcalClient)

		if err := state.WaitForCompletion(ctx); err != nil {
			return nil, fmt.Errorf("onboarding interrupted: %w", err)
		}
		fmt.Println("\n=== Setup Complete ===")
	} else {
		// Already logged in - connect to WhatsApp server
		if err := waClient.WAClient.Connect(); err != nil {
			return nil, fmt.Errorf("failed to connect to WhatsApp: %w", err)
		}
		fmt.Println("WhatsApp connected!")

		state.SetWhatsAppStatus("connected")
		state.SetGCalStatus("connected")
		state.MarkComplete()
	}

	return &Clients{
		WAClient:   waClient,
		GCalClient: gcalClient,
		MsgChan:    handler.MessageChan(),
	}, nil
}

func NeedsSetup(waClient *whatsapp.Client, gcalClient *gcal.Client) bool {
	// WhatsApp must be connected
	if !waClient.IsLoggedIn() {
		return true
	}

	// Google Calendar must be connected (it's mandatory)
	if gcalClient == nil || !gcalClient.IsAuthenticated() {
		return true
	}

	return false
}

// RunWeb runs the web-based onboarding flow, updating state for SSE streaming
func RunWeb(ctx context.Context, state *sse.State, waClient *whatsapp.Client, gcalClient *gcal.Client) {
	// Initialize WhatsApp status
	if waClient.IsLoggedIn() {
		state.SetWhatsAppStatus("connected")
	} else {
		state.SetWhatsAppStatus("needs_qr")
		go runWhatsAppOnboarding(ctx, state, waClient)
	}

	// Initialize Google Calendar status
	state.SetGCalConfigured(true)
	if gcalClient.IsAuthenticated() {
		state.SetGCalStatus("connected")
	} else {
		state.SetGCalStatus("needs_auth")
	}
}

// runWhatsAppOnboarding handles the WhatsApp QR flow for web onboarding
func runWhatsAppOnboarding(ctx context.Context, state *sse.State, waClient *whatsapp.Client) {
	// Get QR channel from WhatsApp client
	qrChan, err := waClient.WAClient.GetQRChannel(ctx)
	if err != nil {
		state.SetWhatsAppError(fmt.Sprintf("Failed to get QR channel: %v", err))
		return
	}

	// Connect (this triggers QR generation)
	if err := waClient.WAClient.Connect(); err != nil {
		state.SetWhatsAppError(fmt.Sprintf("Failed to connect: %v", err))
		return
	}

	state.SetWhatsAppStatus("waiting")

	// Listen for QR events
	for evt := range qrChan {
		switch evt.Event {
		case "code":
			// Generate QR code as data URL
			dataURL, err := whatsapp.GenerateQRDataURL(evt.Code)
			if err != nil {
				state.SetWhatsAppError(fmt.Sprintf("Failed to generate QR: %v", err))
				continue
			}
			state.SetQR(dataURL)
		case "success":
			state.SetWhatsAppStatus("connected")
			fmt.Println("WhatsApp connected successfully!")
			return
		case "timeout":
			state.SetWhatsAppError("QR code expired. Please refresh the page to try again.")
			return
		}
	}
}

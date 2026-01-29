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

// ClientsReadyCallback is called when clients are created (before onboarding completes)
// This allows the server to use the clients during the onboarding flow
type ClientsReadyCallback interface {
	SetGCalClient(client *gcal.Client)
	SetWAClient(client *whatsapp.Client)
}

// Initialize creates WhatsApp and GCal clients without blocking on onboarding.
// The mobile app handles the Smart Calendar setup flow.
func Initialize(ctx context.Context, db *database.DB, cfg *config.Config, state *sse.State, clientsReady ClientsReadyCallback) (*Clients, error) {
	// 1. Create WhatsApp handler and client
	handler := whatsapp.NewHandler(db, cfg.DebugAllMessages, state)
	waClient, err := whatsapp.NewClient(handler, cfg.WhatsAppDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create WhatsApp client: %w", err)
	}

	// 2. Create Google Calendar client (uses embedded credentials)
	gcalClient, err := gcal.NewClient(cfg.GoogleCredentialsFile, cfg.GoogleTokenFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create Google Calendar client: %w", err)
	}

	// 3. Notify server that clients are ready
	if clientsReady != nil {
		clientsReady.SetGCalClient(gcalClient)
		clientsReady.SetWAClient(waClient)
	}

	// 4. Check feature settings to determine what to connect
	featureSettings, err := db.GetFeatureSettings()
	if err != nil {
		fmt.Printf("Warning: Could not load feature settings: %v\n", err)
		// Continue anyway - features will be disabled
	}

	// 5. Connect integrations based on feature settings
	if featureSettings != nil && featureSettings.SmartCalendarEnabled && featureSettings.SmartCalendarSetupComplete {
		// Smart Calendar is enabled and setup is complete - connect integrations

		// Connect WhatsApp if enabled and logged in
		if featureSettings.WhatsAppInputEnabled && waClient.IsLoggedIn() {
			if err := waClient.WAClient.Connect(); err != nil {
				fmt.Printf("Warning: Failed to connect WhatsApp: %v\n", err)
			} else {
				fmt.Println("WhatsApp connected!")
				state.SetWhatsAppStatus("connected")
			}
		}

		// Update GCal status if connected
		if featureSettings.GoogleCalendarEnabled && gcalClient != nil && gcalClient.IsAuthenticated() {
			state.SetGCalStatus("connected")
			fmt.Println("Google Calendar connected!")
		}

		state.MarkComplete()
	} else {
		// Smart Calendar not enabled or setup incomplete
		// Just update status without blocking
		if waClient.IsLoggedIn() {
			state.SetWhatsAppStatus("connected")
		} else {
			state.SetWhatsAppStatus("pending")
		}

		if gcalClient != nil && gcalClient.IsAuthenticated() {
			state.SetGCalStatus("connected")
		} else {
			state.SetGCalStatus("pending")
		}

		state.MarkComplete()
		fmt.Println("App started - configure Smart Calendar in Assistant Capabilities")
	}

	return &Clients{
		WAClient:   waClient,
		GCalClient: gcalClient,
		MsgChan:    handler.MessageChan(),
	}, nil
}

// NeedsSetup checks if Smart Calendar setup is needed based on feature settings
// Returns true if Smart Calendar is enabled but setup is not complete
func NeedsSetup(db *database.DB, waClient *whatsapp.Client, gcalClient *gcal.Client) bool {
	settings, err := db.GetFeatureSettings()
	if err != nil {
		return false // Can't determine, assume no setup needed
	}

	// If Smart Calendar is not enabled, no setup needed
	if !settings.SmartCalendarEnabled {
		return false
	}

	// If setup is already complete, no setup needed
	if settings.SmartCalendarSetupComplete {
		return false
	}

	// Smart Calendar is enabled but setup not complete
	return true
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

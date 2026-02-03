package onboarding

import (
	"context"
	"fmt"

	"github.com/omriShneor/project_alfred/internal/config"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/gcal"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/sse"
	"github.com/omriShneor/project_alfred/internal/whatsapp"
)

type ClientsReadyCallback interface {
	SetGCalClient(client *gcal.Client)
	SetWAClient(client *whatsapp.Client)
}

func Initialize(ctx context.Context, db *database.DB, cfg *config.Config, state *sse.State, clientsReady ClientsReadyCallback, notifyService *notify.Service) (*Clients, error) {
	handler := whatsapp.NewHandler(db, cfg.DebugAllMessages, state)
	waClient, err := whatsapp.NewClient(handler, cfg.WhatsAppDBPath, notifyService)
	if err != nil {
		return nil, fmt.Errorf("failed to create WhatsApp client: %w", err)
	}

	gcalClient, err := gcal.NewClient(cfg.GoogleCredentialsFile, cfg.GoogleTokenFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create Google Calendar client: %w", err)
	}

	if clientsReady != nil {
		clientsReady.SetGCalClient(gcalClient)
		clientsReady.SetWAClient(waClient)
	}

	// Always try to connect WhatsApp to restore session
	// Note: Must connect first, then check IsLoggedIn() - the device JID isn't populated until connection
	// UserID will be set later when user logs in
	if err := waClient.WAClient.Connect(); err != nil {
		fmt.Printf("Warning: Failed to connect WhatsApp: %v\n", err)
		state.SetWhatsAppStatus("pending")
	} else if waClient.IsLoggedIn() {
		// Connection succeeded AND we have an authenticated session
		fmt.Println("WhatsApp: Restored session - already logged in")
		state.SetWhatsAppStatus("connected")
	} else {
		// Connection succeeded but no authenticated session - needs pairing
		fmt.Println("WhatsApp: Connected but not authenticated - needs pairing")
		state.SetWhatsAppStatus("pending")
	}

	// Set Google Calendar status
	if gcalClient != nil && gcalClient.IsAuthenticated() {
		state.SetGCalStatus("connected")
		fmt.Println("Google Calendar: Already authenticated")
	} else {
		state.SetGCalStatus("pending")
	}

	state.MarkComplete()
	fmt.Println("App started - waiting for user login")

	return &Clients{
		WAClient:   waClient,
		GCalClient: gcalClient,
		MsgChan:    handler.MessageChan(),
	}, nil
}

// NeedsSetup is deprecated - onboarding is now per-user after login
// Keeping for backwards compatibility but always returns false
func NeedsSetup(db *database.DB, waClient *whatsapp.Client, gcalClient *gcal.Client) bool {
	return false
}

func RunWeb(ctx context.Context, state *sse.State, waClient *whatsapp.Client, gcalClient *gcal.Client) {
	if waClient.IsLoggedIn() {
		state.SetWhatsAppStatus("connected")
	} else {
		state.SetWhatsAppStatus("needs_qr")
		go runWhatsAppOnboarding(ctx, state, waClient)
	}

	state.SetGCalConfigured(true)
	if gcalClient.IsAuthenticated() {
		state.SetGCalStatus("connected")
	} else {
		state.SetGCalStatus("needs_auth")
	}
}

func runWhatsAppOnboarding(ctx context.Context, state *sse.State, waClient *whatsapp.Client) {
	qrChan, err := waClient.WAClient.GetQRChannel(ctx)
	if err != nil {
		state.SetWhatsAppError(fmt.Sprintf("Failed to get QR channel: %v", err))
		return
	}

	if err := waClient.WAClient.Connect(); err != nil {
		state.SetWhatsAppError(fmt.Sprintf("Failed to connect: %v", err))
		return
	}

	state.SetWhatsAppStatus("waiting")

	for evt := range qrChan {
		switch evt.Event {
		case "code":
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

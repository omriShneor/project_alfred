package whatsapp

import (
	"context"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/omriShneor/project_alfred/internal/sse"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type Client struct {
	WAClient *whatsmeow.Client
	handler  *Handler
}

func NewClient(handler *Handler) (*Client, error) {
	dbLog := waLog.Stdout("Database", "ERROR", true)
	clientLog := waLog.Stdout("Client", "ERROR", true)

	container, err := sqlstore.New(context.Background(), "sqlite3", "file:whatsapp.db?_foreign_keys=on", dbLog)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get device store: %w", err)
	}

	waClient := whatsmeow.NewClient(deviceStore, clientLog)

	c := &Client{
		WAClient: waClient,
		handler:  handler,
	}

	if handler != nil {
		waClient.AddEventHandler(handler.HandleEvent)
	}

	return c, nil
}

func (c *Client) Disconnect() {
	c.WAClient.Disconnect()
}

func (c *Client) IsLoggedIn() bool {
	return c.WAClient.Store.ID != nil
}

// Reconnect disconnects and reconnects to WhatsApp, generating a new QR code
func (c *Client) Reconnect(ctx context.Context, state *sse.State) {
	// Disconnect first
	c.WAClient.Disconnect()

	// Reset status
	state.SetWhatsAppStatus("waiting")
	state.SetWhatsAppError("")

	// Get QR channel
	qrChan, err := c.WAClient.GetQRChannel(ctx)
	if err != nil {
		state.SetWhatsAppError(fmt.Sprintf("Failed to get QR channel: %v", err))
		return
	}

	// Connect (this triggers QR generation)
	if err := c.WAClient.Connect(); err != nil {
		state.SetWhatsAppError(fmt.Sprintf("Failed to connect: %v", err))
		return
	}

	// Listen for QR events
	for evt := range qrChan {
		switch evt.Event {
		case "code":
			// Generate QR code as data URL
			dataURL, err := GenerateQRDataURL(evt.Code)
			if err != nil {
				state.SetWhatsAppError(fmt.Sprintf("Failed to generate QR: %v", err))
				continue
			}
			state.SetQR(dataURL)
		case "success":
			state.SetWhatsAppStatus("connected")
			fmt.Println("WhatsApp reconnected successfully!")
			return
		case "timeout":
			state.SetWhatsAppError("QR code expired. Click retry to try again.")
			return
		}
	}
}

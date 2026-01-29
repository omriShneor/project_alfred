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
	WAClient  *whatsmeow.Client
	handler   *Handler
	container *sqlstore.Container
}

func NewClient(handler *Handler, dbPath string) (*Client, error) {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	clientLog := waLog.Stdout("Client", "DEBUG", true)

	container, err := sqlstore.New(context.Background(), "sqlite3", "file:"+dbPath+"?_foreign_keys=on", dbLog)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get device store: %w", err)
	}

	waClient := whatsmeow.NewClient(deviceStore, clientLog)

	c := &Client{
		WAClient:  waClient,
		handler:   handler,
		container: container,
	}

	if handler != nil {
		waClient.AddEventHandler(handler.HandleEvent)
	}

	return c, nil
}

func (c *Client) Disconnect() {
	// Logout clears the session and deletes device store
	if c.WAClient.Store.ID != nil {
		// Ensure we're connected before trying to logout (required to notify WhatsApp servers)
		if !c.WAClient.IsConnected() {
			if err := c.WAClient.Connect(); err != nil {
				fmt.Printf("Warning: could not connect for logout: %v\n", err)
			}
		}
		// Logout() internally: 1) notifies WhatsApp servers, 2) deletes device store, 3) sets Store.ID = nil
		// Don't call Delete() manually - Logout() handles it and needs device data to notify servers
		if err := c.WAClient.Logout(context.Background()); err != nil {
			fmt.Printf("Warning: logout failed: %v\n", err)
		} else {
			fmt.Println("WhatsApp logged out successfully")
		}
	}

	c.WAClient.Disconnect()
}

func (c *Client) ReinitializeDevice() error {
	ctx := context.Background()

	// Delete all existing devices to clear any stale data
	devices, err := c.container.GetAllDevices(ctx)
	if err != nil {
		fmt.Printf("Warning: could not get existing devices: %v\n", err)
	} else {
		for _, dev := range devices {
			if err := c.container.DeleteDevice(ctx, dev); err != nil {
				fmt.Printf("Warning: failed to delete device %v: %v\n", dev.ID, err)
			}
		}
	}

	// Now GetFirstDevice will create a fresh device since we deleted all existing ones
	deviceStore, err := c.container.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get new device store: %w", err)
	}

	clientLog := waLog.Stdout("Client", "DEBUG", true)
	c.WAClient = whatsmeow.NewClient(deviceStore, clientLog)

	if c.handler != nil {
		c.WAClient.AddEventHandler(c.handler.HandleEvent)
	}

	return nil
}

func (c *Client) PairWithPhone(ctx context.Context, phone string, state *sse.State) (string, error) {
	if err := c.ReinitializeDevice(); err != nil {
		return "", fmt.Errorf("failed to reinitialize device: %w", err)
	}

	qrChan, err := c.WAClient.GetQRChannel(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get QR channel to initialize PairCode: %w", err)
	}

	// Trigger QR channel evt
	if err := c.WAClient.Connect(); err != nil {
		return "", fmt.Errorf("failed to connect: %w", err)
	}

	<-qrChan // Wait for the first qr channel event
	code, err := c.WAClient.PairPhone(
		context.Background(),
		phone,
		true,
		whatsmeow.PairClientChrome,
		"Chrome (Linux)",
	)
	if err != nil {
		return "", fmt.Errorf("PairPhone failed with error: %w", err)
	}

	go func() {
		for evt := range qrChan {
			switch evt.Event {
			case "success":
				if state != nil {
					state.SetWhatsAppStatus("connected")
				}
				fmt.Println("WhatsApp paired successfully!")
				return
			case "timeout":
				if state != nil {
					state.SetWhatsAppError("Pairing timed out")
				}
				return
			}
		}
	}()

	return code, nil
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

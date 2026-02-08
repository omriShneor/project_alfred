package whatsapp

import (
	"context"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/sse"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type Client struct {
	UserID        int64 // User who owns this client (for multi-user support)
	WAClient      *whatsmeow.Client
	handler       *Handler
	container     *sqlstore.Container
	notifyService *notify.Service
}

func NewClient(handler *Handler, dbPath string, preferredDeviceJID string, notifyService *notify.Service) (*Client, error) {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	clientLog := waLog.Stdout("Client", "DEBUG", true)

	// Set device properties so WhatsApp shows "Chrome (Linux)" instead of "Unknown"
	store.DeviceProps.Os = stringPtr("Linux")
	store.DeviceProps.PlatformType = store.DeviceProps.PlatformType.Enum()
	store.DeviceProps.RequireFullSync = boolPtr(false)

	ctx := context.Background()
	container, err := sqlstore.New(ctx, "sqlite3", "file:"+dbPath+"?_foreign_keys=on", dbLog)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	deviceStore, err := getDeviceStore(ctx, container, preferredDeviceJID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device store: %w", err)
	}

	waClient := whatsmeow.NewClient(deviceStore, clientLog)

	c := &Client{
		WAClient:      waClient,
		handler:       handler,
		container:     container,
		notifyService: notifyService,
	}

	if handler != nil {
		waClient.AddEventHandler(handler.HandleEvent)
		handler.SetClient(waClient) // Enable history sync processing
	}

	return c, nil
}

func getDeviceStore(ctx context.Context, container *sqlstore.Container, preferredDeviceJID string) (*store.Device, error) {
	if preferredDeviceJID != "" {
		jid, err := types.ParseJID(preferredDeviceJID)
		if err == nil {
			deviceStore, err := container.GetDevice(ctx, jid)
			if err != nil {
				return nil, fmt.Errorf("failed to get preferred device %s: %w", preferredDeviceJID, err)
			}
			if deviceStore != nil {
				return deviceStore, nil
			}
		}
	}

	return container.GetFirstDevice(ctx)
}

// Disconnect closes the WhatsApp connection without logging out.
// The session is preserved so the user remains authenticated on restart.
func (c *Client) Disconnect() {
	c.WAClient.Disconnect()
	fmt.Println("WhatsApp disconnected (session preserved)")
}

// Logout explicitly logs out from WhatsApp and clears the session.
// Use this only when the user wants to disconnect their WhatsApp account.
func (c *Client) Logout() error {
	if c.WAClient.Store.ID == nil {
		return fmt.Errorf("not logged in")
	}

	// Need to be connected to send logout request
	if !c.WAClient.IsConnected() {
		if err := c.WAClient.Connect(); err != nil {
			return fmt.Errorf("could not connect for logout: %w", err)
		}
	}

	if err := c.WAClient.Logout(context.Background()); err != nil {
		return fmt.Errorf("logout failed: %w", err)
	}

	fmt.Println("WhatsApp logged out successfully (session cleared)")
	return nil
}

func (c *Client) ReinitializeDevice() error {
	ctx := context.Background()

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

	deviceStore, err := c.container.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get new device store: %w", err)
	}

	clientLog := waLog.Stdout("Client", "DEBUG", true)
	c.WAClient = whatsmeow.NewClient(deviceStore, clientLog)

	if c.handler != nil {
		c.WAClient.AddEventHandler(c.handler.HandleEvent)
		c.handler.SetClient(c.WAClient) // Enable history sync processing
	}

	return nil
}

func (c *Client) PairWithPhone(ctx context.Context, phone string, state *sse.State) (string, error) {
	if err := c.ReinitializeDevice(); err != nil {
		return "", fmt.Errorf("failed to reinitialize device: %w", err)
	}

	// Use Background context so connection outlives HTTP request
	qrChan, err := c.WAClient.GetQRChannel(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get QR channel to initialize PairCode: %w", err)
	}

	if err := c.WAClient.Connect(); err != nil {
		return "", fmt.Errorf("failed to connect: %w", err)
	}

	// Wait for first event (indicates connection is ready)
	firstEvt := <-qrChan
	if firstEvt.Event != "code" {
		return "", fmt.Errorf("unexpected first event: %s", firstEvt.Event)
	}

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
				fmt.Println("WhatsApp paired successfully via QR channel!")
				if state != nil {
					state.SetWhatsAppStatus("connected")
				}
				time.Sleep(5 * time.Second) // Wait 5 seconds for the App UI to complete.
				if c.notifyService != nil {
					c.notifyService.NotifyWhatsAppConnected(context.Background(), c.UserID)
				}
				return
			case "timeout":
				fmt.Println("WhatsApp pairing timed out")
				if state != nil {
					state.SetWhatsAppError("Pairing timed out. Please try again.")
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

// SetUserID sets the user ID on both the client and its handler
func (c *Client) SetUserID(userID int64) {
	c.UserID = userID
	if c.handler != nil {
		c.handler.UserID = userID
	}
}

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

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

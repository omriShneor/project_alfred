package whatsapp

import (
	"context"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
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

func (c *Client) Connect(ctx context.Context) error {
	if c.WAClient.Store.ID == nil {
		return c.connectWithQR(ctx)
	}

	if err := c.WAClient.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	fmt.Println("Connected to WhatsApp!")
	return nil
}

func (c *Client) connectWithQR(ctx context.Context) error {
	qrChan, _ := c.WAClient.GetQRChannel(ctx)

	if err := c.WAClient.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	fmt.Println("\nScan this QR code in WhatsApp:")
	fmt.Println("WhatsApp > Settings > Linked Devices > Link a Device\n")

	for evt := range qrChan {
		switch evt.Event {
		case "code":
			DisplayQR(evt.Code)
		case "success":
			fmt.Println("\nSuccessfully logged in!")
			return nil
		case "timeout":
			return fmt.Errorf("QR code timeout - please restart and try again")
		}
	}

	return nil
}

func (c *Client) Disconnect() {
	c.WAClient.Disconnect()
}

func (c *Client) IsLoggedIn() bool {
	return c.WAClient.Store.ID != nil
}

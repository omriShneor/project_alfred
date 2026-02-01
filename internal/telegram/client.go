package telegram

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

// Client manages the Telegram connection
type Client struct {
	apiID        int
	apiHash      string
	sessionPath  string
	client       *telegram.Client
	api          *tg.Client
	handler      *Handler
	connected    bool
	phoneNumber  string
	codeHash     string // Stored during code verification flow
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	updatesChan  chan tg.UpdatesClass
}

// ClientConfig holds configuration for the Telegram client
type ClientConfig struct {
	APIID       int
	APIHash     string
	SessionPath string
	Handler     *Handler
}

// NewClient creates a new Telegram client
func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.APIID == 0 || cfg.APIHash == "" {
		return nil, fmt.Errorf("Telegram API ID and API Hash are required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := &Client{
		apiID:       cfg.APIID,
		apiHash:     cfg.APIHash,
		sessionPath: cfg.SessionPath,
		handler:     cfg.Handler,
		ctx:         ctx,
		cancel:      cancel,
		updatesChan: make(chan tg.UpdatesClass, 100),
	}

	return c, nil
}

// Connect initializes and connects the Telegram client
func (c *Client) Connect() error {
	// Check if already connected (with read lock)
	c.mu.RLock()
	if c.connected {
		c.mu.RUnlock()
		return nil
	}
	// Also check if api is already set (connection in progress or done)
	if c.api != nil {
		c.mu.RUnlock()
		return nil
	}
	c.mu.RUnlock()

	// Acquire write lock to set up the client
	c.mu.Lock()

	// Double-check after acquiring write lock
	if c.connected || c.api != nil {
		c.mu.Unlock()
		return nil
	}

	// Create storage for session persistence
	sessionStorage := &FileSessionStorage{Path: c.sessionPath}

	// Create the Telegram client
	client := telegram.NewClient(c.apiID, c.apiHash, telegram.Options{
		SessionStorage: sessionStorage,
		UpdateHandler:  c,
	})

	c.client = client
	c.mu.Unlock() // Release lock before starting goroutine

	// Start the client in a goroutine
	go func() {
		if err := client.Run(c.ctx, func(ctx context.Context) error {
			// Get the API client
			c.mu.Lock()
			c.api = client.API()
			c.mu.Unlock()

			// Check if already authorized
			status, err := client.Auth().Status(ctx)
			if err != nil {
				return fmt.Errorf("failed to get auth status: %w", err)
			}

			c.mu.Lock()
			c.connected = status.Authorized
			c.mu.Unlock()

			if status.Authorized {
				fmt.Println("Telegram: Already authorized")
			} else {
				fmt.Println("Telegram: Not authorized, waiting for authentication")
			}

			// Block until context is cancelled
			<-ctx.Done()
			return ctx.Err()
		}); err != nil && err != context.Canceled {
			fmt.Printf("Telegram client error: %v\n", err)
		}
	}()

	// Wait for client to initialize with timeout
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for Telegram client to connect")
		case <-ticker.C:
			c.mu.RLock()
			apiReady := c.api != nil
			c.mu.RUnlock()
			if apiReady {
				fmt.Println("Telegram: Client connected and ready")
				return nil
			}
		}
	}
}

// Disconnect closes the Telegram connection
func (c *Client) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancel != nil {
		c.cancel()
	}
	c.connected = false
}

// IsConnected returns whether the client is connected and authenticated
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// SendCode requests a verification code for the given phone number
func (c *Client) SendCode(ctx context.Context, phoneNumber string) error {
	// Check if we need to connect first (without holding lock)
	c.mu.RLock()
	needsConnect := c.api == nil
	c.mu.RUnlock()

	if needsConnect {
		fmt.Println("Telegram: Auto-connecting before sending code...")
		if err := c.Connect(); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.api == nil {
		return fmt.Errorf("client not connected - connection may have failed")
	}

	// Send code request
	sentCode, err := c.api.AuthSendCode(ctx, &tg.AuthSendCodeRequest{
		PhoneNumber: phoneNumber,
		APIID:       c.apiID,
		APIHash:     c.apiHash,
		Settings:    tg.CodeSettings{},
	})
	if err != nil {
		return fmt.Errorf("failed to send code: %w", err)
	}

	// Store phone number and code hash for verification
	c.phoneNumber = phoneNumber
	switch v := sentCode.(type) {
	case *tg.AuthSentCode:
		c.codeHash = v.PhoneCodeHash
	default:
		return fmt.Errorf("unexpected sent code type: %T", sentCode)
	}

	fmt.Printf("Telegram: Verification code sent to %s\n", phoneNumber)
	return nil
}

// VerifyCode verifies the code and completes authentication
func (c *Client) VerifyCode(ctx context.Context, code string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.api == nil {
		return fmt.Errorf("client not connected")
	}

	if c.phoneNumber == "" || c.codeHash == "" {
		return fmt.Errorf("no pending code verification - call SendCode first")
	}

	// Sign in with the code
	authResult, err := c.api.AuthSignIn(ctx, &tg.AuthSignInRequest{
		PhoneNumber:   c.phoneNumber,
		PhoneCodeHash: c.codeHash,
		PhoneCode:     code,
	})
	if err != nil {
		// Check if 2FA is required
		if auth.IsKeyUnregistered(err) {
			return fmt.Errorf("phone number not registered on Telegram")
		}
		return fmt.Errorf("failed to sign in: %w", err)
	}

	switch v := authResult.(type) {
	case *tg.AuthAuthorization:
		c.connected = true
		fmt.Printf("Telegram: Successfully authenticated as %v\n", v.User)
	case *tg.AuthAuthorizationSignUpRequired:
		return fmt.Errorf("account registration required - please sign up on Telegram first")
	default:
		return fmt.Errorf("unexpected auth result: %T", authResult)
	}

	// Clear the pending code
	c.phoneNumber = ""
	c.codeHash = ""

	return nil
}

// Handle implements telegram.UpdateHandler
func (c *Client) Handle(ctx context.Context, u tg.UpdatesClass) error {
	if c.handler == nil {
		return nil
	}

	// Forward updates to handler
	select {
	case c.updatesChan <- u:
	default:
		fmt.Println("Telegram: Updates channel full, dropping update")
	}

	return nil
}

// StartUpdateLoop starts processing updates
func (c *Client) StartUpdateLoop() {
	go func() {
		for {
			select {
			case <-c.ctx.Done():
				return
			case update := <-c.updatesChan:
				c.handler.HandleUpdate(update)
			}
		}
	}()
}

// GetAPI returns the raw Telegram API client
func (c *Client) GetAPI() *tg.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.api
}

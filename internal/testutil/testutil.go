package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/server"
	"github.com/omriShneor/project_alfred/internal/sse"
	"github.com/stretchr/testify/require"
)

// TestServer wraps a server for E2E testing
type TestServer struct {
	Server     *server.Server
	DB         *database.DB
	HTTPServer *httptest.Server
	State      *sse.State
	t          *testing.T

	// Mock clients
	GCalMock     *MockGCalClient
	GmailMock    *MockGmailClient
	WhatsAppMock *MockWhatsAppClient
	TelegramMock *MockTelegramClient
}

// TestServerOption configures a test server
type TestServerOption func(*TestServer)

// NewTestServer creates a fully configured test server for E2E testing
func NewTestServer(t *testing.T, opts ...TestServerOption) *TestServer {
	t.Helper()

	// Create in-memory database
	db, err := database.New(":memory:")
	require.NoError(t, err, "failed to create test database")

	state := sse.NewState()

	ts := &TestServer{
		DB:    db,
		State: state,
		t:     t,
	}

	// Apply options before creating server
	for _, opt := range opts {
		opt(ts)
	}

	// Create server config
	cfg := server.ServerConfig{
		DB:              db,
		OnboardingState: state,
		Port:            0, // Will use httptest server
	}

	ts.Server = server.New(cfg)

	// Create HTTP test server using the server's handler
	ts.HTTPServer = httptest.NewServer(ts.Server.Handler())

	t.Cleanup(func() {
		ts.HTTPServer.Close()
		db.Close()
	})

	return ts
}

// BaseURL returns the test server base URL
func (ts *TestServer) BaseURL() string {
	return ts.HTTPServer.URL
}

// Client returns an HTTP client configured for the test server
func (ts *TestServer) Client() *http.Client {
	return ts.HTTPServer.Client()
}

// WithMockGCal enables Google Calendar mocking
func WithMockGCal() TestServerOption {
	return func(ts *TestServer) {
		ts.GCalMock = NewMockGCalClient()
	}
}

// WithMockGmail enables Gmail mocking
func WithMockGmail() TestServerOption {
	return func(ts *TestServer) {
		ts.GmailMock = NewMockGmailClient()
	}
}

// WithMockWhatsApp enables WhatsApp mocking
func WithMockWhatsApp() TestServerOption {
	return func(ts *TestServer) {
		ts.WhatsAppMock = NewMockWhatsAppClient()
	}
}

// WithMockTelegram enables Telegram mocking
func WithMockTelegram() TestServerOption {
	return func(ts *TestServer) {
		ts.TelegramMock = NewMockTelegramClient()
	}
}

// WithAllMocks enables all external service mocks
func WithAllMocks() TestServerOption {
	return func(ts *TestServer) {
		ts.GCalMock = NewMockGCalClient()
		ts.GmailMock = NewMockGmailClient()
		ts.WhatsAppMock = NewMockWhatsAppClient()
		ts.TelegramMock = NewMockTelegramClient()
	}
}

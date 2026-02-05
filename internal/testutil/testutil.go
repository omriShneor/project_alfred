package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/omriShneor/project_alfred/internal/auth"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/notify"
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
	TestUser   *database.TestUser
	t          *testing.T

	// Mock clients
	GCalMock      *MockGCalClient
	GmailMock     *MockGmailClient
	ClientManager *MockClientManager

	// Deprecated: Use ClientManager.GetWhatsAppClient(userID) instead
	WhatsAppMock *MockWhatsAppClient
	// Deprecated: Use ClientManager.GetTelegramClient(userID) instead
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

	// Create a test user for E2E tests
	testUser := database.CreateTestUser(t, db)

	state := sse.NewState()

	ts := &TestServer{
		DB:       db,
		State:    state,
		TestUser: testUser,
		t:        t,
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

	// Ensure notification service exists so push availability is true in tests.
	// Expo push does not require credentials, so we can wire it unconditionally.
	notifyService := notify.NewService(db, nil, notify.NewExpoPushNotifier())
	ts.Server.InitializeClients(server.ClientsConfig{
		NotifyService: notifyService,
	})

	// Always create MockClientManager for multi-user support
	// Tests can use GetWhatsAppClient(userID) or GetTelegramClient(userID) on the manager
	ts.ClientManager = NewMockClientManager(100)

	// Note: Server.SetClientManager expects *clients.ClientManager
	// Tests that need WhatsApp/Telegram should use ts.ClientManager directly
	// or call GetWhatsAppClient/GetTelegramClient to auto-create per-user clients

	// Create HTTP test server using the server's handler wrapped with test auth
	// This injects the test user into the request context for all requests
	testAuthMiddleware := createTestAuthMiddleware(testUser)
	ts.HTTPServer = httptest.NewServer(testAuthMiddleware(ts.Server.Handler()))

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

// GetWhatsAppClient returns or creates a WhatsApp mock client for the given user
func (ts *TestServer) GetWhatsAppClient(userID int64) (*MockWhatsAppClient, error) {
	if ts.ClientManager == nil {
		ts.ClientManager = NewMockClientManager(100)
	}
	return ts.ClientManager.GetWhatsAppClient(userID)
}

// GetTelegramClient returns or creates a Telegram mock client for the given user
func (ts *TestServer) GetTelegramClient(userID int64) (*MockTelegramClient, error) {
	if ts.ClientManager == nil {
		ts.ClientManager = NewMockClientManager(100)
	}
	return ts.ClientManager.GetTelegramClient(userID)
}

// GetTestUserWhatsAppClient is a convenience method to get WhatsApp client for the test user
func (ts *TestServer) GetTestUserWhatsAppClient() (*MockWhatsAppClient, error) {
	return ts.GetWhatsAppClient(ts.TestUser.ID)
}

// GetTestUserTelegramClient is a convenience method to get Telegram client for the test user
func (ts *TestServer) GetTestUserTelegramClient() (*MockTelegramClient, error) {
	return ts.GetTelegramClient(ts.TestUser.ID)
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

// WithMockWhatsApp enables WhatsApp mocking (deprecated - use ClientManager)
// Creates a WhatsApp client for the test user automatically
func WithMockWhatsApp() TestServerOption {
	return func(ts *TestServer) {
		// Create client for test user via ClientManager
		if ts.ClientManager != nil && ts.TestUser != nil {
			client, _ := ts.ClientManager.GetWhatsAppClient(ts.TestUser.ID)
			ts.WhatsAppMock = client
		}
	}
}

// WithMockTelegram enables Telegram mocking (deprecated - use ClientManager)
// Creates a Telegram client for the test user automatically
func WithMockTelegram() TestServerOption {
	return func(ts *TestServer) {
		// Create client for test user via ClientManager
		if ts.ClientManager != nil && ts.TestUser != nil {
			client, _ := ts.ClientManager.GetTelegramClient(ts.TestUser.ID)
			ts.TelegramMock = client
		}
	}
}

// WithAllMocks enables all external service mocks
func WithAllMocks() TestServerOption {
	return func(ts *TestServer) {
		ts.GCalMock = NewMockGCalClient()
		ts.GmailMock = NewMockGmailClient()
		// WhatsApp and Telegram clients are created via ClientManager on demand
	}
}

// createTestAuthMiddleware creates middleware that injects the test user into requests
func createTestAuthMiddleware(testUser *database.TestUser) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Inject test user into context
			user := &auth.User{
				ID:    testUser.ID,
				Email: testUser.Email,
				Name:  testUser.Name,
			}
			ctx := auth.SetUserInContext(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// NewTestServerWithUser creates a test server with a specific user email.
// This shares the same database as other test servers created in the same test,
// allowing multi-user testing with different authenticated contexts.
func NewTestServerWithUser(t *testing.T, email string) *TestServer {
	t.Helper()

	// Create in-memory database
	db, err := database.New(":memory:")
	require.NoError(t, err, "failed to create test database")

	// Create a test user with specific email
	testUser := database.CreateTestUserWithEmail(t, db, email)

	state := sse.NewState()

	ts := &TestServer{
		DB:       db,
		State:    state,
		TestUser: testUser,
		t:        t,
	}

	// Create server config
	cfg := server.ServerConfig{
		DB:              db,
		OnboardingState: state,
		Port:            0,
	}

	ts.Server = server.New(cfg)

	// Ensure notification service exists so push availability is true in tests.
	// Expo push does not require credentials, so we can wire it unconditionally.
	notifyService := notify.NewService(db, nil, notify.NewExpoPushNotifier())
	ts.Server.InitializeClients(server.ClientsConfig{
		NotifyService: notifyService,
	})

	// Always create MockClientManager for multi-user support
	ts.ClientManager = NewMockClientManager(100)

	// Create HTTP test server with auth middleware for this user
	testAuthMiddleware := createTestAuthMiddleware(testUser)
	ts.HTTPServer = httptest.NewServer(testAuthMiddleware(ts.Server.Handler()))

	t.Cleanup(func() {
		ts.HTTPServer.Close()
		db.Close()
	})

	return ts
}

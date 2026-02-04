package testutil

import (
	"fmt"
	"sync"
	"time"

	"github.com/omriShneor/project_alfred/internal/source"
)

// MockGCalClient simulates Google Calendar API for testing
type MockGCalClient struct {
	mu            sync.Mutex
	authenticated bool
	events        []MockCalendarEvent
	calendars     []MockCalendar
}

// MockCalendarEvent represents a calendar event in the mock
type MockCalendarEvent struct {
	ID          string
	CalendarID  string
	Title       string
	Description string
	StartTime   time.Time
	EndTime     time.Time
	Location    string
}

// MockCalendar represents a calendar in the mock
type MockCalendar struct {
	ID       string
	Summary  string
	TimeZone string
}

// NewMockGCalClient creates a new mock Google Calendar client
func NewMockGCalClient() *MockGCalClient {
	return &MockGCalClient{
		authenticated: true,
		calendars: []MockCalendar{
			{ID: "primary", Summary: "Primary Calendar", TimeZone: "America/Los_Angeles"},
		},
	}
}

// IsAuthenticated returns whether the mock client is authenticated
func (m *MockGCalClient) IsAuthenticated() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.authenticated
}

// SetAuthenticated sets the authentication state
func (m *MockGCalClient) SetAuthenticated(auth bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.authenticated = auth
}

// AddEvent adds an event to the mock
func (m *MockGCalClient) AddEvent(event MockCalendarEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

// GetEvents returns all events in the mock
func (m *MockGCalClient) GetEvents() []MockCalendarEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]MockCalendarEvent{}, m.events...)
}

// ClearEvents clears all events in the mock
func (m *MockGCalClient) ClearEvents() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = nil
}

// MockGmailClient simulates Gmail API for testing
type MockGmailClient struct {
	mu            sync.Mutex
	authenticated bool
	emails        []MockEmail
}

// MockEmail represents an email in the mock
type MockEmail struct {
	ID        string
	ThreadID  string
	From      string
	To        string
	Subject   string
	Body      string
	Timestamp time.Time
}

// NewMockGmailClient creates a new mock Gmail client
func NewMockGmailClient() *MockGmailClient {
	return &MockGmailClient{
		authenticated: true,
	}
}

// IsAuthenticated returns whether the mock client is authenticated
func (m *MockGmailClient) IsAuthenticated() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.authenticated
}

// SetAuthenticated sets the authentication state
func (m *MockGmailClient) SetAuthenticated(auth bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.authenticated = auth
}

// AddEmail adds an email to the mock
func (m *MockGmailClient) AddEmail(email MockEmail) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.emails = append(m.emails, email)
}

// GetEmails returns all emails in the mock
func (m *MockGmailClient) GetEmails() []MockEmail {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]MockEmail{}, m.emails...)
}

// ClearEmails clears all emails in the mock
func (m *MockGmailClient) ClearEmails() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.emails = nil
}

// MockWhatsAppClient simulates WhatsApp connection for testing
type MockWhatsAppClient struct {
	mu        sync.Mutex
	connected bool
	messages  []MockMessage
}

// MockMessage represents a message in the mock
type MockMessage struct {
	ID         string
	SenderJID  string
	SenderName string
	Text       string
	IsGroup    bool
	GroupJID   string
	Timestamp  time.Time
}

// NewMockWhatsAppClient creates a new mock WhatsApp client
func NewMockWhatsAppClient() *MockWhatsAppClient {
	return &MockWhatsAppClient{
		connected: true,
	}
}

// IsConnected returns whether the mock client is connected
func (m *MockWhatsAppClient) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

// SetConnected sets the connection state
func (m *MockWhatsAppClient) SetConnected(connected bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = connected
}

// AddMessage adds a message to the mock
func (m *MockWhatsAppClient) AddMessage(msg MockMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
}

// GetMessages returns all messages in the mock
func (m *MockWhatsAppClient) GetMessages() []MockMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]MockMessage{}, m.messages...)
}

// ClearMessages clears all messages in the mock
func (m *MockWhatsAppClient) ClearMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = nil
}

// MockTelegramClient simulates Telegram connection for testing
type MockTelegramClient struct {
	mu        sync.Mutex
	connected bool
	messages  []MockTelegramMessage
}

// MockTelegramMessage represents a Telegram message in the mock
type MockTelegramMessage struct {
	ID         int64
	ChatID     int64
	SenderID   int64
	SenderName string
	Text       string
	Timestamp  time.Time
}

// NewMockTelegramClient creates a new mock Telegram client
func NewMockTelegramClient() *MockTelegramClient {
	return &MockTelegramClient{
		connected: true,
	}
}

// IsConnected returns whether the mock client is connected
func (m *MockTelegramClient) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

// SetConnected sets the connection state
func (m *MockTelegramClient) SetConnected(connected bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = connected
}

// AddMessage adds a message to the mock
func (m *MockTelegramClient) AddMessage(msg MockTelegramMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
}

// GetMessages returns all messages in the mock
func (m *MockTelegramClient) GetMessages() []MockTelegramMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]MockTelegramMessage{}, m.messages...)
}

// ClearMessages clears all messages in the mock
func (m *MockTelegramClient) ClearMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = nil
}

// MockClientManager simulates the ClientManager for multi-user testing
type MockClientManager struct {
	mu              sync.RWMutex
	whatsappClients map[int64]*MockWhatsAppClient
	telegramClients map[int64]*MockTelegramClient
	msgChan         chan source.Message
	bufferSize      int
}

// NewMockClientManager creates a new mock client manager
func NewMockClientManager(bufferSize int) *MockClientManager {
	if bufferSize == 0 {
		bufferSize = 100 // Default buffer size
	}
	return &MockClientManager{
		whatsappClients: make(map[int64]*MockWhatsAppClient),
		telegramClients: make(map[int64]*MockTelegramClient),
		msgChan:         make(chan source.Message, bufferSize),
		bufferSize:      bufferSize,
	}
}

// GetWhatsAppClient returns or creates a WhatsApp client for the given user
func (m *MockClientManager) GetWhatsAppClient(userID int64) (*MockWhatsAppClient, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if client, exists := m.whatsappClients[userID]; exists {
		return client, nil
	}

	// Create new client for this user
	client := NewMockWhatsAppClient()
	m.whatsappClients[userID] = client
	return client, nil
}

// GetTelegramClient returns or creates a Telegram client for the given user
func (m *MockClientManager) GetTelegramClient(userID int64) (*MockTelegramClient, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if client, exists := m.telegramClients[userID]; exists {
		return client, nil
	}

	// Create new client for this user
	client := NewMockTelegramClient()
	m.telegramClients[userID] = client
	return client, nil
}

// LogoutWhatsApp disconnects and removes the WhatsApp client for a user
func (m *MockClientManager) LogoutWhatsApp(userID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if client, exists := m.whatsappClients[userID]; exists {
		client.SetConnected(false)
		delete(m.whatsappClients, userID)
	}
	return nil
}

// LogoutTelegram disconnects and removes the Telegram client for a user
func (m *MockClientManager) LogoutTelegram(userID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if client, exists := m.telegramClients[userID]; exists {
		client.SetConnected(false)
		delete(m.telegramClients, userID)
	}
	return nil
}

// CleanupUser removes all clients for a user (called on logout)
func (m *MockClientManager) CleanupUser(userID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Disconnect and remove WhatsApp client
	if client, exists := m.whatsappClients[userID]; exists {
		client.SetConnected(false)
		delete(m.whatsappClients, userID)
	}

	// Disconnect and remove Telegram client
	if client, exists := m.telegramClients[userID]; exists {
		client.SetConnected(false)
		delete(m.telegramClients, userID)
	}

	return nil
}

// MessageChan returns the shared message channel
func (m *MockClientManager) MessageChan() <-chan source.Message {
	return m.msgChan
}

// SendMessage simulates sending a message to the channel (for testing)
func (m *MockClientManager) SendMessage(msg source.Message) {
	m.msgChan <- msg
}

// RestoreUserSessions simulates restoring user sessions on startup (no-op in mock)
func (m *MockClientManager) RestoreUserSessions() error {
	// No-op for mock - sessions are created on-demand in tests
	return nil
}

// Shutdown closes the message channel
func (m *MockClientManager) Shutdown() {
	close(m.msgChan)
}

// GetWhatsAppClientCount returns the number of active WhatsApp clients (for testing)
func (m *MockClientManager) GetWhatsAppClientCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.whatsappClients)
}

// GetTelegramClientCount returns the number of active Telegram clients (for testing)
func (m *MockClientManager) GetTelegramClientCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.telegramClients)
}

// SetWhatsAppConnected sets the connection state for a user's WhatsApp client
func (m *MockClientManager) SetWhatsAppConnected(userID int64, connected bool) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.whatsappClients[userID]
	if !exists {
		return fmt.Errorf("WhatsApp client not found for user %d", userID)
	}
	client.SetConnected(connected)
	return nil
}

// SetTelegramConnected sets the connection state for a user's Telegram client
func (m *MockClientManager) SetTelegramConnected(userID int64, connected bool) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.telegramClients[userID]
	if !exists {
		return fmt.Errorf("Telegram client not found for user %d", userID)
	}
	client.SetConnected(connected)
	return nil
}

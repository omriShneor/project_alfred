package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/clients"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/sse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

type stubEventAnalyzer struct{}

func (s stubEventAnalyzer) AnalyzeMessages(ctx context.Context, history []database.MessageRecord, newMessage database.MessageRecord, existingEvents []database.CalendarEvent) (*agent.EventAnalysis, error) {
	return &agent.EventAnalysis{HasEvent: false, Action: "none"}, nil
}

func (s stubEventAnalyzer) AnalyzeEmail(ctx context.Context, email agent.EmailContent) (*agent.EventAnalysis, error) {
	return &agent.EventAnalysis{HasEvent: false, Action: "none"}, nil
}

func (s stubEventAnalyzer) IsConfigured() bool {
	return true
}

func TestStartGlobalProcessorStartsOnce(t *testing.T) {
	db := database.NewTestDB(t)
	state := sse.NewState()

	tmpDir := t.TempDir()
	cm := clients.NewClientManager(db, &clients.ManagerConfig{
		WhatsAppDBBasePath: filepath.Join(tmpDir, "whatsapp.db"),
		TelegramDBBasePath: filepath.Join(tmpDir, "telegram.db"),
	}, nil, state)

	manager := NewUserServiceManager(UserServiceManagerConfig{
		DB:            db,
		EventAnalyzer: stubEventAnalyzer{},
		ClientManager: cm,
	})

	require.NoError(t, manager.StartGlobalProcessor())
	require.True(t, manager.GlobalProcessorRunning())

	require.NoError(t, manager.StartGlobalProcessor())
	require.True(t, manager.GlobalProcessorRunning())

	manager.StopGlobalProcessor()
}

func TestStartServicesForEligibleUsersWhatsAppSession(t *testing.T) {
	db := database.NewTestDB(t)
	user := database.CreateTestUser(t, db)

	err := db.SaveWhatsAppSession(user.ID, "+1234567890", "device@wa", true)
	require.NoError(t, err)

	manager := NewUserServiceManager(UserServiceManagerConfig{
		DB: db,
	})

	manager.StartServicesForEligibleUsers()

	assert.True(t, manager.IsRunningForUser(user.ID))
}

func TestStartServicesForEligibleUsersGmailScope(t *testing.T) {
	os.Setenv("ALFRED_ENCRYPTION_KEY", "test-key-user-services")
	t.Cleanup(func() {
		os.Unsetenv("ALFRED_ENCRYPTION_KEY")
	})

	db := database.NewTestDB(t)
	user := database.CreateTestUser(t, db)

	token := &oauth2.Token{
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(1 * time.Hour),
	}
	err := db.SaveGoogleToken(user.ID, token, user.Email, []string{"https://www.googleapis.com/auth/gmail.readonly"})
	require.NoError(t, err)

	manager := NewUserServiceManager(UserServiceManagerConfig{
		DB: db,
	})

	manager.StartServicesForEligibleUsers()

	assert.True(t, manager.IsRunningForUser(user.ID))

	settings, err := db.GetGmailSettings(user.ID)
	require.NoError(t, err)
	assert.True(t, settings.Enabled)
}

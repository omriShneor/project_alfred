package clients

import (
	"path/filepath"
	"testing"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/sse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogoutWhatsAppUpdatesSessionState(t *testing.T) {
	db := database.NewTestDB(t)
	user := database.CreateTestUser(t, db)

	err := db.SaveWhatsAppSession(user.ID, "+1234567890", "device@wa", true)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	manager := NewClientManager(db, &ManagerConfig{
		WhatsAppDBBasePath: filepath.Join(tmpDir, "whatsapp.db"),
		TelegramDBBasePath: filepath.Join(tmpDir, "telegram.db"),
	}, nil, sse.NewState())

	require.NoError(t, manager.LogoutWhatsApp(user.ID))

	session, err := db.GetWhatsAppSession(user.ID)
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.False(t, session.Connected)
}

func TestLogoutTelegramUpdatesSessionState(t *testing.T) {
	db := database.NewTestDB(t)
	user := database.CreateTestUser(t, db)

	err := db.SaveTelegramSession(user.ID, "+1234567890", true)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	manager := NewClientManager(db, &ManagerConfig{
		WhatsAppDBBasePath: filepath.Join(tmpDir, "whatsapp.db"),
		TelegramDBBasePath: filepath.Join(tmpDir, "telegram.db"),
	}, nil, sse.NewState())

	require.NoError(t, manager.LogoutTelegram(user.ID))

	session, err := db.GetTelegramSession(user.ID)
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.False(t, session.Connected)
}

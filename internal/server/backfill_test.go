package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type countingBackfillEventAnalyzer struct {
	calls atomic.Int64
}

func (a *countingBackfillEventAnalyzer) AnalyzeMessages(
	ctx context.Context,
	history []database.MessageRecord,
	newMessage database.MessageRecord,
	existingEvents []database.CalendarEvent,
) (*agent.EventAnalysis, error) {
	a.calls.Add(1)
	return &agent.EventAnalysis{
		HasEvent: false,
		Action:   "none",
	}, nil
}

func (a *countingBackfillEventAnalyzer) AnalyzeEmail(ctx context.Context, email agent.EmailContent) (*agent.EventAnalysis, error) {
	return &agent.EventAnalysis{
		HasEvent: false,
		Action:   "none",
	}, nil
}

func (a *countingBackfillEventAnalyzer) IsConfigured() bool {
	return true
}

func getChannelInitialBackfillStatus(t *testing.T, db *database.DB, channelID int64) (sql.NullString, sql.NullTime) {
	t.Helper()

	var status sql.NullString
	var at sql.NullTime
	err := db.QueryRow(`
		SELECT initial_backfill_status, initial_backfill_at
		FROM channels
		WHERE id = ?
	`, channelID).Scan(&status, &at)
	require.NoError(t, err)

	return status, at
}

func waitForChannelBackfillStatus(t *testing.T, db *database.DB, channelID int64, expected database.BackfillStatus) {
	t.Helper()

	require.Eventually(t, func() bool {
		status, _ := getChannelInitialBackfillStatus(t, db, channelID)
		return status.Valid && status.String == string(expected)
	}, 2*time.Second, 20*time.Millisecond)
}

func TestStartChannelBackfill_SkippedWithoutAnalyzers(t *testing.T) {
	s := createTestServer(t)
	s.db.SetMaxOpenConns(1)
	user := database.CreateTestUser(t, s.db)

	channel, err := s.db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"skip-backfill@s.whatsapp.net",
		"Skip Backfill",
	)
	require.NoError(t, err)

	s.startChannelBackfill(user.ID, channel)

	status, at := getChannelInitialBackfillStatus(t, s.db, channel.ID)
	require.True(t, status.Valid)
	assert.Equal(t, string(database.BackfillStatusSkipped), status.String)
	assert.True(t, at.Valid, "expected initial_backfill_at to be set for terminal status")
}

func TestStartChannelBackfill_CompletesAndAnalyzesMessages(t *testing.T) {
	s := createTestServer(t)
	s.db.SetMaxOpenConns(1)
	analyzer := &countingBackfillEventAnalyzer{}
	s.eventAnalyzer = analyzer

	user := database.CreateTestUser(t, s.db)
	channel, err := s.db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"complete-backfill@s.whatsapp.net",
		"Complete Backfill",
	)
	require.NoError(t, err)

	_, err = s.db.StoreSourceMessage(
		source.SourceTypeWhatsApp,
		channel.ID,
		"sender@s.whatsapp.net",
		"Sender",
		"hello from history",
		"",
		time.Now().Add(-2*time.Hour),
	)
	require.NoError(t, err)

	s.startChannelBackfill(user.ID, channel)
	waitForChannelBackfillStatus(t, s.db, channel.ID, database.BackfillStatusCompleted)

	assert.Equal(t, int64(1), analyzer.calls.Load())
}

func TestHandleCreateWhatsappChannel_DisableAndReEnableTriggersBackfillWithPreservedHistory(t *testing.T) {
	s := createTestServer(t)
	s.db.SetMaxOpenConns(1)
	analyzer := &countingBackfillEventAnalyzer{}
	s.eventAnalyzer = analyzer

	user := database.CreateTestUser(t, s.db)
	originalChannel, err := s.db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"readd@s.whatsapp.net",
		"Readd Contact",
	)
	require.NoError(t, err)

	_, err = s.db.StoreSourceMessage(
		source.SourceTypeWhatsApp,
		originalChannel.ID,
		"readd@s.whatsapp.net",
		"Readd Contact",
		"historical message",
		"",
		time.Now().Add(-1*time.Hour),
	)
	require.NoError(t, err)

	beforeDeleteCount, err := s.db.CountSourceMessages(user.ID, source.SourceTypeWhatsApp, originalChannel.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, beforeDeleteCount)

	deleteReq := httptest.NewRequest("DELETE", "/api/whatsapp/channel/"+strconv.FormatInt(originalChannel.ID, 10), nil)
	deleteReq = withAuthContext(deleteReq, user)
	deleteReq.SetPathValue("id", strconv.FormatInt(originalChannel.ID, 10))
	deleteW := httptest.NewRecorder()
	s.handleDeleteWhatsappChannel(deleteW, deleteReq)
	require.Equal(t, 200, deleteW.Code)

	disabled, err := s.db.GetSourceChannelByID(user.ID, originalChannel.ID)
	require.NoError(t, err)
	require.NotNil(t, disabled)
	assert.False(t, disabled.Enabled)

	createBody, err := json.Marshal(map[string]string{
		"type":       "sender",
		"identifier": originalChannel.Identifier,
		"name":       "Readd Contact",
	})
	require.NoError(t, err)

	createReq := httptest.NewRequest("POST", "/api/whatsapp/channel", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq = withAuthContext(createReq, user)
	createW := httptest.NewRecorder()
	s.handleCreateWhatsappChannel(createW, createReq)
	require.Equal(t, 200, createW.Code)

	var recreated database.SourceChannel
	err = json.Unmarshal(createW.Body.Bytes(), &recreated)
	require.NoError(t, err)
	assert.Equal(t, originalChannel.ID, recreated.ID)
	assert.True(t, recreated.Enabled)

	waitForChannelBackfillStatus(t, s.db, recreated.ID, database.BackfillStatusCompleted)

	recreatedCount, err := s.db.CountSourceMessages(user.ID, source.SourceTypeWhatsApp, recreated.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, recreatedCount)
	assert.Equal(t, int64(1), analyzer.calls.Load(), "re-enabled channel should backfill preserved history")
}

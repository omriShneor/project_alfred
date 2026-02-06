package server

import (
	"context"
	"fmt"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/gmail"
	"github.com/omriShneor/project_alfred/internal/processor"
)

const (
	backfillWindowDays = 10
	backfillMaxEmails  = 200
)

func (s *Server) startChannelBackfill(userID int64, channel *database.SourceChannel) {
	if s == nil || s.db == nil || channel == nil {
		return
	}

	if s.eventAnalyzer == nil && s.reminderAnalyzer == nil {
		_ = s.db.UpdateChannelInitialBackfillStatus(userID, channel.ID, database.BackfillStatusSkipped)
		return
	}

	go func() {
		if err := s.db.UpdateChannelInitialBackfillStatus(userID, channel.ID, database.BackfillStatusInProgress); err != nil {
			fmt.Printf("Backfill: failed to mark channel in progress: %v\n", err)
		}

		since := time.Now().Add(-backfillWindowDays * 24 * time.Hour)
		messages, err := s.db.GetSourceMessagesSince(userID, channel.SourceType, channel.ID, since)
		if err != nil {
			fmt.Printf("Backfill: failed to load message history for channel %d: %v\n", channel.ID, err)
			_ = s.db.UpdateChannelInitialBackfillStatus(userID, channel.ID, database.BackfillStatusFailed)
			return
		}

		backfillProc := processor.NewBackfillProcessor(s.db, s.eventAnalyzer, s.reminderAnalyzer, s.notifyService)
		if err := backfillProc.ProcessChannelMessages(context.Background(), userID, channel.ID, channel.SourceType, messages); err != nil {
			fmt.Printf("Backfill: failed to process history for channel %d: %v\n", channel.ID, err)
			_ = s.db.UpdateChannelInitialBackfillStatus(userID, channel.ID, database.BackfillStatusFailed)
			return
		}

		_ = s.db.UpdateChannelInitialBackfillStatus(userID, channel.ID, database.BackfillStatusCompleted)
	}()
}

func (s *Server) startEmailSourceBackfill(userID int64, source *database.EmailSource) {
	if s == nil || s.db == nil || source == nil {
		return
	}

	if s.userServiceManager == nil {
		_ = s.db.UpdateEmailSourceInitialBackfillStatus(userID, source.ID, database.BackfillStatusSkipped)
		return
	}

	go func() {
		if err := s.db.UpdateEmailSourceInitialBackfillStatus(userID, source.ID, database.BackfillStatusInProgress); err != nil {
			fmt.Printf("Backfill: failed to mark email source in progress: %v\n", err)
		}

		if err := s.userServiceManager.StartServicesForUser(userID); err != nil {
			fmt.Printf("Backfill: failed to start services for user %d: %v\n", userID, err)
		}

		worker := s.userServiceManager.GetGmailWorkerForUser(userID)
		if worker == nil {
			_ = s.db.UpdateEmailSourceInitialBackfillStatus(userID, source.ID, database.BackfillStatusSkipped)
			return
		}

		gmailSource := &gmail.EmailSource{
			ID:         source.ID,
			Type:       gmail.EmailSourceType(source.Type),
			Identifier: source.Identifier,
			Name:       source.Name,
			Enabled:    source.Enabled,
			CreatedAt:  source.CreatedAt,
			UpdatedAt:  source.UpdatedAt,
		}

		since := time.Now().Add(-backfillWindowDays * 24 * time.Hour)
		if _, err := worker.BackfillSource(context.Background(), gmailSource, since, backfillMaxEmails); err != nil {
			fmt.Printf("Backfill: failed to backfill email source %d: %v\n", source.ID, err)
			_ = s.db.UpdateEmailSourceInitialBackfillStatus(userID, source.ID, database.BackfillStatusFailed)
			return
		}

		_ = s.db.UpdateEmailSourceInitialBackfillStatus(userID, source.ID, database.BackfillStatusCompleted)
	}()
}

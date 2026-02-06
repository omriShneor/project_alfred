package processor

import (
	"context"
	"fmt"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/source"
)

// BackfillProcessor processes historical messages without storing duplicates.
type BackfillProcessor struct {
	db               *database.DB
	eventAnalyzer    agent.EventAnalyzer
	reminderAnalyzer agent.ReminderAnalyzer
	notifyService    *notify.Service
	eventCreator     *EventCreator
	reminderCreator  *ReminderCreator
}

// NewBackfillProcessor creates a new backfill processor.
func NewBackfillProcessor(db *database.DB, eventAnalyzer agent.EventAnalyzer, reminderAnalyzer agent.ReminderAnalyzer, notifyService *notify.Service) *BackfillProcessor {
	return &BackfillProcessor{
		db:               db,
		eventAnalyzer:    eventAnalyzer,
		reminderAnalyzer: reminderAnalyzer,
		notifyService:    notifyService,
		eventCreator:     NewEventCreator(db, notifyService),
		reminderCreator:  NewReminderCreator(db, notifyService),
	}
}

// ProcessChannelMessages analyzes existing messages from history for events and reminders.
func (p *BackfillProcessor) ProcessChannelMessages(ctx context.Context, userID int64, channelID int64, sourceType source.SourceType, messages []database.SourceMessage) error {
	if len(messages) == 0 {
		return nil
	}

	channel, err := p.db.GetSourceChannelByID(userID, channelID)
	if err != nil {
		return fmt.Errorf("failed to get channel: %w", err)
	}
	if channel == nil {
		return fmt.Errorf("channel not found: %d", channelID)
	}
	if !channel.Enabled {
		return nil
	}

	for i, msg := range messages {
		// Limit history window to the last defaultHistorySize messages.
		historyStart := 0
		if i+1 > defaultHistorySize {
			historyStart = i + 1 - defaultHistorySize
		}
		historySlice := messages[historyStart : i+1]

		historyRecords := convertToMessageRecords(historySlice)
		newRecord := convertSourceMessageToRecord(&msg)

		existingEvents, err := p.db.GetActiveEventsForChannel(userID, channelID)
		if err != nil {
			fmt.Printf("Backfill: warning - failed to get existing events: %v\n", err)
			existingEvents = []database.CalendarEvent{}
		}

		existingReminders, err := p.db.GetActiveRemindersForChannel(channelID)
		if err != nil {
			fmt.Printf("Backfill: warning - failed to get existing reminders: %v\n", err)
			existingReminders = []database.Reminder{}
		}

		if p.eventAnalyzer != nil && p.eventAnalyzer.IsConfigured() {
			analysis, err := p.eventAnalyzer.AnalyzeMessages(ctx, historyRecords, newRecord, existingEvents)
			if err != nil {
				fmt.Printf("Backfill event analysis error: %v\n", err)
			} else if analysis.HasEvent && analysis.Action != "none" {
				if err := p.createPendingEvent(channel, msg.ID, analysis, sourceType); err != nil {
					fmt.Printf("Backfill: failed to create pending event: %v\n", err)
				}
			}
		}

		if p.reminderAnalyzer != nil && p.reminderAnalyzer.IsConfigured() {
			analysis, err := p.reminderAnalyzer.AnalyzeMessages(ctx, historyRecords, newRecord, existingReminders)
			if err != nil {
				fmt.Printf("Backfill reminder analysis error: %v\n", err)
			} else if analysis.HasReminder && analysis.Action != "none" {
				if err := p.createPendingReminder(channel, msg.ID, analysis, sourceType); err != nil {
					fmt.Printf("Backfill: failed to create pending reminder: %v\n", err)
				}
			}
		}
	}

	return nil
}

func (p *BackfillProcessor) createPendingEvent(
	channel *database.SourceChannel,
	messageID int64,
	analysis *agent.EventAnalysis,
	sourceType source.SourceType,
) error {
	params := EventCreationParams{
		UserID:     channel.UserID,
		ChannelID:  channel.ID,
		SourceType: sourceType,
		MessageID:  &messageID,
		Analysis:   analysis,
	}

	if analysis.Event != nil && analysis.Event.AlfredEventRef != 0 {
		existing, err := p.db.GetEventByID(analysis.Event.AlfredEventRef)
		if err == nil && existing.Status == database.EventStatusPending {
			params.ExistingEvent = existing
		}
	}

	_, err := p.eventCreator.CreateEventFromAnalysis(ctxOrBackground(), params)
	return err
}

func (p *BackfillProcessor) createPendingReminder(
	channel *database.SourceChannel,
	messageID int64,
	analysis *agent.ReminderAnalysis,
	sourceType source.SourceType,
) error {
	params := ReminderCreationParams{
		UserID:     channel.UserID,
		ChannelID:  channel.ID,
		SourceType: sourceType,
		MessageID:  &messageID,
		Analysis:   analysis,
	}

	_, err := p.reminderCreator.CreateReminderFromAnalysis(ctxOrBackground(), params)
	return err
}

func ctxOrBackground() context.Context {
	return context.Background()
}

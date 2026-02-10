package processor

import (
	"context"
	"fmt"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/agent/intents"
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
	intentRegistry   *intents.Registry
	intentRouter     intents.Router
}

// NewBackfillProcessor creates a new backfill processor.
func NewBackfillProcessor(db *database.DB, eventAnalyzer agent.EventAnalyzer, reminderAnalyzer agent.ReminderAnalyzer, notifyService *notify.Service) *BackfillProcessor {
	registry := intents.NewRegistry()
	if eventAnalyzer != nil && eventAnalyzer.IsConfigured() {
		_ = registry.Register(&intents.EventModule{Analyzer: eventAnalyzer})
	}
	if reminderAnalyzer != nil && reminderAnalyzer.IsConfigured() {
		_ = registry.Register(&intents.ReminderModule{Analyzer: reminderAnalyzer})
	}

	return &BackfillProcessor{
		db:               db,
		eventAnalyzer:    eventAnalyzer,
		reminderAnalyzer: reminderAnalyzer,
		notifyService:    notifyService,
		eventCreator:     NewEventCreator(db, notifyService),
		reminderCreator:  NewReminderCreator(db, notifyService),
		intentRegistry:   registry,
		intentRouter:     intents.NewKeywordRouter(),
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

		if err := p.routeAnalyzeAndPersistBackfill(
			ctx,
			channel,
			msg.ID,
			sourceType,
			intents.MessageInput{
				History:           historyRecords,
				NewMessage:        newRecord,
				ExistingEvents:    existingEvents,
				ExistingReminders: existingReminders,
			},
		); err != nil {
			fmt.Printf("Backfill intent orchestration error: %v\n", err)
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

type backfillIntentPersister struct {
	p         *BackfillProcessor
	channel   *database.SourceChannel
	messageID int64
	source    source.SourceType
}

func (bp *backfillIntentPersister) PersistEvent(ctx context.Context, analysis *agent.EventAnalysis) error {
	return bp.p.createPendingEvent(bp.channel, bp.messageID, analysis, bp.source)
}

func (bp *backfillIntentPersister) PersistReminder(ctx context.Context, analysis *agent.ReminderAnalysis) error {
	return bp.p.createPendingReminder(bp.channel, bp.messageID, analysis, bp.source)
}

func (p *BackfillProcessor) routeAnalyzeAndPersistBackfill(
	ctx context.Context,
	channel *database.SourceChannel,
	messageID int64,
	sourceType source.SourceType,
	input intents.MessageInput,
) error {
	if p.intentRegistry == nil || p.intentRouter == nil {
		return nil
	}

	route := p.intentRouter.RouteMessages(ctx, input)
	intentOrder, unknownRoutedIntent := resolveIntentExecutionOrder(p.intentRegistry, route)
	if len(intentOrder) == 0 {
		return nil
	}

	var firstErr error
	for _, intentName := range intentOrder {
		if err := p.runBackfillIntentModule(ctx, intentName, channel, messageID, sourceType, input); err != nil {
			fmt.Printf("Backfill intent module %s error: %v\n", intentName, err)
			if firstErr == nil {
				firstErr = err
			}
		}
		if unknownRoutedIntent {
			break
		}
	}
	return firstErr
}

func (p *BackfillProcessor) runBackfillIntentModule(
	ctx context.Context,
	intentName string,
	channel *database.SourceChannel,
	messageID int64,
	sourceType source.SourceType,
	input intents.MessageInput,
) error {
	module, ok := p.intentRegistry.Get(intentName)
	if !ok {
		fmt.Printf("Unknown backfill intent '%s' -> no_action\n", intentName)
		msgID := messageID
		_ = p.db.CreateAnalysisTrace(database.AnalysisTrace{
			UserID:           channel.UserID,
			ChannelID:        channel.ID,
			SourceType:       string(sourceType),
			TriggerMessageID: &msgID,
			Intent:           intentName,
			Status:           "unknown_intent",
		})
		return nil
	}

	output, err := module.AnalyzeMessages(ctx, input)
	if err != nil {
		return err
	}
	if output == nil {
		return nil
	}
	if err := module.Validate(ctx, output); err != nil {
		fmt.Printf("Backfill intent validation failed intent=%s: %v\n", intentName, err)
		msgID := messageID
		_ = p.db.CreateAnalysisTrace(database.AnalysisTrace{
			UserID:           channel.UserID,
			ChannelID:        channel.ID,
			SourceType:       string(sourceType),
			TriggerMessageID: &msgID,
			Intent:           intentName,
			Action:           output.Action,
			Confidence:       output.Confidence,
			Reasoning:        output.Reasoning,
			Status:           "validation_failed",
		})
		return nil
	}
	if output.Confidence < minPersistConfidence {
		fmt.Printf("Backfill: skipping low-confidence intent=%s confidence=%.2f\n", intentName, output.Confidence)
		msgID := messageID
		_ = p.db.CreateAnalysisTrace(database.AnalysisTrace{
			UserID:           channel.UserID,
			ChannelID:        channel.ID,
			SourceType:       string(sourceType),
			TriggerMessageID: &msgID,
			Intent:           intentName,
			Action:           output.Action,
			Confidence:       output.Confidence,
			Reasoning:        output.Reasoning,
			Status:           "skipped_low_confidence",
		})
		return nil
	}
	err = module.Persist(ctx, output, &backfillIntentPersister{
		p:         p,
		channel:   channel,
		messageID: messageID,
		source:    sourceType,
	})
	status := "persisted"
	if err != nil {
		status = "persist_error"
	}
	msgID := messageID
	_ = p.db.CreateAnalysisTrace(database.AnalysisTrace{
		UserID:           channel.UserID,
		ChannelID:        channel.ID,
		SourceType:       string(sourceType),
		TriggerMessageID: &msgID,
		Intent:           intentName,
		Action:           output.Action,
		Confidence:       output.Confidence,
		Reasoning:        output.Reasoning,
		Status:           status,
	})
	return err
}

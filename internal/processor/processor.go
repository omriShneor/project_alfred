package processor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/agent/intents"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/source"
)

const (
	defaultHistorySize   = 25
	defaultWorkerCount   = 2
	minPersistConfidence = 0.30
)

// Processor handles incoming messages from any source and detects calendar events and reminders
type Processor struct {
	db               *database.DB
	eventAnalyzer    agent.EventAnalyzer
	reminderAnalyzer agent.ReminderAnalyzer
	intentRegistry   *intents.Registry
	intentRouter     intents.Router
	msgChan          <-chan source.Message
	historySize      int
	notifyService    *notify.Service
	eventCreator     *EventCreator
	reminderCreator  *ReminderCreator
	workerCount      int

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	unknownIntentCount atomic.Uint64
}

// New creates a new event processor
func New(
	db *database.DB,
	eventAnalyzer agent.EventAnalyzer,
	reminderAnalyzer agent.ReminderAnalyzer,
	msgChan <-chan source.Message,
	historySize int,
	notifyService *notify.Service,
) *Processor {
	if historySize <= 0 {
		historySize = defaultHistorySize
	}

	ctx, cancel := context.WithCancel(context.Background())
	registry := intents.NewRegistry()
	if eventAnalyzer != nil && eventAnalyzer.IsConfigured() {
		_ = registry.Register(&intents.EventModule{Analyzer: eventAnalyzer})
	}
	if reminderAnalyzer != nil && reminderAnalyzer.IsConfigured() {
		_ = registry.Register(&intents.ReminderModule{Analyzer: reminderAnalyzer})
	}

	return &Processor{
		db:               db,
		eventAnalyzer:    eventAnalyzer,
		reminderAnalyzer: reminderAnalyzer,
		intentRegistry:   registry,
		intentRouter:     intents.NewKeywordRouter(),
		msgChan:          msgChan,
		historySize:      historySize,
		notifyService:    notifyService,
		eventCreator:     NewEventCreator(db, notifyService),
		reminderCreator:  NewReminderCreator(db, notifyService),
		workerCount:      defaultWorkerCount,
		ctx:              ctx,
		cancel:           cancel,
	}
}

// Start begins processing messages from the channel
func (p *Processor) Start() error {
	fmt.Println("Event processor started")

	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.processLoop()
	}

	return nil
}

// Stop gracefully shuts down the processor
func (p *Processor) Stop() {
	fmt.Println("Stopping event processor...")
	p.cancel()
	p.wg.Wait()
	fmt.Println("Event processor stopped")
}

// processLoop continuously reads messages from the channel and processes them
func (p *Processor) processLoop() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case msg, ok := <-p.msgChan:
			if !ok {
				fmt.Println("Event processor: message channel closed")
				return
			}
			if err := p.processMessage(msg); err != nil {
				fmt.Printf("Event processor: error processing message: %v\n", err)
			}
		}
	}
}

// processMessage handles a single incoming message from any source
func (p *Processor) processMessage(msg source.Message) error {
	fmt.Printf("Processing %s message from channel %d: %s\n", msg.SourceType, msg.SourceID, truncate(msg.Text, 50))

	// Get the channel to find its calendar_id
	channel, err := p.db.GetSourceChannelByID(msg.UserID, msg.SourceID)
	if err != nil {
		return fmt.Errorf("failed to get channel: %w", err)
	}
	if channel == nil {
		return fmt.Errorf("channel not found: %d", msg.SourceID)
	}

	if !channel.Enabled {
		// Channel is disabled, skip processing
		return nil
	}

	// Store the new message in history
	storedMsg, err := p.storeSourceMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to store message: %w", err)
	}

	// Prune old messages to keep only the last N
	if err := p.db.PruneSourceMessages(msg.UserID, msg.SourceType, msg.SourceID, p.historySize); err != nil {
		fmt.Printf("Warning: failed to prune messages: %v\n", err)
	}

	// Get message history for context (shared between analyzers)
	history, err := p.db.GetSourceMessageHistory(msg.UserID, msg.SourceType, msg.SourceID, p.historySize)
	if err != nil {
		return fmt.Errorf("failed to get message history: %w", err)
	}

	// Get existing active events (pending + synced) for this channel
	existingEvents, err := p.db.GetActiveEventsForChannel(msg.UserID, msg.SourceID)
	if err != nil {
		fmt.Printf("Warning: failed to get existing events: %v\n", err)
		existingEvents = []database.CalendarEvent{}
	}

	// Get existing active reminders for this channel
	existingReminders, err := p.db.GetActiveRemindersForChannel(msg.SourceID)
	if err != nil {
		fmt.Printf("Warning: failed to get existing reminders: %v\n", err)
		existingReminders = []database.Reminder{}
	}

	// Convert to database types for analysis (shared context)
	historyRecords := convertToMessageRecords(history)
	newMessageRecord := convertSourceMessageToRecord(storedMsg)
	if err := p.routeAnalyzeAndPersistMessage(
		channel,
		msg.SourceType,
		storedMsg.ID,
		intents.MessageInput{
			History:           historyRecords,
			NewMessage:        newMessageRecord,
			ExistingEvents:    existingEvents,
			ExistingReminders: existingReminders,
		},
	); err != nil {
		fmt.Printf("Intent orchestration error: %v\n", err)
	}

	return nil
}

// convertToMessageRecords converts SourceMessage slice to MessageRecord slice for Claude
func convertToMessageRecords(messages []database.SourceMessage) []database.MessageRecord {
	records := make([]database.MessageRecord, len(messages))
	for i, m := range messages {
		records[i] = database.MessageRecord{
			ID:          m.ID,
			SourceType:  m.SourceType,
			ChannelID:   m.ChannelID,
			SenderJID:   m.SenderID,
			SenderName:  m.SenderName,
			MessageText: m.MessageText,
			Subject:     m.Subject,
			Timestamp:   m.Timestamp,
			CreatedAt:   m.CreatedAt,
		}
	}
	return records
}

// convertSourceMessageToRecord converts a single SourceMessage to MessageRecord for Claude
func convertSourceMessageToRecord(m *database.SourceMessage) database.MessageRecord {
	return database.MessageRecord{
		ID:          m.ID,
		SourceType:  m.SourceType,
		ChannelID:   m.ChannelID,
		SenderJID:   m.SenderID,
		SenderName:  m.SenderName,
		MessageText: m.MessageText,
		Subject:     m.Subject,
		Timestamp:   m.Timestamp,
		CreatedAt:   m.CreatedAt,
	}
}

type messageIntentPersister struct {
	p         *Processor
	channel   *database.SourceChannel
	source    source.SourceType
	messageID int64
}

func (mp *messageIntentPersister) PersistEvent(ctx context.Context, analysis *agent.EventAnalysis) error {
	return mp.p.createPendingEvent(mp.channel, mp.messageID, analysis, mp.source)
}

func (mp *messageIntentPersister) PersistReminder(ctx context.Context, analysis *agent.ReminderAnalysis) error {
	return mp.p.createPendingReminder(mp.channel, mp.messageID, analysis, mp.source)
}

func (p *Processor) routeAnalyzeAndPersistMessage(
	channel *database.SourceChannel,
	sourceType source.SourceType,
	messageID int64,
	input intents.MessageInput,
) error {
	if p.intentRegistry == nil || p.intentRouter == nil {
		return nil
	}

	route := p.intentRouter.RouteMessages(p.ctx, input)
	fmt.Printf("Intent route: intent=%s confidence=%.2f reason=%s\n", route.Intent, route.Confidence, truncate(route.Reasoning, 80))
	msgID := messageID
	_ = p.db.CreateAnalysisTrace(database.AnalysisTrace{
		UserID:           channel.UserID,
		ChannelID:        channel.ID,
		SourceType:       string(sourceType),
		TriggerMessageID: &msgID,
		Intent:           route.Intent,
		RouterConfidence: route.Confidence,
		Action:           "",
		Confidence:       route.Confidence,
		Reasoning:        route.Reasoning,
		Status:           "routed",
	})

	intentOrder, unknownRoutedIntent := resolveIntentExecutionOrder(p.intentRegistry, route)
	if len(intentOrder) == 0 {
		return nil
	}

	var firstErr error
	for _, intentName := range intentOrder {
		if err := p.runMessageIntentModule(intentName, channel, sourceType, messageID, input); err != nil {
			fmt.Printf("Intent module %s error: %v\n", intentName, err)
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

func (p *Processor) runMessageIntentModule(
	intentName string,
	channel *database.SourceChannel,
	sourceType source.SourceType,
	messageID int64,
	input intents.MessageInput,
) error {
	module, ok := p.intentRegistry.Get(intentName)
	if !ok {
		count := p.unknownIntentCount.Add(1)
		fmt.Printf("Unknown intent '%s' -> no_action (count=%d)\n", intentName, count)
		msgID := messageID
		_ = p.db.CreateAnalysisTrace(database.AnalysisTrace{
			UserID:           channel.UserID,
			ChannelID:        channel.ID,
			SourceType:       string(sourceType),
			TriggerMessageID: &msgID,
			Intent:           intentName,
			Status:           "unknown_intent",
			Reasoning:        "intent module not registered",
		})
		return nil
	}

	output, err := module.AnalyzeMessages(p.ctx, input)
	if err != nil {
		return err
	}
	if output == nil {
		return nil
	}

	if err := module.Validate(p.ctx, output); err != nil {
		fmt.Printf("Intent validation failed intent=%s: %v\n", intentName, err)
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
			Details: map[string]any{
				"error": err.Error(),
			},
		})
		return nil
	}
	if output.Confidence < minPersistConfidence {
		fmt.Printf("Skipping low-confidence intent=%s confidence=%.2f\n", intentName, output.Confidence)
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

	persister := &messageIntentPersister{
		p:         p,
		channel:   channel,
		source:    sourceType,
		messageID: messageID,
	}
	err = module.Persist(p.ctx, output, persister)
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

// createPendingEvent creates or updates a pending event from the analysis
func (p *Processor) createPendingEvent(
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

	// Check if we should update an existing pending event
	if analysis.Event != nil && analysis.Event.AlfredEventRef != 0 {
		existing, err := p.db.GetEventByID(analysis.Event.AlfredEventRef)
		if err == nil && existing.Status == database.EventStatusPending {
			params.ExistingEvent = existing
		}
	}

	_, err := p.eventCreator.CreateEventFromAnalysis(p.ctx, params)
	return err
}

// createPendingReminder creates or updates a pending reminder from the analysis
func (p *Processor) createPendingReminder(
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

	_, err := p.reminderCreator.CreateReminderFromAnalysis(p.ctx, params)
	return err
}

// truncate shortens a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

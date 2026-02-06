package processor

import (
	"context"
	"fmt"
	"sync"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/source"
)

const (
	defaultHistorySize = 25
)

// Processor handles incoming messages from any source and detects calendar events and reminders
type Processor struct {
	db               *database.DB
	eventAnalyzer    agent.EventAnalyzer    // Event analyzer
	reminderAnalyzer agent.ReminderAnalyzer // Reminder analyzer
	msgChan          <-chan source.Message
	historySize      int
	notifyService    *notify.Service
	eventCreator     *EventCreator
	reminderCreator  *ReminderCreator

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
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

	return &Processor{
		db:               db,
		eventAnalyzer:    eventAnalyzer,
		reminderAnalyzer: reminderAnalyzer,
		msgChan:          msgChan,
		historySize:      historySize,
		notifyService:    notifyService,
		eventCreator:     NewEventCreator(db, notifyService),
		reminderCreator:  NewReminderCreator(db, notifyService),
		ctx:              ctx,
		cancel:           cancel,
	}
}

// Start begins processing messages from the channel
func (p *Processor) Start() error {
	if p.eventAnalyzer == nil || !p.eventAnalyzer.IsConfigured() {
		fmt.Println("Event processor: EventAnalyzer not configured, processor disabled")
		return nil
	}

	fmt.Println("Event processor started")

	p.wg.Add(1)
	go p.processLoop()

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

	// Run event analyzer (independent goroutine - fire and forget)
	if p.eventAnalyzer != nil && p.eventAnalyzer.IsConfigured() {
		go func() {
			analysis, err := p.eventAnalyzer.AnalyzeMessages(p.ctx, historyRecords, newMessageRecord, existingEvents)
			if err != nil {
				fmt.Printf("Event analysis error: %v\n", err)
				return
			}

			fmt.Printf("Event analysis: action=%s, has_event=%v, confidence=%.2f\n",
				analysis.Action, analysis.HasEvent, analysis.Confidence)

			if analysis.HasEvent && analysis.Action != "none" {
				if err := p.createPendingEvent(channel, storedMsg.ID, analysis, msg.SourceType); err != nil {
					fmt.Printf("Failed to create pending event: %v\n", err)
				}
			}
		}()
	}

	// Run reminder analyzer (independent goroutine - fire and forget)
	if p.reminderAnalyzer != nil && p.reminderAnalyzer.IsConfigured() {
		go func() {
			analysis, err := p.reminderAnalyzer.AnalyzeMessages(p.ctx, historyRecords, newMessageRecord, existingReminders)
			if err != nil {
				fmt.Printf("Reminder analysis error: %v\n", err)
				return
			}

			fmt.Printf("Reminder analysis: action=%s, has_reminder=%v, confidence=%.2f\n",
				analysis.Action, analysis.HasReminder, analysis.Confidence)

			if analysis.HasReminder && analysis.Action != "none" {
				if err := p.createPendingReminder(channel, storedMsg.ID, analysis, msg.SourceType); err != nil {
					fmt.Printf("Failed to create pending reminder: %v\n", err)
				}
			}
		}()
	}

	return nil
}

// convertToMessageRecords converts SourceMessage slice to MessageRecord slice for Claude
func convertToMessageRecords(messages []database.SourceMessage) []database.MessageRecord {
	records := make([]database.MessageRecord, len(messages))
	for i, m := range messages {
		records[i] = database.MessageRecord{
			ID:          m.ID,
			ChannelID:   m.ChannelID,
			SenderJID:   m.SenderID,
			SenderName:  m.SenderName,
			MessageText: m.MessageText,
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
		ChannelID:   m.ChannelID,
		SenderJID:   m.SenderID,
		SenderName:  m.SenderName,
		MessageText: m.MessageText,
		Timestamp:   m.Timestamp,
		CreatedAt:   m.CreatedAt,
	}
}

// createPendingEvent creates or updates a pending event from the analysis
func (p *Processor) createPendingEvent(
	channel *database.SourceChannel,
	messageID int64,
	analysis *agent.EventAnalysis,
	sourceType source.SourceType,
) error {
	params := EventCreationParams{
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

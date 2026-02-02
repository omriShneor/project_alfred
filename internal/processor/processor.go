package processor

import (
	"context"
	"fmt"
	"sync"

	"github.com/omriShneor/project_alfred/internal/claude"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/gcal"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/source"
)

const (
	defaultHistorySize = 25
)

// Processor handles incoming messages from any source and detects calendar events
type Processor struct {
	db            *database.DB
	gcalClient    *gcal.Client
	claudeClient  *claude.Client
	msgChan       <-chan source.Message
	historySize   int
	notifyService *notify.Service
	eventCreator  *EventCreator

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates a new event processor
func New(
	db *database.DB,
	gcalClient *gcal.Client,
	claudeClient *claude.Client,
	msgChan <-chan source.Message,
	historySize int,
	notifyService *notify.Service,
) *Processor {
	if historySize <= 0 {
		historySize = defaultHistorySize
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Processor{
		db:            db,
		gcalClient:    gcalClient,
		claudeClient:  claudeClient,
		msgChan:       msgChan,
		historySize:   historySize,
		notifyService: notifyService,
		eventCreator:  NewEventCreator(db, notifyService),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start begins processing messages from the channel
func (p *Processor) Start() error {
	if p.claudeClient == nil || !p.claudeClient.IsConfigured() {
		fmt.Println("Event processor: Claude API not configured, processor disabled")
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
	channel, err := p.db.GetSourceChannelByID(msg.SourceID)
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
	if err := p.db.PruneSourceMessages(msg.SourceType, msg.SourceID, p.historySize); err != nil {
		fmt.Printf("Warning: failed to prune messages: %v\n", err)
	}

	// Get message history for context
	history, err := p.db.GetSourceMessageHistory(msg.SourceType, msg.SourceID, p.historySize)
	if err != nil {
		return fmt.Errorf("failed to get message history: %w", err)
	}

	// Get existing active events (pending + synced) for this channel
	existingEvents, err := p.db.GetActiveEventsForChannel(msg.SourceID)
	if err != nil {
		fmt.Printf("Warning: failed to get existing events: %v\n", err)
		existingEvents = []database.CalendarEvent{}
	}

	// Convert to database types for Claude analysis
	historyRecords := convertToMessageRecords(history)
	newMessageRecord := convertSourceMessageToRecord(storedMsg)

	// Send to Claude for analysis
	analysis, err := p.claudeClient.AnalyzeMessages(p.ctx, historyRecords, newMessageRecord, existingEvents)
	if err != nil {
		return fmt.Errorf("Claude analysis failed: %w", err)
	}

	fmt.Printf("Claude analysis: action=%s, has_event=%v, confidence=%.2f\n",
		analysis.Action, analysis.HasEvent, analysis.Confidence)

	// If no event detected or action is "none", skip
	if !analysis.HasEvent || analysis.Action == "none" {
		return nil
	}

	// Create pending event in database
	if err := p.createPendingEvent(channel, storedMsg.ID, analysis, msg.SourceType); err != nil {
		return fmt.Errorf("failed to create pending event: %w", err)
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

// createPendingEvent creates or updates a pending event from Claude's analysis
func (p *Processor) createPendingEvent(
	channel *database.SourceChannel,
	messageID int64,
	analysis *claude.EventAnalysis,
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

// truncate shortens a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

package processor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/omriShneor/project_alfred/internal/claude"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/gcal"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/whatsapp"
)

const (
	defaultHistorySize = 25
)

// Processor handles incoming WhatsApp messages and detects calendar events
type Processor struct {
	db            *database.DB
	gcalClient    *gcal.Client
	claudeClient  *claude.Client
	msgChan       <-chan whatsapp.FilteredMessage
	historySize   int
	notifyService *notify.Service

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates a new event processor
func New(
	db *database.DB,
	gcalClient *gcal.Client,
	claudeClient *claude.Client,
	msgChan <-chan whatsapp.FilteredMessage,
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

// processMessage handles a single incoming message
func (p *Processor) processMessage(msg whatsapp.FilteredMessage) error {
	fmt.Printf("Processing message from channel %d: %s\n", msg.SourceID, truncate(msg.Text, 50))

	// Get the channel to find its calendar_id
	channel, err := p.db.GetChannelByID(msg.SourceID)
	if err != nil {
		return fmt.Errorf("failed to get channel: %w", err)
	}

	if !channel.Enabled {
		// Channel is disabled, skip processing
		return nil
	}

	// Store the new message in history
	storedMsg, err := p.storeMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to store message: %w", err)
	}

	// Prune old messages to keep only the last N
	if err := p.db.PruneMessages(msg.SourceID, p.historySize); err != nil {
		fmt.Printf("Warning: failed to prune messages: %v\n", err)
	}

	// Get message history for context
	history, err := p.db.GetMessageHistory(msg.SourceID, p.historySize)
	if err != nil {
		return fmt.Errorf("failed to get message history: %w", err)
	}

	// Get existing synced events for this channel
	existingEvents, err := p.db.GetExistingEventsForChannel(msg.SourceID)
	if err != nil {
		fmt.Printf("Warning: failed to get existing events: %v\n", err)
		existingEvents = []database.CalendarEvent{}
	}

	// Send to Claude for analysis
	analysis, err := p.claudeClient.AnalyzeMessages(p.ctx, history, *storedMsg, existingEvents)
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
	if err := p.createPendingEvent(channel, storedMsg.ID, analysis); err != nil {
		return fmt.Errorf("failed to create pending event: %w", err)
	}

	return nil
}

// createPendingEvent creates a pending event from Claude's analysis
func (p *Processor) createPendingEvent(
	channel *database.Channel,
	messageID int64,
	analysis *claude.EventAnalysis,
) error {
	if analysis.Event == nil {
		return fmt.Errorf("analysis has no event data")
	}

	// Parse start time
	startTime, err := time.Parse(time.RFC3339, analysis.Event.StartTime)
	if err != nil {
		// Try alternative format
		startTime, err = time.Parse("2006-01-02T15:04:05", analysis.Event.StartTime)
		if err != nil {
			return fmt.Errorf("failed to parse start time: %w", err)
		}
	}

	// Parse end time if provided
	var endTime *time.Time
	if analysis.Event.EndTime != "" {
		et, err := time.Parse(time.RFC3339, analysis.Event.EndTime)
		if err != nil {
			et, err = time.Parse("2006-01-02T15:04:05", analysis.Event.EndTime)
			if err == nil {
				endTime = &et
			}
		} else {
			endTime = &et
		}
	}

	// If no end time, default to 1 hour after start
	if endTime == nil {
		et := startTime.Add(time.Hour)
		endTime = &et
	}

	// Determine action type
	var actionType database.EventActionType
	switch analysis.Action {
	case "create":
		actionType = database.EventActionCreate
	case "update":
		actionType = database.EventActionUpdate
	case "delete":
		actionType = database.EventActionDelete
	default:
		return fmt.Errorf("unknown action type: %s", analysis.Action)
	}

	// For updates/deletes, store the reference to the existing event
	var googleEventID *string
	if analysis.Event.UpdateRef != "" {
		googleEventID = &analysis.Event.UpdateRef
	}

	event := &database.CalendarEvent{
		ChannelID:     channel.ID,
		GoogleEventID: googleEventID,
		CalendarID:    channel.CalendarID,
		Title:         analysis.Event.Title,
		Description:   analysis.Event.Description,
		StartTime:     startTime,
		EndTime:       endTime,
		Location:      analysis.Event.Location,
		ActionType:    actionType,
		OriginalMsgID: &messageID,
		LLMReasoning:  analysis.Reasoning,
	}

	created, err := p.db.CreatePendingEvent(event)
	if err != nil {
		return fmt.Errorf("failed to save event: %w", err)
	}

	fmt.Printf("Created pending event: %s (ID: %d, Action: %s)\n",
		created.Title, created.ID, created.ActionType)

	// Send notification (non-blocking, don't fail event creation)
	if p.notifyService != nil {
		go p.notifyService.NotifyPendingEvent(context.Background(), created)
	}

	return nil
}

// truncate shortens a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

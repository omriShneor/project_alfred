package processor

import (
	"context"
	"fmt"
	"time"

	"github.com/omriShneor/project_alfred/internal/claude"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/gmail"
	"github.com/omriShneor/project_alfred/internal/notify"
)

// EmailProcessor processes emails for calendar event detection
type EmailProcessor struct {
	db            *database.DB
	claudeClient  *claude.Client
	notifyService *notify.Service
}

// NewEmailProcessor creates a new email processor
func NewEmailProcessor(db *database.DB, claudeClient *claude.Client, notifyService *notify.Service) *EmailProcessor {
	return &EmailProcessor{
		db:            db,
		claudeClient:  claudeClient,
		notifyService: notifyService,
	}
}

// ProcessEmail processes a single email for event detection
func (p *EmailProcessor) ProcessEmail(ctx context.Context, email *gmail.Email, source *gmail.EmailSource) error {
	if p.claudeClient == nil || !p.claudeClient.IsConfigured() {
		return fmt.Errorf("Claude API not configured")
	}

	fmt.Printf("Processing email: %s (from: %s)\n", truncate(email.Subject, 50), email.From)

	// Clean the email body
	body := gmail.CleanEmailBody(email.Body)

	// Send to Claude for analysis
	analysis, err := p.claudeClient.AnalyzeEmail(ctx, claude.EmailContent{
		Subject: email.Subject,
		From:    email.From,
		To:      email.To,
		Date:    email.Date,
		Body:    body,
	})
	if err != nil {
		return fmt.Errorf("Claude analysis failed: %w", err)
	}

	fmt.Printf("Claude email analysis: action=%s, has_event=%v, confidence=%.2f\n",
		analysis.Action, analysis.HasEvent, analysis.Confidence)

	// If no event detected or action is "none", skip
	if !analysis.HasEvent || analysis.Action == "none" {
		return nil
	}

	// Create pending event in database
	if err := p.createPendingEventFromEmail(source, analysis); err != nil {
		return fmt.Errorf("failed to create pending event: %w", err)
	}

	return nil
}

// createPendingEventFromEmail creates a pending event from Claude's email analysis
func (p *EmailProcessor) createPendingEventFromEmail(source *gmail.EmailSource, analysis *claude.EventAnalysis) error {
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

	// Determine calendar ID from source
	calendarID := "primary"
	if source != nil && source.CalendarID != "" {
		calendarID = source.CalendarID
	}

	// Get or create a placeholder channel for email sources
	// We use a special channel for email-sourced events
	emailChannel, err := p.getOrCreateEmailChannel(source)
	if err != nil {
		return fmt.Errorf("failed to get email channel: %w", err)
	}

	event := &database.CalendarEvent{
		ChannelID:    emailChannel.ID,
		CalendarID:   calendarID,
		Title:        analysis.Event.Title,
		Description:  analysis.Event.Description,
		StartTime:    startTime,
		EndTime:      endTime,
		Location:     analysis.Event.Location,
		ActionType:   database.EventActionCreate,
		LLMReasoning: analysis.Reasoning,
	}

	created, err := p.db.CreatePendingEvent(event)
	if err != nil {
		return fmt.Errorf("failed to save event: %w", err)
	}

	// Update source to gmail
	p.db.Exec(`UPDATE calendar_events SET source = 'gmail', email_source_id = ? WHERE id = ?`, source.ID, created.ID)

	fmt.Printf("Created pending event from email: %s (ID: %d)\n", created.Title, created.ID)

	// Send notification (non-blocking, don't fail event creation)
	if p.notifyService != nil {
		go p.notifyService.NotifyPendingEvent(context.Background(), created)
	}

	return nil
}

// getOrCreateEmailChannel gets or creates a placeholder channel for email-sourced events
// This is needed because calendar_events has a foreign key to channels
func (p *EmailProcessor) getOrCreateEmailChannel(source *gmail.EmailSource) (*database.Channel, error) {
	// Use a channel identifier based on the email source
	identifier := fmt.Sprintf("email:%s:%s", source.Type, source.Identifier)

	// Check if channel exists
	channel, err := p.db.GetChannelByIdentifier(identifier)
	if err != nil {
		return nil, err
	}
	if channel != nil {
		return channel, nil
	}

	// Create new channel for this email source
	return p.db.CreateChannel(database.ChannelTypeSender, identifier, fmt.Sprintf("Email: %s", source.Name))
}

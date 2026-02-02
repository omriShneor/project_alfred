package processor

import (
	"context"
	"fmt"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/gmail"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/source"
)

// EmailProcessor processes emails for calendar event detection
type EmailProcessor struct {
	db            *database.DB
	analyzer      agent.Analyzer
	notifyService *notify.Service
	eventCreator  *EventCreator
}

// NewEmailProcessor creates a new email processor
func NewEmailProcessor(db *database.DB, analyzer agent.Analyzer, notifyService *notify.Service) *EmailProcessor {
	return &EmailProcessor{
		db:            db,
		analyzer:      analyzer,
		notifyService: notifyService,
		eventCreator:  NewEventCreator(db, notifyService),
	}
}

// ProcessEmail processes a single email for event detection
func (p *EmailProcessor) ProcessEmail(ctx context.Context, email *gmail.Email, source *gmail.EmailSource, thread *gmail.Thread) error {
	if p.analyzer == nil || !p.analyzer.IsConfigured() {
		return fmt.Errorf("analyzer not configured")
	}

	threadLen := 0
	if thread != nil {
		threadLen = len(thread.Messages)
	}
	fmt.Printf("Processing email: %s (from: %s, thread: %d messages)\n", truncate(email.Subject, 50), email.From, threadLen)

	// Clean the email body
	body := gmail.CleanEmailBody(email.Body)

	// Build email content with thread context
	emailContent := agent.EmailContent{
		Subject: email.Subject,
		From:    email.From,
		To:      email.To,
		Date:    email.Date,
		Body:    body,
	}

	// Add thread history (excluding the latest message which is the email being analyzed)
	if thread != nil && len(thread.Messages) > 1 {
		for _, msg := range thread.Messages[:len(thread.Messages)-1] {
			emailContent.ThreadHistory = append(emailContent.ThreadHistory, agent.EmailThreadMessage{
				From:    msg.From,
				Date:    msg.Date,
				Subject: msg.Subject,
				Body:    msg.Body,
			})
		}
	}

	// Send to analyzer for event detection
	analysis, err := p.analyzer.AnalyzeEmail(ctx, emailContent)
	if err != nil {
		return fmt.Errorf("email analysis failed: %w", err)
	}

	fmt.Printf("Email analysis: action=%s, has_event=%v, confidence=%.2f\n",
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

// createPendingEventFromEmail creates a pending event from email analysis
func (p *EmailProcessor) createPendingEventFromEmail(emailSource *gmail.EmailSource, analysis *agent.EventAnalysis) error {
	// Get or create a placeholder channel for email sources
	emailChannel, err := p.getOrCreateEmailChannel(emailSource)
	if err != nil {
		return fmt.Errorf("failed to get email channel: %w", err)
	}

	emailSourceID := emailSource.ID
	params := EventCreationParams{
		ChannelID:     emailChannel.ID,
		SourceType:    source.SourceTypeGmail,
		EmailSourceID: &emailSourceID,
		Analysis:      analysis,
	}

	_, err = p.eventCreator.CreateEventFromAnalysis(context.Background(), params)
	return err
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
